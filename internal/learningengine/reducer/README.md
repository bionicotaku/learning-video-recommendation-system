# Learning Engine Reducer

`reducer` 是 Learning Engine 内负责 `learning.*` 表的子模块。它消费已经归一化的 Learning Engine events，维护 append-only ledger，并把事件归约成用户学习状态投影。

## Responsibilities

- Owns `learning.unit_learning_events` and `learning.user_unit_states`.
- Owns Learning Engine migrations under `infrastructure/migration`.
- Validates normalized event contracts.
- Appends normalized events idempotently.
- Reduces newly inserted events into `learning.user_unit_states`.
- Provides direct user-scoped commands that append reducer-owned events, such as
  reset-unlearned.
- Replays user state from `learning.unit_learning_events`.
- Maintains target/control commands and state query usecases.
- Provides the Unit Progress read usecase for frontend mastered/unmastered
  progress lists.

## Boundaries

- Does not read `analytics.*`.
- Does not import `internal/learningengine/normalizer`.
- Does not import `internal/analytics`.
- Does not decide raw fact semantics; that belongs to normalizer.
- Is the only Learning Engine path that writes `learning.*`.
- Batch target commands write through reducer-owned SQL; `EnsureTargetUnits` uses one batch upsert rather than one roundtrip per target.

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
  -> group and preserve accepted ledger order
  -> lock affected user_unit_states in one query
  -> skip non-reset events at or before state latest_reset_boundary_at
  -> batch append learning.unit_learning_events
  -> skip duplicate source events
  -> reduce only newly inserted events
  -> update state projection watermarks
  -> batch upsert learning.user_unit_states
```

The reducer keeps business state-machine logic in Go. SQL is only responsible for idempotent ledger append, row locking and batch persistence. Duplicate normalized events are ignored at append time and are never reduced.

### ReplayUserStates

```text
request
  -> read existing control snapshot
  -> read all unit_learning_events ordered by ledger_seq
  -> delete current user states
  -> replay reducer from empty state
  -> merge control snapshot
  -> batch upsert rebuilt states
```

Replay never re-reads analytics raw facts. It only uses the reducer-owned normalized ledger.
`ledger_seq` is the authoritative replay order; `occurred_at` remains business
event time only. Replay always merges the current control snapshot back into
rebuilt states, including mastered rows.

### ResetUserUnitProgress

```text
request
  -> require existing learning.user_unit_states row for user_id + coarse_unit_id
  -> lookup existing reset event by user_id + client_event_id
  -> compute reset_boundary_at
  -> append reset_unlearned event to learning.unit_learning_events
  -> reduce newly inserted event
  -> upsert learning.user_unit_states
```

Reset is a reducer-owned direct command, not an Analytics raw fact and not a
Normalizer repair target. It accepts any existing user-unit state row, including
`is_target=false` and already mastered rows. The reducer resets progress,
observation, recent quality, streak and schedule fields to unlearned defaults,
while preserving target/control fields such as `is_target`,
`target_source`, `target_source_ref_id` and `target_priority`.

`client_event_id` is user-scoped for reset commands. A duplicate
`user_id + client_event_id` returns the existing `reset_unlearned` event ID and
does not reduce the request body unit again. The database enforces this with a
partial unique index on `learning.unit_learning_events` for
`source_type = 'learning_unit_reset'`.

`reset_boundary_at` is stored only on reset events. It is computed as the max of
the request `occurred_at`,
`learning.user_unit_states.latest_learning_event_occurred_at`, and
`learning.user_unit_states.latest_reset_boundary_at`. `RecordLearningEvents`
locks affected state rows first and skips later non-reset events whose business
time is at or before the state projection boundary. The same state projection
also stores `latest_learning_event_ledger_seq`; replay rebuilds these internal
watermarks from the ledger.

### ListUserUnitProgress

```text
request
  -> validate user_id, bucket, limit and cursor
  -> read learning.user_unit_states joined with semantic.coarse_unit
  -> apply mastered/unmastered bucket filters
  -> keyset paginate by label or progress_percent + label
  -> return frontend display fields and opaque next_cursor
```

This is a read usecase only. Mastered rows are selected by
`status = 'mastered'` without `is_target = true`; unmastered rows are limited to
active targets with status `new`, `learning` or `reviewing`.

## Reducer Effects

- `observe_only`: updates observation fields only.
- `affects_progress`: updates observation, progress, schedule, status and mastery fields.
- `set_mastered`: updates observation and moves the unit to terminal mastered state without changing target/control fields.
- `reset_unlearned`: clears observation/progress/schedule fields and moves the
  unit back to `status = 'new'` without changing target/control fields.

`event_type` describes the normalized business event. `reducer_effect` is the reducer dispatch field.

`affects_progress` events also carry `counts_toward_success_streak`.
Quiz progress events set it to `true`; passive exposure session3 events set it
to `false`. Passive progress still advances progress/schedule, but it does not
increase `consecutive_success_count`. After any progress event is reduced, a
state with `progress_percent >= 100` is forced into terminal mastered state.

Passive exposure session3 events also carry exactly three
`consumed_watch_session_ids`. This typed column is the reducer ledger's source
of truth for which watch sessions have already been consumed by a passive
progress window; metadata only keeps an audit copy.

## Time Handling

- `RecordLearningEvents` normalizes `OccurredAt` to UTC before validation and append.
- Persistence mappers use `internal/platform/postgres/pgtime` to write UTC `time.Time` into `timestamptz`.
- Persistence mappers use `internal/platform/postgres/pgtime` to read UTC `time.Time` back into reducer models.
- UUID、nullable text、numeric 等纯 Postgres 类型转换委托 `internal/platform/postgres/*`；reducer 仍保留本地 mapper 函数作为模块边界。
- Integration fixture uses `internal/platform/postgres/pgtest` for embedded Postgres and template database cloning; reducer `test/fixture` owns the reducer schema plan and seed helpers.
