# 观看进度上报 MVP 设计

本文档描述“用户观看视频进度”的 MVP 设计方案。它是待实施设计，不代表当前仓库已经落地。

当前仓库已经存在 `catalog.video_user_states`，它是用户与视频互动状态的聚合投影表，不是观看流水表。当前没有专门的观看流水表。MVP 不建议新增高频 `play_started / progress / paused / resumed / seeked / completed` 全量事件流水，因为这会带来较高写入量，也会让 reduce 逻辑被 seek、重试、后台切换、重复 progress 上报等细节拖复杂。

推荐方案是新增一张低频观看事件表，并继续维护现有 `catalog.video_user_states`：

```text
analytics.video_watch_events  -- 每次观看 session 的低频摘要事件
catalog.video_user_states     -- 用户 x 视频的聚合状态
```

`analytics.video_watch_events` 用来保存原始观看事实，提供幂等、去重、完成次数依据和有限可追溯能力。`catalog.video_user_states` 继续作为 Recommendation 读取的轻量消费状态投影。

因为本方案还没有执行，后续落地时应直接使用 `analytics.video_watch_events`，不需要保留旧的 `catalog.video_watch_sessions` 表名或兼容迁移。

## 1. 设计目标

本方案的目标是：

1. 支持前端以低频方式上报观看进度。
2. 避免高频事件流水带来的写入和归约复杂度。
3. 让 `watch_count` 和 `completed_count` 有明确去重依据。
4. 保持 Recommendation 只读 `catalog.video_user_states`，不直接依赖观看 session 表。
5. 保持 Analytics / Catalog / Learning engine / Recommendation 边界清晰。

本方案不解决：

1. 精确逐秒观看轨迹分析。
2. 防作弊级别的真实观看时长计算。
3. 点赞、收藏、分享等低频互动事件审计。
4. Recommendation 曝光和冷却状态维护。
5. 学习事件归约。

## 2. 模块边界

观看进度有两层语义：原始观看事实和聚合消费状态。

原始观看事实属于 Analytics owner。`analytics.video_watch_events` 只记录用户观看某个视频的一次播放会话摘要。它不是逐秒播放器日志，也不记录推荐曝光。推荐曝光和冷却仍属于 `recommendation.user_video_serving_states`。

聚合消费状态属于 Catalog owner。`catalog.video_user_states` 继续作为用户与视频的聚合投影。Recommendation 当前只需要读取其中的 `last_watched_at`、`watch_count`、`completed_count`、`last_watch_ratio` 和 `max_watch_ratio` 作为轻量 penalty 输入。

`learning.unit_learning_events` 只记录学习事件，例如正式学习、复习、测验、查词或 exposure。普通观看进度不应自动写入 Learning engine。只有当产品层明确把一次观看行为解释成学习行为时，才应由上层业务另行调用 Learning engine 的学习事件接口。

## 3. 数据库设计

### 3.1 `analytics.video_watch_events`

建议新增表：

```sql
create schema if not exists analytics;

create table analytics.video_watch_events (
  watch_session_id uuid primary key,

  user_id uuid not null references auth.users(id) on delete cascade,
  video_id uuid not null references catalog.videos(video_id) on delete cascade,

  started_at timestamptz not null,
  last_seen_at timestamptz not null,
  completed_at timestamptz,

  last_position_ms integer not null default 0,
  max_position_ms integer not null default 0,
  duration_ms integer,
  max_watch_ratio numeric(6,5) not null default 0,
  is_completed boolean not null default false,

  progress_report_count integer not null default 0,
  source text not null default 'app',
  metadata jsonb not null default '{}'::jsonb,

  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),

  check (last_position_ms >= 0),
  check (max_position_ms >= 0),
  check (duration_ms is null or duration_ms > 0),
  check (max_watch_ratio >= 0 and max_watch_ratio <= 1),
  check (progress_report_count >= 0)
);
```

字段说明：

