# Video Favorites API MVP 设计

## 1. 文档目标

本文定义 `GET /api/video-favorites` 的后端契约。该 API 用于读取当前用户收藏过的视频列表，服务移动端 Me 页进入的 Video Favorites 页面。

本 API 是 Catalog 只读列表能力：返回列表 preview 和收藏 metadata，不返回播放资源、transcript、description、like / favorite count 或当前用户互动状态。点击列表项进入播放页后，由 [Video-Detail-API-MVP设计.md](Video-Detail-API-MVP设计.md) 读取完整详情。

相关文档：

- [Feed-API-MVP设计.md](Feed-API-MVP设计.md)
- [Video-Detail-API-MVP设计.md](Video-Detail-API-MVP设计.md)
- [Video-Interactions-API-MVP设计.md](Video-Interactions-API-MVP设计.md)
- [Catalog-观看进度上报MVP设计.md](Catalog-观看进度上报MVP设计.md)
- [API模块总体设计规范.md](API模块总体设计规范.md)

## 2. 核心结论

```http
GET /api/video-favorites?limit=20&cursor=<opaque_cursor>
Authorization: Bearer <token>
Accept: application/json
```

固定语义：

- `auth: required`。
- request body 为空。
- `user_id` 只来自 trusted principal；前端不能通过 body、query 或 path 指定用户。
- 只返回当前用户仍收藏的视频：`catalog.video_user_states.has_bookmarked = true`。
- 只返回当前仍可展示的视频：active、public、已发布。
- 同一个 `video_id` 最多返回一条。
- 默认按 `favorited_at desc, video_id asc` 排序。
- 使用 cursor keyset pagination，不使用 offset。
- cursor 是 opaque token，前端只透传，不解析。
- 空列表返回 `200 OK`，`items=[]`，`has_more=false`。
- 不新增业务表；MVP 读取既有 `catalog.video_user_states`、`catalog.videos`、`catalog.video_engagement_stats`。
- 为保证分页性能，需要新增面向收藏列表的 partial index。

## 3. Owner 边界

```text
GET /api/video-favorites
  -> internal/api videolibrary.Handler
  -> internal/api VideoLibraryService.ListFavorites
  -> catalog.ListVideoFavoritesUsecase
  -> catalog.VideoLibraryReader.ListVideoFavorites
  -> catalog.video_user_states
  -> catalog.videos
  -> catalog.video_engagement_stats
```

边界说明：

- `internal/api` 负责 principal、query parsing、HTTP error mapping 和 response DTO。
- `catalog` 负责 cursor decode / encode、分页规则、可展示视频过滤、表读取和领域错误。
- `catalog.video_user_states` 是当前用户视频状态投影，不是收藏事件日志。
- `catalog.video_engagement_stats` 只提供 `view_count` preview；缺统计行返回 `0`。
- API 不写 Analytics、Learning Engine 或 Recommendation。
- API 不修改 favorite 状态；favorite 写入仍由 `PUT/DELETE /api/videos/{video_id}/favorite` 负责。

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
- cursor `kind` 不是 `video_favorites` 返回 `400 invalid_request`。
- request body 必须为空；即使客户端带 body，handler 也不读取或信任其中字段。

## 5. 响应契约

```ts
type VideoFavoritesResponse = {
  items: VideoFavoriteItem[];
  page: {
    limit: number;
    has_more: boolean;
    next_cursor: string | null;
  };
};

type VideoFavoriteItem = {
  video_id: string;
  title: string;
  cover_image_url: string | null;
  duration_seconds: number;
  view_count: number;
  favorited_at: string;
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
| `favorited_at` | `catalog.video_user_states.bookmarked_at` | 当前用户最近设置收藏为 true 的时间。 |
| `page.limit` | request normalized limit | 本页实际 limit。 |
| `page.has_more` | `limit + 1` 查询结果 | 是否还有下一页。 |
| `page.next_cursor` | 最后一条返回 item 生成 | 下一页 cursor；没有下一页时为 `null`。 |

`title / cover_image_url / duration_seconds / view_count` 是当前 Catalog preview，不是收藏时快照。当前后端没有 favorite snapshot 表，因此视频元数据变化会反映到下次列表读取。

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
last_position_ms
watch_session_id
favorite_id
```

原因：

- `description / video_url / transcript_url / like_count / favorite_count / user_state` 属于 Video Detail 真值。
- `learning_units / recommendation_run_id / occurrence_index` 属于 Feed occurrence 语义。
- `last_position_ms / watch_session_id` 属于观看进度或播放恢复语义。
- `favorite_id` 不暴露；视频收藏写 API 使用 path `video_id` 加 body `occurred_at` 表达一次幂等 set / unset 动作。

