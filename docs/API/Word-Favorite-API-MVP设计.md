# Word Favorite API MVP 设计

## 1. 目标与边界

Word Favorite API 表达“当前用户收藏一个词 / 字幕 token 对应的学习对象”。它归属 `catalog`，因为它是用户对内容对象的收藏投影；它不写 Analytics，不推进 Learning Engine，不属于 User profile。

本 MVP 提供 4 个 endpoint：

| Method | Path | 用途 | 成功响应 |
|---|---|---|---|
| `POST` | `/api/word-favorites/status` | 查询当前 identity 是否已收藏，可选返回视频句子上下文。 | `200 OK` JSON |
| `PUT` | `/api/word-favorites` | 幂等收藏。 | `204 No Content` |
| `DELETE` | `/api/word-favorites` | 幂等取消收藏。 | `204 No Content` |
| `GET` | `/api/word-favorites` | 分页读取当前用户收藏列表。 | `200 OK` JSON |

非目标：

- 不暴露 `favorite_id` 给前端。
- 不实现 `DELETE /api/word-favorites/{favorite_id}` 或其他兼容路径。
- 不把前端局部错误码作为后端 envelope code；后端继续使用统一 `invalid_request`、`not_found`、`payload_too_large` 等 code。
- 不写 Unit Progress、Video Favorite、Learning Target、Recommendation 或 Analytics 状态。

## 2. 全局传输契约

所有 endpoint 都要求 trusted principal。`user_id` 只来自 API auth middleware 注入的 principal，不接受 body、query、path 或普通 header 中的用户身份。

`POST`、`PUT`、`DELETE` 使用 `application/json`，body 必须是单个 JSON object。unknown field、数组、字符串、多段 JSON、空 body 都返回 `400 invalid_request`。body 超过 1 MiB 返回 `413 payload_too_large`。

`GET /api/word-favorites` 不要求 JSON body，只读取 query。

错误 envelope 沿用 API 总体规范：

```json
{
  "error": {
    "code": "invalid_request",
    "message": "...",
    "details": [],
    "request_id": "req_..."
  }
}
```

## 3. Identity 模型

请求使用 flat identity，不使用嵌套对象：

```json
{
  "coarse_unit_id": 108404,
  "text": "Making",
  "source": "video_transcript",
  "video_id": "00000000-0000-4000-8000-000000000001",
  "sentence_index": 7,
  "token_index": 2
}
```

字段规则：

| 字段 | 类型 | 必需 | 规则 |
|---|---|---:|---|
| `coarse_unit_id` | integer \| null | 条件 | `word_list` 必须非 null 且正整数；`video_transcript` 可为 null，提供时必须正整数。 |
| `text` | string | 是 | trim 后必须非空；只做请求合法性校验，不参与 canonical lookup。 |
| `source` | string | 是 | 只允许 `word_list` / `video_transcript`。 |
| `video_id` | string UUID \| null | 条件 | `video_transcript` 必须提供；`word_list` 忽略。 |
| `sentence_index` | integer \| null | 条件 | `video_transcript` 必须提供，且 `>=0`；`word_list` 忽略。 |
| `token_index` | integer \| null | 条件 | `video_transcript` 必须提供，且 `>=0`；`word_list` 忽略。 |

Canonical lookup：

| 输入 | 收藏 key |
|---|---|
| `source=word_list` + `coarse_unit_id` | `user_id + coarse_unit_id` |
| `source=video_transcript` + `coarse_unit_id` | `user_id + coarse_unit_id` |
| `source=video_transcript` + `coarse_unit_id=null` | `user_id + video_id + sentence_index + token_index` |

`source=video_transcript` 且同时带 `coarse_unit_id` 时，coarse key 优先。后端不校验 token 是否实际映射到该 coarse unit；`video_id`、`sentence_index`、`token_index` 只作为来源上下文字段保存和返回。

## 4. 写入时间与乱序保护

`PUT /api/word-favorites` 和 `DELETE /api/word-favorites` 必须在 identity 字段之外携带：

| 字段 | 类型 | 必需 | 规则 |
|---|---|---:|---|
| `occurred_at` | RFC3339 datetime | 是 | 客户端动作发生时间；必须带 `Z` 或 explicit offset；后端按 UTC 时间点存储和比较。 |

后端使用 `catalog.word_favorites.state_updated_at` 作为 canonical key 的状态裁决水位：

- `occurred_at >= state_updated_at`：请求生效。
- `occurred_at < state_updated_at`：stale no-op，仍返回 `204 No Content`。
- `PUT` 生效时写入 `is_favorited=true`；从 tombstone 恢复收藏时，`favorited_at=occurred_at`。
- 已经是 favorited 时收到更新的同状态 `PUT`，只推进 `state_updated_at`，不刷新 `favorited_at`，避免重试或重复点击改变列表顺序；同一 `occurred_at` 的已生效 PUT 重试不刷新状态。
- `DELETE` 生效时写入 `is_favorited=false`、`favorited_at=null`，保留 tombstone 用来挡住旧 `PUT`。

