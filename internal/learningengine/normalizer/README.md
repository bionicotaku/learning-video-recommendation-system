# Learning Engine Normalizer

`normalizer` 是 Learning Engine 的 raw fact 解释子模块。它和 `internal/learningengine/reducer` 平级，读取 Analytics 原始事实，把可进入学习引擎的事实转换成 reducer 的 `RecordLearningEvents` 输入。

## Boundaries

- Reads `analytics.quiz_events` and `analytics.learning_interaction_events`.
- Does not write Analytics tables.
- Does not write `learning.unit_learning_events` or `learning.user_unit_states` directly.
- Calls the reducer `RecordLearningEvents` usecase for all Learning Engine writes.
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

### NormalizeLearningInteractionsByIDs

```text
NormalizeLearningInteractionsByIDs
  -> read specified analytics.learning_interaction_events by user_id + event_id
  -> map raw facts with domain/rule
  -> group normalized events by user_id
  -> call reducer.RecordLearningEvents
```

This is the main path for the future `POST /api/learning-interactions:batch` API after Analytics raw write returns raw `event_id` values.

### NormalizeQuizAttemptByID

```text
NormalizeQuizAttemptByID
  -> read specified analytics.quiz_events by user_id + event_id
  -> map raw fact with domain/rule
  -> call reducer.RecordLearningEvents
```

This is the main path for the future `POST /api/quiz-attempts` API after Analytics raw write returns `quiz_event_id`.

### NormalizePendingEvents

```text
NormalizePendingEvents
  -> read pending raw facts with anti-join against learning.unit_learning_events
  -> map raw facts with domain/rule
  -> group normalized events by user_id
  -> call reducer.RecordLearningEvents
```

`NormalizePendingEvents` is repair/backfill. `source_kind` supports `all`, `quiz`, and `learning_interaction`; empty value defaults to `all`. `limit=0` defaults to `500`, and values above `1000` are capped to `1000`.

## Persistence

The normalizer owns a separate SQLC package under `infrastructure/persistence/sqlcgen` with package name `normalizersqlc`.

The SQL layer is read-only for Analytics and joins against reducer-owned `learning.unit_learning_events`:

- `raw_quiz_events.sql` reads pending `analytics.quiz_events`.
- `raw_quiz_events.sql` also reads specified quiz raw rows for the by-IDs API path.
- `raw_learning_interaction_events.sql` reads pending `analytics.learning_interaction_events` where `coarse_unit_id is not null`.
- `raw_learning_interaction_events.sql` also reads specified interaction raw rows for the by-IDs API path; unmapped lookup rows can be read and then skipped by the mapper.

Pending queries exclude rows already present in `learning.unit_learning_events` by `user_id + source_type + source_ref_id + coarse_unit_id`. The by-IDs path relies on reducer `RecordLearningEvents` idempotent append, so duplicates are counted and not reduced again.

## Tests

- Unit tests live under `test/unit`.
- Real Postgres tests live under `test/integration`.
- The integration fixture applies minimal external refs, Analytics migrations, and Learning Engine migrations.
