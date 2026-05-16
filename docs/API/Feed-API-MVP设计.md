# Feed API MVP 设计

## 0. 文档信息

文档状态：MVP 已实现，HTTP endpoint 已接入 `cmd/server`
目标读者：前端、后端、Recommendation / Catalog / API 维护者
当前范围：定义 feed 页面获取视频列表的前端契约、后端编排边界、字段来源、Recommendation audit 关系和成功语义。
当前明确不做：不把 Recommendation DTO 直接暴露给前端，不在 feed 响应里返回 quiz 题目，不在 `internal/recommendation` 中补前端展示字段。

关联文档：

- [API模块总体设计规范.md](API模块总体设计规范.md)
- [Catalog-观看进度上报MVP设计.md](Catalog-观看进度上报MVP设计.md)
- [学习事件上报API设计.md](学习事件上报API设计.md)
- [../推荐模块设计.md](../推荐模块设计.md)
- [../题目入库文档.md](../题目入库文档.md)

## 1. 一句话结论

Feed API 是一个前端展示 facade：

```text
POST /api/feed
  -> Recommendation.GenerateVideoRecommendations
  -> 批量补 Catalog 视频展示信息
  -> 批量补视频互动统计
  -> 批量补 semantic.coarse_unit 展示文本
  -> 返回前端 FeedResponse
```

Recommendation 仍只负责生成推荐计划：

```text
run_id + items[].video_id + items[].duration_ms + items[].learning_units
```

Feed API 负责把这份推荐计划转换成前端 feed 页面可直接展示和后续上报可直接使用的结构：

```text
recommendation_run_id + items[]
```

## 2. API 定位

### 2.1 为什么用 POST

Feed 请求会触发 Recommendation run，并写入 Recommendation audit 与 serving state。因此它不是纯读取接口，不应使用 `GET`。

### 2.2 endpoint

```http
POST /api/feed
Content-Type: application/json
```

`user_id` 不由前端传入。HTTP 层必须从认证 principal 中解析当前用户，并传给内部 usecase。

### 2.3 模块边界

| 模块 | 职责 | 不做什么 |
| --- | --- | --- |
| `internal/api` | HTTP handler、请求 validation、调用 Recommendation 和 Catalog unit label 读取能力、组装 FeedResponse。 | 不写 SQL，不拥有业务表，不实现推荐排序规则。 |
| `internal/recommendation` | 生成 `run_id + videos + learning_units`，写 Recommendation audit / serving state。 | 不返回 `title`、`cover_image_url`、`like_count` 等 feed 展示字段。 |
| `internal/catalog` | 提供按 `video_id[]` 批量读取视频展示信息和互动统计的能力。 | 不理解 Recommendation 的 `role`、`is_primary`、`rank`、`score`。 |
| Catalog unit label 读取能力 | 在 Catalog feed lookup reader 中提供按 `coarse_unit_id[]` 批量读取 `semantic.coarse_unit.label`。 | 不参与推荐排序或题目选择。 |

MVP 不新增独立 `internal/feed` 模块。当前 Feed API 只是 API 层组合编排，不是新的业务 owner。只有未来 Feed 开始拥有独立状态或规则，例如 feed session、cursor、运营插入、实验分流、跨推荐源混排，才考虑拆出 feed 模块。

## 3. 请求结构

### 3.1 Request

```json
{
  "target_video_count": 8,
  "preferred_duration_sec": {
    "min": 45,
    "max": 180
  },
  "session_hint": "normal",
  "client_context": {
    "platform": "ios",
    "app_version": "1.3.0",
    "os_version": "18.5",
    "device_model": "iPhone16,2"
  }
}
```

### 3.2 请求字段