`PUT` 的目标存在性 / 可展示性校验只在请求不是 stale 且不是同状态重复 set 时执行。stale `PUT` 或同一次已生效收藏动作的重试，不会因为目标内容后来 inactive、hidden 或缺失而返回 `404`。

`favorited_at` 是列表排序时间，使用生效 `PUT` 的 `occurred_at`；`state_updated_at`、`favorited_at`、`favorite_id` 不暴露给前端。`catalog.word_favorites` 是用户状态投影，tombstone 必须能独立于内容行存在；因此不对 `video_id` / `coarse_unit_id` 建内容 FK。

## 5. `POST /api/word-favorites/status`

用途：查询当前 identity 是否已收藏；可选补视频句子上下文，用于前端在词详情弹层中展示字幕来源。

Owner：Catalog。

Auth：必需。

Request body：

| 字段 | 类型 | 必需 | 默认值 | 说明 |
|---|---|---:|---|---|
| `coarse_unit_id` | integer \| null | 条件 | 无 | 见 identity 规则。 |
| `text` | string | 是 | 无 | trim 后非空；不参与查找。 |
| `source` | string | 是 | 无 | `word_list` / `video_transcript`。 |
| `video_id` | string UUID \| null | 条件 | 无 | `video_transcript` 必须提供。 |
| `sentence_index` | integer \| null | 条件 | 无 | `video_transcript` 必须提供，且 `>=0`。 |
| `token_index` | integer \| null | 条件 | 无 | `video_transcript` 必须提供，且 `>=0`。 |
| `include_video_context` | boolean | 否 | `false` | `true` 时只允许 `source=video_transcript`。 |

Response `200 OK`：

| 字段 | 类型 | 必需 | 说明 |
|---|---|---:|---|
| `is_favorited` | boolean | 是 | 当前 canonical key 是否已收藏。 |
| `video_context` | object | 否 | 仅请求 `include_video_context=true` 且上下文存在时返回。 |
| `video_context.video_id` | string UUID | 是 | 按 request 原样返回。 |
| `video_context.video_title` | string | 是 | 来自 `catalog.videos.title`。 |
| `video_context.video_duration_ms` | integer \| null | 是 | 来自 `catalog.videos.duration_ms`。 |
| `video_context.token_index` | integer | 是 | 按 request 原样返回。 |
| `video_context.sentence_index` | integer | 是 | 按 request 原样返回。 |
| `video_context.sentence_text` | string | 是 | 来自 `catalog.video_transcript_sentences.sentence_text`。 |
| `video_context.sentence_translation` | string \| null | 是 | 来自 `catalog.video_transcript_sentences.sentence_translation`。 |
| `video_context.sentence_start_ms` | integer \| null | 是 | 来自 `catalog.video_transcript_sentences.start_ms`。 |
| `video_context.sentence_end_ms` | integer \| null | 是 | 来自 `catalog.video_transcript_sentences.end_ms`。 |

Validation：

- `include_video_context=false` 或省略时，只查收藏表，不 join 视频 / 字幕 / span 表。
- `include_video_context=true` 只允许 `source=video_transcript`。
- `include_video_context=true` 时按 `video_id + sentence_index` 读取可展示视频和句子；`video_context.video_id`、`sentence_index`、`token_index` 仍按请求原样返回。
- 即使 `is_favorited=false`，只要请求了 context 且视频句子存在，也返回 `video_context`。

Errors：

| HTTP | code | 场景 |
|---:|---|---|
| 400 | `invalid_request` | JSON 非 object、unknown field、非法 source、`text` 为空、required 字段缺失、index 负数或非整数、`word_list` 缺少 `coarse_unit_id`、`include_video_context=true` 但 source 不是 `video_transcript`。 |
| 401 | `unauthorized` | principal 缺失。 |
| 404 | `not_found` | `include_video_context=true` 时视频不可展示或句子不存在。 |
| 413 | `payload_too_large` | body 超过 1 MiB。 |
| 500 | `internal_error` | 数据库或未知服务端错误。 |
| 503 | `service_unavailable` | request 取消 / 超时。 |

Side effects：无。

Retry / idempotency：只读，可安全重试。

## 6. `PUT /api/word-favorites`

用途：幂等收藏当前 identity。

Owner：Catalog。

Auth：必需。

Request body：

