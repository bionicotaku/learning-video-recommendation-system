# 学习引擎 Normalizer 设计

## 0. 文档信息

文档状态：MVP 设计草案  
目标读者：后端、数据、前端、后续接手维护的人  
当前范围：定义 Learning Engine normalizer 的模块边界、输入输出、映射规则、质量分语义和 MVP 实现结构。  
当前明确不做：不新增 checkpoint 表，不新增跨视频 exposure rollup 表，不让弱互动直接推进 progress。

本文承接：

- [学习引擎设计.md](学习引擎设计.md)
- [学习互动信号语义设计.md](学习互动信号语义设计.md)
- [学习互动信号架构图.md](学习互动信号架构图.md)
- [题目入库文档.md](题目入库文档.md)

## 1. 一句话结论

Normalizer 是 Learning Engine 的 raw fact 解释子模块。

它 read-only 读取 `analytics.*` 原始互动事实，把可解释、可绑定 `coarse_unit_id` 的事实转换成 Learning Engine normalized event，然后调用现有 `RecordLearningEvents` 链路写入：

```text
analytics raw fact
        ↓
internal/learningengine/normalizer
        ↓
RecordLearningEvents
        ↓
learning.unit_learning_events
        ↓
reducer
        ↓
learning.user_unit_states
```

Normalizer 不直接写 `learning.unit_learning_events`，不直接更新 `learning.user_unit_states`，不绕过 reducer。

## 2. Owner 与模块边界

### 2.1 为什么属于 Learning Engine

`analytics` 的职责是保存原始事实：

- 用户看了什么；
- 点了什么；
- 查了什么；
- 答了什么；
- 用了多久；
- 当时客户端上下文是什么。

这些事实本身不是学习结论。

Normalizer 的职责是解释这些事实对学习状态意味着什么：

- 能不能绑定 `coarse_unit_id`；
- 是 `exposure`、`lookup`、`quiz` 还是 `self_mark_mastered`；
- 是否推进 progress；
- 如果推进，`progress_quality` 是多少；
- 如果是 self mark，是否直接 `set_mastered`。

这些解释规则属于 Learning Engine 的学习语义，因此 normalizer 放在 `internal/learningengine/normalizer`，而不是放在 `internal/analytics`。

### 2.2 和 Learning Engine reducer 的关系

Normalizer 不是 reducer 的一部分。

Reducer 只消费已经归一化的 event：

```text
event_type
reducer_effect
progress_quality
source_type
source_ref_id
metadata
```

Reducer 不读取 `analytics.*`，不重新解释 raw fact。

Normalizer 负责在 reducer 之前做解释：

```text
analytics.quiz_events
analytics.learning_interaction_events
        ↓
normalizer rules
        ↓
dto.RecordLearningEventsRequest
```

这样可以保持两层语义清晰：

- normalizer 负责 raw fact -> normalized event；
- reducer 负责 normalized event -> user unit state。

### 2.3 和 Analytics 的关系

Normalizer 可以 read-only 读取 Analytics owner 的表：

- `analytics.quiz_events`
- `analytics.learning_interaction_events`

Normalizer 不修改 analytics raw 表。

当前 MVP 不新增 `normalized_at`、`normalization_status` 之类字段到 analytics 表，也不新增 checkpoint 表。

### 2.4 和 Recommendation 的关系

Recommendation 不调用 normalizer，也不写 Learning Engine 表。

Recommendation 只通过 `learning.user_unit_states` 消费归约后的学习状态。

Recommendation 返回给前端的 `learning_units` 会影响前端产生哪些 exposure 候选，但 Recommendation 不负责解释 exposure 是否推进学习进度。

## 3. 设计目标