| 字段 | 含义 | 维护规则 |
| --- | --- | --- |
| `watch_session_id` | 一次播放会话 ID。由前端生成，并在一次播放器生命周期内复用。 | 主键；同一个 session 重复上报走 upsert。 |
| `user_id` | 当前登录用户。 | 从认证上下文获取，不接受前端传入。 |
| `video_id` | 被观看的视频。 | 从 URL path 获取，必须存在于 `catalog.videos`。 |
| `started_at` | 该 session 第一次上报时间。 | 首次 insert 时写入；后续不覆盖。 |
| `last_seen_at` | 该 session 最近一次有效上报时间。 | 每次 progress 上报时更新。 |
| `completed_at` | 该 session 首次完成时间。 | 第一次 `is_completed = true` 时写入；后续不覆盖。 |
| `last_position_ms` | 最近一次上报的播放位置。 | 每次上报覆盖；允许小于 `max_position_ms`，因为用户可能 seek 回退。 |
| `max_position_ms` | 该 session 到达过的最大播放位置。 | 只增不减，取历史值和本次 `position_ms` 的较大值。 |
| `duration_ms` | 前端播放器看到的视频总时长。 | 可为空；有值时必须大于 0。后续上报可更新为最新可信值。 |
| `max_watch_ratio` | 该 session 达到过的最大观看比例。 | 服务端用 `max_position_ms / duration_ms` 计算并裁剪到 `[0, 1]`。 |
| `is_completed` | 该 session 是否已完成。 | 只允许从 `false` 变为 `true`。 |
| `progress_report_count` | 该 session 累计收到的进度上报次数。 | 每次成功处理请求加 1。 |
| `source` | 客户端来源。 | 例如 `web`、`ios`、`android`；默认 `app`。 |
| `metadata` | 可选调试上下文。 | 只放非核心上下文，例如页面 surface、播放器版本。 |
| `created_at` | 行创建时间。 | 数据库默认值。 |
| `updated_at` | 行最近更新时间。 | 每次 upsert 更新。 |

推荐索引：

```sql
create index idx_video_watch_events_user_video_updated_at
on analytics.video_watch_events (user_id, video_id, updated_at desc);

create index idx_video_watch_events_user_updated_at
on analytics.video_watch_events (user_id, updated_at desc);

create index idx_video_watch_events_video_updated_at
on analytics.video_watch_events (video_id, updated_at desc);
```

索引用途：

| 索引 | 用途 |
| --- | --- |
| `(user_id, video_id, updated_at desc)` | 查询某用户对某视频最近的观看 session。 |
| `(user_id, updated_at desc)` | 查询某用户最近观看历史。 |
| `(video_id, updated_at desc)` | 查询某视频最近消费情况，用于排查或后台统计。 |

### 3.2 `catalog.video_user_states`

现有 `catalog.video_user_states` 继续保留为聚合投影。观看进度 API 只维护观看相关字段，不处理点赞和收藏字段。

字段语义建议固定如下：

| 字段 | 含义 | 观看进度 API 维护规则 |
| --- | --- | --- |
| `user_id` | 用户 ID。 | 从认证上下文获取。 |
| `video_id` | 视频 ID。 | 从 URL path 获取。 |
| `has_liked` | 用户当前是否点赞。 | 观看进度 API 不修改。 |
| `has_bookmarked` | 用户当前是否收藏。 | 观看进度 API 不修改。 |
| `has_watched` | 用户是否至少观看过该视频。 | 第一次成功上报时置为 `true`。 |
| `liked_at` | 最近点赞时间。 | 观看进度 API 不修改。 |
| `bookmarked_at` | 最近收藏时间。 | 观看进度 API 不修改。 |
| `first_watched_at` | 用户第一次观看该视频的时间。 | 为空时写入新 session 的 `started_at`。 |
| `last_watched_at` | 用户最近观看该视频的时间。 | 每次成功上报后更新为最新 `last_seen_at`。 |
| `watch_count` | 观看 session 数。 | 每个新 `watch_session_id` 只加 1。 |
| `completed_count` | 完成 session 数。 | 每个 session 首次完成只加 1。 |
| `last_watch_ratio` | 最近一次 session 当前达到的最大观看比例。 | 更新为当前 session 的 `max_watch_ratio`。 |
| `max_watch_ratio` | 用户历史看该视频达到过的最大观看比例。 | 只增不减，取历史值和当前 session `max_watch_ratio` 的较大值。 |
| `updated_at` | 聚合状态最近更新时间。 | 每次成功处理请求更新。 |

## 4. API 设计

### 4.1 上报观看进度

```http
POST /api/catalog/videos/{video_id}/watch-progress
```

路径参数：

| 参数 | 含义 |
| --- | --- |
| `video_id` | 被观看的视频 ID。服务端必须校验该视频存在。 |