| 字段 | 类型 | 必需 | 默认值 | 说明 |
|---|---|---:|---|---|
| `coarse_unit_id` | integer \| null | 条件 | 无 | 见 identity 规则。 |
| `text` | string | 是 | 无 | trim 后非空；不参与查找。 |
| `source` | string | 是 | 无 | `word_list` / `video_transcript`。 |
| `video_id` | string UUID \| null | 条件 | 无 | `video_transcript` 必须提供；`word_list` 忽略。 |
| `sentence_index` | integer \| null | 条件 | 无 | `video_transcript` 必须提供，且 `>=0`。 |
| `token_index` | integer \| null | 条件 | 无 | `video_transcript` 必须提供，且 `>=0`。 |
| `occurred_at` | RFC3339 datetime | 是 | 无 | 客户端收藏动作发生时间；必须带 `Z` 或 explicit offset。 |

Response：`204 No Content`，无 JSON body。

Validation：

- coarse-key 写入：`word_list + coarse_unit_id` 或 `video_transcript + coarse_unit_id` 都按 `user_id + coarse_unit_id` 收藏；非 stale 请求校验 coarse unit 存在且 active。
- token-only 写入：`video_transcript + coarse_unit_id=null` 按 `user_id + video_id + sentence_index + token_index` 收藏；非 stale 请求校验视频可展示、句子存在、token/span 存在，保证列表可展示。
- `source=video_transcript + coarse_unit_id` 会保存来源 token 字段供列表展示，但不校验 token 是否映射到该 coarse unit。
- stale `PUT` 和同一 `occurred_at` 的已生效 PUT 重试，在目标已经 inactive、hidden 或缺失时仍是 no-op，返回 `204 No Content`。

Errors：

| HTTP | code | 场景 |
|---:|---|---|
| 400 | `invalid_request` | JSON 非 object、unknown field、非法 source、`text` 为空、required 字段缺失、index 负数或非整数、`word_list` 缺少 `coarse_unit_id`、`occurred_at` 缺失或非法。 |
| 401 | `unauthorized` | principal 缺失。 |
| 404 | `not_found` | coarse unit 不存在 / inactive；token-only 目标视频、句子、span 不存在或视频不可展示。 |
| 413 | `payload_too_large` | body 超过 1 MiB。 |
| 500 | `internal_error` | 数据库或未知服务端错误。 |
| 503 | `service_unavailable` | request 取消 / 超时。 |

Side effects：

- 对 canonical key upsert 当前状态投影。
- 不存在时插入 `is_favorited=true`、`favorited_at=occurred_at`、`state_updated_at=occurred_at`。
- 已存在且 `occurred_at >= state_updated_at` 时更新当前状态；从 tombstone 恢复时刷新 `favorited_at=occurred_at`。
- 已经是 favorited 时较新同状态 `PUT` 只推进 `state_updated_at`，保留原 `favorited_at`；同一 `occurred_at` 的已生效 PUT 重试不刷新状态，也不重新要求目标存在。
- `occurred_at < state_updated_at` 时 stale no-op。

Retry / idempotency：幂等。重试同一次收藏动作必须复用同一个 `occurred_at`；旧请求不会覆盖更新状态。

## 7. `DELETE /api/word-favorites`

用途：幂等取消收藏当前 identity。

Owner：Catalog。

Auth：必需。

Request body：字段同 `PUT /api/word-favorites`，包括必填 `occurred_at`。

Response：`204 No Content`，无 JSON body。

Validation：

- 只做请求语法和 identity validation。
- 按 canonical key unset。
- 不要求目标 coarse unit、视频、句子或 token 当前仍存在，避免内容下架后无法取消收藏。

Errors：

| HTTP | code | 场景 |
|---:|---|---|
| 400 | `invalid_request` | JSON 非 object、unknown field、非法 source、`text` 为空、required 字段缺失、index 负数或非整数、`word_list` 缺少 `coarse_unit_id`、`occurred_at` 缺失或非法。 |
| 401 | `unauthorized` | principal 缺失。 |
| 413 | `payload_too_large` | body 超过 1 MiB。 |
| 500 | `internal_error` | 数据库或未知服务端错误。 |
| 503 | `service_unavailable` | request 取消 / 超时。 |

Side effects：不物理删除。生效时 upsert tombstone：`is_favorited=false`、`favorited_at=null`、`state_updated_at=occurred_at`。不存在时也可写 tombstone，用来挡住更旧的 `PUT`；目标内容不存在时仍返回 `204 No Content`。

Retry / idempotency：幂等。重复取消收藏返回 `204 No Content`；旧请求不会覆盖更新状态。

## 8. `GET /api/word-favorites`

用途：分页读取当前用户收藏的词 / 字幕 token 展示列表。

Owner：Catalog。

Auth：必需。

Query：

| 字段 | 类型 | 必需 | 默认值 | 规则 |
|---|---|---:|---|---|
| `limit` | integer | 否 | `50` | `1..100`；非法返回 `400 invalid_request`。 |
| `cursor` | string | 否 | 无 | opaque cursor；trim 后空字符串等同不传。 |