1. 让 raw fact 到 Learning Engine event 的解释规则有唯一 owner。
2. 保持 `learning.unit_learning_events` 仍然是 replay 的唯一事实来源。
3. 让所有 normalized event 写入都复用 `RecordLearningEvents`。
4. 让 quiz 成为 MVP 主要 progress 信号。
5. 让 lookup / exposure 保守地只更新 observation。
6. 让 self mark 使用 `set_mastered`，不伪装成 quality 5。
7. MVP 不引入 checkpoint 表，先依赖 source 幂等约束和可重扫查询。

## 4. 非目标

MVP 不做：

- 不按跨视频 exposure 自动推进 progress。
- 不新增 `learning.user_unit_states.exposure_without_lookup_count`。
- 不新增 `analytics.user_unit_signal_rollups`。
- 不新增 `learning.normalization_checkpoints`。
- 不把 lookup 停留、音频播放、练一下直接解释成 progress。
- 不让前端上传 `progress_quality`。
- 不让前端判断 `reducer_effect`。
- 不把完整题目、选项、用户答案塞进 Learning Engine ledger。

## 5. 输入表

### 5.1 `analytics.quiz_events`

`analytics.quiz_events` 是答题原始事实。当前可用于 normalizer 的关键字段：

| 字段 | 用途 |
| --- | --- |
| `event_id` | normalized event 的 `source_ref_id`。 |
| `user_id` | Learning Engine event 的用户。 |
| `question_id` | 写入 metadata，便于追溯题目。 |
| `coarse_unit_id` | Learning Engine event 的学习单元。 |
| `video_id` | 可写入 normalized event 的 `video_id`。 |
| `recommendation_run_id` | 写入 metadata，便于追溯推荐来源。 |
| `trigger_type` | 写入 metadata，区分 video_end / lookup_practice / feed_review / mid_video / manual。 |
| `selected_option_ids` | 写入 metadata；用于确认是否首次正确。 |
| `selection_interval_ms` | 写入 metadata；当前不直接参与 quality。 |
| `is_first_try_correct` | quality 主要输入。 |
| `total_elapsed_ms` | quality 主要输入。 |
| `shown_at` | 写入 metadata。 |
| `completed_at` | normalized event 的 `occurred_at`。 |

### 5.2 `analytics.learning_interaction_events`

`analytics.learning_interaction_events` 保存 exposure、lookup、self mark 等非答题互动原始事实。当前可用于 normalizer 的关键字段：

| 字段 | 用途 |
| --- | --- |
| `event_id` | normalized event 的 `source_ref_id`。 |
| `user_id` | Learning Engine event 的用户。 |
| `event_type` | raw 类型：`exposure` / `lookup` / `self_mark_mastered`。 |
| `source_surface` | 写入 metadata，说明来源界面。 |
| `video_id` | 可写入 normalized event 的 `video_id`。 |
| `watch_session_id` | 写入 metadata。 |
| `recommendation_run_id` | 写入 metadata。 |
| `related_quiz_event_id` | 写入 metadata；不直接影响 progress。 |
| `coarse_unit_id` | 为空则不能进入 Learning Engine。 |
| `token_text` | 写入 metadata。 |
| `sentence_index` | 写入 metadata。 |
| `span_index` | 写入 metadata。 |
| `occurred_at` | normalized event 的 `occurred_at`。 |
| `exposure_start_ms` / `exposure_end_ms` / `exposure_count` | exposure metadata。 |
| `lookup_visible_ms` | lookup 附加字段 metadata。 |
| `lookup_sentence_audio_replay_count` | lookup 附加行为 metadata。 |
| `lookup_word_audio_play_count` | lookup 附加行为 metadata。 |
| `lookup_practice_now_clicked` | lookup 附加字段 metadata，不直接生成 progress。 |
| `event_payload` | 作为 metadata 的补充来源。 |

## 6. 输出契约

Normalizer 输出的是 `RecordLearningEvents` 可消费的 DTO，不是 SQL row。

核心字段映射：