请求体：

```json
{
  "watch_session_id": "9c68402b-2c53-4478-94ec-e4cb7e682458",
  "position_ms": 83420,
  "duration_ms": 312000,
  "is_completed": false,
  "occurred_at": "2026-05-08T20:12:34.200Z",
  "source": "web",
  "metadata": {
    "surface": "fullscreen"
  }
}
```

请求字段说明：

| 字段 | 必填 | 含义 | 服务端规则 |
| --- | --- | --- | --- |
| `watch_session_id` | 是 | 一次播放会话 ID。 | 必须是 UUID；同一 session 后续上报必须指向同一 `user_id + video_id`。 |
| `position_ms` | 是 | 当前播放位置，毫秒。 | 必须大于等于 0；可小于历史最大位置。 |
| `duration_ms` | 建议必填 | 视频总时长，毫秒。 | 有值时必须大于 0；用于计算 `max_watch_ratio`。 |
| `is_completed` | 否 | 本次上报是否表示完成播放。 | 默认 `false`；session 只允许首次完成贡献一次 `completed_count`。 |
| `occurred_at` | 否 | 客户端事件发生时间。 | 可用于 `started_at / last_seen_at`，但服务端应限制不能离当前时间过远。 |
| `source` | 否 | 客户端来源。 | 例如 `web`、`ios`、`android`；默认 `app`。 |
| `metadata` | 否 | 非核心调试上下文。 | 必须是 JSON object；不参与核心业务规则。 |

不建议前端传 `watch_ratio`。服务端应根据 `position_ms` 和 `duration_ms` 计算，避免客户端传入不可信 ratio。

成功响应：

```json
{
  "accepted": true,
  "watch_session_id": "9c68402b-2c53-4478-94ec-e4cb7e682458",
  "created_session": false,
  "completed_session": false,
  "video_user_state": {
    "has_watched": true,
    "watch_count": 1,
    "completed_count": 0,
    "last_watch_ratio": 0.26737,
    "max_watch_ratio": 0.26737,
    "last_watched_at": "2026-05-08T20:12:34.200Z"
  }
}
```

响应字段说明：

| 字段 | 含义 |
| --- | --- |
| `accepted` | 服务端已接受并处理本次上报。 |
| `watch_session_id` | 被处理的观看 session ID。 |
| `created_session` | 本次是否新建 session；为 `true` 时 `watch_count` 会增加。 |
| `completed_session` | 本次是否首次把该 session 标记为完成；为 `true` 时 `completed_count` 会增加。 |
| `video_user_state` | 更新后的用户-视频观看聚合状态，方便前端立即刷新 UI。 |

错误响应建议：

| HTTP 状态 | 场景 |
| --- | --- |
| `400 Bad Request` | 请求 JSON 格式错误、字段类型错误、`position_ms < 0`、`duration_ms <= 0`。 |
| `401 Unauthorized` | 未登录或认证失效。 |
| `404 Not Found` | `video_id` 不存在。 |
| `409 Conflict` | `watch_session_id` 已存在，但绑定的 `user_id` 或 `video_id` 与本次请求不一致。 |
| `422 Unprocessable Entity` | `occurred_at` 明显异常，例如过早或超过当前时间太多。 |

### 4.2 给前端的接口说明

前端只需要调用一个接口来上报视频观看进度：

```http
POST /api/catalog/videos/{video_id}/watch-progress
```

`video_id` 放在 URL path 里，表示“正在观看哪个视频”。请求 body 里不需要再重复传 `video_id`。用户身份也不需要传，后端会从登录态或 token 中解析当前用户。

接口调用时机：

1. 视频开始播放时调用一次。
2. 播放过程中每 10 到 15 秒调用一次。
3. 暂停、切换视频、退出播放页、播放结束时补调用一次。
4. 页面卸载时可以用 `navigator.sendBeacon` 尝试补发，但不能只依赖 unload 场景。

请求示例：

```ts
await fetch(`/api/catalog/videos/${videoId}/watch-progress`, {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  credentials: "include",
  body: JSON.stringify({
    watch_session_id: watchSessionId,
    position_ms: Math.round(video.currentTime * 1000),
    duration_ms: Math.round(video.duration * 1000),
    is_completed: video.ended,
    occurred_at: new Date().toISOString(),
    source: "web",
    metadata: {
      surface: "fullscreen"
    }
  })
});
```

