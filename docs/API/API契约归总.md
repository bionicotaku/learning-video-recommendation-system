# API 契约归总

本文归总当前 `internal/api` 已注册的业务 HTTP API 契约。口径按“当前实现优先，`docs/API` 业务文档补充语义”维护：

- 路由清单以 `internal/api/infrastructure/http/handler/**/handler.go` 的 `RegisterRoutes` 为准。
- 请求 / 响应字段以 handler、API DTO、业务 usecase DTO 和 integration test 为准。
- 业务语义、owner 边界、重试建议和非目标范围以各业务 API 设计文档为准。
- 本文不定义未实现的 OpenAPI / SDK，也不替代单个 API 文档中的背景设计。

## 1. 全局契约

### 1.1 认证

所有当前业务 endpoint 都要求登录。`user_id` 永远来自 trusted principal，不接受 body、query、path 或普通 header 中的用户身份。

生产链路：

```text
Authorization: Bearer <JWT>
  -> GCP API Gateway / auth provider verify
  -> X-Apigateway-Api-Userinfo
  -> internal/api principal extraction
```

实现确认：

- 后端优先解析 `X-Apigateway-Api-Userinfo`。
- 仅当 `DEV_MODE=true` 且 gateway userinfo header 缺失时，才从 `Authorization: Bearer <JWT>` 的 payload 解码 `sub`，用于本地 / 联调。
- gateway userinfo header 存在但非法时，不 fallback 到 Authorization。
- principal 缺失或 principal 指向不存在的 Auth user 时，返回 `401 unauthorized`。

### 1.2 Content-Type、body 和 unknown field

JSON 写 API 必须带：

```http
Content-Type: application/json
```

JSON body 必须是单个 JSON object。空 body、数组、字符串、多段 JSON 或 unknown field 默认返回 `400 invalid_request`。`PATCH /api/me/profile` 使用 map 解析后手工拒绝未允许字段；效果同样是 unknown field 返回 `400 invalid_request`。

`GET` API 当前不读取 request body。

唯一 multipart API 是：

```http
POST /api/feedback
Content-Type: multipart/form-data
```

### 1.3 Body size

实现确认：

| 范围 | 限制 |
|---|---:|
| `/api/feedback` | 5 MiB |
| 其他会读取 request body 的当前 API | 1 MiB |

`/api/feedback` 超过 5 MiB 返回 `413 payload_too_large`。其他会读取 request body 的当前 API 超过 1 MiB 也返回 `413 payload_too_large`。GET API 当前不读取 request body，因此不会主动按 body 大小触发该错误。

### 1.4 时间字段

全局规范要求 request 中的 `*_at` 使用 RFC3339 / RFC3339Nano，并带显式 `Z` 或 offset。handler 会把时间转为 UTC 传给 usecase。

允许：

```json
"2026-05-15T17:00:01Z"
"2026-05-15T10:00:01-07:00"
```

不允许：

```json
"2026-05-15T10:00:01"
"2026-05-15"
```

实现差异需要注意：

- Video Interactions 的 `occurred_at` 通过 Go `time.Time` JSON unmarshal 解析，仍要求 RFC3339 时间点。
- Watch Progress 的 `occurred_at` 是可选字段；缺省时 Catalog usecase 使用服务端当前时间。

### 1.5 错误 envelope

所有 API 错误使用统一 JSON：

```json
{
  "error": {
    "code": "invalid_request",
    "message": "events must not be empty",
    "details": [],
    "request_id": "req_..."
  }
}
```

实现确认：`details` 当前总是输出数组；没有字段级详情时为 `[]`。

通用错误码：

| HTTP | code | 通用场景 |
|---:|---|---|
| 400 | `invalid_request` | JSON、字段类型、required、UUID、时间、枚举、数组大小、cursor、query limit 等请求契约错误。 |
| 401 | `unauthorized` | trusted principal 缺失或认证主体不可用。 |
| 404 | `not_found` | 资源不存在或业务文档要求隐藏不可访问资源。 |
| 409 | `conflict` | 请求与当前资源状态冲突，原样重试不会自然成功。 |
| 413 | `payload_too_large` | request body 超过当前 endpoint 限制；`/api/feedback` 是 5 MiB，其他当前 API 是 1 MiB。 |
| 422 | `unprocessable_entity` | 请求格式正确，但业务状态或引用对象拒绝；包括真实业务主键不存在、Catalog 时间异常等。 |
| 500 | `internal_error` | 数据库、URL 组装、数据一致性或未知服务端错误。 |
| 503 | `service_unavailable` | request context 取消 / 超时，或 API application service 明确返回依赖不可用。 |

## 2. 已实现 API 总表

当前实现注册 26 个业务 endpoint。

| Method | Path | 分组 | Owner / 编排 | 成功边界 |
|---|---|---|---|---|
| `POST` | `/api/feed` | Feed | API facade -> Recommendation + Catalog + Semantic label | 生成推荐 plan 并补齐 feed preview 字段后返回。 |
| `GET` | `/api/videos/{video_id}` | Video Detail | Catalog | 读取单个可展示视频详情和当前用户互动 base state。 |
| `GET` | `/api/video-favorites` | Video Library | Catalog | 分页读取当前用户仍收藏且仍可展示的视频。 |
| `GET` | `/api/video-history` | Video Library | Catalog | 分页读取当前用户最近观看且仍可展示的视频。 |
| `POST` | `/api/videos/end-quiz` | End Quiz | Catalog | 按 `video_id + coarse_unit_ids` 只读获取 quiz 候选。 |
| `GET` | `/api/me` | Me | User | 读取 profile、累计 stats 和 7 天 activity calendar；可 lazy repair / 顺手更新 timezone。 |
| `PATCH` | `/api/me/profile` | Me | User | 更新允许前端维护的 profile 字段并返回 profile 子集。 |
| `GET` | `/api/unit-collections` | Unit Collections | API facade -> Semantic + Learning Engine | 读取 active 词书列表和当前 active slug / null。 |
| `GET` | `/api/learning-targets/active-coarse-unit-ids` | Learning Targets | Learning Engine | 读取当前未 mastered target coarse unit ids。 |
| `PUT` | `/api/learning-targets/active-collection` | Learning Targets | API facade -> Learning Engine + User | 同事务激活词书 target projection，并更新 onboarding status。 |
| `PUT` | `/api/videos/{video_id}/like` | Video Interactions | Catalog | 幂等设置当前用户已点赞。 |
| `DELETE` | `/api/videos/{video_id}/like` | Video Interactions | Catalog | 幂等设置当前用户未点赞。 |
| `PUT` | `/api/videos/{video_id}/favorite` | Video Interactions | Catalog | 幂等设置当前用户已收藏。 |
| `DELETE` | `/api/videos/{video_id}/favorite` | Video Interactions | Catalog | 幂等设置当前用户未收藏。 |
| `POST` | `/api/word-favorites/status` | Word Favorites | Catalog | 查询当前词 / 字幕 token identity 是否已收藏，可选返回视频句子上下文。 |
| `PUT` | `/api/word-favorites` | Word Favorites | Catalog | 幂等收藏当前词 / 字幕 token identity。 |
| `DELETE` | `/api/word-favorites` | Word Favorites | Catalog | 幂等取消收藏当前词 / 字幕 token identity。 |
| `GET` | `/api/word-favorites` | Word Favorites | Catalog | 分页读取当前用户收藏的词 / 字幕 token 展示列表。 |
| `POST` | `/api/video-watch-progress` | Watch Progress | Catalog + User stats projection | 写 watch session ledger、视频消费投影和活动统计。 |
| `POST` | `/api/learning-interactions:batch` | Learning Events | Analytics + Learning Engine best-effort normalizer | 整批写入 exposure / lookup raw facts；HTTP success 只承诺 raw accepted。 |
| `POST` | `/api/quiz-attempts` | Learning Events | Analytics + Learning Engine best-effort normalizer + User stats | 写入 completed quiz attempt raw fact。 |
| `POST` | `/api/learning-units:mark-mastered` | Learning Events | Analytics + Learning Engine best-effort normalizer | 写入 self mark mastered raw fact；要求已有 user-unit state。 |
| `POST` | `/api/learning-units:reset-unlearned` | Learning Events | Learning Engine reducer | 直接写 reset normalized event 并同步重置状态。 |
| `GET` | `/api/learning/unit-progress/mastered` | Unit Progress | Learning Engine + Semantic read model | 分页读取已掌握学习单元。 |
| `GET` | `/api/learning/unit-progress/unmastered` | Unit Progress | Learning Engine + Semantic read model | 分页读取尚未掌握的目标学习单元。 |
| `POST` | `/api/feedback` | Feedback | User | 原子写入 feedback submission 与 JPEG 图片。 |