| 输出字段 | 规则 |
| --- | --- |
| `user_id` | 来自 raw fact。 |
| `coarse_unit_id` | 来自 raw fact，必须非空。 |
| `video_id` | raw fact 有则带上。 |
| `event_type` | `exposure` / `lookup` / `quiz` / `self_mark_mastered`。 |
| `reducer_effect` | `observe_only` / `affects_progress` / `set_mastered`。 |
| `progress_quality` | 仅 `affects_progress` 使用。 |
| `is_correct` | quiz 填 `is_first_try_correct`；其他事件为空。 |
| `source_type` | `quiz_event` 或 `learning_interaction_event`。 |
| `source_ref_id` | raw fact 的 `event_id` 字符串。 |
| `metadata` | raw 上下文的裁剪版 JSON object。 |
| `occurred_at` | quiz 用 `completed_at`；互动事件用 `occurred_at`。 |

幂等依赖 Learning Engine 现有唯一约束：

```sql
unique (user_id, source_type, source_ref_id, coarse_unit_id)
```

因此同一 raw fact 被 normalizer 重试时，不应产生重复 Learning Engine event。

## 7. Normalizer 规则总表

| Raw fact / 场景 | 是否进入 Learning Engine | normalized `event_type` | `reducer_effect` | `progress_quality` | `is_correct` | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `self_mark_mastered`，有 `coarse_unit_id` | 是 | `self_mark_mastered` | `set_mastered` | `null` | `null` | 用户主动声明已掌握，直接 terminal mastered。 |
| quiz 首次答对，`total_elapsed_ms <= 5000` | 是 | `quiz` | `affects_progress` | `5` | `true` | 快速首次正确，表示稳定回忆。 |
| quiz 首次答对，`total_elapsed_ms > 5000` | 是 | `quiz` | `affects_progress` | `4` | `true` | 慢速首次正确，正常通过。 |
| quiz 首次答错，`total_elapsed_ms <= 5000` | 是 | `quiz` | `affects_progress` | `2` | `false` | 快速失败；未通过但有明确验证反馈。 |
| quiz 首次答错，`total_elapsed_ms > 5000` | 是 | `quiz` | `affects_progress` | `1` | `false` | 慢速失败；比快速失败更弱。 |
| quiz 超时 / 跳过 / 放弃 | 当前 schema 暂不支持 | `quiz` | `affects_progress` | `0` | `false` | 未来有明确字段后再启用。 |
| mapped lookup | 是 | `lookup` | `observe_only` | `null` | `null` | 只要 lookup 能绑定 `coarse_unit_id` 就生成一条 observe-only event；停留、音频、练一下等附加字段只进 metadata，不改变入库判断。 |
| unmapped lookup | 否 | - | - | - | - | 只保留 analytics raw fact。 |
| 推荐目标 unit 的有效 exposure | 是 | `exposure` | `observe_only` | `null` | `null` | 只表示用户可能接触过。 |
| 非推荐目标 unit 的普通 exposure | MVP 不生成 | - | - | - | - | 避免全视频词表污染学习状态。 |
| 跨视频多次 exposure 且从未 lookup | MVP 不进 | - | - | - | - | 语义上是 passive familiarity candidate，MVP 不推进 progress。 |

## 8. Quiz quality policy

### 8.1 质量分语义

`progress_quality` 表示“这次验证是否足以推进记忆状态”，不是“用户最后是否点到了正确答案”。

MVP 规则：

```text
if is_first_try_correct = true and total_elapsed_ms <= 5000:
  progress_quality = 5

if is_first_try_correct = true and total_elapsed_ms > 5000:
  progress_quality = 4

if is_first_try_correct = false and total_elapsed_ms <= 5000:
  progress_quality = 2

if is_first_try_correct = false and total_elapsed_ms > 5000:
  progress_quality = 1
```

含义：

- `5`：快速首次正确，强通过。
- `4`：慢速首次正确，正常通过。
- `2`：快速首次错误，不能算通过。
- `1`：慢速首次错误，失败程度更强。

Reducer 当前以 `progress_quality >= 3` 作为 pass，所以只有首次正确会推进 pass path。

