# 学习事件上报 API 设计

## 0. 文档信息

文档状态：MVP 当前实现说明
目标读者：前端、后端、数据、后续接手维护的人
当前范围：定义已落地学习事件上报 API 的前端上传契约、字段语义、raw fact / normalized ledger 落库语义、幂等响应、业务成功边界和后端内部归一化链路。
当前明确不做：不引入 queue / checkpoint / dead-letter / `normalized_at`；除 reset-unlearned 这类 reducer 直接命令外，不让 HTTP success 承诺 Learning Engine 已完成归约。

关联文档：

- [API模块总体设计规范.md](API模块总体设计规范.md)：定义 `internal/api` 的统一入口规范、错误响应、认证、validation 和跨模块编排规则。
- [学习互动信号语义设计.md](../学习互动信号语义设计.md)：定义 exposure / lookup / self mark / quiz 的学习语义。
- [学习引擎Normalizer设计.md](../学习引擎Normalizer设计.md)：定义 raw fact 如何转成 Learning Engine normalized event。
- [题目入库文档.md](../题目入库文档.md)：定义题目内容与 quiz attempt 的存储边界。

## 1. 一句话结论

学习事件上报当前拆成四条 API，HTTP 层已落在 `internal/api`：

```http
POST /api/learning-interactions:batch
POST /api/quiz-attempts
POST /api/learning-units:mark-mastered
POST /api/learning-units:reset-unlearned
```

前三条是 raw fact 写入入口：前端只上传 raw fact，不上传学习结论；后端先把 raw fact 幂等写入 Analytics，再在请求内同步尝试归一化本批 raw IDs。`reset-unlearned` 是 Learning Engine reducer 直接命令，用于把已有 user-unit 状态重置为未学习；它不写 Analytics，也不经过 Normalizer。

```text
POST /api/learning-interactions:batch
  -> internal/analytics RecordLearningInteractionsBatch
  -> internal/learningengine/normalizer NormalizeLearningInteractionsByIDs
  -> internal/learningengine/reducer RecordLearningEvents

POST /api/quiz-attempts
  -> internal/analytics RecordQuizAttempt
  -> internal/learningengine/normalizer NormalizeQuizAttemptByID
  -> internal/learningengine/reducer RecordLearningEvents

POST /api/learning-units:mark-mastered
  -> internal/analytics RecordSelfMarkMastered
  -> internal/learningengine/normalizer NormalizeSelfMarkMasteredByID
  -> internal/learningengine/reducer RecordLearningEvents

POST /api/learning-units:reset-unlearned
  -> internal/learningengine/reducer ResetUserUnitProgress
  -> append learning.unit_learning_events(reset_unlearned)
  -> reduce learning.user_unit_states in the same user transaction
```

前三条 raw API 的成功响应只承诺 raw fact 已接收并持久化，或因 `(user_id, client_event_id)` 已存在而幂等存在。Learning Engine 是否已经更新学习状态是后端内部状态，不暴露为前端成功条件。`reset-unlearned` 成功响应承诺 reset normalized event 已在 `learning.unit_learning_events` 幂等存在，且新插入时对应 `learning.user_unit_states` 已同步归约。

## 2. API 定位

### 2.1 Principal 与用户来源

本 API 遵守 [API模块总体设计规范.md](API模块总体设计规范.md) 的 principal 规则。生产认证默认由网关 / Auth provider 完成；`internal/api` 只从可信 principal 中解析 `user_id`。

请求体不接收可信 `user_id`。handler 必须把 principal 中的 `user_id` 传给 `internal/analytics`，不能从前端 payload 选择写入用户。

### 2.2 成功语义

前三条 raw API 成功只表示：

```text
raw fact accepted = 已新插入 analytics raw row 或已幂等存在
```

成功不表示：

- `learning.unit_learning_events` 已经写入；
- `learning.user_unit_states` 已经更新；
- self mark 一定已经从目标列表消失；
- quiz quality 已经完成归约；
- Recommendation 已看到最新学习状态。

后端会同步尝试归一化本次 raw IDs。若内部归一化失败，repair/backfill 会通过 `NormalizePendingEvents` 补偿。

`POST /api/learning-units:reset-unlearned` 不写 raw fact。成功表示：

```text
reset normalized event accepted = 已新插入 learning.unit_learning_events 或已幂等存在
```

新插入时，reducer 会在同一个 user-scoped transaction 内把 `learning.user_unit_states` 重置为未学习状态。

### 2.3 学习事件专属链路

```text
internal/api
  -> internal/analytics raw fact write
  -> internal/learningengine/normalizer by-ID normalize
  -> internal/learningengine/reducer RecordLearningEvents
```

Analytics 只保存 raw fact，不生成 `reducer_effect`，不计算 `progress_quality`，不写 `learning.*`。