## 3. Endpoint 详情

### 3.1 `POST /api/feed`

文档来源：[Feed-API-MVP设计.md](Feed-API-MVP设计.md)；总体约束来源：[API模块总体设计规范.md](API模块总体设计规范.md)。

用途：获取当前用户 feed 推荐视频列表。Feed 只返回列表 preview 和本轮 learning context，不返回播放资源、transcript、description、like/favorite count 或当前用户互动状态。

Auth：必需。

Owner：API facade 编排 Recommendation、Catalog feed lookup、Catalog/Semantic unit label lookup。

Request body：

| 字段 | 类型 | 必需 | 默认值 | 规则 |
|---|---|---:|---|---|
| `target_video_count` | integer | 否 | `8` | 合法范围 `1..20`；省略时默认 `8`，显式 `0`、负数或 `>20` 返回 `400 invalid_request`。 |
| `client_context` | object | 否 | `{}` | 只校验 JSON object，字段集合可扩展。 |

Response `200 OK`：

| 字段 | 类型 | 说明 |
|---|---|---|
| `recommendation_run_id` | string UUID | 本次推荐运行 ID，用于 exposure / lookup / quiz 归因。 |
| `items` | array | Feed item 列表。 |
| `items[].video_id` | string UUID | 视频 ID。 |
| `items[].title` | string | 列表标题。 |
| `items[].cover_image_url` | string \| null | 封面 URL；缺失或空路径返回 `null`。 |
| `items[].duration_seconds` | integer | `duration_ms` 向上取整。 |
| `items[].view_count` | integer | 全局观看数；缺统计行返回 `0`。 |
| `items[].learning_units` | array | 本轮 feed learning context。空数组合法。 |
| `items[].learning_units[].coarse_unit_id` | integer | 学习单元 ID。 |
| `items[].learning_units[].text` | string | 展示文本，来自 unit label lookup。 |
| `items[].learning_units[].role` | string | `hard_review` / `new_now` / `soft_review` / `near_future` 等 Recommendation role。 |
| `items[].learning_units[].is_primary` | boolean | 是否主学习单元。 |
| `items[].learning_units[].evidence_sentence_index` | integer | 字幕句索引。 |
| `items[].learning_units[].evidence_span_index` | integer | span 索引。 |
| `items[].learning_units[].evidence_start_ms` | integer | evidence 开始时间。 |
| `items[].learning_units[].evidence_end_ms` | integer | evidence 结束时间。 |

Errors：

| HTTP | code | 场景 |
|---:|---|---|
| 400 | `invalid_request` | 非 JSON object、unknown field、`target_video_count` 越界、`client_context` 不是 object。 |
| 401 | `unauthorized` | principal 缺失。 |
| 500 | `internal_error` | 缺视频 preview、`duration_ms <= 0`、非空 learning unit evidence 不完整、非空 unit label 缺失、URL 组装失败或未知错误。 |
| 503 | `service_unavailable` | request 取消 / 超时或依赖不可用。 |

Side effects：Recommendation 会写 audit / serving state；API facade 不写数据库。

Retry：读式获取接口，可重试；同一次 feed 失败后重试可能生成新的 `recommendation_run_id`。

### 3.2 `GET /api/videos/{video_id}`

文档来源：[Video-Detail-API-MVP设计.md](Video-Detail-API-MVP设计.md)；总体约束来源：[API模块总体设计规范.md](API模块总体设计规范.md)。

用途：fullscreen 播放页读取单个视频详情、播放资源、transcript URL、互动计数和当前用户 base state。

Auth：必需。

Owner：Catalog read usecase；API 负责 public asset URL 组装。

Path：

| 参数 | 类型 | 必需 | 规则 |
|---|---|---:|---|
| `video_id` | string UUID | 是 | 非 UUID 返回 `400 invalid_request`。 |

Response `200 OK`：

| 字段 | 类型 | 说明 |
|---|---|---|
| `video_id` | string UUID | 视频 ID。 |
| `title` | string | 视频标题。 |
| `description` | string | 视频详情文案；数据库空值映射为空字符串。 |
| `video_url` | string | 播放资源 URL；空 object path 或 base URL 缺失是服务端错误。 |
| `cover_image_url` | string \| null | 封面 URL；缺失或空路径返回 `null`。 |
| `transcript_url` | string \| null | transcript asset JSON URL；缺 transcript 行或空路径返回 `null`。 |
| `duration_seconds` | integer | 视频时长秒数，按 `duration_ms` 向上取整。 |
| `view_count` | integer | 全局观看数，缺 stats 行返回 `0`。 |
| `like_count` | integer | 全局点赞数，缺 stats 行返回 `0`。 |
| `favorite_count` | integer | 全局收藏数，缺 stats 行返回 `0`。 |
| `user_state.has_liked` | boolean | 当前用户是否已点赞；缺状态行返回 `false`。 |
| `user_state.has_favorited` | boolean | 当前用户是否已收藏；缺状态行返回 `false`。 |

Errors：

| HTTP | code | 场景 |
|---:|---|---|
| 400 | `invalid_request` | `video_id` 不是 UUID。 |
| 401 | `unauthorized` | principal 缺失。 |
| 404 | `not_found` | 视频不存在、inactive、private 或 future publish。 |
| 500 | `internal_error` | `video_object_path` 为空、URL 组装失败、数据库或未知错误。 |
| 503 | `service_unavailable` | request 取消 / 超时。 |

Side effects：无。

Retry：可安全重试。

### 3.3 `GET /api/video-favorites`

文档来源：[Video-Favorites-API-MVP设计.md](Video-Favorites-API-MVP设计.md)；总体约束来源：[API模块总体设计规范.md](API模块总体设计规范.md)。

用途：分页读取当前用户仍收藏且仍可展示的视频列表。该 API 只返回列表 preview 和 `favorited_at`，不返回播放详情字段。

Auth：必需。

Owner：Catalog read usecase；API 负责 public asset URL 组装。

Query：

| 参数 | 类型 | 必需 | 默认值 | 规则 |
|---|---|---:|---|---|
| `limit` | integer | 否 | `20` | `1..100`；非整数、`0`、`>100` 返回 `400 invalid_request`。 |
| `cursor` | string | 否 | 无 | opaque cursor；空白会 trim；解码失败或 cursor kind 不是 `video_favorites` 返回 `400 invalid_request`。 |

Response `200 OK`：

| 字段 | 类型 | 说明 |
|---|---|---|
| `items` | array | 收藏视频列表。 |
| `items[].video_id` | string UUID | 视频 ID。 |
| `items[].title` | string | 标题。 |
| `items[].cover_image_url` | string \| null | 封面 URL。 |
| `items[].duration_seconds` | integer | 时长秒数。 |
| `items[].view_count` | integer | 全局观看数。 |
| `items[].favorited_at` | string datetime | 当前用户收藏时间，UTC JSON。 |
| `page.limit` | integer | 实际 page size。 |
| `page.has_more` | boolean | 是否还有下一页。 |
| `page.next_cursor` | string \| null | 下一页 opaque cursor。 |

Errors：

| HTTP | code | 场景 |
|---:|---|---|
| 400 | `invalid_request` | `limit` 非法、cursor 无法解码、cursor kind 不匹配、cursor 字段非法。 |
| 401 | `unauthorized` | principal 缺失。 |
| 500 | `internal_error` | 数据库、URL 组装或未知错误。 |
| 503 | `service_unavailable` | request 取消 / 超时。 |

Side effects：无。

Retry：可安全重试；翻页重试必须复用同一个 cursor。

### 3.4 `GET /api/video-history`

文档来源：[Video-History-API-MVP设计.md](Video-History-API-MVP设计.md)；总体约束来源：[API模块总体设计规范.md](API模块总体设计规范.md)。

用途：分页读取当前用户最近观看且仍可展示的视频列表。该 API 读取 Catalog 消费投影，不直接读取 Analytics watch event ledger。

Auth：必需。

Owner：Catalog read usecase；API 负责 public asset URL 组装。

Query：

| 参数 | 类型 | 必需 | 默认值 | 规则 |
|---|---|---:|---|---|
| `limit` | integer | 否 | `20` | `1..100`。 |
| `cursor` | string | 否 | 无 | opaque cursor；cursor kind 必须是 `video_history`。 |

Response `200 OK`：