### 8.2 为什么快慢阈值是 5000ms

MVP 推荐固定：

```text
quiz_speed_threshold_ms = 5000
```

原因：

- 当前 schema 稳定提供 `is_first_try_correct` 和 `total_elapsed_ms`。
- 5000ms 对当前选择题交互更保守，既能识别明显快答，也避免把正常阅读题干的答题误判为慢速。
- 固定阈值方便产品、数据和后端共同观察效果。
- 后续可以在不改 reducer 的情况下调整 normalizer policy。

未来如果题型扩展，可以改成按题型配置：

| 题型 | 未来可选快慢阈值 |
| --- | --- |
| definition choice | `4000ms` |
| context choice | `5000ms` |
| cloze / fill blank | `6000ms` |
| typing / production | `8000ms` |

但 MVP 不做题型差异化。

### 8.3 错一次和错多次不区分

当前 `analytics.quiz_events` 保存 `selected_option_ids`，可以知道用户选了多少次。

MVP 不区分：

```text
wrong_selection_count = 1
wrong_selection_count > 1
```

只要 `is_first_try_correct = false`，错误次数不影响 quality；只按总耗时区分快慢：

```text
total_elapsed_ms <= 5000 -> progress_quality = 2
total_elapsed_ms > 5000 -> progress_quality = 1
```

这样可以避免 normalizer 把 UI 交互细节过度放大。错误次数仍然可以写入 metadata，供后续分析。

## 9. Lookup policy

Lookup 的语义是主动关注或疑似困难。

它不能直接代表：

```text
用户理解了
用户记住了
用户已经掌握
```

因此 MVP 中 lookup 永远不推进 progress：

```text
event_type = lookup
reducer_effect = observe_only
progress_quality = null
```

### 9.1 mapped lookup

当 `analytics.learning_interaction_events.event_type = 'lookup'` 且 `coarse_unit_id is not null`：

```text
source_type = learning_interaction_event
source_ref_id = learning_interaction_events.event_id
event_type = lookup
reducer_effect = observe_only
progress_quality = null
```

metadata 建议包含：

```json
{
  "source_surface": "video_subtitle",
  "token_text": "barely",
  "sentence_index": 12,
  "span_index": 3,
  "lookup_visible_ms": 5200,
  "lookup_sentence_audio_replay_count": 1,
  "lookup_word_audio_play_count": 2,
  "lookup_practice_now_clicked": false,
  "recommendation_run_id": "..."
}
```

### 9.2 unmapped lookup

当 lookup 无法绑定 `coarse_unit_id`：

```text
不生成 Learning Engine event
只保留 analytics raw fact
```

原因是 Learning Engine 的最小归约单位是 `user_id + coarse_unit_id`。没有稳定 unit 绑定，就不能进入 reducer。

### 9.3 lookup 附加字段

lookup 附加字段不参与是否生成 Learning Engine event 的判断。MVP 只有一个分支：

```text
coarse_unit_id is not null -> 生成 lookup observe_only
coarse_unit_id is null -> 只保留 analytics raw fact
```

以下字段只原样写入 metadata：

- `lookup_visible_ms`
- `lookup_sentence_audio_replay_count`
- `lookup_word_audio_play_count`
- `lookup_practice_now_clicked`

它们未来可用于：

- 练习触发分析；
- 推荐解释；
- difficulty / interest 派生特征；
- normalizer 规则迭代。

但当前不产生额外 lookup 分支，不单独生成 Learning Engine event，也不写 `affects_progress`。

## 10. Exposure policy

Exposure 的语义是“用户可能接触过这个 unit”。

单次 exposure 不能证明用户注意到了，也不能证明用户理解了。

MVP 中 exposure 只写 observation：

```text
event_type = exposure
reducer_effect = observe_only
progress_quality = null
```

### 10.1 有效 exposure

一个 exposure 进入 Learning Engine，必须至少满足：

