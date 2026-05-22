# Video History API MVP 设计

## 1. 文档目标

本文定义 `GET /api/video-history` 的后端契约。该 API 用于读取当前用户最近观看过的视频列表，服务移动端 Me 页进入的 Video History 页面。

本 API 是 Catalog 只读列表能力：返回列表 preview 和观看历史 metadata，不返回播放资源、transcript、description、like / favorite count 或当前用户互动状态。点击列表项进入播放页后，由 [Video-Detail-API-MVP设计.md](Video-Detail-API-MVP设计.md) 读取完整详情。

相关文档：

- [Feed-API-MVP设计.md](Feed-API-MVP设计.md)
- [Video-Detail-API-MVP设计.md](Video-Detail-API-MVP设计.md)
- [Catalog-观看进度上报MVP设计.md](Catalog-观看进度上报MVP设计.md)
- [Video-Interactions-API-MVP设计.md](Video-Interactions-API-MVP设计.md)
- [API模块总体设计规范.md](API模块总体设计规范.md)

## 2. 核心结论

```http
GET /api/video-history?limit=20&cursor=<opaque_cursor>
Authorization: Bearer <token>
Accept: application/json
```

固定语义：

- `auth: required`。
- request body 为空。
- `user_id` 只来自 trusted principal；前端不能通过 body、query 或 path 指定用户。
- 同一个 `video_id` 最多返回一条，表示当前用户对该视频的最新观看投影。
- 列表读取 `catalog.video_user_states` 投影，不直接读取 `analytics.video_watch_events`。
- 只返回当前仍可展示的视频：active、public、已发布。
- 默认按 `last_watched_at desc, video_id asc` 排序。
- 使用 cursor keyset pagination，不使用 offset。
- cursor 是 opaque token，前端只透传，不解析。
- 空列表返回 `200 OK`，`items=[]`，`has_more=false`。
- 不新增业务表；MVP 读取既有 `catalog.video_user_states`、`catalog.videos`、`catalog.video_engagement_stats`。
- 为保证分页性能，建议把现有 history index 升级为带 tie-breaker 和 partial predicate 的索引。

## 3. Owner 边界

```text
GET /api/video-history
  -> internal/api videolibrary.Handler
  -> internal/api VideoLibraryService.ListHistory
  -> catalog.ListVideoHistoryUsecase
  -> catalog.VideoLibraryReader.ListVideoHistory
  -> catalog.video_user_states
  -> catalog.videos
  -> catalog.video_engagement_stats
```

边界说明：

- `internal/api` 负责 principal、query parsing、HTTP error mapping 和 response DTO。
- `catalog` 负责 cursor decode / encode、分页规则、可展示视频过滤、表读取和领域错误。
- `catalog.video_user_states` 是用户 x 视频的当前消费状态投影，是本 API 的读取来源。
- `analytics.video_watch_events` 是低频观看 session summary ledger，不是 history list 的热路径读取来源。
- `catalog.video_engagement_stats` 只提供 `view_count` preview；缺统计行返回 `0`。
- API 不写 Analytics、Learning Engine 或 Recommendation。
- API 不创建新的 watch session；点击历史列表进入播放页后，播放器会通过 watch-progress 写入口创建或继续自己的 session。

## 4. 请求契约

Query 参数：

| 参数 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `limit` | integer | 否 | `20` | 每页数量，合法范围 `1..100`。 |
| `cursor` | string | 否 | 无 | 上一页返回的 opaque cursor。第一页不传。 |

解析规则：

- `limit` 省略或空字符串时使用默认值 `20`。
- `limit` 非整数返回 `400 invalid_request`。
- `limit < 1` 或 `limit > 100` 返回 `400 invalid_request`，不静默截断。
- `cursor` 会 trim；空字符串等同于不传。
- cursor 解码失败返回 `400 invalid_request`。
- cursor `kind` 不是 `video_history` 返回 `400 invalid_request`。
- request body 必须为空；即使客户端带 body，handler 也不读取或信任其中字段。

## 5. 响应契约

```ts
type VideoHistoryResponse = {
  items: VideoHistoryItem[];
  page: {
    limit: number;
    has_more: boolean;
    next_cursor: string | null;
  };
};

type VideoHistoryItem = {
  video_id: string;
  title: string;
  cover_image_url: string | null;
  duration_seconds: number;
  view_count: number;
  last_position_ms: number;
  last_watched_at: string;
};
```

字段语义：