前端字段说明：

| 字段 | 前端怎么取 | 说明 |
| --- | --- | --- |
| `watch_session_id` | 打开一次视频播放页或创建一次播放器会话时生成 `crypto.randomUUID()`。 | 同一次观看过程中保持不变。不要每次 progress 都生成新 ID，否则后端会认为是多次观看。 |
| `position_ms` | `Math.round(video.currentTime * 1000)`。 | 当前播放器位置，单位毫秒。它表示这次上报发生时播到哪里，不表示新增观看时长。seek 回退后它可以变小。 |
| `duration_ms` | `Math.round(video.duration * 1000)`。 | 视频总时长，单位毫秒。前端拿不到时先不要上报，等 metadata loaded 后再报。 |
| `is_completed` | `video.ended` 或 `position_ms / duration_ms >= 0.9`。 | 是否认为这次 session 已完成。后端会保证同一个 session 只计一次完成。 |
| `occurred_at` | `new Date().toISOString()`。 | 客户端本次上报发生时间。 |
| `source` | 固定传当前客户端，例如 `web`。 | 用于后端排查来源。 |
| `metadata.surface` | 当前页面或播放场景，例如 `feed`、`fullscreen`、`detail`。 | 非核心字段，只用于调试和分析。 |

响应示例：

```json
{
  "accepted": true,
  "watch_session_id": "9c68402b-2c53-4478-94ec-e4cb7e682458",
  "created_session": false,
  "completed_session": false,
  "video_user_state": {
    "has_watched": true,
    "watch_count": 1,
    "completed_count": 0,
    "last_watch_ratio": 0.26737,
    "max_watch_ratio": 0.26737,
    "last_watched_at": "2026-05-08T20:12:34.200Z"
  }
}
```

前端通常只需要关心：

| 字段 | 用途 |
| --- | --- |
| `accepted` | 为 `true` 表示后端已处理。 |
| `created_session` | 调试用；为 `true` 表示这是这个 `watch_session_id` 的第一次上报。 |
| `completed_session` | 调试用；为 `true` 表示这次首次把 session 标记为完成。 |
| `video_user_state.max_watch_ratio` | 可用于本地 UI 立即显示“看过多少”。 |
| `video_user_state.completed_count` | 可用于判断是否至少完成过一次。 |
| `video_user_state.last_watched_at` | 可用于本地展示最近观看时间。 |

前端注意事项：

1. 同一个视频播放会话必须复用同一个 `watch_session_id`。
2. 切换到另一个视频时必须生成新的 `watch_session_id`。
3. 不要在 body 里传 `user_id`，也不要在 body 里重复传 `video_id`。
4. 不要传 `watch_ratio`，后端会用 `position_ms / duration_ms` 计算。
5. `position_ms` 是当前播放位置，不是新增观看时长。
6. 请求失败时可以重试；同一个 `watch_session_id` 重试不会重复增加观看次数。
7. 进度上报不需要每秒发送，10 到 15 秒一次足够。

## 5. 后端处理流程

服务端处理 `POST /api/catalog/videos/{video_id}/watch-progress` 时应按以下顺序执行：

1. 从认证上下文获取 `user_id`。
2. 校验 `video_id` 存在。
3. 校验请求体字段。
4. 在短事务内 upsert `analytics.video_watch_events`。
5. 判断本次是否创建了新 session。
6. 判断本次是否首次完成该 session。
7. 在同一事务内 upsert `catalog.video_user_states`。
8. 返回更新后的聚合状态。

session upsert 规则：

```text
new_max_position_ms = greatest(existing.max_position_ms, request.position_ms)
new_duration_ms = request.duration_ms if present else existing.duration_ms
new_max_watch_ratio = clamp(new_max_position_ms / new_duration_ms, 0, 1)
new_is_completed = existing.is_completed or request.is_completed
new_completed_at = existing.completed_at or occurred_at when request.is_completed
```

聚合投影更新规则：

```text
has_watched = true
first_watched_at = coalesce(existing.first_watched_at, session.started_at)
last_watched_at = greatest(existing.last_watched_at, session.last_seen_at)
watch_count += 1 only when created_session = true
completed_count += 1 only when completed_session = true
last_watch_ratio = session.max_watch_ratio
max_watch_ratio = greatest(existing.max_watch_ratio, session.max_watch_ratio)
updated_at = now()
```

