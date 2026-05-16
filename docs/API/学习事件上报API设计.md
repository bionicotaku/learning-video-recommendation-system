# 学习事件上报 API 设计

## 0. 文档信息

文档状态：MVP 设计，HTTP handler 尚未实现
目标读者：前端、后端、数据、后续接手维护的人
当前范围：定义未来学习事件上报 API 的前端上传契约、字段语义、raw fact 落库语义、幂等响应、业务成功边界和后端内部归一化链路。
当前明确不做：本轮不实现 HTTP handler，不创建真实路由，不引入 queue / checkpoint / dead-letter / `normalized_at`。

关联文档：

- [API模块总体设计规范.md](API模块总体设计规范.md)：定义 `internal/api` 的统一入口规范、错误响应、认证、validation 和跨模块编排规则。
- [学习互动信号语义设计.md](../学习互动信号语义设计.md)：定义 exposure / lookup / self mark / quiz 的学习语义。
- [学习引擎Normalizer设计.md](../学习引擎Normalizer设计.md)：定义 raw fact 如何转成 Learning Engine normalized event。
- [题目入库文档.md](../题目入库文档.md)：定义题目内容与 quiz attempt 的存储边界。

## 1. 一句话结论

未来学习事件上报拆成两条 API，HTTP 层放在 `internal/api`：

```http
POST /api/learning-interactions:batch
POST /api/quiz-attempts
```

前端只上传 raw fact，不上传学习结论。后端先把 raw fact 幂等写入 Analytics，再在请求内同步尝试归一化本批 raw IDs：

```text
POST /api/learning-interactions:batch
  -> internal/analytics RecordLearningInteractionsBatch
  -> internal/learningengine/normalizer NormalizeLearningInteractionsByIDs
  -> internal/learningengine/reducer RecordLearningEvents

POST /api/quiz-attempts
  -> internal/analytics RecordQuizAttempt
  -> internal/learningengine/normalizer NormalizeQuizAttemptByID
  -> internal/learningengine/reducer RecordLearningEvents
```

API 成功响应只承诺 raw fact 已接收并持久化，或因 `(user_id, client_event_id)` 已存在而幂等存在。Learning Engine 是否已经更新学习状态是后端内部状态，不暴露为前端成功条件。

## 2. API 定位

### 2.1 Principal 与用户来源

本 API 遵守 [API模块总体设计规范.md](API模块总体设计规范.md) 的 principal 规则。生产认证默认由网关 / Auth provider 完成；`internal/api` 只从可信 principal 中解析 `user_id`。

请求体不接收可信 `user_id`。未来 handler 必须把 principal 中的 `user_id` 传给 `internal/analytics`，不能从前端 payload 选择写入用户。

### 2.2 成功语义

成功只表示：

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

### 2.3 学习事件专属链路

```text
internal/api
  -> internal/analytics raw fact write
  -> internal/learningengine/normalizer by-ID normalize
  -> internal/learningengine/reducer RecordLearningEvents
```

Analytics 只保存 raw fact，不生成 `reducer_effect`，不计算 `progress_quality`，不写 `learning.*`。

Learning Engine normalizer 不写 Analytics，不直接写 `learning.*`，只调用 reducer 的 `RecordLearningEvents`。

## 3. 支持的事件范围

| API | 写入表 | 事件类型 | 是否可能进入 Learning Engine |
| --- | --- | --- | --- |
| `POST /api/learning-interactions:batch` | `analytics.learning_interaction_events` | `exposure` | 是，observe-only。 |
| `POST /api/learning-interactions:batch` | `analytics.learning_interaction_events` | `lookup` | mapped lookup 是，observe-only；unmapped lookup 只留 Analytics。 |
| `POST /api/learning-interactions:batch` | `analytics.learning_interaction_events` | `self_mark_mastered` | 是，set-mastered。 |
| `POST /api/quiz-attempts` | `analytics.quiz_events` | completed quiz attempt | 是，affects-progress。 |

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
  "events": []
}
```

顶层字段：

| 字段 | 类型 | 必需 | 说明 |
| --- | --- | --- | --- |
| `client_context` | object | 否 | 请求级客户端环境上下文。建议使用当前四个基础字段；缺省按 `{}` 处理。 |
| `events` | array | 是 | interaction 事件数组。整批先 validation，任意一条非法则整批拒绝。 |

`events[]` 通用字段：

| 字段 | 类型 | 必需 | 说明 |
| --- | --- | --- | --- |
| `client_event_id` | string | 是 | 前端生成的幂等 ID。 |
| `event_type` | string | 是 | `exposure` / `lookup` / `self_mark_mastered`。 |
| `source_surface` | string | 是 | 事件发生的业务界面，例如 `video_subtitle`、`word_detail`。 |
| `video_id` | string UUID | 否 | 关联视频。 |
| `watch_session_id` | string UUID | 否 | 关联观看 session。 |
| `recommendation_run_id` | string UUID | 否 | 关联推荐 run。 |
| `related_quiz_event_id` | string UUID | 否 | 关联 quiz raw event。通常 MVP 不需要前端填写。 |
| `coarse_unit_id` | integer | `exposure` / `self_mark_mastered` 必需；mapped `lookup` 必需 | 学习单元 ID。unmapped lookup 可以为空，只留 Analytics。 |
| `token_text` | string | `lookup` 必需 | 用户 lookup 的原始 token 文本。 |
| `sentence_index` | integer | 否 | 字幕句子 index。 |
| `span_index` | integer | 否 | token/span index。 |
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
  "events": [
    {
      "client_event_id": "01JY_LOOKUP_0001",
      "event_type": "lookup",
      "source_surface": "video_subtitle",
      "video_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
      "watch_session_id": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
      "recommendation_run_id": "cccccccc-cccc-cccc-cccc-cccccccccccc",
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
      "video_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
      "watch_session_id": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
      "recommendation_run_id": "cccccccc-cccc-cccc-cccc-cccccccccccc",
      "coarse_unit_id": 102,
      "sentence_index": 13,
      "span_index": 1,
      "occurred_at": "2026-05-15T17:00:05Z",
      "exposure_start_ms": 142000,
      "exposure_end_ms": 146300,
      "exposure_count": 1
    },
    {
      "client_event_id": "01JY_SELF_MARK_0001",
      "event_type": "self_mark_mastered",
      "source_surface": "word_detail",
      "video_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
      "coarse_unit_id": 103,
      "token_text": "trivial",
      "occurred_at": "2026-05-15T17:00:08Z"
    }
  ]
}
```