| 字段 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `target_video_count` | integer | 否 | `8` | 希望返回的视频数量。MVP 建议限制为 `1..20`。 |
| `preferred_duration_sec.min` | integer | 否 | `45` | 偏好的最短视频时长，单位秒。字段缺失时补默认值；传入时必须为正数。 |
| `preferred_duration_sec.max` | integer | 否 | `180` | 偏好的最长视频时长，单位秒。字段缺失时补默认值；传入时必须为正数且大于等于 `min`。 |
| `session_hint` | string | 否 | 空字符串 | 推荐会话提示，例如 `normal`、`quick_review`、`deep_practice`。MVP 可先透传给 Recommendation，不做强语义承诺。 |
| `client_context` | object | 否 | `{}` | 客户端环境上下文，写入 Recommendation request context，用于排障和后续分析。 |

`client_context` 只要求是 JSON object，不固定字段集合。推荐继续使用 `platform`、`app_version`、`os_version`、`device_model` 四个基础字段。

## 4. 返回结构

### 4.1 Response

```json
{
  "recommendation_run_id": "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
  "items": [
    {
      "video_id": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
      "title": "How to stay focused while learning",
      "description": "A short clip about staying focused during practice.",
      "video_url": "https://cdn.example.com/hls/clip-001/playlist.m3u8",
      "cover_image_url": "https://cdn.example.com/covers/clip-001.webp",
      "duration_seconds": 92,
      "view_count": 1204,
      "like_count": 85,
      "favorite_count": 17,
      "learning_units": [
        {
          "coarse_unit_id": 101,
          "text": "focus",
          "role": "hard_review",
          "is_primary": true,
          "evidence_sentence_index": 12,
          "evidence_span_index": 4,
          "evidence_start_ms": 42310,
          "evidence_end_ms": 42880
        }
      ]
    }
  ]
}
```

### 4.2 顶层字段

| 字段 | 类型 | 来源 | 说明 |
| --- | --- | --- | --- |
| `recommendation_run_id` | string UUID | `GenerateVideoRecommendationsResponse.run_id` | 本次 feed 请求对应的推荐运行 ID。前端不解析，只在 exposure / lookup / quiz attempt / feedback 上报时原样带回。 |
| `items` | array | Recommendation 返回顺序 + Catalog/semantic 补齐 | Feed 视频列表。数组顺序就是展示顺序，不额外暴露 `rank`。 |

### 4.3 `FeedItem`

| 字段 | 类型 | 来源 | 说明 |
| --- | --- | --- | --- |
| `video_id` | string UUID | Recommendation item | 视频稳定 ID，用于列表 key、路由、fullscreen 定位和互动上报。 |
| `title` | string | `catalog.videos.title` | 视频标题，用于 feed 卡片、fullscreen、详情页展示。 |
| `description` | string | `catalog.videos.description` | 视频描述。数据库可空；API 缺省返回空字符串，保持前端字段稳定。 |
| `video_url` | string | `catalog.videos.hls_master_playlist_path` + URL 组装规则 | 视频播放地址。MVP 应返回前端可直接播放的 HLS master playlist URL。 |
| `cover_image_url` | string \| null | `catalog.videos.thumbnail_url` + URL 组装规则 | 视频封面地址；为空时前端显示 fallback。 |
| `duration_seconds` | integer | Recommendation `items[].duration_ms` | 视频总时长，单位秒。Feed facade 由毫秒向上取整，保证非 0 视频不会显示为 0 秒。Recommendation 已为推荐决策读取 `duration_ms`，这里不重复从 Catalog 读取。 |
| `view_count` | integer | `catalog.video_engagement_stats.view_count` | 全局观看数，只用于展示。无统计行时返回 `0`。 |
| `like_count` | integer | `catalog.video_engagement_stats.like_count` | 全局点赞数，作为 action rail 基础 count。无统计行时返回 `0`。 |
| `favorite_count` | integer | `catalog.video_engagement_stats.favorite_count` | 全局收藏数，作为 action rail 基础 count。无统计行时返回 `0`。 |
| `learning_units` | array | Recommendation item + semantic label 补齐 | 本视频在本次推荐中预期承载的学习单元。不是完整词表，最多约 `1..8` 个。 |

### 4.4 `FeedLearningUnit`