| 字段 | 类型 | 说明 |
|---|---|---|
| `items` | array | 观看历史视频列表。 |
| `items[].video_id` | string UUID | 视频 ID。 |
| `items[].title` | string | 标题。 |
| `items[].cover_image_url` | string \| null | 封面 URL。 |
| `items[].duration_seconds` | integer | 时长秒数。 |
| `items[].view_count` | integer | 全局观看数。 |
| `items[].last_position_ms` | integer | 当前用户最近播放位置。 |
| `items[].last_watched_at` | string datetime | 最近观看时间，UTC JSON。 |
| `page.limit` | integer | 实际 page size。 |
| `page.has_more` | boolean | 是否还有下一页。 |
| `page.next_cursor` | string \| null | 下一页 opaque cursor。 |

Errors 同 `/api/video-favorites`，但 cursor kind 不匹配时按 `video_history` 判断。

Side effects：无。

Retry：可安全重试；翻页重试必须复用同一个 cursor。

### 3.5 `POST /api/videos/end-quiz`

文档来源：[End-Quiz-批量取题API-MVP设计.md](End-Quiz-批量取题API-MVP设计.md)；总体约束来源：[API模块总体设计规范.md](API模块总体设计规范.md)。

用途：视频末尾按 `video_id + coarse_unit_ids` 批量获取 quiz 题。每个 coarse unit 最多返回一道题，优先 video-context 题，fallback 到 unit-generic 题。

Auth：必需。

Owner：Catalog read usecase。

Request body：

| 字段 | 类型 | 必需 | 默认值 | 规则 |
|---|---|---:|---|---|
| `video_id` | string UUID | 是 | 无 | 当前视频 ID；必须可展示，否则 `404 not_found`。 |
| `coarse_unit_ids` | integer[] | 是 | 无 | 非空；每项正整数；按首次出现顺序去重；去重后最多 8 个。 |
| `recommendation_run_id` | string UUID | 否 | 空 | 可选推荐归因上下文；只校验 UUID，不要求 run 已存在或属于当前用户；不参与取题选择、不写取题日志。 |
| `client_context` | object | 否 | `{}` | 只校验 JSON object。 |

Response `200 OK`：

| 字段 | 类型 | 说明 |
|---|---|---|
| `video_id` | string UUID | 请求中的视频 ID。 |
| `items` | array | 命中题目的列表。 |
| `missing_coarse_unit_ids` | integer[] | 没有合法题目的 coarse unit IDs。 |
| `items[].coarse_unit_id` | integer | 题目考察的学习单元。 |
| `items[].question_id` | string UUID | 题目 ID，quiz attempt 上报必须带回。 |
| `items[].source` | string | `video_context` 或 `unit_generic`。 |
| `items[].question_type` | string | 题型。 |
| `items[].target_text` | string | 考察词或表达。 |
| `items[].question` | string | 前端展示问题文本。 |
| `items[].context_text` | string \| null | 视频上下文或通用题上下文。 |
| `items[].options` | array | 选择项。 |
| `items[].options[].option_id` | string | 选项 ID；正确项固定为 `correct`。 |
| `items[].options[].text` | string | 选项文本。 |
| `items[].explanation` | string \| null | 解释文本。 |
| `items[].context_sentence_index` | integer \| null | 视频上下文字幕句索引。 |
| `items[].context_span_index` | integer \| null | 视频上下文 span 索引。 |
| `items[].context_start_ms` | integer \| null | 上下文开始时间。 |
| `items[].context_end_ms` | integer \| null | 上下文结束时间。 |

Errors：

| HTTP | code | 场景 |
|---:|---|---|
| 400 | `invalid_request` | Content-Type / JSON / unknown field 错误；`video_id` 或 `recommendation_run_id` 非 UUID；`coarse_unit_ids` 为空、含非正整数或超过 8 个。 |
| 401 | `unauthorized` | principal 缺失。 |
| 404 | `not_found` | 视频不存在或不可用于 end quiz。 |
| 409 | `conflict` | Catalog owner 返回冲突。 |
| 422 | `unprocessable_entity` | Catalog owner 返回业务状态拒绝。 |
| 500 | `internal_error` | 数据库、题目 payload 数据异常或未知错误。 |
| 503 | `service_unavailable` | request 取消 / 超时。 |

Side effects：无取题 audit、无 quiz session、无学习进度写入。

Retry：可安全重试；返回题目选择按当前 Catalog 候选稳定规则决定。

### 3.6 `GET /api/me`

文档来源：[Me-API-MVP设计.md](Me-API-MVP设计.md)；总体约束来源：[API模块总体设计规范.md](API模块总体设计规范.md)。

用途：读取当前用户 profile、累计活动统计和内嵌 7 天 activity calendar。启动路径使用。

Auth：必需。

Owner：User module。

Headers：

| Header | 必需 | 规则 |
|---|---:|---|
| `X-Client-Timezone` | 否 | 合法 IANA timezone 时，后端可顺手更新 profile timezone；非法值被忽略，不返回 `400`。 |

Request body：无。

Response `200 OK`：

| 字段 | 类型 | 说明 |
|---|---|---|
| `user_id` | string UUID | 当前用户 ID。 |
| `email` | string \| null | Auth 派生邮箱缓存。 |
| `email_confirmed` | boolean | 邮箱是否确认。 |
| `display_name` | string | 展示名。 |
| `avatar_url` | string \| null | 头像 URL。 |
| `locale` | string | 用户 locale。 |
| `timezone` | string \| null | profile timezone。 |
| `onboarding_status` | string | `new` / `collection_selected` / `completed`。 |
| `birth_date` | string date \| null | `YYYY-MM-DD`。 |
| `gender` | string \| null | `male` / `female` / `other` / `prefer_not_to_say`。 |
| `education_stage` | string \| null | `primary_school` / `middle_school` / `high_school` / `undergraduate` / `graduate` / `phd` / `working` / `other`。 |
| `ip_region` | string \| null | 预留地区字段。 |
| `stats.total_watch_seconds` | integer | 累计观看秒数。 |
| `stats.quiz_attempt_count` | integer | 累计 quiz attempt 数。 |
| `stats.started_unit_count` | integer | 累计开始学习的 unit 数。 |
| `activity_calendar.timezone` | string | 本次 calendar 使用的 timezone。 |
| `activity_calendar.today` | string date | timezone 下的今天。 |
| `activity_calendar.current_streak_days` | integer | 当前连续活跃天数。 |
| `activity_calendar.days` | array | 固定 7 天，日期升序，补齐空日期。 |
| `activity_calendar.days[].local_date` | string date | 本地日期。 |
| `activity_calendar.days[].watch_seconds` | integer | 当日观看秒数。 |
| `activity_calendar.days[].quiz_attempt_count` | integer | 当日 quiz 次数。 |
| `activity_calendar.days[].learning_interaction_count` | integer | 当日 learning interaction 次数。 |

Errors：

| HTTP | code | 场景 |
|---:|---|---|
| 401 | `unauthorized` | principal 缺失、无法解析，或 principal 指向不存在的 Auth user。 |
| 500 | `internal_error` | 数据库或未知服务端错误。 |
| 503 | `service_unavailable` | request 取消 / 超时。 |

Side effects：可能 lazy repair `app_user.user_profiles`；合法 `X-Client-Timezone` 可更新 profile timezone；会 ensure activity stats row。

Retry：可安全重试；合法 timezone header 可能重复写同值。

### 3.7 `PATCH /api/me/profile`

文档来源：[Me-Profile-Update-API-MVP设计.md](Me-Profile-Update-API-MVP设计.md)；总体约束来源：[API模块总体设计规范.md](API模块总体设计规范.md)。

用途：修改当前用户可编辑 profile 字段。只更新 `app_user.user_profiles`，不返回 stats 或 activity calendar。

Auth：必需。

Owner：User module。

Request body：

| 字段 | 类型 | 必需 | 规则 |
|---|---|---:|---|
| `display_name` | string | 否 | 不允许 `null`；trim 后 2-20 个 Unicode 字符；只允许 Unicode 字母、数字和下划线。 |
| `birth_date` | string date \| null | 否 | `YYYY-MM-DD`；范围 `1900-01-01..today`；`null` 清空。 |
| `gender` | string \| null | 否 | `male` / `female` / `other` / `prefer_not_to_say`；`null` 清空。 |
| `education_stage` | string \| null | 否 | `primary_school` / `middle_school` / `high_school` / `undergraduate` / `graduate` / `phd` / `working` / `other`；`null` 清空。 |
| `timezone` | string \| null | 否 | 合法 IANA timezone；`null` 清空。非法值返回 `400 invalid_request`。 |