## 7. 可展示视频过滤

Catalog reader 必须使用与 Feed / Video Detail 一致的可展示视频 predicate：

```sql
v.status = 'active'
and v.visibility_status = 'public'
and (v.publish_at is null or v.publish_at <= now())
```

如果用户收藏的视频后来被下架、隐藏或设置为未来发布时间，本 API 不返回该视频。这样可以避免列表展示一个点击后 Video Detail 返回 `404 not_found` 的 item。

## 8. Cursor 分页

### 8.1 Cursor 格式

对外返回 opaque token，例如 base64url 编码后的 JSON。前端不解析 cursor。

内部 payload：

```json
{
  "kind": "video_favorites",
  "favorited_at": "2026-05-22T07:01:02.123Z",
  "video_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
}
```

字段规则：

- `kind` 用于防止不同列表 endpoint 的 cursor 混用。
- `favorited_at` 必须是有效 timestamp。
- `video_id` 必须是 UUID。
- cursor 中的字段只用于分页定位，不用于鉴权或跳过当前 endpoint 的过滤条件。

### 8.2 排序和下一页条件

固定排序：

```sql
order by state.bookmarked_at desc, state.video_id asc
```

下一页条件：

```sql
and (
  state.bookmarked_at < sqlc.arg(cursor_favorited_at)
  or (
    state.bookmarked_at = sqlc.arg(cursor_favorited_at)
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
  state.bookmarked_at as favorited_at
from catalog.video_user_states state
join catalog.videos v
  on v.video_id = state.video_id
left join catalog.video_engagement_stats stats
  on stats.video_id = v.video_id
where state.user_id = sqlc.arg(user_id)::uuid
  and state.has_bookmarked = true
  and state.bookmarked_at is not null
  and v.status = 'active'
  and v.visibility_status = 'public'
  and (v.publish_at is null or v.publish_at <= now())
order by state.bookmarked_at desc, state.video_id asc
limit sqlc.arg(limit_plus_one);
```

`duration_seconds` 在 service / mapper 层由 `duration_ms` 向上取整；`thumbnail_url` 在 API facade 层组装为公开 URL。

## 10. 索引建议

MVP 不新增业务表，但建议新增 partial index：

```sql
create index if not exists idx_video_user_states_favorites_page
on catalog.video_user_states (user_id, bookmarked_at desc, video_id asc)
where has_bookmarked = true and bookmarked_at is not null;
```

原因：

- `catalog.video_user_states` 以 `(user_id, video_id)` 为主键，不覆盖按收藏时间倒序分页。
- partial index 只覆盖当前收藏列表，避免把未收藏或空时间状态混进索引。
- `video_id` 作为 tie-breaker，保证同一时间戳下分页稳定。

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
- `has_bookmarked = true` 但 `bookmarked_at is null` 的历史脏数据不返回。
- 已收藏但视频不可展示时不返回，不单独报错。

## 12. 与 Video Interactions / Video Detail 的关系

- `PUT /api/videos/{video_id}/favorite` 和 `DELETE /api/videos/{video_id}/favorite` 是写入口；写请求必须携带 `occurred_at`，旧时间请求不会回滚当前收藏状态。
- `GET /api/video-favorites` 是当前用户收藏投影的列表读取入口。
- `GET /api/videos/{video_id}` 是播放页完整详情和当前用户 action rail base state 的权威读取入口。
- Favorite list 成功读取后可以给前端 seed video preview，但不应替代 Video Detail cache。
- 播放页取消收藏后，已打开的 favorite list 不要求实时删除旧 row；用户 refresh 后读取最新投影。

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
- history cursor 传给 favorites 返回 `400 invalid_request`。
- 空收藏列表返回 `200` 和空数组。
- 只返回当前用户 `has_bookmarked = true` 的视频。
- 不返回 inactive / private / future publish 视频。
- 缺 stats 返回 `view_count = 0`。
- 分页返回 `limit` 条、`has_more=true`、`next_cursor != null`。
- 第二页用 `next_cursor` 不重复、不漏掉同时间戳视频。

Catalog integration：

- 按 `bookmarked_at desc, video_id asc` 排序。
- 同一时间戳 tie-breaker 稳定。
- `bookmarked_at is null` 的脏数据不返回。
- cursor 解码后 SQL 条件正确推进。
- index migration 存在并能覆盖主查询路径。

成功标准：

1. 前端可以用 `GET /api/video-favorites` 分页读取当前用户收藏视频列表。
2. 列表 item 只包含 preview 和 `favorited_at`。
3. 列表与 Feed / Video Detail 的可展示视频规则一致。
4. 分页是 keyset cursor，不使用 offset。
5. Video Detail 继续作为播放详情和互动 base state 的唯一权威读取入口。