注意事项：

1. 同一个 `watch_session_id` 不允许跨用户或跨视频复用。
2. 同一个 session 多次传 `is_completed = true`，只能让 `completed_count` 增加一次。
3. seek 回退只影响 `last_position_ms`，不降低 `max_position_ms` 和 `max_watch_ratio`。
4. 前端重复上报同一 session 不会重复增加 `watch_count`。
5. 该接口不写 `learning.unit_learning_events`，也不写 `recommendation.user_video_serving_states`。

## 6. 前端调用示例

基础调用函数：

```ts
export async function reportWatchProgress(input: {
  videoId: string;
  watchSessionId: string;
  positionMs: number;
  durationMs: number;
  isCompleted?: boolean;
}) {
  const response = await fetch(
    `/api/catalog/videos/${input.videoId}/watch-progress`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({
        watch_session_id: input.watchSessionId,
        position_ms: input.positionMs,
        duration_ms: input.durationMs,
        is_completed: input.isCompleted ?? false,
        occurred_at: new Date().toISOString(),
        source: "web",
        metadata: { surface: "fullscreen" }
      })
    }
  );

  if (!response.ok) {
    throw new Error(`watch progress failed: ${response.status}`);
  }

  return response.json() as Promise<{
    accepted: boolean;
    watch_session_id: string;
    created_session: boolean;
    completed_session: boolean;
    video_user_state: {
      has_watched: boolean;
      watch_count: number;
      completed_count: number;
      last_watch_ratio: number;
      max_watch_ratio: number;
      last_watched_at: string;
    };
  }>;
}
```

播放器使用方式：

```ts
const watchSessionId = crypto.randomUUID();

await reportWatchProgress({
  videoId,
  watchSessionId,
  positionMs: 0,
  durationMs
});

await reportWatchProgress({
  videoId,
  watchSessionId,
  positionMs,
  durationMs
});

await reportWatchProgress({
  videoId,
  watchSessionId,
  positionMs,
  durationMs,
  isCompleted: positionMs / durationMs >= 0.9
});
```

推荐上报策略：

1. 播放开始时上报一次。
2. 播放中每 10 到 15 秒上报一次。
3. 暂停、退出页面、切换视频、播放结束时补一次。
4. 页面卸载时可以用 `navigator.sendBeacon` 做 best-effort flush，但不能把它作为唯一可靠路径。

## 7. 测试与验收标准

数据库层应验证：

1. `video_watch_events` 可以创建、重复 upsert，并保持 `watch_session_id` 唯一。
2. `position_ms` 回退不会降低 `max_position_ms` 和 `max_watch_ratio`。
3. 同一个 session 首次完成会设置 `completed_at`，后续完成上报不重复计数。
4. 同一个 `watch_session_id` 绑定不同用户或不同视频时返回冲突。

API / usecase 层应验证：

1. 第一次上报创建 session，并让 `video_user_states.watch_count = 1`。
2. 同一个 session 重复上报不重复增加 `watch_count`。
3. 首次完成让 `completed_count += 1`。
4. 重复完成不重复增加 `completed_count`。
5. `max_watch_ratio` 只增不减。
6. 未登录、视频不存在、非法字段都返回明确错误。

集成验收：

1. 前端按 10 到 15 秒频率上报时，数据库只保留 session summary，不产生高频事件流水。
2. Recommendation 继续只读 `catalog.video_user_states`，无需感知 `analytics.video_watch_events`。
3. 普通观看进度不会自动写入 `learning.unit_learning_events`。

## 8. 后续扩展

如果未来需要更细粒度分析，可以在不破坏当前 API 的前提下新增：

1. `analytics.video_player_events`：高频播放器事件流水，仅用于分析，不进入 MVP。
2. `analytics.video_reaction_events`：点赞、收藏、分享等低频互动审计。
3. session 级有效观看时长估算：基于服务端规则限制 progress 上报间隔与单次增量。
4. 后台重建任务：从 `analytics.video_watch_events` 重算 `catalog.video_user_states`。

这些扩展都不应改变当前边界：Analytics 负责原始行为事实，Catalog 负责视频消费状态投影，Learning engine 负责学习证据与学习状态，Recommendation 负责推荐曝光、冷却与审计。
