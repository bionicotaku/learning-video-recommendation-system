# Feed API MVP 设计

## 1. 文档目标

本文定义后端 `POST /api/feed` 的当前前端契约。Feed API 只负责推荐列表 preview 和本轮 learning context；播放资源、transcript URL、description、互动计数和当前用户 like / favorite 状态不属于 Feed。

相关文档：

- [Video-Detail-API-MVP设计.md](Video-Detail-API-MVP设计.md)
- [Video-Interactions-API-MVP设计.md](Video-Interactions-API-MVP设计.md)
- [API模块总体设计规范.md](API模块总体设计规范.md)

## 2. 核心结论

```http
POST /api/feed
Content-Type: application/json
Authorization: Bearer <token>
```

Feed API 当前固定语义：

- 请求体只接受 `target_video_count` 和 `client_context`。
- response 顶层返回 `recommendation_run_id` 和 `items[]`。
- `items[]` 只返回列表 preview 字段、`view_count` 和 `learning_units[]`。
- 不返回 item-level recommendation id；前端用 `recommendation_run_id + item index` 派生本地 feed occurrence identity。
- 不暴露 Recommendation 内部字段：`rank / score / reason_codes / explanation / selector_mode / underfilled`。
- 不暴露播放详情字段：`description / video_url / transcript_url`。
- 不暴露 fullscreen 互动字段：`like_count / favorite_count / has_liked / has_favorited`。

播放详情由 `GET /api/videos/{video_id}` 提供。

## 3. 请求契约

```ts
type FeedRequest = {
  target_video_count?: number;
  client_context?: Record<string, unknown>;
};
```

字段语义：

| 字段 | 类型 | 必填 | 语义 |
|---|---:|---:|---|
| `target_video_count` | integer | 否 | 希望返回的视频数量。省略时后端默认 `8`；合法范围 `1..20`，显式传 `0` 返回 `400 invalid_request`。 |
| `client_context` | object | 否 | 前端环境上下文。省略时按空 object 处理。 |

Feed API 不接受 `preferred_duration_sec` 或 `session_hint`。请求 JSON 中出现未定义字段时返回 `400 invalid_request`。

## 4. 响应契约

```ts
type FeedLearningUnit = {
  coarse_unit_id: number;
  text: string;
  role: 'hard_review' | 'new_now' | 'soft_review' | 'near_future';
  is_primary: boolean;
  evidence_sentence_index: number;
  evidence_span_index: number;
  evidence_start_ms: number;
  evidence_end_ms: number;
};

type FeedItem = {
  video_id: string;
  title: string;
  cover_image_url: string | null;
  duration_seconds: number;
  view_count: number;
  learning_units: FeedLearningUnit[];
};

type FeedResponse = {
  recommendation_run_id: string;
  items: FeedItem[];
};
```

字段语义：

| 字段 | 来源 | 语义 |
|---|---|---|
| `recommendation_run_id` | Recommendation run | 本次推荐运行 ID，用于前端后续 exposure / lookup / quiz 等上报关联审计。 |
| `video_id` | Recommendation item | 视频稳定标识。 |
| `title` | `catalog.videos.title` | 列表 preview 标题。 |
| `cover_image_url` | `catalog.videos.thumbnail_url` 经 API URL 组装 | 列表 preview 封面；空路径或缺值返回 `null`。 |
| `duration_seconds` | Recommendation item `duration_ms` 向上取整 | 列表 preview 时长。 |
| `view_count` | `catalog.video_engagement_stats.view_count` | 全局观看数；缺统计行返回 `0`。 |
| `learning_units` | Recommendation item + `semantic.coarse_unit.label` | 本轮 feed 的 learning context；不是视频永久元数据。 |

## 5. 后端调用链

```text
POST /api/feed
  -> internal/api FeedService
  -> recommendation.GenerateVideoRecommendations
  -> catalog.FeedVideoLookupUsecase
  -> catalog.VideoPresentationReader.ListFeedVideosByIDs
  -> catalog.UnitLabelLookupUsecase
  -> catalog.UnitLabelReader.ListUnitLabelsByIDs
```

模块边界：

| 模块 | 职责 | 不负责 |
|---|---|---|
| `internal/api` | HTTP 解析、principal、调用 Recommendation 和 Catalog、组装前端 response。 | 不直接读写数据库，不拥有业务规则。 |
| `internal/recommendation` | 生成 `run_id + items[].video_id + duration_ms + learning_units`，写 audit / serving state。 | 不返回前端展示字段。 |
| `internal/catalog` | 批量读取 active/public/published 视频 preview 字段和 `view_count`。 | 不理解 Recommendation rank/role，不写状态。 |
| `internal/semantic` | 提供 active coarse unit label。 | 不参与推荐选择。 |

`ListFeedVideosByIDs` 只返回可展示视频：`catalog.videos.status = active`、`visibility_status = public`、且 `publish_at` 为空或已发布。

## 6. 错误与一致性边界

- 缺少 principal：`401 unauthorized`。
- 非 JSON object、非法字段、`target_video_count` 缺省以外的非 `1..20` 值：`400 invalid_request`。
- Recommendation 或下游超时 / 取消：`503 service_unavailable`。
- Recommendation 已写 audit / serving state 后，API facade 不静默丢弃 item。
- 缺少视频 preview 数据、`duration_ms <= 0`、非空 learning unit evidence 不完整、非空 unit label 缺失，都是后端一致性错误，返回 `500 internal_error`。
- `learning_units=[]` 合法，表示该 item 是 video-level 补全视频。

## 7. 成功标准

1. `POST /api/feed` 成功返回 `200`，且每个 item 只包含 preview 字段与 `learning_units[]`。
2. Feed response 不包含 `description / video_url / transcript_url / like_count / favorite_count / has_liked / has_favorited`。
3. Recommendation audit / serving state 与前端收到的 item 数量保持一致。
4. 播放详情和 action rail base state 统一由 `GET /api/videos/{video_id}` 获取。