字段省略表示不修改。空 object `{}` 返回 `400 invalid_request`。未列出的字段全部返回 `400 invalid_request`。

Response `200 OK`：返回更新后的 profile 子集：

| 字段 | 类型 |
|---|---|
| `user_id` | string UUID |
| `email` | string \| null |
| `email_confirmed` | boolean |
| `display_name` | string |
| `avatar_url` | string \| null |
| `locale` | string |
| `timezone` | string \| null |
| `onboarding_status` | string |
| `birth_date` | string date \| null |
| `gender` | string \| null |
| `education_stage` | string \| null |
| `ip_region` | string \| null |

Errors：

| HTTP | code | 场景 |
|---:|---|---|
| 400 | `invalid_request` | JSON 非 object、空 patch、字段类型错误、字段值非法、包含不允许修改字段、非法 timezone。 |
| 401 | `unauthorized` | principal 缺失或 Auth user 不存在。 |
| 500 | `internal_error` | 数据库或未知服务端错误。 |
| 503 | `service_unavailable` | request 取消 / 超时。 |

Side effects：profile 缺失时先 lazy repair，再应用 patch。

Retry：同 body 重试会重复设置同一 profile 字段；无额外幂等 key。

### 3.8 `GET /api/unit-collections`

文档来源：[Unit-Collections-API-MVP设计.md](Unit-Collections-API-MVP设计.md)；总体约束来源：[API模块总体设计规范.md](API模块总体设计规范.md)。

用途：读取 active 词书列表，以及当前用户 active collection slug / null。

Auth：必需。

Owner：API facade 编排 Semantic list usecase 和 Learning Engine active collection reader。

Request body：无。

Response `200 OK`：

| 字段 | 类型 | 说明 |
|---|---|---|
| `items` | array | active unit collections。 |
| `items[].collection_id` | string UUID | 词书 ID。 |
| `items[].slug` | string | 词书 slug。 |
| `items[].name` | string | 展示名称。 |
| `items[].description` | string \| null | 描述。 |
| `items[].category` | string | 分类。 |
| `items[].coarse_unit_count` | integer | collection 中 coarse unit 总数。 |
| `items[].word_unit_count` | integer | word unit 数量。 |
| `active_collection` | string \| null | 当前用户 active collection slug；没有 profile 或 active collection 不在 active list 中时返回 `null`。 |

Errors：

| HTTP | code | 场景 |
|---:|---|---|
| 401 | `unauthorized` | principal 缺失。 |
| 400 | `invalid_request` | facade / owner 返回可修正输入错误。 |
| 500 | `internal_error` | 数据库或未知错误。 |
| 503 | `service_unavailable` | request 取消 / 超时。 |

Side effects：无。

Retry：可安全重试。

### 3.9 `GET /api/learning-targets/active-coarse-unit-ids`

文档来源：[Active-Learning-Targets-API-MVP设计.md](Active-Learning-Targets-API-MVP设计.md)；总体约束来源：[API模块总体设计规范.md](API模块总体设计规范.md)。

用途：读取当前用户仍可用于 exposure 过滤的 active target coarse unit ids。

Auth：必需。

Owner：Learning Engine reducer read model。

Request body：无。实现确认不读取 query string；`user_id`、`collection_slug` 或分页参数即使传入也会被忽略，`user_id` 只从 trusted principal 获取。

Response `200 OK`：

| 字段 | 类型 | 说明 |
|---|---|---|
| `active_collection` | string \| null | 当前用户 active collection slug；没有 learning profile 时为 `null`。 |
| `target_count` | integer | `coarse_unit_ids.length`。 |
| `coarse_unit_ids` | integer[] | 当前用户 `is_target=true AND status in ('new','learning','reviewing')` 的 ids，升序返回。 |

无 active profile 时返回：

```json
{
  "active_collection": null,
  "target_count": 0,
  "coarse_unit_ids": []
}
```

Errors：

| HTTP | code | 场景 |
|---:|---|---|
| 401 | `unauthorized` | principal 缺失。 |
| 500 | `internal_error` | 数据库或未知错误。 |
| 503 | `service_unavailable` | request 取消 / 超时。 |

Side effects：无。

Retry：可安全重试。前端在词书切换成功后应清空该 query cache。

### 3.10 `PUT /api/learning-targets/active-collection`

文档来源：[Unit-Collections-API-MVP设计.md](Unit-Collections-API-MVP设计.md)；总体约束来源：[API模块总体设计规范.md](API模块总体设计规范.md)。

用途：为当前用户激活一本词书。API facade 在同一用户级事务中维护 Learning Engine target projection，并把 User onboarding 状态更新为 `collection_selected`。

Auth：必需。

Owner：API facade 同事务编排 Learning Engine 与 User。

Request body：

| 字段 | 类型 | 必需 | 规则 |
|---|---|---:|---|
| `collection_slug` | string | 是 | trim 后必须匹配 `^[a-z0-9][a-z0-9-]{0,80}$`；仅小写字母、数字、连字符，首字符必须字母或数字。 |

Response `200 OK`：

| 字段 | 类型 | 说明 |
|---|---|---|
| `collection_id` | string UUID | 被激活 collection ID。 |
| `collection_slug` | string | 被激活 collection slug。 |
| `target_count` | integer | 激活后的目标单元数量。 |

Errors：

| HTTP | code | 场景 |
|---:|---|---|
| 400 | `invalid_request` | Content-Type / JSON / unknown field 错误；`collection_slug` 缺失或格式非法。 |
| 401 | `unauthorized` | principal 缺失或 Auth user 不存在。 |
| 404 | `not_found` | collection 不存在或 inactive。 |
| 500 | `internal_error` | 数据库或未知错误。 |
| 503 | `service_unavailable` | request 取消 / 超时。 |

Side effects：同事务更新 Learning Engine target projection、active collection 状态和 User onboarding status。

Retry：同一 slug 重试会再次收敛到同一 active collection；无客户端幂等 key。

### 3.11 `PUT /api/videos/{video_id}/like`

文档来源：[Video-Interactions-API-MVP设计.md](Video-Interactions-API-MVP设计.md)；总体约束来源：[API模块总体设计规范.md](API模块总体设计规范.md)。

用途：幂等设置当前用户已点赞单个视频。

Auth：必需。

Owner：Catalog 写侧。

Path / body：

| 字段 | 类型 | 必需 | 规则 |
|---|---|---:|---|
| `video_id` | string UUID path | 是 | 视频必须 active / public / 已发布。 |
| `occurred_at` | RFC3339 datetime | 是 | 客户端动作发生时间。 |

Response `200 OK`：

| 字段 | 类型 |
|---|---|
| `video_id` | string UUID |
| `has_liked` | boolean |
| `like_count` | integer |

Errors：

| HTTP | code | 场景 |
|---:|---|---|
| 400 | `invalid_request` | `video_id` 非 UUID、Content-Type / JSON 错误、unknown field、`occurred_at` 缺失或非法。 |
| 401 | `unauthorized` | principal 缺失。 |
| 404 | `not_found` | 视频不存在或不可交互。 |
| 409 | `conflict` | Catalog owner 返回冲突。 |
| 422 | `unprocessable_entity` | Catalog owner 返回业务状态拒绝。 |
| 500 | `internal_error` | 数据库或未知错误。 |
| 503 | `service_unavailable` | request 取消 / 超时。 |

Idempotency / side effects：set，不是 toggle。只有从 false 变 true 时 `like_count + 1`；重复 set 不重复增加 count。旧 `occurred_at` 请求是 stale no-op，仍返回当前状态。

Retry：重试同一次点击必须复用同一个 `occurred_at`。

### 3.12 `DELETE /api/videos/{video_id}/like`

文档来源：[Video-Interactions-API-MVP设计.md](Video-Interactions-API-MVP设计.md)；总体约束来源：[API模块总体设计规范.md](API模块总体设计规范.md)。

用途：幂等设置当前用户未点赞单个视频。

Auth、path/body、errors 同 `PUT /api/videos/{video_id}/like`。

Response `200 OK`：

| 字段 | 类型 |
|---|---|
| `video_id` | string UUID |
| `has_liked` | boolean |
| `like_count` | integer |

Idempotency / side effects：unset，不是 toggle。只有当前 true 时才 `like_count - 1`；重复 unset 不重复减少 count。旧 `occurred_at` 请求 stale no-op。计数更新使用非负防御。