```text
event_type = exposure
coarse_unit_id is not null
video_id is not null
watch_session_id is not null
```

并且它应该来自前端或后端 watch session aggregator 对当前 `learning_units` 的聚合结果。

MVP 语义：

```text
一次 watch session 中，一个 video + coarse_unit_id 最多形成一次 exposure 候选。
```

### 10.2 非推荐目标 unit 的 exposure

MVP 不对视频中所有 token 做全量 exposure。

只有当前视频推荐返回的 `learning_units` 才应该形成 exposure 候选。普通非目标 token 即使在字幕中出现，也不生成 Learning Engine exposure。

## 11. Self mark policy

`self_mark_mastered` 是用户主动声明“我已经会了”。

它不是 quiz quality，不等于 `progress_quality = 5`。

Normalizer 必须生成：

```text
event_type = self_mark_mastered
reducer_effect = set_mastered
progress_quality = null
source_type = learning_interaction_event
source_ref_id = analytics.learning_interaction_events.event_id
```

Reducer 会把该 unit 收敛为：

```text
status = mastered
progress_percent = 100
mastery_score = 1
next_review_at = null
is_target = false
suspended_reason = ''
```

## 12. 跨视频 exposure 语义

跨视频 exposure 是一个未来可用的解释上下文，不是 MVP progress 输入。

语义上可以定义：

```text
passive_familiarity_candidate:
  user_id + coarse_unit_id
  distinct_video_exposure_count >= 3
  lookup_count = 0
  quiz_failure_count = 0
  self_mark_mastered does not exist
```

含义：

用户在多个视频中自然接触过这个 unit，并且没有主动 lookup 或失败反馈，可能已经有被动熟悉度。

但 MVP 不做以下事情：

- 不新增 rollup 表。
- 不把它转成 `progress_quality = 3`。
- 不进入 SM-2 schedule path。
- 不修改 `learning.user_unit_states` schema。

如果未来需要使用，优先顺序是：

1. normalizer 或分析任务直接查询 `analytics.learning_interaction_events`。
2. 如果查询成本或一致性成为问题，再建立 rollup 表。
3. rollup 表优先服务 normalizer 或 analytics，不污染 `learning.user_unit_states`。
4. 真要影响学习状态，应新增明确语义，而不是把 passive familiarity 伪装成 quiz pass。

未来可能的 rollup 字段：

```text
user_id
coarse_unit_id
exposure_count
distinct_exposure_video_count
last_exposure_at
lookup_count
last_lookup_at
quiz_attempt_count
quiz_failure_count
last_quiz_at
```

当前不落表。

## 13. 暂不加 checkpoint 的幂等策略

MVP 不新增：

```text
learning.normalization_checkpoints
analytics.normalized_at
analytics.normalization_status
```

### 13.1 如何避免重复写入

依赖 `learning.unit_learning_events` 的唯一约束：

```sql
unique (user_id, source_type, source_ref_id, coarse_unit_id)
```

Normalizer 每次生成 event 时固定：

```text
quiz:
  source_type = quiz_event
  source_ref_id = analytics.quiz_events.event_id

learning interaction:
  source_type = learning_interaction_event
  source_ref_id = analytics.learning_interaction_events.event_id
```

同一 raw fact 重试不会产生重复 normalized event。

### 13.2 如何查询 pending raw facts

由于没有 checkpoint，pending 查询应优先使用 anti-join：

```sql
select q.*
from analytics.quiz_events q
where not exists (
  select 1
  from learning.unit_learning_events e
  where e.user_id = q.user_id
    and e.source_type = 'quiz_event'
    and e.source_ref_id = q.event_id::text
    and e.coarse_unit_id = q.coarse_unit_id
)
order by q.completed_at asc, q.event_id asc
limit $1;
```

Learning interaction 也是同样模式：

