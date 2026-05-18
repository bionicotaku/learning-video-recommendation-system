# 观看进度上报 MVP 设计

本文档描述“用户观看视频进度”的 MVP 设计方案。当前 API 已实现为：

```http
POST /api/video-watch-progress
```

当前实现已经收敛为三层数据模型：

```text
analytics.video_watch_events  -- 一次观看 session 的低频摘要事实
catalog.video_user_states     -- 用户 x 视频的聚合状态
catalog.video_engagement_stats -- 视频级全局互动统计投影
```

`analytics.video_watch_events` 保存 session 事实，提供幂等、去重、完成次数依据和有限可追溯能力。`catalog.video_user_states` 保存单个用户对单个视频的当前聚合状态。`catalog.video_engagement_stats` 保存视频级全局统计，供 feed/detail 快速读取，避免热路径实时聚合。

## 1. 设计目标

本方案的目标是：

1. 支持前端以低频方式上报观看进度。
2. 避免高频播放器事件流水带来的写入和归约复杂度。
3. 让 `watch_count`、`completed_count` 和 `total_watch_ms` 有明确去重依据。
4. 保持 Recommendation 只读 `catalog.video_user_states`，不直接依赖观看 session 表。
5. 保持 Analytics / Catalog / Learning engine / Recommendation 边界清晰。

本方案不解决：

1. 精确逐秒观看轨迹分析。
2. 防作弊级别的真实观看时长计算。
3. 点赞、收藏、分享等低频互动事件审计。
4. Recommendation 曝光和冷却状态维护。
5. 学习事件归约。

## 2. 模块边界

原始观看事实属于 Analytics owner。`analytics.video_watch_events` 只记录用户观看某个视频的一次播放会话摘要。它不是逐秒播放器日志，也不记录推荐曝光。

聚合消费状态属于 Catalog owner。`catalog.video_user_states` 是用户与视频的聚合投影，`catalog.video_engagement_stats` 是视频级全局统计投影。Recommendation 当前只允许读取 `catalog.video_user_states` 中的 `last_watched_at`、`watch_count`、`completed_count`、`last_position_ms`、`max_position_ms` 作为轻量 penalty 输入；观看比例不再持久化，而是用 `max_position_ms / catalog.videos.duration_ms` 派生。

后端业务 owner 是 `internal/catalog` 的 `RecordVideoWatchProgress` usecase。该命令会在同一事务内写入 `analytics.video_watch_events` 与两个 Catalog 投影；这是 watch-progress 的 session ledger 与消费投影原子一致性要求，不表示 Catalog 泛化拥有 Analytics raw fact 表。`internal/api` 只负责 HTTP 入口、principal、transport validation 和 DTO 映射，不直接读写数据库。

`learning.unit_learning_events` 只记录学习事件。普通观看进度不应自动写入 Learning engine。只有当产品层明确把一次观看行为解释成学习行为时，才应由上层业务另行调用 Learning engine 的学习事件接口。

## 3. 数据库设计

### 3.1 `analytics.video_watch_events`

```sql
create table analytics.video_watch_events (
  watch_session_id uuid primary key,

  user_id uuid not null references auth.users(id) on delete cascade,
  video_id uuid not null references catalog.videos(video_id) on delete cascade,

  started_at timestamptz not null,
  last_seen_at timestamptz not null,
  completed_at timestamptz,

  last_position_ms integer not null default 0,
  max_position_ms integer not null default 0,
  active_watch_ms bigint not null default 0,
  is_completed boolean not null default false,

  progress_report_count integer not null default 0,
  client_context jsonb not null default '{}'::jsonb,
  metadata jsonb not null default '{}'::jsonb,

  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),

  check (last_position_ms >= 0),
  check (max_position_ms >= 0),
  check (active_watch_ms >= 0),
  check (progress_report_count >= 0),
  check (jsonb_typeof(client_context) = 'object'),
  check (jsonb_typeof(metadata) = 'object')
);
```

字段说明：