### 3.13 `PUT /api/videos/{video_id}/favorite`

文档来源：[Video-Interactions-API-MVP设计.md](Video-Interactions-API-MVP设计.md)；总体约束来源：[API模块总体设计规范.md](API模块总体设计规范.md)。

用途：幂等设置当前用户已收藏单个视频。

Auth：必需。

Owner：Catalog 写侧。

Path / body：

| 字段 | 类型 | 必需 | 规则 |
|---|---|---:|---|
| `video_id` | string UUID path | 是 | 视频必须 active / public / 已发布。 |
| `occurred_at` | RFC3339 datetime | 是 | 客户端动作发生时间。 |

Response `200 OK`：

| 字段 | 类型 |
|---|---|
| `video_id` | string UUID |
| `has_favorited` | boolean |
| `favorite_count` | integer |

Errors 同 like API。

Idempotency / side effects：set，不是 toggle。数据库字段使用既有 `has_bookmarked` / `bookmarked_at` / `favorite_state_updated_at`，API 产品语义统一叫 favorite。重复 set 不重复增加 count；旧时间 stale no-op。

Retry：重试同一次点击必须复用同一个 `occurred_at`。

### 3.14 `DELETE /api/videos/{video_id}/favorite`

文档来源：[Video-Interactions-API-MVP设计.md](Video-Interactions-API-MVP设计.md)；总体约束来源：[API模块总体设计规范.md](API模块总体设计规范.md)。

用途：幂等设置当前用户未收藏单个视频。

Auth、path/body、errors 同 `PUT /api/videos/{video_id}/favorite`。

Response `200 OK`：

| 字段 | 类型 |
|---|---|
| `video_id` | string UUID |
| `has_favorited` | boolean |
| `favorite_count` | integer |

Idempotency / side effects：unset，不是 toggle。只有当前 true 时才 `favorite_count - 1`；重复 unset 不重复减少 count。旧 `occurred_at` 请求 stale no-op。

### 3.15 `POST /api/video-watch-progress`

文档来源：[Catalog-观看进度上报MVP设计.md](Catalog-观看进度上报MVP设计.md)；总体约束来源：[API模块总体设计规范.md](API模块总体设计规范.md)。

用途：上报视频观看进度。Catalog 同事务维护 watch session ledger、视频消费投影和 User watch stats。

Auth：必需。

Owner：Catalog 写侧，User stats projection。

Request body：

| 字段 | 类型 | 必需 | 默认值 | 规则 |
|---|---|---:|---|---|
| `video_id` | string UUID | 是 | 无 | 视频必须存在。 |
| `watch_session_id` | string UUID | 是 | 无 | 前端生成的观看 session correlation key；同一用户同一 session 后续上报必须绑定同一 `video_id`，不同用户可复用同一个客户端 session key。 |
| `position_ms` | integer | 是 | 无 | 必须非负；可小于历史最大位置。 |
| `active_watch_ms` | integer | 是 | 无 | 必须非负；重复上报按 delta 去重。 |
| `occurred_at` | RFC3339 datetime | 否 | 服务端当前时间 | 提供时必须带 offset；不能超过当前时间太多，也不能过早。 |
| `source_surface` | string | 否 | 空字符串 | 当前实现不校验枚举；建议传 `fullscreen` / `feed` / `detail` 等。 |
| `client_context` | object | 否 | `{}` | 客户端环境上下文。 |
| `metadata` | object | 否 | `{}` | watch-progress 专属扩展调试上下文。 |

Response `200 OK`：

| 字段 | 类型 | 说明 |
|---|---|---|
| `accepted` | boolean | 成功时为 `true`。 |

Errors：

| HTTP | code | 场景 |
|---:|---|---|
| 400 | `invalid_request` | Content-Type / JSON / unknown field 错误；UUID 非法；`position_ms` 或 `active_watch_ms` 缺失 / 为负；`client_context` 或 `metadata` 不是 object。 |
| 401 | `unauthorized` | principal 缺失。 |
| 404 | `not_found` | `video_id` 不存在。 |
| 409 | `conflict` | 当前用户的 `watch_session_id` 已存在，但绑定的 `video_id` 与本次请求不一致。 |
| 422 | `unprocessable_entity` | `occurred_at` 明显异常，例如超过当前时间太多或过早。 |
| 500 | `internal_error` | 数据库或未知错误。 |
| 503 | `service_unavailable` | request 取消 / 超时。 |

Side effects：写 `analytics.video_watch_events`、`catalog.video_user_states`、`catalog.video_engagement_stats` 和 User 活动统计相关投影。

Retry：同一 session 重试必须复用 `watch_session_id`；session 绑定冲突不可通过原样重试解决。

### 3.16 `POST /api/learning-interactions:batch`

文档来源：[学习事件上报API设计.md](学习事件上报API设计.md)；总体约束来源：[API模块总体设计规范.md](API模块总体设计规范.md)。

用途：批量写入 exposure / lookup raw learning interaction facts。整批 validation，任意一条非法则整批拒绝且不调用 usecase。

Auth：必需。

Owner：Analytics raw fact；Learning Engine normalizer 同步 best-effort。

Request body：

| 字段 | 类型 | 必需 | 默认值 | 规则 |
|---|---|---:|---|---|
| `client_context` | object | 否 | `{}` | 请求级客户端环境上下文。 |
| `video_id` | string UUID | 是 | 无 | 当前 batch 所属视频。 |
| `watch_session_id` | string UUID | 是 | 无 | 前端生成的观看 session correlation key；只校验 UUID，不要求 `analytics.video_watch_events` 已存在。 |
| `recommendation_run_id` | string UUID | 否 | 空 | 推荐归因上下文；只校验 UUID，不要求 run 已存在或属于当前用户。 |
| `events` | array | 是 | 无 | 必须非空；实现确认当前 handler 未设置最大 batch size。 |

`events[]`：

| 字段 | 类型 | 必需 | 规则 |
|---|---|---:|---|
| `client_event_id` | string | 是 | 当前用户维度幂等 ID。 |
| `event_type` | string | 是 | 只支持 `exposure` / `lookup`；`self_mark_mastered` 必须走单点 API。 |
| `source_surface` | string | 是 | 事件发生界面；当前不校验枚举。 |
| `coarse_unit_id` | integer | exposure 必需；lookup 可选 | exposure 必须正整数；lookup 提供时必须正整数。 |
| `token_text` | string | lookup 必需 | lookup 原始 token 文本。 |
| `sentence_index` | integer | 是 | exposure / lookup 均必需。 |
| `span_index` | integer | 是 | exposure / lookup 均必需。 |
| `occurred_at` | RFC3339 datetime | 是 | 必须带 explicit offset。 |
| `exposure_start_ms` | integer | 否 | 非负。 |
| `exposure_end_ms` | integer | 否 | 非负；若同时有 start，必须 `>= exposure_start_ms`。 |
| `exposure_count` | integer | 否 | 提供时必须 `>= 1`。 |
| `lookup_visible_ms` | integer | 否 | 非负。 |
| `lookup_sentence_audio_replay_count` | integer | 否 | 缺省 `0`；非负。 |
| `lookup_word_audio_play_count` | integer | 否 | 缺省 `0`；非负。 |
| `lookup_practice_now_clicked` | boolean | 否 | 缺省 `false`。 |
| `event_payload` | object | 否 | 缺省 `{}`。 |

Response `200 OK`：

| 字段 | 类型 |
|---|---|
| `accepted_count` | integer |
| `inserted_count` | integer |
| `duplicate_count` | integer |
| `events[].client_event_id` | string |
| `events[].learning_interaction_event_id` | string UUID |
| `events[].inserted` | boolean |

Errors：

| HTTP | code | 场景 |
|---:|---|---|
| 400 | `invalid_request` | 任意 transport validation 或 owner validation 失败。 |
| 401 | `unauthorized` | principal 缺失。 |
| 422 | `unprocessable_entity` | `video_id`、exposure `coarse_unit_id` 这类真实业务主键不存在。 |
| 500 | `internal_error` | raw write、数据库或未知错误。 |
| 503 | `service_unavailable` | request 取消 / 超时。 |

Idempotency / side effects：Analytics 使用 `(user_id, client_event_id)` 幂等；跨请求重试命中既有 raw row 时 duplicate 不是错误，返回 `duplicate_count` 和 `inserted=false`。同一 batch 内重复 `client_event_id` 属于请求自身非法，返回 `400 invalid_request`，整批不写 raw。HTTP success 只承诺 raw accepted；normalizer 失败会记录日志，由 pending repair/backfill 补偿，HTTP 仍返回成功。