```sql
select i.*
from analytics.learning_interaction_events i
where i.coarse_unit_id is not null
  and i.event_type in ('exposure', 'lookup', 'self_mark_mastered')
  and not exists (
    select 1
    from learning.unit_learning_events e
    where e.user_id = i.user_id
      and e.source_type = 'learning_interaction_event'
      and e.source_ref_id = i.event_id::text
      and e.coarse_unit_id = i.coarse_unit_id
  )
order by i.occurred_at asc, i.event_id asc
limit $1;
```

### 13.3 skipped raw facts 怎么处理

没有 checkpoint 时，skipped fact 分两类：

1. 永久不可 normalize 的事实，例如 `coarse_unit_id is null`。
2. 当前策略选择不进入 Learning Engine 的 learning interaction fact。

MVP 查询应尽量在 SQL 层排除这些事实，避免每轮重复读：

- quiz 当前 schema 本身要求 `coarse_unit_id not null`。
- learning interaction pending 查询只取 `coarse_unit_id is not null`。

如果未来出现“有 coarse_unit_id 但策略上 skipped”的大量事实，再补 checkpoint 或 skip ledger。

### 13.4 失败重试

如果调用 `RecordLearningEvents` 失败：

- 不写 checkpoint；
- 不标记 raw fact；
- 下次 normalizer 扫描仍会读到它；
- 幂等约束保证已成功写入的 event 不重复。

这意味着 MVP 的失败语义是 at-least-once normalize attempt + idempotent write。

## 14. 应用层用例

推荐 usecase：

```text
NormalizePendingLearningEvents
```

输入 DTO：

```text
user_id optional
source_kind optional: quiz / learning_interaction / all
limit
occurred_before optional
```

输出 DTO：

```text
read_raw_count
normalized_event_count
skipped_count
recorded_event_count
error_count
```

MVP 可以先按批次处理：

```text
1. 读取 pending quiz raw facts。
2. 调用 QuizMapper 生成 normalized events。
3. 按 user_id 分组，调用 RecordLearningEvents。
4. 读取 pending learning interaction raw facts。
5. 调用 LearningInteractionMapper 生成 normalized events。
6. 按 user_id 分组，调用 RecordLearningEvents。
```

如果同一批里同一个用户同一个 unit 同时有 exposure、lookup、quiz，`RecordLearningEvents` 会按 `coarse_unit_id` 分组并按 `occurred_at` 排序。

## 15. 领域规则结构

建议在 `internal/learningengine/normalizer/domain` 中拆成三类。

### 15.1 Policy

稳定参数和判定策略：

```text
domain/policy/quiz_quality_policy.go
domain/policy/interaction_effect_policy.go
domain/policy/exposure_policy.go
```

规则示例：

```text
quiz_speed_threshold_ms = 5000
```

### 15.2 Rule / Mapper

raw fact 到 normalized event 的纯映射：

```text
domain/rule/quiz_mapper.go
domain/rule/learning_interaction_mapper.go
```

Mapper 输入领域模型，输出：

```text
NormalizationResult
  - normalized event
  - skipped reason
```

### 15.3 Model

定义 normalizer 自己的 raw read model 和输出模型：

```text
domain/model/raw_quiz_event.go
domain/model/raw_learning_interaction.go
domain/model/normalization_result.go
```

这些 model 不应直接暴露 `sqlc` 类型。

## 16. 推荐代码结构

建议新增子模块：

```text
internal/learningengine/normalizer/
  README.md
  doc.go

  application/
    dto/
      normalize_events.go
    repository/
      raw_learning_interaction_reader.go
      raw_quiz_event_reader.go
    service/
      normalize_pending_events.go
      normalize_quiz_events.go
      normalize_learning_interactions.go
    usecase/
      normalize_pending_events.go

  domain/
    enum/
      raw_event_type.go
    model/
      raw_learning_interaction.go
      raw_quiz_event.go
      normalization_result.go
    policy/
      quiz_quality_policy.go
      interaction_effect_policy.go
      exposure_policy.go
    rule/
      quiz_mapper.go
      learning_interaction_mapper.go

  infrastructure/
    persistence/
      mapper/
      query/
        raw_quiz_events.sql
        raw_learning_interaction_events.sql
      repository/
      schema/
      sqlcgen/

  test/
    fixture/
    unit/
      domain/
      application/
    integration/
      infrastructure/
      application/
```