| 字段 | 来源 | 语义 |
| --- | --- | --- |
| `video_id` | `catalog.videos.video_id` | 视频稳定 ID。 |
| `title` | `catalog.videos.title` | 当前 Catalog preview 标题。 |
| `cover_image_url` | `catalog.videos.thumbnail_url` 经 API URL 组装 | 当前 Catalog preview 封面；缺值或空路径返回 `null`。 |
| `duration_seconds` | `catalog.videos.duration_ms` 向上取整 | 视频时长秒数。 |
| `view_count` | `catalog.video_engagement_stats.view_count` | 全局观看数；缺统计行返回 `0`。 |
| `last_position_ms` | `catalog.video_user_states.last_position_ms` | 当前用户最近一次 session 的最后播放位置。 |
| `last_watched_at` | `catalog.video_user_states.last_watched_at` | 当前用户最近一次成功 watch-progress 上报时间。 |
| `page.limit` | request normalized limit | 本页实际 limit。 |
| `page.has_more` | `limit + 1` 查询结果 | 是否还有下一页。 |
| `page.next_cursor` | 最后一条返回 item 生成 | 下一页 cursor；没有下一页时为 `null`。 |

`title / cover_image_url / duration_seconds / view_count` 是当前 Catalog preview，不是观看时快照。当前后端没有 history snapshot 表，因此视频元数据变化会反映到下次列表读取。

`last_position_ms` 不因为接近视频尾部而在本 API 中归零。是否从该位置恢复、是否对接近尾部位置做归零，是播放器或 Single Video Playback 的产品策略。

## 6. 不返回字段

本 API 不返回：

```ts
description
video_url
transcript_url
like_count
favorite_count
user_state
learning_units
recommendation_run_id
occurrence_index
watch_session_id
max_position_ms
total_watch_ms
completed_count
```

原因：

- `description / video_url / transcript_url / like_count / favorite_count / user_state` 属于 Video Detail 真值。
- `learning_units / recommendation_run_id / occurrence_index` 属于 Feed occurrence 语义。
- `watch_session_id` 是 watch-progress 上报和 ledger 身份；历史列表点击会进入新的播放上下文，不复用旧 session。
- `max_position_ms / total_watch_ms / completed_count` 属于统计或推荐 penalty 语义，不是 MVP 历史列表展示字段。

## 7. 可展示视频过滤

Catalog reader 必须使用与 Feed / Video Detail 一致的可展示视频 predicate：

```sql
v.status = 'active'
and v.visibility_status = 'public'
and (v.publish_at is null or v.publish_at <= now())
```

如果用户看过的视频后来被下架、隐藏或设置为未来发布时间，本 API 不返回该视频。这样可以避免列表展示一个点击后 Video Detail 返回 `404 not_found` 的 item。

## 8. Cursor 分页

### 8.1 Cursor 格式

对外返回 opaque token，例如 base64url 编码后的 JSON。前端不解析 cursor。

内部 payload：

```json
{
  "kind": "video_history",
  "last_watched_at": "2026-05-22T07:01:02.123Z",
  "video_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
}
```

字段规则：

- `kind` 用于防止不同列表 endpoint 的 cursor 混用。
- `last_watched_at` 必须是有效 timestamp。
- `video_id` 必须是 UUID。
- cursor 中的字段只用于分页定位，不用于鉴权或跳过当前 endpoint 的过滤条件。

### 8.2 排序和下一页条件

固定排序：

```sql
order by state.last_watched_at desc, state.video_id asc
```

下一页条件：

```sql
and (
  state.last_watched_at < sqlc.arg(cursor_last_watched_at)
  or (
    state.last_watched_at = sqlc.arg(cursor_last_watched_at)
    and state.video_id > sqlc.arg(cursor_video_id)
  )
)
```

第一页不带 cursor 时不加该条件。

### 8.3 limit + 1

实际实现读取 `limit + 1` 条：

- 如果返回行数大于 `limit`，`has_more = true`。
- 对前端只返回前 `limit` 条。
- `next_cursor` 使用返回给前端的最后一条 item 生成。
- 如果 `has_more = false`，`next_cursor = null`。

## 9. 查询草案

以下是不带 cursor 的第一页查询草案。后续页追加第 8.2 节 cursor 条件。

```sql
select
  v.video_id,
  v.title,
  v.thumbnail_url,
  v.duration_ms,
  coalesce(stats.view_count, 0)::bigint as view_count,
  state.last_position_ms,
  state.last_watched_at
from catalog.video_user_states state
join catalog.videos v
  on v.video_id = state.video_id
left join catalog.video_engagement_stats stats
  on stats.video_id = v.video_id
where state.user_id = sqlc.arg(user_id)::uuid
  and state.has_watched = true
  and state.last_watched_at is not null
  and v.status = 'active'
  and v.visibility_status = 'public'
  and (v.publish_at is null or v.publish_at <= now())
order by state.last_watched_at desc, state.video_id asc
limit sqlc.arg(limit_plus_one);
```

`duration_seconds` 在 service / mapper 层由 `duration_ms` 向上取整；`thumbnail_url` 在 API facade 层组装为公开 URL。

## 10. 索引建议

为了稳定 keyset pagination 和减少无关状态扫描，建议使用 partial index：