Learning Engine normalizer 不写 Analytics，不直接写 `learning.*`，只调用 reducer 的 `RecordLearningEvents`。

`reset-unlearned` 是本 API 分组中的 reducer 直接命令：

```text
internal/api
  -> internal/learningengine/reducer ResetUserUnitProgress
  -> learning.unit_learning_events
  -> learning.user_unit_states
```

它的业务前置条件是当前用户必须已经存在对应 `learning.user_unit_states` 行；不要求 `is_target=true`，也不要求当前状态不是 `mastered`。

## 3. 支持的事件范围

| API | 写入表 | 事件类型 | 是否可能进入 Learning Engine |
| --- | --- | --- | --- |
| `POST /api/learning-interactions:batch` | `analytics.learning_interaction_events` | `exposure` | raw 保留；满 3 个不同 watch session 且最近无 lookup 时生成 passive affects-progress。 |
| `POST /api/learning-interactions:batch` | `analytics.learning_interaction_events` | `lookup` | mapped lookup 是，observe-only；unmapped lookup 只留 Analytics。 |
| `POST /api/quiz-attempts` | `analytics.quiz_events` | completed quiz attempt | 是，affects-progress。 |
| `POST /api/learning-units:mark-mastered` | `analytics.learning_interaction_events` | `self_mark_mastered` | 是，set-mastered。 |
| `POST /api/learning-units:reset-unlearned` | `learning.unit_learning_events` | `reset_unlearned` | 是，reset-unlearned；不写 Analytics，不经 Normalizer。 |

明确不属于本 API 的事件：

- video watch completed / watch progress：继续走 watch-progress API。
- favorite / 收藏单词：不是学习状态信号，不进入 Learning Engine。
- 手动加入 / 移出 target：是 Learning Engine control command，不是 raw fact。
- lookup 后点“练一下”：不单独作为学习事件；后续完成 quiz attempt 后再上报 `POST /api/quiz-attempts`。

## 4. 共享字段

### 4.1 `client_context`

`client_context` 描述客户端运行环境，不描述业务入口。业务入口使用每条 interaction 的 `source_surface` 或 quiz 的 `trigger_type`。

learning interaction batch 和 quiz attempt 的前端上传样例统一携带以下基础字段：

```json
{
  "client_context": {
    "platform": "ios",
    "app_version": "1.3.0",
    "os_version": "18.5",
    "device_model": "iPhone16,2"
  }
}
```

| 字段 | 类型 | 必需 | 说明 |
| --- | --- | --- | --- |
| `platform` | string | 建议 | `ios` / `android` / `web` 等。 |
| `app_version` | string | 建议 | App 版本。 |
| `os_version` | string | 建议 | 系统版本。 |
| `device_model` | string | 建议 | 设备型号。 |

后端只要求 `client_context` 是 JSON object，不能是 array / string / number；字段集合不在 application / DB 层固定，后续可以随客户端遥测演进扩展。

所有 `*_at` 时间字段必须自己携带 `Z` 或 offset；后端不会用客户端时区补解释事件时间。

### 4.2 `client_event_id`

`client_event_id` 是前端生成的幂等键：

- 同一个用户内必须唯一。
- 同一次重试必须复用同一个 `client_event_id`。
- 不同用户可以有相同 `client_event_id`。
- 后端以 `(user_id, client_event_id)` 幂等写入 raw fact。
- 同一 batch 请求内不能重复使用同一个 `client_event_id`；重复时整个 batch 返回 `invalid_request`，不写入 raw fact。

推荐格式是 UUID / ULID / nanoid。后端不从 `client_event_id` 推断时间、用户或业务类型。

## 5. Learning Interaction 批量 API

### 5.1 Endpoint

```http
POST /api/learning-interactions:batch
Content-Type: application/json
```

### 5.2 请求结构

```json
{
  "client_context": {
    "platform": "ios",
    "app_version": "1.3.0",
    "os_version": "18.5",
    "device_model": "iPhone16,2"
  },
  "video_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
  "watch_session_id": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
  "recommendation_run_id": "cccccccc-cccc-cccc-cccc-cccccccccccc",
  "events": []
}
```

顶层字段：

| 字段 | 类型 | 必需 | 说明 |
| --- | --- | --- | --- |
| `client_context` | object | 否 | 请求级客户端环境上下文。建议使用当前四个基础字段；缺省按 `{}` 处理。 |
| `video_id` | string UUID | 是 | 当前 batch 所属视频。 |
| `watch_session_id` | string UUID | 是 | 当前 batch 所属观看 session。 |
| `recommendation_run_id` | string UUID | 否 | 如果本次视频来自推荐结果，记录对应 run。 |
| `events` | array | 是 | interaction 事件数组。整批先 validation，任意一条非法则整批拒绝。 |