| 字段 | 含义 | 维护规则 |
| --- | --- | --- |
| `watch_session_id` | 一次播放会话 ID。 | 主键；同一个 session 重复上报走 upsert。 |
| `user_id` | 当前登录用户。 | 从认证上下文获取，不接受前端传入。 |
| `video_id` | 被观看的视频。 | 从请求 body 获取，必须存在于 `catalog.videos`。 |
| `started_at` | 该 session 第一次上报时间。 | 首次 insert 时写入；后续不覆盖。 |
| `last_seen_at` | 该 session 最近一次有效上报时间。 | 每次 progress 上报时更新。 |
| `completed_at` | 该 session 首次完成时间。 | 服务端首次判定完成时写入；后续不覆盖。 |
| `last_position_ms` | 最近一次上报的播放位置。 | 每次上报覆盖；允许小于 `max_position_ms`。 |
| `max_position_ms` | 该 session 到达过的最大播放位置。 | 只增不减。 |
| `active_watch_ms` | 该 session 累计有效播放时长。 | 前端上报 session 内累计值；服务端用新旧值差值维护投影。 |
| `is_completed` | 该 session 是否已完成。 | 服务端根据累计有效播放时长和最大播放位置计算；只允许从 `false` 变为 `true`。 |
| `progress_report_count` | 该 session 累计收到的进度上报次数。 | 每次成功处理请求加 1。 |
| `client_context` | 客户端环境上下文。 | 保存客户端环境 JSON object；当前前端样例使用 `platform`、`app_version`、`os_version`、`device_model`。 |
| `metadata` | 可选调试上下文。 | 只放 watch-progress 专属扩展信息，例如 `source_surface`、播放器版本。 |
| `created_at` | 行创建时间。 | 数据库默认值。 |
| `updated_at` | 行最近更新时间。 | 每次 upsert 更新。 |

`duration_ms` 和 `max_watch_ratio` 不再保存在本表。视频时长只来自 `catalog.videos.duration_ms`；观看比例由 `max_position_ms / duration_ms` 派生。

推荐索引：

```sql
create index idx_video_watch_events_user_video_updated_at
on analytics.video_watch_events (user_id, video_id, updated_at desc);

create index idx_video_watch_events_user_updated_at
on analytics.video_watch_events (user_id, updated_at desc);

create index idx_video_watch_events_video_updated_at
on analytics.video_watch_events (video_id, updated_at desc);
```

### 3.2 `catalog.video_user_states`

`catalog.video_user_states` 是用户与视频的聚合投影。观看进度 API 只维护观看相关字段，不处理点赞和收藏字段。

| 字段 | 含义 | 观看进度 API 维护规则 |
| --- | --- | --- |
| `user_id` | 用户 ID。 | 从认证上下文获取。 |
| `video_id` | 视频 ID。 | 从请求 body 获取。 |
| `has_liked` | 用户当前是否点赞。 | 观看进度 API 不修改。 |
| `has_bookmarked` | 用户当前是否收藏。 | 观看进度 API 不修改。 |
| `has_watched` | 用户是否至少观看过该视频。 | 第一次成功上报时置为 `true`。 |
| `liked_at` | 最近点赞时间。 | 观看进度 API 不修改。 |
| `bookmarked_at` | 最近收藏时间。 | 观看进度 API 不修改。 |
| `first_watched_at` | 用户第一次观看该视频的时间。 | 为空时写入新 session 的 `started_at`。 |
| `last_watched_at` | 用户最近观看该视频的时间。 | 每次成功上报后更新为最新 `last_seen_at`。 |
| `watch_count` | 观看 session 数。 | 每个新 `watch_session_id` 只加 1。 |
| `completed_count` | 完成 session 数。 | 每个 session 首次完成只加 1。 |
| `last_position_ms` | 最近一次 session 最后停留位置。 | 更新为当前 session 的 `last_position_ms`。 |
| `max_position_ms` | 用户历史到达过的最远播放位置。 | 只增不减。 |
| `total_watch_ms` | 用户对该视频累计有效播放时长。 | 按 `active_watch_ms` 的 session delta 累加。 |
| `updated_at` | 聚合状态最近更新时间。 | 每次成功处理请求更新。 |

`last_watch_ratio` 和 `max_watch_ratio` 不再持久化。需要展示或推荐 penalty 时，用 `position_ms / catalog.videos.duration_ms` 派生。

### 3.3 `catalog.video_engagement_stats`

`catalog.video_engagement_stats` 是视频级全局统计投影。

```sql
create table catalog.video_engagement_stats (
  video_id uuid primary key references catalog.videos(video_id) on delete cascade,
  view_count bigint not null default 0,
  like_count bigint not null default 0,
  favorite_count bigint not null default 0,
  completed_count bigint not null default 0,
  total_watch_ms bigint not null default 0,
  updated_at timestamptz not null default now(),

  check (view_count >= 0),
  check (like_count >= 0),
  check (favorite_count >= 0),
  check (completed_count >= 0),
  check (total_watch_ms >= 0)
);
```

