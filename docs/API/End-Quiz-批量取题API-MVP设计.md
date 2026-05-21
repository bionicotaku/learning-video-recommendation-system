# End Quiz 批量取题 API MVP 设计

## 0. 文档信息

文档状态：MVP 已实现
目标读者：前端、后端、Catalog / Analytics / Learning Engine / API 维护者
当前范围：定义视频末尾按 `video_id + coarse_unit_ids` 批量获取 quiz 题目的轻量 API 契约、fallback 规则、字段语义和与 quiz attempt 上报的边界。
当前明确不做：不创建取题审计表，不创建 quiz session，不固定题目 assignment，不在 Feed API 内联返回 quiz，不在取题响应里隐藏正确选项。

关联文档：

- [API模块总体设计规范.md](API模块总体设计规范.md)
- [Feed-API-MVP设计.md](Feed-API-MVP设计.md)
- [学习事件上报API设计.md](学习事件上报API设计.md)
- [../题目入库文档.md](../题目入库文档.md)

## 1. 一句话结论

视频末尾 quiz 取题 API 保持精简：

```http
POST /api/videos/end-quiz
```

请求只需要：

```text
video_id + coarse_unit_ids[]
```

后端对每个 `coarse_unit_id` 取一道题：

```text
1. 优先查 video_id + coarse_unit_id 的视频上下文题
2. 没有则 fallback 到 coarse_unit_id 的通用题
3. 仍没有则跳过该 unit
```

`recommendation_run_id` 可以作为可选字段传入，HTTP 层只校验 UUID 格式；当前实现不写取题日志、不参与取题选择。后续 quiz attempt 上报时建议继续带上做推荐归因。

## 2. API 定位

### 2.1 什么时候调用

前端在用户真正进入某个视频并接近结尾时调用本 API。

推荐调用时机：

- 视频播放到 `80%..90%` 时预取；
- 或视频结束事件触发后请求；
- 如果用户从 fullscreen 退出，不需要请求。

Feed API 不返回 quiz。原因是 feed 列表中大多数视频不会被点开，也不会被看到结尾；提前在 feed 中返回 quiz 会让 payload 变重，并把视频内学习体验耦进列表展示 API。

### 2.2 endpoint

```http
POST /api/videos/end-quiz
Content-Type: application/json
```

`user_id` 不由前端传入。MVP 取题本身不依赖用户身份，但 HTTP 层仍应遵守统一认证规则，避免给匿名客户端开放题库批量读取能力。

### 2.3 模块边界

| 模块 | 职责 | 不做什么 |
| --- | --- | --- |
| `internal/api` | HTTP handler、请求 validation、调用 Catalog 取题能力、返回 response。 | 不写 SQL，不记录取题 audit，不计算学习进度。 |
| `internal/catalog` | 提供按 `video_id + coarse_unit_ids[]` 批量取题并 fallback 的 read usecase / repository。 | 不记录用户答题结果，不写 Analytics。 |
| `internal/analytics` | 在 quiz attempt 上报时记录答题事实。 | 不参与取题。 |
| `internal/learningengine` | 通过 normalizer 消费 quiz attempt raw fact 并更新学习状态。 | 不参与取题。 |

## 3. 请求结构

### 3.1 Request

```json
{
  "video_id": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
  "coarse_unit_ids": [101, 205, 309],
  "recommendation_run_id": "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
  "client_context": {
    "platform": "ios",
    "app_version": "1.3.0",
    "os_version": "18.5",
    "device_model": "iPhone16,2"
  }
}
```

### 3.2 请求字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `video_id` | string UUID | 是 | 当前视频 ID。 |
| `coarse_unit_ids` | integer[] | 是 | 本次视频末尾要测试的学习单元列表。通常来自 Feed API 返回的 `learning_units[].coarse_unit_id`。 |
| `recommendation_run_id` | string UUID | 否 | 本视频来自推荐 feed 时可带上。取题不依赖该字段；后续 quiz attempt 上报时建议继续带上做推荐归因。 |
| `client_context` | object | 否 | 客户端环境上下文。当前实现只做 JSON object 校验，不写入数据库。 |

`coarse_unit_ids` 规则：

- 必须非空。调用方应只传 Feed item 中非空 `learning_units[].coarse_unit_id`；若当前视频是补全视频且 `learning_units=[]`，前端应跳过 end quiz 请求。
- 每个值必须为正整数。
- 后端应去重，但 response 顺序以去重后的首次出现顺序为准。
- MVP 建议限制最多 `8` 个，因为 Feed API 的学习推荐视频最多约 `1..8` 个 learning units；补全视频为空数组。

请求中不需要传 `role`、`is_primary`、evidence 或题型偏好。取题只依赖 `video_id + coarse_unit_ids[]`。

## 4. 返回结构

### 4.1 Response