`events[]` 通用字段：

| 字段 | 类型 | 必需 | 说明 |
| --- | --- | --- | --- |
| `client_event_id` | string | 是 | 前端生成的幂等 ID。 |
| `event_type` | string | 是 | `exposure` / `lookup`。`self_mark_mastered` 不允许放入 batch，必须调用单点 mark-mastered API。 |
| `source_surface` | string | 是 | 事件发生的业务界面，例如 `video_subtitle`、`word_detail`。 |
| `coarse_unit_id` | integer | `exposure` 必需；mapped `lookup` 必需 | 学习单元 ID。填写时必须为正整数。exposure 不要求它来自本次 Recommendation `learning_units`，但前端应只上报当前用户未 mastered 且 target 的 coarse unit。unmapped lookup 可以为空，只留 Analytics。 |
| `token_text` | string | `lookup` 必需 | 用户 lookup 的原始 token 文本。 |
| `sentence_index` | integer | `exposure` / `lookup` 必需 | 字幕句子 index。当前 batch 只支持 `exposure` / `lookup`，所以必须提供；未来新增其他 `event_type` 时可按类型单独定义是否必需。 |
| `span_index` | integer | `exposure` / `lookup` 必需 | token/span index。当前 batch 只支持 `exposure` / `lookup`，所以必须提供；未来新增其他 `event_type` 时可按类型单独定义是否必需。 |
| `occurred_at` | RFC3339 datetime with explicit offset | 是 | 事件实际发生时间。必须带 `Z` 或 offset，后端按 UTC 时间点存储。 |
| `event_payload` | object | 否 | 附加原始上下文。缺省 `{}`。 |

`exposure` 额外字段：

| 字段 | 类型 | 必需 | 说明 |
| --- | --- | --- | --- |
| `exposure_start_ms` | integer | 否 | 在视频内的开始时间。必须非负。 |
| `exposure_end_ms` | integer | 否 | 在视频内的结束时间。必须非负，且不能小于 `exposure_start_ms`。 |
| `exposure_count` | integer | 否 | 聚合曝光次数。填写时必须 `>= 1`。 |

`lookup` 额外字段：

| 字段 | 类型 | 必需 | 说明 |
| --- | --- | --- | --- |
| `lookup_visible_ms` | integer | 否 | lookup 面板可见时长。必须非负。 |
| `lookup_sentence_audio_replay_count` | integer | 否 | 重放全句音频次数。缺省 `0`。 |
| `lookup_word_audio_play_count` | integer | 否 | 播放单词发音次数。缺省 `0`。 |
| `lookup_practice_now_clicked` | boolean | 否 | 是否点过“练一下”。该字段本身不推进 Learning Engine。 |

### 5.3 前端上传样例

```json
{
  "client_context": {
    "platform": "ios",
    "app_version": "1.3.0",
    "os_version": "18.5",
    "device_model": "iPhone16,2"
  },
  "video_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
  "watch_session_id": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
  "recommendation_run_id": "cccccccc-cccc-cccc-cccc-cccccccccccc",
  "events": [
    {
      "client_event_id": "01JY_LOOKUP_0001",
      "event_type": "lookup",
      "source_surface": "video_subtitle",
      "coarse_unit_id": 101,
      "token_text": "constrain",
      "sentence_index": 12,
      "span_index": 4,
      "occurred_at": "2026-05-15T17:00:01Z",
      "lookup_visible_ms": 7200,
      "lookup_sentence_audio_replay_count": 1,
      "lookup_word_audio_play_count": 2,
      "lookup_practice_now_clicked": false,
      "event_payload": {
        "displayed_base_form": "constrain"
      }
    },
    {
      "client_event_id": "01JY_EXPOSURE_0001",
      "event_type": "exposure",
      "source_surface": "video_subtitle",
      "coarse_unit_id": 102,
      "sentence_index": 13,
      "span_index": 1,
      "occurred_at": "2026-05-15T17:00:05Z",
      "exposure_start_ms": 142000,
      "exposure_end_ms": 146300,
      "exposure_count": 1
    }
  ]
}
```

### 5.4 响应结构

```json
{
  "accepted_count": 2,
  "inserted_count": 1,
  "duplicate_count": 1,
  "events": [
    {
      "client_event_id": "01JY_LOOKUP_0001",
      "learning_interaction_event_id": "11111111-1111-1111-1111-111111111111",
      "inserted": true
    },
    {
      "client_event_id": "01JY_EXPOSURE_0001",
      "learning_interaction_event_id": "22222222-2222-2222-2222-222222222222",
      "inserted": false
    }
  ]
}
```