```sql
create index if not exists idx_video_user_states_history_page
on catalog.video_user_states (user_id, last_watched_at desc, video_id asc)
where has_watched = true and last_watched_at is not null;
```

原因：

- `video_id` 是同时间戳下的稳定 tie-breaker。
- partial predicate 与 history list 查询一致。
- 避免扫描从未观看或时间为空的用户视频状态。

## 11. 错误与边界

| HTTP | code | 场景 |
| --- | --- | --- |
| `200 OK` | - | 成功返回列表；空列表也是成功。 |
| `400 Bad Request` | `invalid_request` | `limit` 非整数或越界；cursor 无法解码；cursor kind 不匹配；cursor 字段非法。 |
| `401 Unauthorized` | `unauthorized` | trusted principal 缺失。 |
| `500 Internal Server Error` | `internal_error` | 数据库、URL 组装或未知服务端错误。 |
| `503 Service Unavailable` | `service_unavailable` | 下游超时或 request context 取消。 |

数据缺失规则：

- 缺 `catalog.video_engagement_stats` 行不是错误，`view_count = 0`。
- 缺 thumbnail 或空 thumbnail path 不是错误，`cover_image_url = null`。
- `has_watched = true` 但 `last_watched_at is null` 的历史脏数据不返回。
- 已观看但视频不可展示时不返回，不单独报错。

## 12. 与 Watch Progress / Video Detail 的关系

- `POST /api/video-watch-progress` 是观看进度写入口。
- `GET /api/video-history` 是当前用户观看投影的列表读取入口。
- `GET /api/videos/{video_id}` 是播放页完整详情和当前用户 action rail base state 的权威读取入口。
- History list 成功读取后可以给前端 seed video preview，并把 `last_position_ms` 作为播放恢复 hint。
- 播放页 watch-progress 上报成功后，已打开的 history list 不要求实时更新；用户 refresh 后读取最新投影。

后端不从 `analytics.video_watch_events` 实时 group by 生成列表。该表是 session ledger，用于幂等、去重、审计和后续重建投影；热路径列表读取以 `catalog.video_user_states` 为准。

## 13. 实现位置建议

新增后端结构：

```text
internal/api/application/dto/video_library.go
internal/api/application/service/video_library.go
internal/api/infrastructure/http/handler/videolibrary/

internal/catalog/application/dto/video_library.go
internal/catalog/application/repository/video_library_reader.go
internal/catalog/application/service/video_library.go
internal/catalog/domain/model/video_library.go
internal/catalog/infrastructure/persistence/query/video_library.sql
internal/catalog/infrastructure/persistence/repository/video_library_reader.go
```

HTTP 层使用 `videolibrary.Handler` 统一注册 Video Favorites 与 Video History 两条 route，但每条 route 保持独立方法，并分别调用 `ListVideoFavoritesUsecase` / `ListVideoHistoryUsecase`。Catalog reader 可以共用 `VideoLibraryReader`，cursor kind、排序字段和 metadata 边界由独立 usecase 保持隔离。

## 14. 测试要求

API integration：

- missing principal 返回 `401 unauthorized`。
- `limit` 非整数、`0`、`101` 返回 `400 invalid_request`。
- malformed cursor 返回 `400 invalid_request`。
- favorites cursor 传给 history 返回 `400 invalid_request`。
- 空历史列表返回 `200` 和空数组。
- 只返回当前用户 `has_watched = true` 且 `last_watched_at is not null` 的视频。
- 不返回 inactive / private / future publish 视频。
- 缺 stats 返回 `view_count = 0`。
- 分页返回 `limit` 条、`has_more=true`、`next_cursor != null`。
- 第二页用 `next_cursor` 不重复、不漏掉同时间戳视频。

Catalog integration：

- 按 `last_watched_at desc, video_id asc` 排序。
- 同一时间戳 tie-breaker 稳定。
- `last_watched_at is null` 的脏数据不返回。
- cursor 解码后 SQL 条件正确推进。
- index migration 存在并能覆盖主查询路径。

E2E：

- 通过 `POST /api/video-watch-progress` 写入观看投影后，`GET /api/video-history` 能读到该视频。
- 重复同一 `watch_session_id` 上报不重复制造列表 row。
- 新 session 更新 `last_watched_at` 后，列表排序随投影更新。
- 点击历史列表进入 Video Detail 时，目标视频仍能通过 `GET /api/videos/{video_id}` 读取详情。

成功标准：

1. 前端可以用 `GET /api/video-history` 分页读取当前用户观看历史列表。
2. 列表 item 只包含 preview、`last_position_ms` 和 `last_watched_at`。
3. 列表与 Feed / Video Detail 的可展示视频规则一致。
4. 分页是 keyset cursor，不使用 offset。
5. Video Detail 继续作为播放详情和互动 base state 的唯一权威读取入口。