```json
{
  "video_id": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
  "items": [
    {
      "coarse_unit_id": 101,
      "question_id": "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb",
      "source": "video_context",
      "question_type": "context_meaning_choice",
      "target_text": "focus",
      "question": "这里的 “focus” 最接近什么意思？",
      "context_text": "Try to focus on one sentence at a time.",
      "options": [
        { "option_id": "correct", "text": "集中注意力" },
        { "option_id": "wrong_1", "text": "快速移动" },
        { "option_id": "wrong_2", "text": "完全忘记" },
        { "option_id": "wrong_3", "text": "大声重复" }
      ],
      "explanation": "focus 在这里表示把注意力集中在一件事情上。",
      "context_sentence_index": 12,
      "context_span_index": 4,
      "context_start_ms": 42310,
      "context_end_ms": 42880
    }
  ],
  "missing_coarse_unit_ids": [309]
}
```

### 4.2 顶层字段

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `video_id` | string UUID | 请求中的视频 ID。 |
| `items` | array | 成功匹配到题目的列表。每个 coarse unit 最多返回一道题。 |
| `missing_coarse_unit_ids` | integer[] | 既没有视频上下文题、也没有通用题的 unit。前端可直接跳过。 |

### 4.3 `EndQuizItem`

| 字段 | 类型 | 来源 | 说明 |
| --- | --- | --- | --- |
| `coarse_unit_id` | integer | `catalog.questions.coarse_unit_id` | 题目考察的学习单元。 |
| `question_id` | string UUID | `catalog.questions.question_id` | 题目 ID。后续 quiz attempt 上报必须带回。 |
| `source` | string | 取题策略派生 | `video_context` 表示来自 `scope_type = 'video_unit'`；`unit_generic` 表示 fallback 到 `scope_type = 'unit'`。 |
| `question_type` | string | `catalog.questions.question_type` | 题型，例如 `context_meaning_choice`、`unit_meaning_choice`。 |
| `target_text` | string | `catalog.questions.target_text` | 题目考察的词或表达。 |
| `question` | string | `content_payload.question` | 前端展示的问题文本。字段名使用 `question`，不使用 `prompt`。 |
| `context_text` | string \| null | `content_payload.context_text` | 视频上下文或题目上下文文本。通用题可以为空。 |
| `options` | array | `content_payload.options` | 选择项。MVP 保留 `correct` / `wrong_*` option id，前端可自行打乱展示顺序。 |
| `options[].option_id` | string | `content_payload.options[].id` | 选项稳定 ID。正确项固定为 `correct`。 |
| `options[].text` | string | `content_payload.options[].text` | 选项展示文本。 |
| `explanation` | string \| null | `content_payload.explanation` | 中文解释正确选项为什么对。用户最终选对后展示。 |
| `context_sentence_index` | integer \| null | `catalog.questions.context_sentence_index` | 视频上下文题的字幕句索引。通用题为空。 |
| `context_span_index` | integer \| null | `catalog.questions.context_span_index` | 视频上下文题的 span 索引。通用题为空。 |
| `context_start_ms` | integer \| null | `catalog.questions.context_start_ms` | 视频上下文题的上下文开始时间。通用题为空。 |
| `context_end_ms` | integer \| null | `catalog.questions.context_end_ms` | 视频上下文题的上下文结束时间。通用题为空。 |

## 5. 取题规则

### 5.1 输入去重与顺序

后端先对 `coarse_unit_ids` 按首次出现顺序去重：

```text
[101, 205, 101, 309] -> [101, 205, 309]
```

返回 `items[]` 按去重后的 unit 顺序排列。若某个 unit 缺题，则不占位，放入 `missing_coarse_unit_ids`。

### 5.2 优先级

对每个 unit：

```text
优先级 1: catalog.questions where scope_type = 'video_unit'
  and video_id = request.video_id
  and coarse_unit_id = unit_id
  and status = 'active'

优先级 2: catalog.questions where scope_type = 'unit'
  and video_id is null
  and coarse_unit_id = unit_id
  and status = 'active'
```

`source` 映射：

| 命中来源 | `source` |
| --- | --- |
| `scope_type = 'video_unit'` | `video_context` |
| `scope_type = 'unit'` | `unit_generic` |

### 5.3 多题选择

当前实现使用稳定选择规则：

```sql
order by coarse_unit_id, created_at desc, question_id asc
```

Repository 不在 SQL 里对每个 unit `limit 1`，而是返回候选列表给 Go 层。Go 层会跳过 payload 不合法的候选，并继续尝试同一 unit 的下一道题；第一个合法候选即为本 unit 的返回题。这样实现仍然简单、结果可复现，同时避免坏题阻断 fallback。

### 5.4 题型限制

MVP 不需要请求体传题型偏好。后端按可用题选择：

视频上下文题优先：

```text
context_meaning_choice
context_cloze_choice
```

通用 fallback 题：

```text
unit_meaning_choice
reverse_identification_choice
```

如果同一优先级下有多个题型，MVP 可先按 `created_at desc, question_id asc` 选最新题；后续再补题型权重。

## 6. 不做取题审计表

MVP 不为本 API 新增 `quiz_deliveries`、`quiz_sessions` 或 `quiz_assignments`。

原因：