### 5.4 响应结构

```json
{
  "accepted_count": 3,
  "inserted_count": 2,
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
    },
    {
      "client_event_id": "01JY_SELF_MARK_0001",
      "learning_interaction_event_id": "33333333-3333-3333-3333-333333333333",
      "inserted": true
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
  "trigger_type": "practice_now",
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
| `coarse_unit_id` | integer | 是 | 本题对应学习单元。 |
| `video_id` | string UUID | 否 | 触发题目的视频。 |
| `recommendation_run_id` | string UUID | 否 | 触发题目的推荐 run。 |
| `trigger_type` | string | 是 | 触发来源，例如 `practice_now`、`scheduled_review`。 |
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
  "trigger_type": "practice_now",
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
  "trigger_type": "practice_now",
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

## 7. Normalizer 语义

### 7.1 Learning Interaction

```text
NormalizeLearningInteractionsByIDs(user_id, learning_interaction_event_ids)
```

| raw `event_type` | normalized `event_type` | `reducer_effect` | `progress_quality` | 说明 |
| --- | --- | --- | --- | --- |
| `exposure` | `exposure` | `observe_only` | `null` | 只更新 observation，不推进 progress。 |
| `lookup` | `lookup` | `observe_only` | `null` | mapped lookup 进入 Learning Engine；unmapped lookup skipped。 |
| `self_mark_mastered` | `self_mark_mastered` | `set_mastered` | `null` | 直接进入 terminal mastered。 |

### 7.2 Quiz Attempt

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

## 8. 错误与补偿语义

### 8.1 Validation error

任意 validation 失败都不入库。

interaction batch 是整批拒绝，不 partial success。quiz attempt 是单条拒绝。

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

### 8.2 Duplicate

本 API 的幂等命中不是错误。后端返回已有 raw event ID，并标记 `inserted=false`。

### 8.3 Internal normalize failure

如果 raw write 成功，但同步 normalizer 失败，HTTP 仍可以返回 raw accepted。后端需要记录错误日志，后续由 `NormalizePendingEvents` 修复。

前端不需要因为 Learning Engine 内部归约失败而重试；如果前端因网络失败无法确认 raw accepted，才使用同一个 `client_event_id` 重试。

## 9. 前端队列建议

### 9.1 Interaction queue

learning interaction 可以本地排队并批量 flush：

- lookup / self mark 需要尽快 flush。
- exposure 可以短时间聚合后 flush。
- 同一事件重试必须复用 `client_event_id`。
- 不同事件不能共享 `client_event_id`。
- 失败重试时保持原始 `occurred_at`，不要改成重试时间。

### 9.2 Quiz submit

quiz 不进入 interaction batch。完成一道题后直接调用 `POST /api/quiz-attempts`。

如果网络失败，使用同一个 `client_event_id` 重试同一 completed attempt。不要把每次选项点击拆成单独事件上传。

## 10. TypeScript 契约草稿

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
      video_id?: string;
      watch_session_id?: string;
      recommendation_run_id?: string;
      coarse_unit_id: number;
      sentence_index?: number;
      span_index?: number;
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
      video_id?: string;
      watch_session_id?: string;
      recommendation_run_id?: string;
      coarse_unit_id?: number;
      token_text: string;
      sentence_index?: number;
      span_index?: number;
      occurred_at: string;
      lookup_visible_ms?: number;
      lookup_sentence_audio_replay_count?: number;
      lookup_word_audio_play_count?: number;
      lookup_practice_now_clicked?: boolean;
      event_payload?: Record<string, unknown>;
    }
  | {
      client_event_id: string;
      event_type: "self_mark_mastered";
      source_surface: string;
      video_id?: string;
      watch_session_id?: string;
      recommendation_run_id?: string;
      coarse_unit_id: number;
      token_text?: string;
      sentence_index?: number;
      span_index?: number;
      occurred_at: string;
      event_payload?: Record<string, unknown>;
    };

export type RecordLearningInteractionsBatchRequest = {
  client_context?: ClientContext;
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

export type RecordQuizAttemptRequest = {
  client_context?: ClientContext;
  client_event_id: string;
  question_id: string;
  coarse_unit_id: number;
  video_id?: string;
  recommendation_run_id?: string;
  trigger_type: string;
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
```

## 11. 当前实现映射

当前本轮只落应用层和 normalizer 前置结构，不落 HTTP handler。

未来 `internal/api` 的学习事件 handler 应只做该 API 的薄适配：

- 从可信 principal 取 `user_id`。
- 把 JSON request 映射到 `internal/analytics` DTO。
- raw write 成功后，把 raw event IDs 传给 `internal/learningengine/normalizer`。
- 返回 raw accepted response。

通用认证、错误 envelope、状态码、request id、body size、CORS、日志和 handler 目录规则不在本文重复定义，统一遵守 [API模块总体设计规范.md](API模块总体设计规范.md)。不要在 HTTP 层生成 `progress_quality`、`reducer_effect` 或直接写 `learning.*`。
