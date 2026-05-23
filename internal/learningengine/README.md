# Learning Engine

`internal/learningengine` 是学习域的边界容器，不直接承载完整的 `application/domain/infrastructure/test` 骨架。

当前只有两个平级子模块：

- `reducer`：维护 `learning.*` 表、normalized event ledger、状态归约、replay、target/control 和状态查询。
- `normalizer`：只读 `analytics.*` raw facts，把 quiz / interaction 解释成 reducer 可消费的 normalized events。

## Boundaries

- `reducer` 是 `learning.*` 数据库对象的 owner。
- `normalizer` 是 raw fact 到 normalized event 的学习语义解释层。
- `normalizer` 可以调用 `reducer.RecordLearningEvents`，但不能直接写 `learning.*`。
- `reducer` 不读取 `analytics.*`，也不能 import `normalizer` 或 `analytics`。
- Recommendation 只读取 `learning.user_unit_states`，不能回写 Learning Engine 业务表。
- `semantic.unit_collections` 定义系统词书；reducer 只拥有用户 active collection profile 和 target projection。

## Directory Structure

```text
internal/learningengine/
  README.md
  doc.go
  reducer/
    application/
    domain/
    infrastructure/
    test/
  normalizer/
    application/
    domain/
    infrastructure/
    test/
```

根目录只解释模块边界。新增实现应进入 `reducer` 或 `normalizer`，不要在根目录重新创建 `application/domain/infrastructure/test`。

## Main Flows

### Future API Path

```text
internal/api
  -> internal/analytics raw fact write
  -> internal/learningengine/normalizer by-ID normalize
  -> internal/learningengine/reducer RecordLearningEvents
  -> learning.unit_learning_events
  -> learning.user_unit_states
```

HTTP 层未来放在 `internal/api`。API 成功响应只承诺 raw fact accepted，不把 `progress_quality`、`reducer_effect` 或 reducer 状态细节暴露给前端。

### Repair / Backfill Path

```text
analytics raw facts
  -> normalizer NormalizePendingEvents
  -> reducer RecordLearningEvents
  -> learning.unit_learning_events
  -> learning.user_unit_states
```

当前 normalizer 不维护 checkpoint / rollup / `normalized_at`。补偿依赖 pending anti-join 和 reducer 的 source 幂等约束。

### Reducer Path

```text
RecordLearningEvents
  -> validate normalized events
  -> lock affected user_unit_states and read projection watermarks
  -> skip non-reset events at or before state latest_reset_boundary_at
  -> append learning.unit_learning_events idempotently
  -> reduce only newly inserted events
  -> upsert learning.user_unit_states with updated projection watermarks
```

`ReplayUserStates` 同样只从 `learning.unit_learning_events` 按 `ledger_seq` 重建状态，不重新解释 analytics raw facts。`occurred_at` 是业务时间，不是 replay 排序字段。

### Active Collection Path

```text
internal/api
  -> API facade active collection transaction
  -> internal/learningengine/reducer TargetStateCommandRepository
  -> internal/user ProfileRepository
  -> learning.user_learning_profiles
  -> learning.user_unit_states
  -> app_user.user_profiles.onboarding_status
```

Active collection API 由 `internal/api` facade 打开用户级事务。事务内，Learning Engine repository 读取 active collection members，upsert 当前用户 learning profile，关闭旧 `target_source='unit_collection'` 且不属于新集合的 targets，并批量 upsert 新集合 members；User repository 同事务把 onboarding 状态更新为 `collection_selected`。Learning Engine 只更新 target control 字段，不重置 `status`、progress、mastery 或 schedule 字段。

## Local Checks

- `make quick-check`：日常快速检查。
- `make learningengine-test-integration`：只跑 reducer 真实数据库测试。
- `make normalizer-test-integration`：只跑 normalizer 真实数据库测试。
- `make check`：仓库级验收，包含 quick check 与模块 integration tests。