- 取题请求只是读题，不改变学习状态。
- 当前前端允许用户一直选到正确，不需要服务端实时判题。
- 学习状态真正依据的是完成后的 quiz attempt 上报。
- Recommendation audit 已经记录 feed run 和当次 `learning_units`，不需要为取题再复制一份推荐快照。

只有出现以下需求时，才新增取题分配表：

| 需求 | 可能新增 |
| --- | --- |
| 必须知道前端实际拿到了哪道题 | `quiz_deliveries` |
| 防止用户提交没被发过的题 | delivery/session id |
| 刷新后必须固定同一组题 | `quiz_assignments` |
| 分析题目曝光但未作答 | delivery audit |
| 做题目轮换、A/B、难度实验 | assignment + strategy snapshot |

## 7. 与 quiz attempt 上报的关系

本 API 只负责取题。用户完成题目后，仍通过 [学习事件上报API设计.md](学习事件上报API设计.md) 中的 quiz attempt API 上报：

```http
POST /api/quiz-attempts
```

quiz attempt 请求应带回：

```json
{
  "client_event_id": "01JY_QUIZ_0001",
  "question_id": "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb",
  "coarse_unit_id": 101,
  "video_id": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
  "recommendation_run_id": "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
  "selected_option_ids": ["wrong_1", "correct"],
  "selection_interval_ms": [1800, 2600],
  "completed_at": "2026-05-16T18:20:10Z",
  "trigger_type": "video_end",
  "client_context": {
    "platform": "ios",
    "app_version": "1.3.0"
  }
}
```

其中 `recommendation_run_id` 不是取题必需字段，但如果本题来自某次 feed 推荐的视频，quiz attempt 上报时建议带上，用于分析推荐效果。

## 8. 错误响应

错误 envelope、request id、principal 规则统一遵守 [API模块总体设计规范.md](API模块总体设计规范.md)。

| HTTP 状态 | 场景 |
| --- | --- |
| `400 Bad Request` | JSON 格式错误、字段类型错误、`video_id` 非 UUID、`coarse_unit_ids` 为空、unit id 非正整数、数组过长。 |
| `401 Unauthorized` | 未登录或 principal 缺失。 |
| `404 Not Found` | `video_id` 不存在或不可用。 |
| `500 Internal Server Error` | Catalog 取题查询失败或其他未知服务端错误。 |

如果部分 unit 没有题，不返回错误；这些 unit 放入 `missing_coarse_unit_ids`。

如果某道 active 题的 `content_payload` 不合法，后端跳过该候选并继续尝试同一 unit 的其他候选。若同一 unit 没有任何合法题，则该 unit 进入 `missing_coarse_unit_ids`，不因为单道坏题让整个请求失败。

如果所有 unit 都没有题，仍返回 `200 OK`：

```json
{
  "video_id": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
  "items": [],
  "missing_coarse_unit_ids": [101, 205]
}
```

前端看到 `items` 为空时，直接不展示结尾 quiz。若 Feed item 本身 `learning_units=[]`，这是 Recommendation 的 video-level 补全视频，不是本轮学习任务；前端应在请求前跳过，不需要用空 `coarse_unit_ids` 调用本 API。

## 9. 后端实现建议

### 9.1 推荐内部接口

Catalog 当前提供批量 read usecase：

```text
EndQuizQuestionLookupUsecase.Execute(video_id, coarse_unit_ids) -> EndQuizQuestionLookupResponse
```

API 层调用该 usecase 后直接返回 HTTP response。`recommendation_run_id` 与 `client_context` 不传入 Catalog，因为当前取题不依赖这些字段。

### 9.2 查询策略

推荐两次批量查询，而不是对每个 unit 单独查询：

```text
1. 一次查 video_unit active questions:
   where video_id = $1 and coarse_unit_id = any($2)

2. 一次查 unit active questions:
   where video_id is null and coarse_unit_id = any($2)
```

Go 层按 `coarse_unit_id` 分组，并按请求去重后的 unit 顺序选择：先尝试视频上下文题候选，再尝试通用题候选。

### 9.3 content payload validation

返回前必须确认：

- `content_payload.question` 是非空 string；
- `content_payload.options` 是非空 array；
- 每个 option 有非空 `id` 与 `text`；
- 至少存在一个 `option.id = "correct"`；
- `explanation` 如果存在，必须是 string。

payload 不合法的题目不返回给前端；服务端继续尝试同一 unit 的其他 active 题。若没有可用题，则该 unit 进入 `missing_coarse_unit_ids`。

## 10. 成功标准

实现本 API 时至少满足：

1. 请求只依赖 `video_id + coarse_unit_ids[]` 完成取题。
2. `recommendation_run_id` 不参与取题选择。
3. 每个 unit 最多返回一道题。
4. 优先返回视频上下文题，没有再 fallback 通用题。
5. 部分缺题不失败，统一通过 `missing_coarse_unit_ids` 表达。
6. 不新增取题审计表，不写 Learning Engine，不写 Analytics。
7. 答题结果只通过 quiz attempt API 上报。
