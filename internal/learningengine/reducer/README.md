# Learning Engine Reducer

`reducer` 是 Learning Engine 内负责 `learning.*` 表的子模块。它消费已经归一化的 Learning Engine events，维护 append-only ledger，并把事件归约成用户学习状态投影。

## Responsibilities

- Owns `learning.unit_learning_events` and `learning.user_unit_states`.
- Owns Learning Engine migrations under `infrastructure/migration`.
- Validates normalized event contracts.
- Appends normalized events idempotently.
- Reduces newly inserted events into `learning.user_unit_states`.
- Replays user state from `learning.unit_learning_events`.
- Maintains target/control commands and state query usecases.

## Boundaries

- Does not read `analytics.*`.
- Does not import `internal/learningengine/normalizer`.
- Does not import `internal/analytics`.
- Does not decide raw fact semantics; that belongs to normalizer.
- Is the only Learning Engine path that writes `learning.*`.

## Directory Structure

```text
internal/learningengine/reducer/
  application/
    dto/
    repository/
    service/
    usecase/
  domain/
    aggregate/
    enum/
    model/
    policy/
  infrastructure/
    migration/
    persistence/
      mapper/
      query/
      repository/
      schema/
      sqlcgen/
      tx/
  test/
    fixture/
    unit/
    integration/
```

## Main Flows

### RecordLearningEvents

```text
request
  -> validate normalized events
  -> group and sort by coarse_unit_id / occurred_at
  -> batch append learning.unit_learning_events
  -> skip duplicate source events
  -> lock affected user_unit_states in one query
  -> reduce only newly inserted events
  -> batch upsert learning.user_unit_states
```

The reducer keeps business state-machine logic in Go. SQL is only responsible for idempotent ledger append, row locking and batch persistence. Duplicate normalized events are ignored at append time and are never reduced.

### ReplayUserStates

```text
request
  -> read existing control snapshot
  -> read all unit_learning_events ordered by occurred_at, event_id
  -> delete current user states
  -> replay reducer from empty state
  -> merge control snapshot
  -> batch upsert rebuilt states
```

Replay never re-reads analytics raw facts. It only uses the reducer-owned normalized ledger.

## Reducer Effects

- `observe_only`: updates observation fields only.
- `affects_progress`: updates observation, progress, schedule, status and mastery fields.
- `set_mastered`: updates observation and moves the unit to terminal mastered state.

`event_type` describes the normalized business event. `reducer_effect` is the reducer dispatch field.

## Time Handling

- `RecordLearningEvents` normalizes `OccurredAt` to UTC before validation and append.
- Persistence mappers use `internal/platform/postgres/pgtime` to write UTC `time.Time` into `timestamptz`.
- Persistence mappers use `internal/platform/postgres/pgtime` to read UTC `time.Time` back into reducer models.
- UUID、nullable text、numeric 等纯 Postgres 类型转换委托 `internal/platform/postgres/*`；reducer 仍保留本地 mapper 函数作为模块边界。
- Integration fixture uses `internal/platform/postgres/pgtest` for embedded Postgres and template database cloning; reducer `test/fixture` owns the reducer schema plan and seed helpers.
