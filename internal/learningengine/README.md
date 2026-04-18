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
  test/
    fixture/
    unit/
    integration/
```

## 主要边界

- Learning engine 不暴露 Recommendation-specific 的读取接口语义
- `ListUserUnitStates` 使用 Learning engine 自己的过滤条件
- `SuspendTargetUnit` / `ResumeTargetUnit` 通过状态读取 + upsert 完成，不保留错误的 SQL 直写语义
- Replay 依赖的基础能力以“全量状态读取 + control slice 抽取 + delete by user + batch upsert”为准
- 在线写入与 Replay 共用同一个 reducer
- 同一 `user_id` 的所有状态写入 usecase 通过数据库 advisory xact lock 串行化，避免 `ReplayUserStates` 与在线写入并发破坏最终一致性
- 仅由学习事件创建的新状态默认 `is_target = false`；target 只能来自显式 control 命令，不能由学习事件隐式产生
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

### Empty-State Initialization

```text
first learning event for unseen unit
  -> create new UserUnitState
  -> default is_target = false
  -> default target_priority = 0
  -> apply reducer progression fields
```

这个约束保证：

- “有学习事件”不等于“进入推荐目标集合”
- Recommendation 只消费显式 target/control 选中的 unit
- 非 target 的学习历史不会误进入 Recommendation active demand

## 测试布局

当前测试统一收口到模块 `test/` 目录：

- `test/unit`
  - 纯内存 / fake / stub 测试
  - 当前包括：
    - `application/service`
    - `domain/aggregate`
- `test/integration`
  - 真实 Postgres、真实 repository、真实 tx、真实 usecase 测试
  - 当前包括：
    - `application/service`
    - `infrastructure/persistence/repository`
    - `infrastructure/persistence/tx`
- `test/fixture`
  - 模块共享测试基座
  - 负责 shared embedded Postgres、template database、schema apply 和 seed helper

当前模块内真实数据库测试使用 shared embedded Postgres server + 测试级独立 database：

- 每个 integration 测试包只启动一次 embedded Postgres
- schema / migration 只在 template database 上初始化一次
- 每个测试 case 从 template clone 自己的数据库
- 测完关闭连接并删除该数据库

这样做的目的是：

- 保持真实 Postgres 验证不变
- 避免每个测试 case 各自起库、各自跑 migration
- 在不共享脏数据的前提下缩短 `learningengine` 集成测试耗时

当前推荐的本地检查方式：

- `make quick-check`
  - 日常编码时快速执行 `gofmt + go vet + go test ./...`
  - 不包含带 `integration` tag 的真实数据库慢测试
- `make learningengine-test-integration`
  - 单独执行 Learning engine 模块内真实数据库测试
- `make check`
  - 作为完整仓库级验收
  - 先执行 `quick-check`
  - 再通过一次 `go test -tags=integration ...` 调用并行调度 Learning engine 与 Recommendation 的模块内 integration 测试

## 当前约束

- 当前没有 HTTP/API 层
- 当前没有用户级 partial replay
- 当前没有复杂乱序历史修复工具
- 当前不持有 Recommendation 派生概念或投放字段
