# Video Detail API MVP 设计

## 1. 文档目标

本文定义 `GET /api/videos/{video_id}` 的后端契约。Video Detail API 是 fullscreen 播放页读取单个视频详情的权威来源，负责播放资源、transcript URL、description、互动计数和当前用户 like / favorite 状态。

相关文档：

- [Feed-API-MVP设计.md](Feed-API-MVP设计.md)
- [Video-Interactions-API-MVP设计.md](Video-Interactions-API-MVP设计.md)
- [Catalog-观看进度上报MVP设计.md](Catalog-观看进度上报MVP设计.md)

## 2. 核心结论

```http
GET /api/videos/{video_id}
Authorization: Bearer <token>
```

固定语义：

- `auth: required`。
- request body 为空。
- `video_id` 来自 path，必须是 UUID。
- 只返回 active / public / 已发布视频。
- 不写 Analytics、Learning Engine 或 Recommendation。
- 不修改 like / favorite 状态；写入仍由 Video Interactions API 负责。

## 3. 响应契约

```ts
type VideoDetailResponse = {
  video_id: string;
  title: string;
  description: string;
  video_url: string;
  cover_image_url: string | null;
  transcript_url: string | null;
  duration_seconds: number;
  view_count: number;
  like_count: number;
  favorite_count: number;
  user_state: {
    has_liked: boolean;
    has_favorited: boolean;
  };
};
```

字段语义：

| 字段 | 来源 | 语义 |
|---|---|---|
| `video_id` | `catalog.videos.video_id` | 视频稳定标识。 |
| `title` | `catalog.videos.title` | fullscreen/detail 标题真值。 |
| `description` | `catalog.videos.description` | fullscreen 详情文案；数据库空值返回空字符串。 |
| `video_url` | `catalog.videos.video_object_path` 经 API URL 组装 | 播放器使用的播放资源 URL；空路径是后端数据错误。 |
| `cover_image_url` | `catalog.videos.thumbnail_url` 经 API URL 组装 | 封面 URL；空路径或缺值返回 `null`。 |
| `transcript_url` | `catalog.video_transcripts.transcript_object_path` 经 API URL 组装 | transcript asset JSON URL；缺 transcript 行或空路径返回 `null`。 |
| `duration_seconds` | `catalog.videos.duration_ms` 向上取整 | 视频时长。 |
| `view_count` | `catalog.video_engagement_stats.view_count` | 全局观看数；缺统计行返回 `0`。 |
| `like_count` | `catalog.video_engagement_stats.like_count` | 全局点赞数；缺统计行返回 `0`。 |
| `favorite_count` | `catalog.video_engagement_stats.favorite_count` | 全局收藏数；缺统计行返回 `0`。 |
| `user_state.has_liked` | `catalog.video_user_states.has_liked` | 当前用户是否已点赞；缺用户状态行返回 `false`。 |
| `user_state.has_favorited` | `catalog.video_user_states.has_bookmarked` | 当前用户是否已收藏；缺用户状态行返回 `false`。 |

## 4. 后端调用链

```text
GET /api/videos/{video_id}
  -> internal/api videodetail.Handler
  -> internal/api VideoDetailService
  -> catalog.GetVideoDetailUsecase
  -> catalog.VideoPresentationReader.GetVideoDetailByID
  -> catalog.videos
  -> catalog.video_transcripts
  -> catalog.video_engagement_stats
  -> catalog.video_user_states
```

Catalog reader 使用与 Feed / End Quiz 一致的可展示视频 predicate：

```sql
status = 'active'
and visibility_status = 'public'
and (publish_at is null or publish_at <= now())
```

## 5. 错误与边界

- 缺少 principal：`401 unauthorized`。
- `video_id` 不是 UUID：`400 invalid_request`。
- 视频不存在、inactive、private 或 future publish：`404 not_found`。
- 下游超时 / 取消：`503 service_unavailable`。
- `video_object_path` 为空或 URL 组装失败：`500 internal_error`。
- 缺 transcript、缺 stats、缺 user state 都不是错误，分别按 `null`、`0`、`false` 返回。

## 6. 与其他 API 的关系

- Feed API 只返回列表 preview 和 learning context，不返回播放详情或互动 base state。
- Video Detail API 返回 action rail 的 base count 与当前用户 base flags。
- Video Interactions API 只负责携带 `occurred_at` 的 like / favorite 幂等写入，并返回单类写入结果；旧时间写请求不会回滚当前状态。它不替代 Video Detail 的完整读取快照。
- Watch Progress API 可以维护 `view_count` 等消费投影，但不修改 like / favorite 字段。

## 7. 成功标准

1. `GET /api/videos/{video_id}` 对可展示视频返回完整 detail response。
2. 缺 transcript 时 `transcript_url` 为 `null`。
3. 缺 stats / user state 时 count 为 `0`，flags 为 `false`。
4. inactive / private / future publish 视频不可读取。