响应字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `accepted_count` | integer | 本次请求中 raw accepted 的事件数。成功时等于请求 `events.length`。 |
| `inserted_count` | integer | 新插入 raw row 数。 |
| `duplicate_count` | integer | 因 `(user_id, client_event_id)` 已存在而幂等命中的事件数。 |
| `events[]` | array | 每个 `client_event_id` 对应的 raw event ID。 |
| `events[].learning_interaction_event_id` | string UUID | `analytics.learning_interaction_events.event_id`。 |
| `events[].inserted` | boolean | `true` 表示新插入；`false` 表示幂等命中已有 row。 |

## 6. Quiz Attempt 单点 API

### 6.1 Endpoint

```http
POST /api/quiz-attempts
Content-Type: application/json
```

### 6.2 请求结构

```json
{
  "client_context": {
    "platform": "ios",
    "app_version": "1.3.0",
    "os_version": "18.5",
    "device_model": "iPhone16,2"
  },
  "client_event_id": "01JY_QUIZ_0001",
  "question_id": "44444444-4444-4444-4444-444444444444",
  "coarse_unit_id": 101,
  "video_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
  "recommendation_run_id": "cccccccc-cccc-cccc-cccc-cccccccccccc",
  "trigger_type": "lookup_practice",
  "selected_option_ids": ["wrong-option-id", "correct"],
  "selection_interval_ms": [1800, 2200],
  "is_first_try_correct": false,
  "total_elapsed_ms": 4000,
  "shown_at": "2026-05-15T17:01:00Z",
  "completed_at": "2026-05-15T17:01:04Z"
}
```

字段：

| 字段 | 类型 | 必需 | 说明 |
| --- | --- | --- | --- |
| `client_context` | object | 否 | 请求级客户端环境上下文。建议使用当前四个基础字段；缺省按 `{}` 处理。 |
| `client_event_id` | string | 是 | 前端生成的幂等 ID。 |
| `question_id` | string UUID | 是 | `catalog.questions.question_id`。 |
| `coarse_unit_id` | integer | 是 | 本题对应学习单元，必须为正整数。 |
| `video_id` | string UUID | 否 | 触发题目的视频。 |
| `recommendation_run_id` | string UUID | 否 | 触发题目的推荐 run。 |
| `trigger_type` | string | 是 | 触发来源，必须使用 DB 枚举：`video_end` / `lookup_practice` / `feed_review` / `mid_video` / `manual`。 |
| `selected_option_ids` | string[] | 是 | 用户选择过的 option ID 列表。最后一项必须表示正确答案。 |
| `selection_interval_ms` | integer[] | 是 | 每次选择前的耗时，长度必须等于 `selected_option_ids`。每项非负。 |
| `is_first_try_correct` | boolean | 是 | 第一项选择是否正确。必须和 `selected_option_ids[0]` 一致。 |
| `total_elapsed_ms` | integer | 是 | 从展示题目到完成的总耗时，非负。 |
| `shown_at` | RFC3339 datetime with explicit offset | 是 | 题目展示时间。必须带 `Z` 或 offset，后端按 UTC 时间点存储。 |
| `completed_at` | RFC3339 datetime with explicit offset | 是 | 题目完成时间，必须 `>= shown_at`。必须带 `Z` 或 offset，后端按 UTC 时间点存储。 |

MVP 中前端可以一直选到正确再提交，因此 quiz 是“一次 completed attempt”，不是 clickstream。错误一次和错误多次对 Learning Engine quality 没区别；normalizer 只看 `is_first_try_correct` 和 `total_elapsed_ms`。

### 6.3 前端上传样例

首次快速答对：

```json
{
  "client_context": {
    "platform": "ios",
    "app_version": "1.3.0",
    "os_version": "18.5",
    "device_model": "iPhone16,2"
  },
  "client_event_id": "01JY_QUIZ_FAST_CORRECT",
  "question_id": "44444444-4444-4444-4444-444444444444",
  "coarse_unit_id": 101,
  "video_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
  "recommendation_run_id": "cccccccc-cccc-cccc-cccc-cccccccccccc",
  "trigger_type": "lookup_practice",
  "selected_option_ids": ["correct"],
  "selection_interval_ms": [3200],
  "is_first_try_correct": true,
  "total_elapsed_ms": 3200,
  "shown_at": "2026-05-15T17:02:00Z",
  "completed_at": "2026-05-15T17:02:03.200Z"
}
```

首次答错后完成：

```json
{
  "client_context": {
    "platform": "ios",
    "app_version": "1.3.0",
    "os_version": "18.5",
    "device_model": "iPhone16,2"
  },
  "client_event_id": "01JY_QUIZ_WRONG_THEN_CORRECT",
  "question_id": "55555555-5555-5555-5555-555555555555",
  "coarse_unit_id": 102,
  "video_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
  "trigger_type": "lookup_practice",
  "selected_option_ids": ["option-a", "correct"],
  "selection_interval_ms": [2100, 1900],
  "is_first_try_correct": false,
  "total_elapsed_ms": 4000,
  "shown_at": "2026-05-15T17:03:00Z",
  "completed_at": "2026-05-15T17:03:04Z"
}
```