Retry：失败重试必须复用原 `client_event_id`。

### 3.17 `POST /api/quiz-attempts`

文档来源：[学习事件上报API设计.md](学习事件上报API设计.md)；总体约束来源：[API模块总体设计规范.md](API模块总体设计规范.md)。

用途：写入一次 completed quiz attempt raw fact。Quiz attempt 不进入 interaction batch。

Auth：必需。

Owner：Analytics raw fact；Learning Engine normalizer 同步 best-effort；User stats projection。

Request body：

| 字段 | 类型 | 必需 | 默认值 / 规则 |
|---|---|---:|---|
| `client_context` | object | 否 | 缺省 `{}`。 |
| `client_event_id` | string | 是 | 当前用户维度幂等 ID。 |
| `question_id` | string UUID | 是 | `catalog.questions.question_id`。 |
| `coarse_unit_id` | integer | 是 | 必须正整数。 |
| `video_id` | string UUID | 否 | 提供时必须 UUID。 |
| `recommendation_run_id` | string UUID | 否 | 推荐归因上下文；提供时只校验 UUID。 |
| `trigger_type` | string | 是 | `video_end` / `lookup_practice` / `feed_review` / `mid_video` / `manual`。 |
| `selected_option_ids` | string[] | 是 | 非空；最后一项必须是 `correct`。 |
| `selection_interval_ms` | integer[] | 是 | 长度必须等于 `selected_option_ids`；每项非负。 |
| `is_first_try_correct` | boolean | 是 | 必须与 `selected_option_ids[0] == "correct"` 一致。 |
| `total_elapsed_ms` | integer | 是 | 非负。 |
| `shown_at` | RFC3339 datetime | 是 | 必须带 explicit offset。 |
| `completed_at` | RFC3339 datetime | 是 | 必须带 explicit offset，且 `>= shown_at`。 |

Response `200 OK`：

| 字段 | 类型 |
|---|---|
| `accepted` | boolean |
| `quiz_event_id` | string UUID |
| `inserted` | boolean |

Errors：

| HTTP | code | 场景 |
|---:|---|---|
| 400 | `invalid_request` | transport validation、枚举、时间顺序、选择序列一致性或 owner validation 失败。 |
| 401 | `unauthorized` | principal 缺失。 |
| 422 | `unprocessable_entity` | `question_id`、`coarse_unit_id`、`video_id` 这类真实业务主键不存在。 |
| 500 | `internal_error` | raw write、数据库或未知错误。 |
| 503 | `service_unavailable` | request 取消 / 超时。 |

Idempotency / side effects：Analytics 使用 `(user_id, client_event_id)` 幂等；duplicate 不重复增加 User stats。HTTP success 只承诺 raw accepted；normalizer failure 不改变 HTTP success。

Retry：失败重试必须复用原 `client_event_id`。

### 3.18 `POST /api/learning-units:mark-mastered`

文档来源：[学习事件上报API设计.md](学习事件上报API设计.md)；总体约束来源：[API模块总体设计规范.md](API模块总体设计规范.md)。

用途：用户明确把一个已有 user-unit state 标记为已掌握。不是 quiz attempt，也不放入 interaction batch。

Auth：必需。

Owner：Analytics raw fact；Learning Engine self-mark normalizer 同步 best-effort。

Request body：

| 字段 | 类型 | 必需 | 默认值 / 规则 |
|---|---|---:|---|
| `client_context` | object | 否 | 缺省 `{}`。 |
| `client_event_id` | string | 是 | 当前用户维度幂等 ID。 |
| `coarse_unit_id` | integer | 是 | 必须正整数，且当前用户必须已有对应 `learning.user_unit_states` 行。 |
| `source_surface` | string | 是 | 触发界面；当前不校验枚举。 |
| `video_id` | string UUID | 否 | 提供时必须 UUID；作为真实业务主键，提供时必须存在。 |
| `watch_session_id` | string UUID | 否 | 前端生成的观看 session correlation key；只校验 UUID，不要求 `analytics.video_watch_events` 已存在。 |
| `recommendation_run_id` | string UUID | 否 | 推荐归因上下文；提供时只校验 UUID。 |
| `related_quiz_event_id` | string UUID | 否 | 可选来源上下文；只校验 UUID，不要求 `analytics.quiz_events` 已存在。 |
| `token_text` | string | 否 | 原始 token 文本。 |
| `sentence_index` | integer | 否 | 字幕句索引。 |
| `span_index` | integer | 否 | span 索引。 |
| `occurred_at` | RFC3339 datetime | 是 | 必须带 explicit offset。 |
| `event_payload` | object | 否 | 缺省 `{}`。 |

Response `200 OK`：

| 字段 | 类型 |
|---|---|
| `accepted` | boolean |
| `learning_interaction_event_id` | string UUID |
| `inserted` | boolean |

Errors：

| HTTP | code | 场景 |
|---:|---|---|
| 400 | `invalid_request` | 请求字段非法，或当前用户没有对应 `user_unit_state`。 |
| 401 | `unauthorized` | principal 缺失。 |
| 422 | `unprocessable_entity` | `video_id` / `coarse_unit_id` 这类真实业务主键不存在。 |
| 500 | `internal_error` | raw write、数据库或未知错误。 |
| 503 | `service_unavailable` | request 取消 / 超时。 |

Idempotency / side effects：raw fact 使用 `(user_id, client_event_id)` 幂等；success 只承诺 raw accepted。归一化后的 `set_mastered` 只修改学习状态，不修改 target/control 字段。

Retry：失败重试必须复用原 `client_event_id`。

### 3.19 `POST /api/learning-units:reset-unlearned`

文档来源：[学习事件上报API设计.md](学习事件上报API设计.md)；总体约束来源：[API模块总体设计规范.md](API模块总体设计规范.md)。

用途：把当前用户已有 user-unit state 重置为未学习。该 API 不写 Analytics，不经 Normalizer，直接写 Learning Engine reducer ledger。

Auth：必需。

Owner：Learning Engine reducer。

Request body：字段结构与 mark-mastered 相同：

| 字段 | 类型 | 必需 | 默认值 / 规则 |
|---|---|---:|---|
| `client_context` | object | 否 | 缺省 `{}`。 |
| `client_event_id` | string | 是 | 当前用户维度幂等 ID，不是 user-unit 维度。 |
| `coarse_unit_id` | integer | 是 | 必须正整数，且当前用户必须已有对应 `learning.user_unit_states` 行。 |
| `source_surface` | string | 是 | 触发界面。 |
| `video_id` | string UUID | 否 | 提供时必须 UUID；作为真实业务主键，提供时必须存在。 |
| `watch_session_id` | string UUID | 否 | 前端生成的观看 session correlation key；只校验 UUID，不要求 `analytics.video_watch_events` 已存在。 |
| `recommendation_run_id` | string UUID | 否 | 推荐归因上下文；提供时只校验 UUID。 |
| `related_quiz_event_id` | string UUID | 否 | 可选来源上下文；只校验 UUID，不要求 `analytics.quiz_events` 已存在。 |
| `token_text` | string | 否 | 原始 token 文本。 |
| `sentence_index` | integer | 否 | 字幕句索引。 |
| `span_index` | integer | 否 | span 索引。 |
| `occurred_at` | RFC3339 datetime | 是 | 必须带 explicit offset。 |
| `event_payload` | object | 否 | 缺省 `{}`。 |

Response `200 OK`：

| 字段 | 类型 |
|---|---|
| `accepted` | boolean |
| `unit_learning_event_id` | string UUID |
| `inserted` | boolean |

Errors：

| HTTP | code | 场景 |
|---:|---|---|
| 400 | `invalid_request` | 请求字段非法，或当前用户没有对应 `user_unit_state`。 |
| 401 | `unauthorized` | principal 缺失。 |
| 422 | `unprocessable_entity` | `video_id` / `coarse_unit_id` 这类真实业务主键不存在。 |
| 500 | `internal_error` | reducer write、数据库或未知错误。 |
| 503 | `service_unavailable` | request 取消 / 超时。 |

Idempotency / side effects：`client_event_id` 在当前用户维度幂等。duplicate 返回已有 `unit_learning_event_id` 和 `inserted=false`，不会对本次 body 中另一个 `coarse_unit_id` 再执行 reset；但实现仍会先校验本次 `coarse_unit_id` 已存在 state row。

Retry：失败重试必须复用原 `client_event_id`。

