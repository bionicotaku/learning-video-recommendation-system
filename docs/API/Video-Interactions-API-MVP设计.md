# Video Interactions API MVP 设计

## 1. API 定位

Video Interactions API 维护当前用户对单个视频的点赞和收藏状态，并同步维护视频级全局计数。

它属于 Catalog 写侧能力：

```text
internal/api videointeractions.Handler
  -> catalog.SetVideoLikeUsecase / SetVideoFavoriteUsecase
  -> catalog.video_user_states
  -> catalog.video_engagement_stats
```

本 API 不是学习事件，不进入 Learning Engine；不是推荐曝光，不进入 Recommendation；MVP 不新增点赞/收藏审计表，不写 Analytics，不记录成功业务 log。

现有表已经满足 MVP：

| 表 | 字段 | 用途 |
|---|---|---|
| `catalog.video_user_states` | `has_liked`, `liked_at` | 当前用户是否点赞该视频及最近设置时间。 |
| `catalog.video_user_states` | `has_bookmarked`, `bookmarked_at` | 当前用户是否收藏该视频及最近设置时间。 |
| `catalog.video_engagement_stats` | `like_count` | 当前点赞该视频的用户数。 |
| `catalog.video_engagement_stats` | `favorite_count` | 当前收藏该视频的用户数。 |

## 2. Endpoints

```http
PUT    /api/videos/{video_id}/like
DELETE /api/videos/{video_id}/like
PUT    /api/videos/{video_id}/favorite
DELETE /api/videos/{video_id}/favorite
```

请求不需要 body，也不要求 `Content-Type`。

`user_id` 只能从 trusted principal 获取。客户端不能通过 body、query 或 path 传入用户身份。

`video_id` 必须是 UUID。视频必须可交互：

```sql
catalog.videos.status = 'active'
and catalog.videos.visibility_status = 'public'
and (catalog.videos.publish_at is null or catalog.videos.publish_at <= now())
```

不存在、未发布、非 public、非 active 的视频统一返回 `404 not_found`。

## 3. Response

点赞 API 只返回点赞相关字段：

```ts
type VideoLikeResponse = {
  video_id: string;
  has_liked: boolean;
  like_count: number;
};
```

收藏 API 只返回收藏相关字段：

```ts
type VideoFavoriteResponse = {
  video_id: string;
  has_favorited: boolean;
  favorite_count: number;
};
```

示例：

```json
{
  "video_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
  "has_liked": true,
  "like_count": 86
}
```

```json
{
  "video_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
  "has_favorited": false,
  "favorite_count": 17
}
```

`favorite` 是 API 产品语义；数据库字段继续使用既有 `has_bookmarked` / `bookmarked_at` / `favorite_count`。

## 4. Semantics

四个接口都是幂等 set / unset，不做 toggle。

| Endpoint | 目标状态 | 重复请求 |
|---|---|---|
| `PUT /api/videos/{video_id}/like` | `has_liked = true` | 不重复增加 `like_count`。 |
| `DELETE /api/videos/{video_id}/like` | `has_liked = false` | 不重复减少 `like_count`。 |
| `PUT /api/videos/{video_id}/favorite` | `has_bookmarked = true` | 不重复增加 `favorite_count`。 |
| `DELETE /api/videos/{video_id}/favorite` | `has_bookmarked = false` | 不重复减少 `favorite_count`。 |

Repository 在单个数据库事务内更新用户状态和全局计数：

- `PUT` 在插入新状态行，或从 false 改成 true 时，计数 `+1`。
- `PUT` 如果当前已经是 true，计数 delta 为 `0`。
- `DELETE` 仅当当前是 true 时改成 false，计数 `-1`。
- `DELETE` 如果没有用户状态行或当前已经是 false，计数 delta 为 `0`。
- `DELETE` 不创建空的 `catalog.video_user_states` 行。
- 计数更新使用 `greatest(0, count + delta)` 语义防御历史脏数据。

## 5. Errors

| HTTP | code | 场景 |
|---|---|---|
| `200 OK` | - | 状态已设置，或本来就是目标状态。 |
| `400 Bad Request` | `invalid_request` | `video_id` 不是 UUID。 |
| `401 Unauthorized` | `unauthorized` | trusted principal 缺失。 |
| `404 Not Found` | `not_found` | 视频不存在或不可交互。 |
| `500 Internal Server Error` | `internal_error` | 数据库或未知服务端错误。 |

错误响应使用 API 模块统一 JSON error envelope。

## 6. 前端调用视角

Fullscreen action rail 使用 `GET /api/videos/{video_id}` 返回的 `like_count`、`favorite_count`、`user_state.has_liked` 和 `user_state.has_favorited` 初始化全局计数与当前用户状态。用户点击后调用对应 set / unset API，并用响应中的单类状态覆盖本地 UI：

- 点赞按钮只消费 `VideoLikeResponse`。
- 收藏按钮只消费 `VideoFavoriteResponse`。
- 前端不要依赖一次点赞请求顺带刷新收藏状态，反之亦然。
- 失败时前端保留或回滚 optimistic UI，由客户端交互策略决定。

## 7. 不做事项

MVP 不新增：

- `analytics.video_interaction_events`
- `catalog.video_like_events`
- `catalog.video_favorite_events`
- delivery / session / audit 表
- 成功请求业务 log
- toggle API
- body/query 传入的 `user_id`

后续如果需要历史轨迹、反作弊、增长分析、计数重建，再单独设计 append-only event 表。

## 8. 当前实现入口

| 层 | 文件 |
|---|---|
| HTTP handler | `internal/api/infrastructure/http/handler/videointeractions` |
| Server wiring | `cmd/server/wiring.go` |
| Catalog DTO | `internal/catalog/application/dto/video_interactions.go` |
| Catalog usecase | `internal/catalog/application/service/set_video_like.go`, `set_video_favorite.go` |
| Repository port | `internal/catalog/application/repository/video_interaction_writer.go` |
| Domain model | `internal/catalog/domain/model/video_interaction.go` |
| SQL | `internal/catalog/infrastructure/persistence/query/video_interactions.sql` |
| Repository impl | `internal/catalog/infrastructure/persistence/repository/video_interaction_writer.go` |

## 9. 测试要求

目标测试：

```bash
go test ./internal/api/test/integration/videointeractions
go test ./internal/catalog/test/unit/application/service -run VideoInteraction
go test ./internal/catalog/test/integration/repository -tags=integration -run VideoInteraction
```

验收点：

- 首次 `PUT /like` 返回 `has_liked=true`，`like_count + 1`。
- 重复 `PUT /like` 不重复增加 count。
- `DELETE /like` 返回 `has_liked=false`，`like_count - 1`。
- 重复 `DELETE /like` 不重复减少 count。
- favorite set / unset / idempotency 同样覆盖。
- like response 不包含 favorite 字段。
- favorite response 不包含 like 字段。
- invalid `video_id` 返回 `400 invalid_request`。
- missing principal 返回 `401 unauthorized`。
- inactive / private / future / missing video 返回 `404 not_found`。
- repository integration 验证 user state 与 engagement stats 同事务结果一致。