### 6.4 响应结构

```json
{
  "accepted": true,
  "quiz_event_id": "66666666-6666-6666-6666-666666666666",
  "inserted": true
}
```

响应字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `accepted` | boolean | 成功时固定 `true`。 |
| `quiz_event_id` | string UUID | `analytics.quiz_events.event_id`。 |
| `inserted` | boolean | `true` 表示新插入；`false` 表示幂等命中已有 row。 |

## 7. Self Mark Mastered 单点 API

### 7.1 Endpoint

```http
POST /api/learning-units:mark-mastered
Content-Type: application/json
```

`coarse_unit_id` 放在 body 中，表示用户明确声明已掌握的学习单元。它不是 quiz 作答事实，也不放入 interaction batch。后端只接受该用户已经存在 `learning.user_unit_states` 行的 unit；该行可以是 `is_target=false`，也可以已经是 `status=mastered`。

### 7.2 请求结构

```json
{
  "client_context": {
    "platform": "ios",
    "app_version": "1.3.0",
    "os_version": "18.5",
    "device_model": "iPhone16,2"
  },
  "client_event_id": "01JY_SELF_MARK_0001",
  "coarse_unit_id": 103,
  "source_surface": "word_detail",
  "video_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
  "watch_session_id": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
  "recommendation_run_id": "cccccccc-cccc-cccc-cccc-cccccccccccc",
  "related_quiz_event_id": "66666666-6666-6666-6666-666666666666",
  "token_text": "trivial",
  "sentence_index": 12,
  "span_index": 4,
  "occurred_at": "2026-05-15T17:04:00Z",
  "event_payload": {
    "entry": "lookup_sheet"
  }
}
```

字段：

| 字段 | 类型 | 必需 | 说明 |
| --- | --- | --- | --- |
| `client_context` | object | 否 | 请求级客户端环境上下文。建议使用当前四个基础字段；缺省按 `{}` 处理。 |
| `client_event_id` | string | 是 | 前端生成的幂等 ID。 |
| `coarse_unit_id` | integer | 是 | 要标记为已掌握的学习单元 ID，必须为正整数，且当前用户必须已经存在对应 `learning.user_unit_states` 行。 |
| `source_surface` | string | 是 | 用户点击“已学会”的界面，例如 `word_detail`、`quiz_result`。 |
| `video_id` | string UUID | 否 | 关联视频。 |
| `watch_session_id` | string UUID | 否 | 关联观看 session。 |
| `recommendation_run_id` | string UUID | 否 | 关联推荐 run。 |
| `related_quiz_event_id` | string UUID | 否 | 如果按钮出现在 quiz 页面，可关联刚写入的 `analytics.quiz_events.event_id`。 |
| `token_text` | string | 否 | 用户看到或点击的原始 token 文本。 |
| `sentence_index` | integer | 否 | 字幕句子 index。 |
| `span_index` | integer | 否 | token/span index。 |
| `occurred_at` | RFC3339 datetime with explicit offset | 是 | 用户点击“已学会”的实际时间。必须带 `Z` 或 offset，后端按 UTC 时间点存储。 |
| `event_payload` | object | 否 | 附加原始上下文。缺省 `{}`。 |

### 7.3 响应结构

```json
{
  "accepted": true,
  "learning_interaction_event_id": "33333333-3333-3333-3333-333333333333",
  "inserted": true
}
```

响应字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `accepted` | boolean | 成功时固定 `true`。 |
| `learning_interaction_event_id` | string UUID | `analytics.learning_interaction_events.event_id`。 |
| `inserted` | boolean | `true` 表示新插入；`false` 表示幂等命中已有 row。 |

Self mark 的 API 成功语义仍然只是 raw accepted。raw 写入前会先校验该用户已有对应 `learning.user_unit_states` 行；缺失时返回 `invalid_request`，且不写入 raw fact。后端会同步尝试 `NormalizeSelfMarkMasteredByID`；即使同步归一化失败，也由 pending repair/backfill 最终补偿。

`mark-mastered` 的业务语义是“设置学习状态为 mastered”，不是“移出当前 target”。归一化后的 `set_mastered` 只收敛 `status/progress/mastery/schedule`，不修改 `is_target` 或其他 target/control 字段。

## 8. Reset Unlearned 单点 API

### 8.1 Endpoint

```http
POST /api/learning-units:reset-unlearned
Content-Type: application/json
```

`coarse_unit_id` 放在 body 中，表示用户明确要求把该学习单元重置为未学习。后端只接受当前用户已经存在 `learning.user_unit_states` 行的 unit；该行可以是 `is_target=false`，也可以已经是 `status=mastered`。缺少对应 state row 时返回 `400 invalid_request`，不写 `learning.unit_learning_events`。