### 3.20 `GET /api/learning/unit-progress/mastered`

文档来源：[Unit-Progress-API-MVP设计.md](Unit-Progress-API-MVP设计.md)；总体约束来源：[API模块总体设计规范.md](API模块总体设计规范.md)。

用途：分页读取当前用户已掌握学习单元。

Auth：必需。

Owner：Learning Engine reducer read model join Semantic display metadata。

Query：

| 参数 | 类型 | 必需 | 默认值 | 规则 |
|---|---|---:|---|---|
| `limit` | integer | 否 | `50` | `1..100`。 |
| `cursor` | string | 否 | 无 | opaque cursor；bucket 必须是 `mastered`，不得包含 unmastered-only `progress_percent`。 |

Response `200 OK`：

| 字段 | 类型 |
|---|---|
| `items[].coarse_unit_id` | integer |
| `items[].kind` | string |
| `items[].label` | string |
| `items[].pos` | string \| null |
| `items[].chinese_label` | string \| null |
| `items[].chinese_def` | string \| null |
| `items[].progress_percent` | number |
| `items[].last_progress_at` | string datetime \| null |
| `page.limit` | integer |
| `page.has_more` | boolean |
| `page.next_cursor` | string \| null |

Selection / sorting：`status = 'mastered'`；不按 `is_target` 过滤；按 `lower(label), label, coarse_unit_id` 升序分页。

Errors：

| HTTP | code | 场景 |
|---:|---|---|
| 400 | `invalid_request` | `limit` 非法、cursor 解码失败、cursor bucket 不匹配或 cursor 字段非法。 |
| 401 | `unauthorized` | principal 缺失。 |
| 500 | `internal_error` | 数据库或未知错误。 |
| 503 | `service_unavailable` | request 取消 / 超时。 |

Side effects：无。

Retry：可安全重试；翻页重试复用 cursor。

### 3.21 `GET /api/learning/unit-progress/unmastered`

文档来源：[Unit-Progress-API-MVP设计.md](Unit-Progress-API-MVP设计.md)；总体约束来源：[API模块总体设计规范.md](API模块总体设计规范.md)。

用途：分页读取当前用户尚未掌握的目标学习单元。

Auth：必需。

Owner：Learning Engine reducer read model join Semantic display metadata。

Query：

| 参数 | 类型 | 必需 | 默认值 | 规则 |
|---|---|---:|---|---|
| `limit` | integer | 否 | `50` | `1..100`。 |
| `cursor` | string | 否 | 无 | opaque cursor；bucket 必须是 `unmastered`，且 cursor payload 必须包含 `progress_percent`。 |

Response `200 OK`：结构同 mastered。

Selection / sorting：`is_target = true AND status in ('new','learning','reviewing')`；按 `progress_percent DESC, lower(label), label, coarse_unit_id` 分页。

Errors、side effects、retry 同 mastered。

### 3.22 `POST /api/feedback`

文档来源：[User-Feedback-API-MVP设计.md](User-Feedback-API-MVP设计.md)；总体约束来源：[API模块总体设计规范.md](API模块总体设计规范.md)。

用途：提交当前用户产品反馈。接收一个自定义 JSON object payload 和最多 5 张 JPEG 图片。

Auth：必需。

Owner：User module，写 `app_user.feedback_submissions` 和 `app_user.feedback_images`。

Content-Type：`multipart/form-data`。完整 request body 限制 5 MiB。

Multipart fields：

| 字段 | 类型 | 必需 | 规则 |
|---|---|---:|---|
| `payload` | string | 是 | 必须提供一次；内容必须是 JSON object 字符串；空 object 合法。 |
| `client_feedback_id` | string UUID | 否 | 最多提供一次；同一用户维度幂等 ID。 |
| `images` | file[] | 否 | 同名 file field 可重复；最多 5 个；只接受 JPEG。 |

拒绝任何未列出的 form field 或 file field。

图片校验：

- multipart file header `Content-Type` 必须是 `image/jpeg` 或 `image/jpg`；
- 文件内容 magic bytes 必须以 `FF D8 FF` 开头；
- Go `image/jpeg.DecodeConfig` 必须能解析宽高；
- width / height 必须大于 0；
- 通过后统一保存 `content_type = image/jpeg`；
- 不接受 PNG、WebP、HEIC、GIF、视频或仅靠文件名伪装的 JPEG。

Response `200 OK`：

| 字段 | 类型 | 说明 |
|---|---|---|
| `feedback_id` | string UUID | submission ID。 |
| `accepted` | boolean | 成功时 `true`。 |
| `image_count` | integer | 本次 submission 图片数量。 |
| `created_at` | string datetime | UTC `Z` 格式。 |

Errors：

| HTTP | code | 场景 |
|---:|---|---|
| 400 | `invalid_request` | Content-Type 非 multipart；multipart form 非法；`payload` 缺失 / 重复 / 非 object；`client_feedback_id` 非 UUID；未知字段；图片超过 5 个；图片非 JPEG 或无法解析。 |
| 401 | `unauthorized` | principal 缺失。 |
| 413 | `payload_too_large` | 完整 multipart request body 超过 5 MiB。 |
| 500 | `internal_error` | 数据库或未知错误。 |
| 503 | `service_unavailable` | request 取消 / 超时。 |

Idempotency / side effects：如果同一用户此前已成功提交相同 `client_feedback_id`，重试返回已有 submission 的 `feedback_id`、`image_count` 和 `created_at`，不新增 submission，也不重复写图片。无 `client_feedback_id` 时每次成功请求都是新 submission。

Retry：前端应为可重试提交生成并复用 `client_feedback_id`。

### 3.23 `POST /api/word-favorites/status`

文档来源：[Word-Favorite-API-MVP设计.md](Word-Favorite-API-MVP设计.md)；总体约束来源：[API模块总体设计规范.md](API模块总体设计规范.md)。

用途：查询当前词 / 字幕 token identity 是否已收藏；可选返回视频句子上下文。

Auth：必需。

Owner：Catalog。

Request body：

| 字段 | 类型 | 必需 | 默认值 / 规则 |
|---|---|---:|---|
| `coarse_unit_id` | integer \| null | 条件 | `word_list` 必须为正整数；`video_transcript` 可为 null，提供时必须正整数。 |
| `text` | string | 是 | trim 后必须非空；只做 validation，不参与查找。 |
| `source` | string | 是 | 只允许 `word_list` / `video_transcript`。 |
| `video_id` | string UUID \| null | 条件 | `video_transcript` 必须提供；`word_list` 忽略。 |
| `sentence_index` | integer \| null | 条件 | `video_transcript` 必须提供且 `>=0`；`word_list` 忽略。 |
| `token_index` | integer \| null | 条件 | `video_transcript` 必须提供且 `>=0`；`word_list` 忽略。 |
| `include_video_context` | boolean | 否 | 缺省 `false`；`true` 时只允许 `source=video_transcript`。 |

Canonical lookup：

| 输入 | 收藏 key |
|---|---|
| `source=word_list` + `coarse_unit_id` | `user_id + coarse_unit_id` |
| `source=video_transcript` + `coarse_unit_id` | `user_id + coarse_unit_id` |
| `source=video_transcript` + `coarse_unit_id=null` | `user_id + video_id + sentence_index + token_index` |

`source=video_transcript` 同时带 `coarse_unit_id` 时不校验 token 是否实际映射到该 coarse unit；coarse key 优先。

Response `200 OK`：

| 字段 | 类型 | 说明 |
|---|---|---|
| `is_favorited` | boolean | 当前 canonical key 是否已收藏。 |
| `video_context.video_id` | string UUID | 请求 `include_video_context=true` 且上下文存在时返回；按 request 原样返回。 |
| `video_context.video_title` | string | 来自 DB。 |
| `video_context.video_duration_ms` | integer \| null | 来自 DB。 |
| `video_context.token_index` | integer | 按 request 原样返回。 |
| `video_context.sentence_index` | integer | 按 request 原样返回。 |
| `video_context.sentence_text` | string | 来自 DB。 |
| `video_context.sentence_translation` | string \| null | 来自 DB。 |
| `video_context.sentence_start_ms` | integer \| null | 来自 DB。 |
| `video_context.sentence_end_ms` | integer \| null | 来自 DB。 |

Validation：`include_video_context=false` 或省略时只查收藏表，不查视频 / 字幕 / span 表。`include_video_context=true` 时按 `video_id + sentence_index` 查可展示视频和句子；即使 `is_favorited=false`，只要上下文存在也返回 `video_context`。

Errors：