字段说明：

| 字段 | 含义 |
| --- | --- |
| `video_id` | 视频 ID。 |
| `view_count` | 全局观看 session 数，语义等同于所有用户 `watch_count` 的聚合。 |
| `like_count` | 当前点赞该视频的用户数。 |
| `favorite_count` | 当前收藏该视频的用户数，对应 `has_bookmarked = true`。 |
| `completed_count` | 全局完成观看 session 数。 |
| `total_watch_ms` | 全局累计有效播放时长。 |
| `updated_at` | 统计投影最近更新时间。 |

## 4. API 设计

### 4.1 上报观看进度

```http
POST /api/video-watch-progress
```

请求体：

```json
{
  "video_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
  "watch_session_id": "9c68402b-2c53-4478-94ec-e4cb7e682458",
  "position_ms": 83420,
  "active_watch_ms": 62000,
  "occurred_at": "2026-05-08T20:12:34.200Z",
  "source_surface": "fullscreen",
  "client_context": {
    "platform": "ios",
    "app_version": "1.3.0",
    "os_version": "18.5",
    "device_model": "iPhone16,2"
  }
}
```

请求字段说明：

| 字段 | 必填 | 含义 | 服务端规则 |
| --- | --- | --- | --- |
| `video_id` | 是 | 被观看的视频 ID。 | 必须是 UUID；必须存在于 `catalog.videos`。 |
| `watch_session_id` | 是 | 一次播放会话 ID。 | 必须是 UUID；同一 session 后续上报必须指向同一 `user_id + video_id`。 |
| `position_ms` | 是 | 当前播放位置，毫秒。 | 必须大于等于 0；可小于历史最大位置。 |
| `active_watch_ms` | 是 | 本 session 累计有效播放时长。 | 必须大于等于 0；重复上报用 delta 去重。 |
| `occurred_at` | 否 | 客户端事件发生时间。 | RFC3339 datetime with explicit offset。可用于 `started_at / last_seen_at`，但服务端应限制不能离当前时间过远。 |
| `source_surface` | 建议必填 | 观看进度发生的产品入口，例如 `fullscreen`、`feed`、`detail`。 | 服务端可写入 `metadata.source_surface`。 |
| `client_context` | 否 | 客户端环境上下文。 | 必须是 JSON object；建议使用当前四个基础字段，服务端缺省 `{}`。 |
| `metadata` | 否 | watch-progress 专属扩展调试上下文。 | 必须是 JSON object；不参与核心业务规则。 |

不建议前端传 `watch_ratio` 或 `duration_ms`。服务端应从 `catalog.videos.duration_ms` 获取视频时长，并在需要时派生观看比例。

成功响应：

```json
{
  "accepted": true
}
```

成功响应只表示本次 watch-progress 上报已通过 validation 并完成后端处理。该接口不返回 `catalog.video_user_states` 或 `catalog.video_engagement_stats` 投影；前端如果需要展示观看状态，应通过 feed / detail 等读取接口获取。

错误响应建议：

| HTTP 状态 | 场景 |
| --- | --- |
| `400 Bad Request` | 请求 JSON 格式错误、字段类型错误、`position_ms < 0`、`active_watch_ms < 0`。 |
| `401 Unauthorized` | 未登录或认证失效。 |
| `404 Not Found` | `video_id` 不存在。 |
| `409 Conflict` | `watch_session_id` 已存在，但绑定的 `user_id` 或 `video_id` 与本次请求不一致。 |
| `422 Unprocessable Entity` | `occurred_at` 明显异常，例如过早或超过当前时间太多。 |

### 4.2 给前端的接口说明

前端只需要调用一个接口来上报视频观看进度。`video_id` 放在请求 body 里，用户身份由后端从登录态解析。

建议调用时机：

1. 视频开始播放时调用一次。
2. 播放过程中每 10 到 15 秒调用一次。
3. 切换视频、退出播放页、播放接近结束或结束时尽量补调用一次。
4. 页面卸载时可以用 `navigator.sendBeacon` 尝试补发，但不能只依赖 unload 场景；若使用 `sendBeacon`，payload 必须用 `Blob` 设置 `application/json`，否则当前 API 会因为 Content-Type 不符合请求契约而返回 `400`。

以上是前端 flush 策略建议，不是后端强制契约。后端只要求请求字段满足本节定义；是否在 pause / resume 时 flush 由前端 runtime 自己决定。

请求示例：