| 字段 | 类型 | 来源 | 说明 |
| --- | --- | --- | --- |
| `coarse_unit_id` | integer | Recommendation `learning_units[].coarse_unit_id` | 学习单元 ID，对应 `semantic.coarse_unit.id`。用于 exposure / lookup / quiz attempt 绑定学习对象。 |
| `text` | string | `semantic.coarse_unit.label` | 学习单元文本，例如单词、短语或表达本身。 |
| `role` | string | Recommendation `learning_units[].role` | 本轮推荐中的学习角色：`hard_review` / `new_now` / `soft_review` / `near_future`。 |
| `is_primary` | boolean | Recommendation `learning_units[].is_primary` | 是否是这个视频的主学习目标。 |
| `evidence_sentence_index` | integer | Recommendation `learning_units[].evidence.sentence_index` | 最佳证据命中的字幕句子序号。 |
| `evidence_span_index` | integer | Recommendation `learning_units[].evidence.span_index` | 最佳证据命中的句内 span 序号。 |
| `evidence_start_ms` | integer | Recommendation `learning_units[].evidence.start_ms` | 最佳证据开始时间，单位毫秒。用于字幕高亮、跳转、exposure 候选定位。 |
| `evidence_end_ms` | integer | Recommendation `learning_units[].evidence.end_ms` | 最佳证据结束时间，单位毫秒。 |

MVP Feed API 对前端返回的 `FeedLearningUnit` 要求 evidence 字段完整。若 Recommendation 返回某个 unit 但 evidence 缺失，说明 Catalog evidence 链路存在数据质量问题。Feed facade 不做静默裁剪，必须返回 `500 internal_error` 并记录服务端 error。

## 5. 字段裁剪规则

Recommendation plan response 只包含以下字段：

```text
run_id
items[].video_id
items[].duration_ms
items[].learning_units
```

Feed API 只暴露：

```text
recommendation_run_id
items[] order
items[].video_id
items[].duration_ms -> duration_seconds
items[].learning_units
```

以下字段不返回前端：

| 字段 | 不返回原因 |
| --- | --- |
| `selector_mode` | 推荐供给状态和选择策略，属于后端调试 / audit 信息。 |
| `underfilled` | feed 页面只关心实际拿到的 items；供给不足由后端监控和 audit 处理。 |
| `rank` | 数组顺序已经表达展示顺序，不需要让前端消费 rank。 |
| `score` | 排序分是后端内部信号，前端不应展示或分支依赖。 |
| `reason_codes` | 当前 feed 页面不展示推荐解释；保留在 audit 中即可。 |
| `explanation` | 当前 feed 页面不展示推荐解释；如未来需要解释卡片，再设计独立展示契约。 |

## 6. 后端编排流程

```text
1. HTTP handler 从 principal 解析 user_id。
2. 校验请求 body：
   - target_video_count 在允许范围内；
   - preferred_duration_sec 合法；
   - client_context 是 JSON object。
3. 调用 Recommendation.GenerateVideoRecommendations。
   - Recommendation 生成 run_id；
   - 写 recommendation.video_recommendation_runs；
   - 写 recommendation.video_recommendation_items；
   - 更新 Recommendation serving state。
4. 从 Recommendation response 收集：
   - video_ids；
   - duration_ms；
   - coarse_unit_ids；
   - learning_units evidence。
5. 批量读取视频展示信息：
   - catalog.videos 的 `title`、`description`、`hls_master_playlist_path`、`thumbnail_url`；
   - catalog.video_engagement_stats。
6. 批量读取 unit 展示文本：
   - semantic.coarse_unit.label。
7. 按 Recommendation response 的 items[] 顺序组装 Feed items[]。
8. 裁剪 Recommendation 内部字段，返回 FeedResponse。
```

URL 组装规则：

- `hls_master_playlist_path` 或 `thumbnail_url` 已是 `http://` / `https://` 绝对 URL 时原样返回。
- 相对路径使用 `PUBLIC_ASSET_BASE_URL` 作为前缀，并清理重复 `/`。
- `PUBLIC_ASSET_BASE_URL` 是 `cmd/server` 必填配置，必须是绝对 http(s) URL。