| HTTP | code | 场景 |
|---:|---|---|
| 400 | `invalid_request` | JSON 非 object、unknown field、非法 source、`text` 为空、required 字段缺失、index 非整数或负数、`include_video_context=true` 但 source 不是 `video_transcript`。 |
| 401 | `unauthorized` | principal 缺失。 |
| 404 | `not_found` | `include_video_context=true` 时视频不可展示或句子不存在。 |
| 413 | `payload_too_large` | body 超过 1 MiB。 |
| 500 | `internal_error` | 数据库或未知错误。 |
| 503 | `service_unavailable` | request 取消 / 超时。 |

Side effects：无。

Retry：只读，可安全重试。

### 3.24 `PUT /api/word-favorites`

文档来源：[Word-Favorite-API-MVP设计.md](Word-Favorite-API-MVP设计.md)；总体约束来源：[API模块总体设计规范.md](API模块总体设计规范.md)。

用途：幂等收藏当前词 / 字幕 token identity。

Auth：必需。

Owner：Catalog。

Request body：同 `POST /api/word-favorites/status` 的 identity 字段，不包含 `include_video_context`，并额外要求写入时间：

| 字段 | 类型 | 必需 | 默认值 / 规则 |
|---|---|---:|---|
| `occurred_at` | RFC3339 datetime | 是 | 客户端收藏动作发生时间；必须带 `Z` 或 explicit offset。 |

Response：`204 No Content`，无 JSON body。

Validation：

- `source=word_list + coarse_unit_id`：按 `user_id + coarse_unit_id` 收藏；非 stale 请求校验 coarse unit 存在且 active。
- `source=video_transcript + coarse_unit_id`：按 `user_id + coarse_unit_id` 收藏，同时保存来源 token 字段供列表展示；不校验 token 是否映射到该 coarse unit。
- `source=video_transcript + coarse_unit_id=null`：按 `user_id + video_id + sentence_index + token_index` 收藏；非 stale 请求校验视频可展示、句子存在、token/span 存在。
- `text` 只做非空 validation，不参与查找或写入 key。
- stale `PUT` 或同一 `occurred_at` 的已生效 PUT 重试，不因目标内容后来 inactive、hidden 或缺失而返回 `404`；它们仍是 `204` no-op。

Errors：

| HTTP | code | 场景 |
|---:|---|---|
| 400 | `invalid_request` | JSON 非 object、unknown field、非法 source、`text` 为空、required 字段缺失、index 非整数或负数、`occurred_at` 缺失或非法。 |
| 401 | `unauthorized` | principal 缺失。 |
| 404 | `not_found` | coarse unit 不存在 / inactive；token-only 目标视频、句子、span 不存在或视频不可展示。 |
| 413 | `payload_too_large` | body 超过 1 MiB。 |
| 500 | `internal_error` | 数据库或未知错误。 |
| 503 | `service_unavailable` | request 取消 / 超时。 |

Idempotency / side effects：upsert `catalog.word_favorites` 当前状态投影。生效请求设置 `is_favorited=true`，不存在或从 tombstone 恢复时 `favorited_at=occurred_at`；已收藏状态下较新 PUT 只推进 `state_updated_at`，不刷新 `favorited_at`；同一 `occurred_at` 的已生效 PUT 重试不刷新状态，也不重新要求目标存在。`occurred_at < state_updated_at` 是 stale no-op，仍返回 `204`。tombstone 可独立于内容行存在；`catalog.word_favorites` 不对 `video_id` / `coarse_unit_id` 建内容 FK。

Retry：重试同一次收藏动作必须复用同一个 `occurred_at`；旧请求不会覆盖更新状态。

### 3.25 `DELETE /api/word-favorites`

文档来源：[Word-Favorite-API-MVP设计.md](Word-Favorite-API-MVP设计.md)；总体约束来源：[API模块总体设计规范.md](API模块总体设计规范.md)。

用途：幂等取消收藏当前词 / 字幕 token identity。

Auth：必需。

Owner：Catalog。

Request body：同 `PUT /api/word-favorites`，包括必填 `occurred_at`。

Response：`204 No Content`，无 JSON body。

Validation：只做请求语法、identity validation 和 `occurred_at` 解析。unset 按 canonical key 执行，不要求目标 coarse unit、视频、句子或 token 当前仍存在，避免内容下架后无法取消收藏。

Errors：

| HTTP | code | 场景 |
|---:|---|---|
| 400 | `invalid_request` | JSON 非 object、unknown field、非法 source、`text` 为空、required 字段缺失、index 非整数或负数、`occurred_at` 缺失或非法。 |
| 401 | `unauthorized` | principal 缺失。 |
| 413 | `payload_too_large` | body 超过 1 MiB。 |
| 500 | `internal_error` | 数据库或未知错误。 |
| 503 | `service_unavailable` | request 取消 / 超时。 |

Idempotency / side effects：不物理删除。生效请求 upsert tombstone：`is_favorited=false`、`favorited_at=null`、`state_updated_at=occurred_at`；收藏不存在也返回 `204 No Content`。`occurred_at < state_updated_at` 是 stale no-op。

Retry：重试同一次取消收藏动作必须复用同一个 `occurred_at`；旧请求不会覆盖更新状态。

### 3.26 `GET /api/word-favorites`

文档来源：[Word-Favorite-API-MVP设计.md](Word-Favorite-API-MVP设计.md)；总体约束来源：[API模块总体设计规范.md](API模块总体设计规范.md)。

用途：分页读取当前用户收藏的词 / 字幕 token 展示列表。

Auth：必需。

Owner：Catalog。

Query：

| 参数 | 类型 | 必需 | 默认值 | 规则 |
|---|---|---:|---|---|
| `limit` | integer | 否 | `50` | `1..100`。 |
| `cursor` | string | 否 | 无 | opaque cursor；trim 后空字符串等同不传；kind 必须是 `word_favorites`。 |

Response `200 OK`：

| 字段 | 类型 | 说明 |
|---|---|---|
| `items[].coarse_unit_id` | integer \| null | coarse-key 收藏返回 coarse unit id；token-only 可为 null。 |
| `items[].label` | string \| null | coarse unit label。 |
| `items[].pos` | string \| null | coarse unit POS。 |
| `items[].chinese_label` | string \| null | coarse unit 中文 label。 |
| `items[].chinese_def` | string \| null | coarse unit 中文释义。 |
| `items[].source` | string | `word_list` / `video_transcript`。 |
| `items[].video_id` | string UUID \| null | 来源视频 id。 |
| `items[].sentence_index` | integer \| null | 来源句索引。 |
| `items[].token_index` | integer \| null | 来源 token 索引。 |
| `items[].source_text` | string \| null | 来源 token/span 文本。 |
| `items[].source_translation` | string \| null | 来源句翻译。 |
| `items[].source_dictionary` | string \| null | 来源 span dictionary。 |
| `items[].source_explanation` | string \| null | 来源 span explanation。 |
| `page.limit` | integer | 本次生效 limit。 |
| `page.has_more` | boolean | 是否还有下一页。 |
| `page.next_cursor` | string \| null | 下一页 opaque cursor。 |

不返回 `favorite_id`、`favorited_at`、`created_at`、`updated_at` 给前端。

Selection / sorting：只返回 `is_favorited=true` 且 `favorited_at is not null` 的当前收藏，tombstone 不返回。coarse-key 收藏要求 coarse unit active；token-only 收藏要求对应视频、句子、span 仍可展示。排序为 `favorited_at DESC, favorite_id ASC`。

Cursor：base64 raw URL encoding；payload 由后端拥有，格式为 `{"kind":"word_favorites","favorited_at":"...","favorite_id":"..."}`。前端必须视为 opaque string。

Errors：

| HTTP | code | 场景 |
|---:|---|---|
| 400 | `invalid_request` | `limit` 非法、cursor 解码失败、cursor kind 不匹配或 cursor 字段非法。 |
| 401 | `unauthorized` | principal 缺失。 |
| 500 | `internal_error` | 数据库或未知错误。 |
| 503 | `service_unavailable` | request 取消 / 超时。 |

Side effects：无。

Retry：只读，可安全重试；翻页重试复用 cursor。

## 4. 当前没有单列的无法证明项

本轮归总已对照 `docs/API`、当前 route registration、handler validation、API / owner DTO、application service 和 API tests。没有发现必须在本文单列为“无法从文档或代码证明”的字段或 endpoint。存在实现与设计文档文字有差异的地方，已在对应 endpoint 的“实现确认 / 实现差异”中说明。