```ts
await fetch(`/api/video-watch-progress`, {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  credentials: "include",
  body: JSON.stringify({
    video_id: videoId,
    watch_session_id: watchSessionId,
    position_ms: Math.round(video.currentTime * 1000),
    active_watch_ms: activeWatchMs,
    occurred_at: new Date().toISOString(),
    source_surface: "fullscreen",
    client_context: {
      platform: "ios",
      app_version: "1.3.0",
      os_version: "18.5",
      device_model: "iPhone16,2"
    }
  })
});
```

前端字段说明：

| 字段 | 前端怎么取 | 说明 |
| --- | --- | --- |
| `video_id` | 当前播放器或视频详情页绑定的视频 ID。 | 每次上报必须带；同一个 `watch_session_id` 后续上报必须保持同一视频。 |
| `watch_session_id` | 打开一次视频播放页或创建一次播放器会话时生成 `crypto.randomUUID()`。 | 同一次观看过程中保持不变。 |
| `position_ms` | `Math.round(video.currentTime * 1000)`。 | 当前播放器位置，不表示新增观看时长。 |
| `active_watch_ms` | 播放器处于真实播放状态时累计 elapsed time。 | 不用 `Date.now() - startedAt` 直接推导，暂停、后台、seek 不应累计。 |
| `occurred_at` | `new Date().toISOString()`。 | 客户端本次上报发生时间，必须带 `Z` 或 offset。 |
| `source_surface` | 当前页面或播放场景，例如 `feed`、`fullscreen`、`detail`。 | 表示业务入口，不表示平台。 |
| `client_context.*` | 从全局 telemetry client context 读取。 | 当前建议上传 `platform`、`app_version`、`os_version`、`device_model`；后续可随客户端遥测扩展。 |

前端注意事项：

1. 同一个视频播放会话必须复用同一个 `watch_session_id`。
2. 切换到另一个视频时必须生成新的 `watch_session_id`。
3. 不要在 body 里传 `user_id`；必须在 body 里传 `video_id`。
4. 不要传 `watch_ratio` 或 `duration_ms`。
5. `position_ms` 是当前播放位置，`active_watch_ms` 是累计有效播放时长。
6. 请求失败时可以重试；同一个 `watch_session_id` 重试不会重复增加观看次数。
7. 进度上报不需要每秒发送，10 到 15 秒一次足够。

### 4.3 前端 pending state 设计

观看进度上报不需要使用通用 event queue。它的语义不是“保留每次 progress 变化”，而是“把当前 watch session 的最新观看状态同步给后端”。

前端可以维护一个可替换的 pending state：

```ts
type PendingWatchProgress = {
  videoId: string;
  watchSessionId: string;
  positionMs: number;
  activeWatchMs: number;
  occurredAt: string;
  sourceSurface: string;
  clientContext: {
    platform: string;
    app_version: string;
    os_version: string;
    device_model: string;
  };
};
```

更新规则：

1. 普通 progress sample 到达时，覆盖 `positionMs`、`activeWatchMs`、`occurredAt`、`sourceSurface`、`clientContext`。
2. 同一个 `videoId + watchSessionId` 同一时刻只保留一个 pending state。
3. 切换视频或生成新 `watchSessionId` 前，先 flush 当前 pending state。
4. 视频接近结束或结束时建议立即 flush，但不需要在 payload 里携带完成判断。

## 5. 后端处理流程

服务端处理 `POST /api/video-watch-progress` 时应按以下顺序执行：

1. 从认证上下文获取 `user_id`。
2. 校验 `video_id` 存在，并读取 `catalog.videos.duration_ms`。
3. 校验请求体字段。
4. 在短事务内用条件 upsert 写入 `analytics.video_watch_events`，并通过 `RETURNING` 拿到本次是否创建新 session、是否首次完成、delta 与 upsert 后 session state。
5. 在同一事务内 upsert `catalog.video_user_states`。
6. 在同一事务内 upsert `catalog.video_engagement_stats`。
7. 返回 `{ "accepted": true }`。

这里允许读取 `catalog.videos.duration_ms`，因为完成判定需要视频时长；但不应在应用层先 `SELECT analytics.video_watch_events` 再由业务代码计算后 `UPDATE / INSERT`。同一个 `watch_session_id` 的旧 session state 应由数据库侧条件 upsert 完成；首次并发插入遇到冲突时，repository 只允许做一次同事务内重试，让第二次语句读取已存在 session 并继续走数据库侧计算，不把状态计算搬到 Go 层。