Response `200 OK`：

| 字段 | 类型 | 必需 | 说明 |
|---|---|---:|---|
| `items` | array | 是 | 当前页条目。 |
| `items[].coarse_unit_id` | integer \| null | 是 | coarse-key 收藏返回 coarse unit id；token-only 可为 null。 |
| `items[].label` | string \| null | 是 | coarse unit label；token-only 无 coarse unit 时为 null。 |
| `items[].pos` | string \| null | 是 | coarse unit POS。 |
| `items[].chinese_label` | string \| null | 是 | coarse unit 中文 label。 |
| `items[].chinese_def` | string \| null | 是 | coarse unit 中文释义。 |
| `items[].source` | string | 是 | `word_list` / `video_transcript`。 |
| `items[].video_id` | string UUID \| null | 是 | 来源视频 id；word-list 收藏可为 null。 |
| `items[].sentence_index` | integer \| null | 是 | 来源句索引。 |
| `items[].token_index` | integer \| null | 是 | 来源 token 索引。 |
| `items[].source_text` | string \| null | 是 | 来源 token/span 文本；来自视频 span 读模型。 |
| `items[].source_translation` | string \| null | 是 | 来源句翻译。 |
| `items[].source_dictionary` | string \| null | 是 | 来源 span dictionary 字段。 |
| `items[].source_explanation` | string \| null | 是 | 来源 span explanation 字段。 |
| `page.limit` | integer | 是 | 本次生效 limit。 |
| `page.has_more` | boolean | 是 | 是否还有下一页。 |
| `page.next_cursor` | string \| null | 是 | `has_more=true` 时返回下一页 cursor。 |

不返回 `favorite_id`、`favorited_at`、`created_at`、`updated_at` 给前端。

Selection / projection：

- 列表只返回仍可展示的收藏。
- 仅返回 `is_favorited=true` 且 `favorited_at is not null` 的当前收藏；tombstone 不进入列表。
- coarse-key 收藏 left join `semantic.coarse_unit` 展示字段，并要求 coarse unit active。
- 带来源 token 字段的收藏 left join 视频 span / 句子字段，补 `source_*` 字段。
- token-only 收藏要求对应视频、句子、span 仍可展示。

Pagination：

- 排序：`favorited_at DESC, favorite_id ASC`。
- SQL 读取 `limit + 1` 条，返回前 `limit` 条。
- `has_more=true` 时，`next_cursor` 使用本页最后一个返回 item 的 `favorited_at` 和 `favorite_id` 生成。
- cursor 使用 base64 raw URL encoding，payload 由后端拥有：

```json
{
  "kind": "word_favorites",
  "favorited_at": "2026-05-24T10:20:30.123Z",
  "favorite_id": "00000000-0000-4000-8000-000000000001"
}
```

前端必须把 cursor 视为 opaque string，不解析、不构造。

Errors：

| HTTP | code | 场景 |
|---:|---|---|
| 400 | `invalid_request` | `limit` 非法、cursor 解码失败、cursor kind 不匹配或 cursor 字段非法。 |
| 401 | `unauthorized` | principal 缺失。 |
| 500 | `internal_error` | 数据库或未知服务端错误。 |
| 503 | `service_unavailable` | request 取消 / 超时。 |

Side effects：无。

Retry / idempotency：只读，可安全重试；翻页重试复用同一个 cursor。

## 9. 前端状态期望

- 详情页或弹层需要单点状态时调用 `POST /api/word-favorites/status`，不需要为了状态查询拉列表。
- 收藏按钮调用 `PUT /api/word-favorites`；取消收藏调用 `DELETE /api/word-favorites`。两个写入口都必须传点击发生时的 `occurred_at`，返回 204 后前端以 HTTP success 更新本地状态。
- 重试同一次点击必须复用同一个 `occurred_at`；新点击生成新的 `occurred_at`。
- 列表页调用 `GET /api/word-favorites`，使用 `page.next_cursor` 继续加载。
- `text` 可以保留原始点击文本，后端只用它做非空校验；UI 展示应优先使用列表 / context response 中的 DB-derived 字段。

## 10. 实现来源

- API handler：`internal/api/infrastructure/http/handler/wordfavorites`
- API wiring：`internal/api/infrastructure/http/router`、`cmd/server/wiring_video.go`
- Catalog usecase：`internal/catalog/application/service/word_favorites.go`
- Catalog repository：`internal/catalog/infrastructure/persistence/repository/word_favorite_repository.go`
- SQLC query：`internal/catalog/infrastructure/persistence/query/word_favorites.sql`
- Migration：`internal/catalog/infrastructure/migration/000001_baseline.up.sql` 中的 `catalog.word_favorites`