当前不需要：

```text
normalizer/infrastructure/migration/
normalizer/infrastructure/persistence/query/normalization_checkpoints.sql
normalizer/application/repository/normalization_checkpoint_repository.go
```

因为本轮明确暂不加 checkpoint。

## 17. Repository ports

### 17.1 `RawQuizEventReader`

职责：

```text
ListPendingQuizEvents(ctx, filter) ([]RawQuizEvent, error)
```

只读 `analytics.quiz_events`，并 anti-join `learning.unit_learning_events` 排除已 normalized 的 raw fact。

### 17.2 `RawLearningInteractionReader`

职责：

```text
ListPendingLearningInteractions(ctx, filter) ([]RawLearningInteraction, error)
```

只读 `analytics.learning_interaction_events`，并 anti-join `learning.unit_learning_events` 排除已 normalized 的 raw fact。

### 17.3 `LearningEventRecorder`

Normalizer 不应该直接依赖 repository 写 `unit_learning_events`。

更合适的 port 是应用层 usecase：

```text
RecordLearningEvents(ctx, dto.RecordLearningEventsRequest) (dto.RecordLearningEventsResponse, error)
```

实现可以直接注入现有 `learningengine/application/usecase.RecordLearningEventsUsecase`。

## 18. Metadata 契约

Learning Engine reducer 不依赖 metadata 做核心分发，但 metadata 对审计、排障和后续策略迭代重要。

### 18.1 quiz metadata

建议：

```json
{
  "question_id": "...",
  "trigger_type": "video_end",
  "recommendation_run_id": "...",
  "selected_option_ids": ["wrong_1", "correct"],
  "selection_interval_ms": [1200, 2600],
  "wrong_selection_count": 1,
  "total_elapsed_ms": 3800,
  "shown_at": "2026-05-15T10:00:00Z",
  "completed_at": "2026-05-15T10:00:03.8Z",
  "quality_policy": {
    "name": "quiz_first_try_speed_v1",
    "quiz_speed_threshold_ms": 5000
  }
}
```

### 18.2 lookup metadata

建议：

```json
{
  "source_surface": "video_subtitle",
  "video_id": "...",
  "watch_session_id": "...",
  "recommendation_run_id": "...",
  "token_text": "barely",
  "sentence_index": 12,
  "span_index": 3,
  "lookup_visible_ms": 5200,
  "lookup_sentence_audio_replay_count": 1,
  "lookup_word_audio_play_count": 2,
  "lookup_practice_now_clicked": false
}
```

### 18.3 exposure metadata

建议：

```json
{
  "source_surface": "video_player",
  "video_id": "...",
  "watch_session_id": "...",
  "recommendation_run_id": "...",
  "exposure_start_ms": 12000,
  "exposure_end_ms": 14500,
  "exposure_count": 1,
  "aggregation": {
    "scope": "watch_session_video_unit",
    "max_once_per_session": true
  }
}
```

### 18.4 self mark metadata

建议：

```json
{
  "source_surface": "lookup_modal",
  "video_id": "...",
  "watch_session_id": "...",
  "recommendation_run_id": "...",
  "token_text": "barely",
  "sentence_index": 12,
  "span_index": 3
}
```

## 19. 错误处理

### 19.1 raw fact 不合法

所有 raw fact 在进入 mapper 前都必须先通过基础 validation。validation 失败的 raw fact 不生成 normalized event，也不进入 Learning Engine。

常见 validation 失败包括：

```text
coarse_unit_id missing
user_id missing
occurred_at missing
event time fields invalid
event payload shape invalid
event type unsupported
source context inconsistent
```

MVP 行为：