## 7. 数据来源

| Feed 字段 | 数据来源 |
| --- | --- |
| `recommendation_run_id` | `recommendation.video_recommendation_runs.run_id` / usecase response `run_id` |
| `items[].video_id` | `recommendation.video_recommendation_items.video_id` / usecase response |
| `title` | `catalog.videos.title` |
| `description` | `catalog.videos.description` |
| `video_url` | `catalog.videos.hls_master_playlist_path` 经 API 层 URL 组装 |
| `cover_image_url` | `catalog.videos.thumbnail_url` 经 API 层 URL 组装 |
| `duration_seconds` | `ceil(RecommendationPlanItem.duration_ms / 1000)` |
| `view_count` | `catalog.video_engagement_stats.view_count` |
| `like_count` | `catalog.video_engagement_stats.like_count` |
| `favorite_count` | `catalog.video_engagement_stats.favorite_count` |
| `learning_units[].coarse_unit_id` | Recommendation `learning_units` |
| `learning_units[].text` | `semantic.coarse_unit.label` |
| `learning_units[].role` | Recommendation `learning_units` |
| `learning_units[].is_primary` | Recommendation `learning_units` |
| `learning_units[].evidence_*` | Recommendation `learning_units[].evidence` |

Feed facade 的 batch 补齐范围只包括：

```text
title
description
video_url
cover_image_url
view_count
like_count
favorite_count
learning_units[].text
```

`duration_seconds` 不在这个 batch 查询范围内，直接来自 Recommendation plan item 的 `duration_ms`。

## 8. 错误响应

错误 envelope、request id、principal 规则统一遵守 [API模块总体设计规范.md](API模块总体设计规范.md)。

| HTTP 状态 | 场景 |
| --- | --- |
| `400 Bad Request` | JSON 格式错误、字段类型错误、`target_video_count` 越界、`preferred_duration_sec` 非法。 |
| `401 Unauthorized` | 未登录或 principal 缺失。 |
| `500 Internal Server Error` | Recommendation pipeline、Catalog 批量读取或 semantic 批量读取失败。 |
| `503 Service Unavailable` | 请求超时或 context canceled。 |

MVP 不把“推荐结果少于请求数量”作为错误。Recommendation 自己会记录 `underfilled` 到 audit；Feed API 返回 Recommendation plan 中实际生成的 items。

但 Feed facade 对 Recommendation plan 采用完整补齐语义：缺少视频展示数据、`duration_ms <= 0`、unit evidence 不完整、unit label 缺失、URL 组装失败，都是后端数据一致性错误，返回 `500 internal_error`。这样 Recommendation audit / serving state 不会和前端实际收到的 feed item 静默分叉。

## 9. 前端使用约定

Feed 页面只使用本 API 做列表展示和进入 fullscreen 的初始数据。

前端点进某个视频后，视频文件、字幕文件继续走各自的内容读取路径，不由 Feed API 内联返回。

前端应保存当前 item 的：

```text
recommendation_run_id
video_id
learning_units[]
```

这些字段用于后续：

- 字幕高亮与 exposure 候选判断；
- lookup / exposure 上报；
- 视频末尾批量 quiz 取题；
- quiz attempt 上报时的推荐来源归因。

## 10. 成功标准

实现本 API 时至少满足：

1. 一次成功请求会生成 Recommendation run，并写入 `recommendation.video_recommendation_runs` 与 `recommendation.video_recommendation_items`。
2. response 不暴露 `rank`、`score`、`reason_codes`、`explanation`、`selector_mode`、`underfilled`。
3. response 中每个 `FeedItem` 都包含前端展示所需视频字段和至少一个 evidence 完整的 `learning_unit`。
4. `items[]` 顺序严格保持 Recommendation 返回顺序。
5. Catalog / semantic 补充读取必须按 batch 完成，不允许按 item 逐条查询。