`client_event_id` 对 reset-unlearned 也是当前用户维度的幂等键，而不是当前 user-unit 维度的幂等键。若同一用户同一 `client_event_id` 已经写过 reset event，后端返回已有 `unit_learning_event_id` 和 `inserted=false`，不会对本次 body 中的另一个 `coarse_unit_id` 再执行 reset。即使是 duplicate request，后端仍会先校验本次 body 的 `coarse_unit_id` 已存在对应 state row；缺失时仍返回 `invalid_request`。

### 8.2 请求结构

请求结构与 `POST /api/learning-units:mark-mastered` 完全一致：

```json
{
  "client_context": {
    "platform": "ios",
    "app_version": "1.3.0",
    "os_version": "18.5",
    "device_model": "iPhone16,2"
  },
  "client_event_id": "01JY_RESET_UNLEARNED_0001",
  "coarse_unit_id": 103,
  "source_surface": "word_detail",
  "video_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
  "watch_session_id": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
  "recommendation_run_id": "cccccccc-cccc-cccc-cccc-cccccccccccc",
  "related_quiz_event_id": "66666666-6666-6666-6666-666666666666",
  "token_text": "trivial",
  "sentence_index": 12,
  "span_index": 4,
  "occurred_at": "2026-05-15T17:04:00Z",
  "event_payload": {
    "entry": "word_detail"
  }
}
```

字段要求同 self mark mastered：`client_event_id`、`coarse_unit_id`、`source_surface`、`occurred_at` 必填；`client_context` 和 `event_payload` 必须是 JSON object，缺省按 `{}` 处理；可选 UUID 字段必须是合法 UUID。

### 8.3 响应结构

```json
{
  "accepted": true,
  "unit_learning_event_id": "33333333-3333-3333-3333-333333333333",
  "inserted": true
}
```

响应字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `accepted` | boolean | 成功时固定 `true`。 |
| `unit_learning_event_id` | string UUID | `learning.unit_learning_events.event_id`。 |
| `inserted` | boolean | `true` 表示新插入并同步 reducer；`false` 表示同一用户同一 `client_event_id` 幂等命中已有 reset event。 |

`reset-unlearned` 直接写 reducer normalized ledger：

```text
event_type = reset_unlearned
reducer_effect = reset_unlearned
progress_quality = null
source_type = learning_unit_reset
source_ref_id = client_event_id
occurred_at = request.occurred_at
reset_boundary_at = 后端计算的 reset 边界
```

`occurred_at` 是客户端业务发生时间。后端另存内部字段 `reset_boundary_at = max(request.occurred_at, learning.user_unit_states.latest_learning_event_occurred_at, learning.user_unit_states.latest_reset_boundary_at)`，用于屏蔽 reset 前旧 raw fact。`reset_boundary_at` 不出现在 public response；`latest_*` 字段是 reducer 内部 projection watermark，不属于 public API。

reducer 新插入时把状态重置为未学习：`status = new`、`progress_percent = 0`、`mastery_score = 0`，清空观察、进度、最近质量、成功/失败计数、schedule 和 `next_review_at`。它不改变 target/control 字段本身；因此如果 reset 前该 row 是 `is_target=false`，reset 后仍是 `is_target=false`；如果 reset 前仍是 `is_target=true`，reset 后仍是 `is_target=true`。该事件会进入 `learning.unit_learning_events`，后续 `ReplayUserStates` 按 `ledger_seq` 重放并得到一致结果。重复 `client_event_id` 只返回已有 event，不重新 reduce。

## 9. Normalizer 语义

### 9.1 Learning Interaction

```text
NormalizeLearningInteractionsByIDs(user_id, learning_interaction_event_ids)
```

| raw `event_type` | normalized `event_type` | `reducer_effect` | `progress_quality` | 说明 |
| --- | --- | --- | --- | --- |
| `exposure` | - | - | - | raw exposure 不再逐条写 normalized observe-only event；normalizer 只提取受影响的 `user_id + coarse_unit_id` 执行 session3 聚合检查。 |
| 3 个未消费过的不同 watch session exposure，且晚于 latest lookup/reset boundary | `exposure` | `affects_progress` | `4` | 生成一条 synthetic `exposure_session3_v1` passive progress；`source_ref_id` 是三 session 组合 hash，typed `consumed_watch_session_ids` 记录被消费的 3 个 session，`counts_toward_success_streak=false`。 |
| `lookup` | `lookup` | `observe_only` | `null` | mapped lookup 进入 Learning Engine；unmapped lookup skipped。 |