- 不生成 normalized event；
- 不写 checkpoint；
- 不修改 raw 表；
- 在 normalizer response / log 中返回 skipped reason。

### 19.2 RecordLearningEvents 失败

如果 `RecordLearningEvents` 失败：

- 整批或当前用户组失败；
- 不做部分成功假设；
- 下一轮通过 anti-join 自动跳过已经成功写入的 event；
- 未写入的 event 会再次被扫描。

### 19.3 late progress event

Quiz 是 `affects_progress`，如果发生时间早于当前 `last_progress_at`，reducer 会拒绝 late progress event。

Normalizer 不应吞掉这个错误。

MVP 可以把该错误返回给调用方或记录日志。由于没有 checkpoint，该 raw fact 下次仍会被扫描；如果 late event 长期存在，后续需要 checkpoint 或 dead-letter 机制。

## 20. 调用方式

MVP 可先支持同步调用：

```text
POST /internal/learningengine/normalizer/run
```

或 CLI / job 调用：

```text
learningengine-normalize --source=all --limit=500
```

本文不定 API 路径。实现时应优先先落 application usecase，再由 API 或 job 调用。

## 21. 测试策略

### 21.1 Domain unit tests

Quiz quality:

- 首次正确且 `total_elapsed_ms = 5000` -> quality 5。
- 首次正确且 `total_elapsed_ms = 5001` -> quality 4。
- 首次错误且 `total_elapsed_ms = 5000` -> quality 2。
- 首次错误且 `total_elapsed_ms = 5001` -> quality 1。
- 错一次和错多次不影响 quality。

Learning interaction:

- mapped lookup -> observe_only。
- unmapped lookup -> skipped。
- lookup 附加字段不改变 reducer_effect。
- valid exposure -> observe_only。
- self_mark_mastered -> set_mastered。
- validation 失败的 raw fact 不调用 mapper。

### 21.2 Application unit tests

- pending quiz 被映射并调用 `RecordLearningEvents`。
- pending interaction 被映射并调用 `RecordLearningEvents`。
- skipped raw fact 不调用 recorder。
- 同一批按 `user_id` 分组调用 recorder。
- recorder 返回错误时 usecase 返回错误。

### 21.3 Integration tests

- anti-join 能排除已经写入 `learning.unit_learning_events` 的 quiz raw fact。
- anti-join 能排除已经写入的 learning interaction raw fact。
- real Postgres 下 quiz -> Learning Engine state 更新成功。
- self_mark -> terminal mastered 成功。
- lookup/exposure 只更新 observation，不更新 progress fields。

## 22. MVP 实施顺序

推荐分三步。

第一步：纯 domain mapper。

```text
RawQuizEvent -> LearningEventInput
RawLearningInteraction -> LearningEventInput / skipped
```

第二步：normalizer application usecase。

```text
read pending raw facts
map
group by user
call RecordLearningEvents
return summary
```

第三步：infrastructure reader。

```text
sqlc query with anti-join
repository mapper
integration tests
```

暂不做：

- checkpoint；
- async worker；
- retry table；
- rollup table；
- exposure-based progress。

## 23. 当前设计决策

1. Normalizer 是 `internal/learningengine/normalizer` 子模块。
2. Analytics 表只读，不由 normalizer 修改。
3. Normalizer 不直接写 Learning Engine 表，只调用 `RecordLearningEvents`。
4. Quiz 是 MVP 唯一普通 progress 信号。
5. `self_mark_mastered` 走 `set_mastered`，不使用 quality。
6. Lookup 和 exposure 只写 `observe_only`。
7. Quiz 快慢阈值固定为 `5000ms`。
8. Quiz 首次快速答对给 `5`，首次慢速答对给 `4`，首次快速答错给 `2`，首次慢速答错给 `1`。
9. 跨视频 exposure 只保留语义，不在 MVP 推进 progress。
10. 暂不新增 checkpoint 表，依赖 source 幂等约束和 anti-join 查询。
