# Learning Engine Normalizer

`normalizer` 是 Learning Engine 的 raw fact 解释子模块。它读取 Analytics 原始事实，把可进入学习引擎的事实转换成 `RecordLearningEvents` 输入。

## Boundaries

- Reads `analytics.quiz_events` and `analytics.learning_interaction_events`.
- Does not write Analytics tables.
- Does not write `learning.unit_learning_events` or `learning.user_unit_states` directly.
- Calls the existing `RecordLearningEvents` usecase for all Learning Engine writes.
- Does not maintain checkpoint or rollup tables in MVP.

## Directory Structure

```text
internal/learningengine/normalizer/
  application/
    dto/
    repository/
    service/
    usecase/
  domain/
    model/
    policy/
    rule/
  infrastructure/
    persistence/
      mapper/
      query/
      repository/
      schema/
      sqlcgen/
  test/
    fixture/
    unit/
    integration/
```

## Current Rules

- Quiz is the only ordinary progress signal.
- Quiz quality uses `quiz_speed_threshold_ms = 5000`:
  - first-try correct and `total_elapsed_ms <= 5000` -> `5`
  - first-try correct and `total_elapsed_ms > 5000` -> `4`
  - first-try wrong and `total_elapsed_ms <= 5000` -> `2`
  - first-try wrong and `total_elapsed_ms > 5000` -> `1`
- Lookup and exposure are `observe_only`.
- Self mark is `set_mastered`.

## Current Flow

```text
NormalizePendingEvents
  -> read pending raw facts with anti-join against learning.unit_learning_events
  -> map raw facts with domain/rule
  -> group normalized events by user_id
  -> call RecordLearningEvents
```

`source_kind` supports `all`, `quiz`, and `learning_interaction`; empty value defaults to `all`. `limit=0` defaults to `500`, and values above `1000` are capped to `1000`.

## Persistence

The normalizer owns a separate SQLC package under `infrastructure/persistence/sqlcgen` with package name `normalizersqlc`.

The SQL layer is read-only for Analytics:

- `raw_quiz_events.sql` reads pending `analytics.quiz_events`.
- `raw_learning_interaction_events.sql` reads pending `analytics.learning_interaction_events` where `coarse_unit_id is not null`.

Both queries exclude rows already present in `learning.unit_learning_events` by `user_id + source_type + source_ref_id + coarse_unit_id`.

## Tests

- Unit tests live under `test/unit`.
- Real Postgres tests live under `test/integration`.
- The integration fixture applies minimal external refs, Analytics migrations, and Learning Engine migrations.
