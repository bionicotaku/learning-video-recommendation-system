# Learning Engine

`learningengine` 负责学习事件、学习状态、target/control 字段和 replay 相关边界。

## 当前职责

当前模块只负责：

- `learning.unit_learning_events`
- `learning.user_unit_states`
- target/control 字段
- replay 相关边界

当前模块不负责：

- Recommendation 的任何投放状态
- Recommendation run / items
- 视频推荐策略
- Catalog 内容事实

## 当前实现状态

当前 `learningengine` 已经具备：

- 模块标准骨架
- Learning engine owner 的 migration
- SQL/query/sqlc 基础层
- domain 规则：
  - 事件校验
  - 强弱事件分类
  - 简化 SM-2
  - `progress_percent`
  - `mastery_score`
  - 状态迁移
  - 统一 reducer
- application service / usecase 实现：
  - `EnsureTargetUnits`
  - `SetTargetInactive`
  - `SuspendTargetUnit`
  - `ResumeTargetUnit`
  - `ListUserUnitStates`
  - `RecordLearningEvents`
  - `ReplayUserStates`
- 模块内数据库测试：
  - usecase real Postgres 测试
  - repository real Postgres 测试
  - 事务回滚测试

## 目录结构

```text
internal/learningengine/
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
  testutil/
```

## 主要边界

- Learning engine 不暴露 Recommendation-specific 的读取接口语义
- `ListUserUnitStates` 使用 Learning engine 自己的过滤条件
- `SuspendTargetUnit` / `ResumeTargetUnit` 通过状态读取 + upsert 完成，不保留错误的 SQL 直写语义
- Replay 依赖的基础能力以“全量状态读取 + control slice 抽取 + delete by user + batch upsert”为准
- 在线写入与 Replay 共用同一个 reducer
- 同一 `user_id` 的所有状态写入 usecase 通过数据库 advisory xact lock 串行化，避免 `ReplayUserStates` 与在线写入并发破坏最终一致性
- `recommendation` 只能读取 `learning.user_unit_states`，不能回写 Learning engine 业务表

## 主要调用链

### RecordLearningEvents

```text
request
  -> request validation
  -> map dto -> domain events
  -> group by coarse_unit_id
  -> sort by occurred_at
  -> tx begin
  -> acquire user-scoped advisory lock
  -> append learning.unit_learning_events
  -> load current state (for update)
  -> reduce event stream
  -> batch upsert learning.user_unit_states
  -> tx commit
```

### ReplayUserStates

```text
request
  -> tx begin
  -> acquire user-scoped advisory lock
  -> list current user_unit_states
  -> extract control slice
  -> list unit_learning_events ordered by occurred_at, event_id
  -> delete user states
  -> replay reducer from empty state
  -> merge progression with control slice
  -> batch upsert rebuilt states
  -> tx commit
```

## 测试布局

当前测试分三层：

- 领域测试：`domain/aggregate`
- usecase 测试：`application/service`
- real Postgres 测试：
  - `application/service/*_integration_test.go`
  - `infrastructure/persistence/repository/*_integration_test.go`
  - `infrastructure/persistence/tx/*_integration_test.go`

测试数据库当前使用 embedded Postgres。当前环境没有可用 Docker daemon，因此模块内连库测试没有使用容器，但仍然是对真实 Postgres 的数据库验证。

## 当前约束

- 当前没有 HTTP/API 层
- 当前没有用户级 partial replay
- 当前没有复杂乱序历史修复工具
- 当前不持有 Recommendation 派生概念或投放字段