`NormalizeLearningInteractionsByIDs` 是 batch API 专属入口，只允许 exposure / lookup raw row。self mark raw row 必须走 `NormalizeSelfMarkMasteredByID`。同一 `watch_session_id` 内同一个 `coarse_unit_id` 的多条 exposure raw row 只计为一次 session exposure；已被 session3 event 的 typed `consumed_watch_session_ids` 消费的 `watch_session_id` 不会再次计入后续 window；lookup 会重置之后的 session3 计数窗口，`reset_unlearned.reset_boundary_at` 也会作为窗口起点并屏蔽旧 raw。

### 9.2 Quiz Attempt

```text
NormalizeQuizAttemptByID(user_id, quiz_event_id)
```

| raw 条件 | `progress_quality` |
| --- | --- |
| `is_first_try_correct = true` 且 `total_elapsed_ms <= 5000` | `5` |
| `is_first_try_correct = true` 且 `total_elapsed_ms > 5000` | `4` |
| `is_first_try_correct = false` 且 `total_elapsed_ms <= 5000` | `2` |
| `is_first_try_correct = false` 且 `total_elapsed_ms > 5000` | `1` |

normalized event 固定为：

```text
event_type = quiz
reducer_effect = affects_progress
source_type = quiz_event
source_ref_id = analytics.quiz_events.event_id
```

### 9.3 Self Mark Mastered

```text
NormalizeSelfMarkMasteredByID(user_id, learning_interaction_event_id)
```

raw row 必须满足 `event_type = self_mark_mastered`。如果传入 exposure / lookup 的 raw ID，该用例返回错误且不调用 reducer。

API 层已经保证 self mark raw row 只来自已有 `learning.user_unit_states` 的 unit。normalizer 固定生成 `set_mastered`，reducer 不再检查该 state 是否仍是 target：已有 state 无论 `is_target=true/false` 或是否已经 mastered，最终都收敛为 `status=mastered`，但保留原有 target/control 字段。

normalized event 固定为：

```text
event_type = self_mark_mastered
reducer_effect = set_mastered
progress_quality = null
source_type = learning_interaction_event
source_ref_id = analytics.learning_interaction_events.event_id
```

`reset-unlearned` 不属于 Normalizer 职责，不会出现在 `NormalizePendingEvents` repair/backfill 中；它已直接持久化为 reducer normalized event。

## 10. 错误与补偿语义

### 10.1 Validation error

任意 validation 失败都不入库。

interaction batch 是整批拒绝，不 partial success。quiz attempt、self mark mastered 和 reset-unlearned 是单条拒绝。

错误 envelope、状态码、`request_id` 遵守 [API模块总体设计规范.md](API模块总体设计规范.md)。示例：

```json
{
  "error": {
    "code": "invalid_request",
    "message": "events[1].occurred_at is required",
    "details": [
      {
        "field": "events[1].occurred_at",
        "reason": "required"
      }
    ],
    "request_id": "req_01HY..."
  }
}
```

### 10.2 Duplicate

本 API 的跨请求幂等命中不是错误。前三条 raw API 返回已有 raw event ID；`reset-unlearned` 按 `(user_id, client_event_id)` 返回已有 `unit_learning_event_id`；都标记 `inserted=false`。

`POST /api/learning-interactions:batch` 额外要求同一 batch 内 `events[].client_event_id` 不重复；同批重复是当前请求自身非法，返回 `invalid_request`，整批不写入 raw fact。跨请求重试复用同一个 `client_event_id` 才是幂等命中。

### 10.3 Internal normalize failure

如果 raw write 成功，但同步 normalizer 失败，HTTP 仍可以返回 raw accepted。后端需要记录错误日志，后续由 `NormalizePendingEvents` 修复。

前端不需要因为 Learning Engine 内部归约失败而重试；如果前端因网络失败无法确认 raw accepted，才使用同一个 `client_event_id` 重试。

`reset-unlearned` 不走 Normalizer；如果返回 200，新插入时 ledger 和 state 已经在同一事务提交。网络失败无法确认时，前端仍使用同一个 `client_event_id` 重试。

## 11. 前端队列建议

### 11.1 Interaction queue

learning interaction 可以本地排队并批量 flush：

- lookup 需要尽快 flush。
- exposure 可以短时间聚合后 flush。
- 同一事件重试必须复用 `client_event_id`。
- 不同事件不能共享 `client_event_id`。
- 失败重试时保持原始 `occurred_at`，不要改成重试时间。

### 11.2 Quiz submit

quiz 不进入 interaction batch。完成一道题后直接调用 `POST /api/quiz-attempts`。

如果网络失败，使用同一个 `client_event_id` 重试同一 completed attempt。不要把每次选项点击拆成单独事件上传。

### 11.3 Self mark submit

self mark 不进入 interaction batch。用户点击“已学会”后直接调用 `POST /api/learning-units:mark-mastered`。