session upsert 规则：

```text
new_last_position_ms = request.position_ms
new_max_position_ms = greatest(existing.max_position_ms, request.position_ms)
new_active_watch_ms = greatest(existing.active_watch_ms, request.active_watch_ms)
delta_active_watch_ms = new_active_watch_ms - existing.active_watch_ms
computed_completed =
  new_active_watch_ms > 10_000
  and duration_ms > 0
  and new_max_position_ms / duration_ms >= 0.9
completed_session = existing.is_completed = false and computed_completed = true
new_is_completed = existing.is_completed or computed_completed
new_completed_at = existing.completed_at or occurred_at when completed_session
```

完成判定使用服务端 upsert 后的单调字段：`new_active_watch_ms` 与 `new_max_position_ms` 都只增不减。这样 seek 回退、旧请求重试或乱序上报不会让完成状态回落，也不会重复增加完成计数。

用户聚合投影更新规则：

```text
has_watched = true
first_watched_at = coalesce(existing.first_watched_at, session.started_at)
last_watched_at = greatest(existing.last_watched_at, session.last_seen_at)
watch_count += 1 only when created_session = true
completed_count += 1 only when completed_session = true
last_position_ms = session.last_position_ms
max_position_ms = greatest(existing.max_position_ms, session.max_position_ms)
total_watch_ms += delta_active_watch_ms
updated_at = now()
```

视频全局统计投影更新规则：

```text
view_count += 1 only when created_session = true
completed_count += 1 only when completed_session = true
total_watch_ms += delta_active_watch_ms
updated_at = now()
```

`Video Interactions API` 维护 `like_count` 与 `favorite_count`，观看进度 API 不修改这两个字段。

注意事项：

1. 同一个 `watch_session_id` 不允许跨用户或跨视频复用。
2. session 完成状态由服务端计算，不能由前端请求直接指定。
3. seek 回退只影响 `last_position_ms`，不降低 `max_position_ms`。
4. 前端重复上报同一 session 不会重复增加 `watch_count` 或 `view_count`。
5. 该接口不写 `learning.unit_learning_events`，也不写 `recommendation.user_video_serving_states`。

## 6. 测试与验收标准

数据库层应验证：

1. `video_watch_events` 可以创建、重复 upsert，并保持 `watch_session_id` 唯一。
2. `position_ms` 回退不会降低 `max_position_ms`。
3. `active_watch_ms` 重复上报不会重复累计。
4. `active_watch_ms <= 10_000` 时不会完成，即使播放位置达到 90%。
5. `new_max_position_ms / duration_ms < 0.9` 时不会完成，即使 `active_watch_ms` 足够。
6. 两个完成条件都满足时首次完成会设置 `completed_at`，后续上报不重复计数。
7. 同一个 `watch_session_id` 绑定不同用户或不同视频时返回冲突。
8. session 状态更新使用条件 upsert 原子完成，应用层不对 `analytics.video_watch_events` 做 pre-read。

API / usecase 层应验证：

1. 第一次上报创建 session，并让 `video_user_states.watch_count = 1`、`video_engagement_stats.view_count = 1`。
2. 同一个 session 重复上报不重复增加 `watch_count` 和 `view_count`。
3. 首次满足服务端完成阈值时让用户级和视频级 `completed_count += 1`。
4. 重复达到完成阈值不重复增加 `completed_count`。
5. `max_position_ms` 和 `active_watch_ms` 只增不减。
6. 未登录、视频不存在、非法字段都返回明确错误。

集成验收：

1. 前端按 10 到 15 秒频率上报时，数据库只保留 session summary，不产生高频事件流水。
2. Recommendation 继续只读 `catalog.video_user_states`，无需感知 `analytics.video_watch_events`。
3. 普通观看进度不会自动写入 `learning.unit_learning_events`。

## 7. 后续扩展

如果未来需要更细粒度分析，可以在不破坏当前 API 的前提下新增：

1. `analytics.video_player_events`：高频播放器事件流水，仅用于分析，不进入 MVP。
2. `analytics.video_reaction_events`：点赞、收藏、分享等低频互动审计。
3. 后台重建任务：从 `analytics.video_watch_events` 重算 `catalog.video_user_states` 和 `catalog.video_engagement_stats`。

这些扩展都不应改变当前边界：Analytics 负责原始行为事实，Catalog 负责视频消费状态投影，Learning engine 负责学习证据与学习状态，Recommendation 负责推荐曝光、冷却与审计。