前端可以做乐观 UI 更新，但服务端响应只表示 raw fact accepted；最终状态由同步 best-effort normalize 加 pending repair/backfill 保证。

### 11.4 Reset unlearned submit

reset-unlearned 不进入 interaction batch。用户点击“重置为未学习”后直接调用 `POST /api/learning-units:reset-unlearned`。

前端可以做乐观 UI 更新；服务端 200 表示 reset event 已幂等存在。若该用户没有对应 `learning.user_unit_states` 行，服务端返回 `400 invalid_request`，前端应回滚本次乐观状态。

## 12. TypeScript 契约草稿

```ts
export type ClientContext = {
  platform?: "ios" | "android" | "web" | string;
  app_version?: string;
  os_version?: string;
  device_model?: string;
  [key: string]: unknown;
};

export type LearningInteractionEvent =
  | {
      client_event_id: string;
      event_type: "exposure";
      source_surface: string;
      coarse_unit_id: number;
      sentence_index: number;
      span_index: number;
      occurred_at: string;
      exposure_start_ms?: number;
      exposure_end_ms?: number;
      exposure_count?: number;
      event_payload?: Record<string, unknown>;
    }
  | {
      client_event_id: string;
      event_type: "lookup";
      source_surface: string;
      coarse_unit_id?: number;
      token_text: string;
      sentence_index: number;
      span_index: number;
      occurred_at: string;
      lookup_visible_ms?: number;
      lookup_sentence_audio_replay_count?: number;
      lookup_word_audio_play_count?: number;
      lookup_practice_now_clicked?: boolean;
      event_payload?: Record<string, unknown>;
    };

export type RecordLearningInteractionsBatchRequest = {
  client_context?: ClientContext;
  video_id: string;
  watch_session_id: string;
  recommendation_run_id?: string;
  events: LearningInteractionEvent[];
};

export type RecordLearningInteractionsBatchResponse = {
  accepted_count: number;
  inserted_count: number;
  duplicate_count: number;
  events: Array<{
    client_event_id: string;
    learning_interaction_event_id: string;
    inserted: boolean;
  }>;
};

export type QuizTriggerType =
  | "video_end"
  | "lookup_practice"
  | "feed_review"
  | "mid_video"
  | "manual";

export type RecordQuizAttemptRequest = {
  client_context?: ClientContext;
  client_event_id: string;
  question_id: string;
  coarse_unit_id: number;
  video_id?: string;
  recommendation_run_id?: string;
  trigger_type: QuizTriggerType;
  selected_option_ids: string[];
  selection_interval_ms: number[];
  is_first_try_correct: boolean;
  total_elapsed_ms: number;
  shown_at: string;
  completed_at: string;
};

export type RecordQuizAttemptResponse = {
  accepted: true;
  quiz_event_id: string;
  inserted: boolean;
};

export type RecordSelfMarkMasteredRequest = {
  client_context?: ClientContext;
  client_event_id: string;
  coarse_unit_id: number;
  source_surface: string;
  video_id?: string;
  watch_session_id?: string;
  recommendation_run_id?: string;
  related_quiz_event_id?: string;
  token_text?: string;
  sentence_index?: number;
  span_index?: number;
  occurred_at: string;
  event_payload?: Record<string, unknown>;
};

export type RecordSelfMarkMasteredResponse = {
  accepted: true;
  learning_interaction_event_id: string;
  inserted: boolean;
};

export type ResetUserUnitProgressRequest = RecordSelfMarkMasteredRequest;

export type ResetUserUnitProgressResponse = {
  accepted: true;
  unit_learning_event_id: string;
  inserted: boolean;
};
```

## 13. 当前实现映射

当前已落 `internal/api` HTTP handler、API application service、Analytics raw write、normalizer by-ID 调用链，以及 reset-unlearned 的 reducer 直接写入链路。

`internal/api` 的学习事件 handler 只做该 API 的薄适配：

- 从可信 principal 取 `user_id`。
- 把 JSON request 映射到对应 application DTO。
- 对 raw API，raw write 成功后把 raw event IDs 传给 `internal/learningengine/normalizer`，并返回 raw accepted response。
- 对 reset-unlearned，直接调用 `internal/learningengine/reducer`，并返回 reducer event accepted response。

`reset-unlearned` handler 同样只做 HTTP 薄适配，但 API application service 会直接调用 reducer `ResetUserUnitProgress`，由 reducer 在 user-scoped transaction 内写 `learning.unit_learning_events` 并归约 `learning.user_unit_states`。

通用认证、错误 envelope、状态码、request id、body size、日志和 handler 目录规则不在本文重复定义，统一遵守 [API模块总体设计规范.md](API模块总体设计规范.md)。不要在 HTTP 层生成 `progress_quality`、`reducer_effect` 或直接写 `learning.*`。
