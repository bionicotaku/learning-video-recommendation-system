# Learningengine 实施步骤与进度记录

状态：DRAFT  
更新时间：2026-04-16  
适用范围：`internal/learningengine`

## 文档目的

本文档是 `learningengine` 模块的实施步骤与进度记录文档，不是设计文档，也不是当前实现说明文档。

本文档的作用是：

1. 把 `learningengine` 的后续实现工作拆成可直接执行的大步骤
2. 为每一步提供明确的实施内容、测试方案和验收标准
3. 作为实现过程中的唯一进度真相

后续执行 `learningengine` 实现时，必须以本文档为准推进。

## 实施范围

本轮只实现 `internal/learningengine`，不实现 `internal/recommendation`，也不依赖 `recommendation` 已实现的代码。

本轮目标是把 `learningengine` 从“只有 migration、query/sqlc、DTO、repository port、repository 壳”的状态推进到“核心业务逻辑、主用例和模块内测试可用”的状态。

## 明确不做

本轮明确不做以下内容：

1. 不实现 `recommendation` 模块
2. 不做跨模块集成测试
3. 不做 `internal/test/e2e` 下的端到端测试
4. 不依赖 `recommendation` 已实现
5. 不实现 partial / scoped replay
6. 不实现复杂乱序历史事件修复工具
7. 不引入 Recommendation 派生概念或 Recommendation 专用字段

## 测试范围

本轮测试范围固定为：

1. 领域单测
2. usecase 单测
3. `learningengine` 模块内连库测试

本轮允许做 `learningengine` 自己的数据库测试，但这些测试只验证 `learningengine` 自己的 migration、repository、事务和 usecase，不做跨模块联调。

## 执行规则

执行本实施文档时必须遵守以下规则：

1. 每个大步骤开始前，先把该步骤状态改为 `IN_PROGRESS`
2. 每个大步骤验收通过后，必须立刻更新本文档，把状态改为 `ACCEPTED`
3. 未更新本文档前，禁止进入下一步
4. 若某一步被阻塞，则将状态改为 `BLOCKED`，并记录阻塞原因和当前结论
5. 每个大步骤验收通过后，必须补齐：
   - 实际改动摘要
   - 实际测试命令
   - 测试结果
   - 与设计文档偏差
6. 若某一步与设计文档存在偏差，必须写明偏差内容、原因和处理决定；若无偏差，则明确写“无偏差”
7. 本文档是过程性文档，后续实现期间持续维护，不得用聊天记录替代文档记录

## 总进度总表

| 步骤 | 名称 | 状态 |
| --- | --- | --- |
| 1 | 契约与骨架收正 | ACCEPTED |
| 2 | 领域规则与 reducer | ACCEPTED |
| 3 | target/control 与读取 usecase | ACCEPTED |
| 4 | RecordLearningEvents | ACCEPTED |
| 5 | ReplayUserStates | ACCEPTED |
| 6 | 模块内数据库测试 | ACCEPTED |
| 7 | 文档同步与最终验收 | ACCEPTED |

---

## 步骤 1：契约与骨架收正

### 目标

先把当前 `learningengine` 骨架中与设计文档不一致的契约缺口收正，确保后续业务实现建立在正确边界之上。

### 当前缺口

当前已知缺口包括：

1. `ListRecommendationStates` 这类命名把 Recommendation 消费语义泄漏进了 Learning engine
2. `ResumeTargetUnit` 当前实现语义错误，恢复挂起时直接回写 `new`，与设计文档要求的“按当前学习进展字段重算状态”冲突
3. Replay 所需的 repository/query 能力不完整，缺少 delete/control-snapshot 等支撑能力
4. 现有 DTO、repository、query、README 仍停留在骨架状态，还没有完全收敛到最终设计文档的稳定语义

### 实施内容

本步骤实施内容固定为：

1. 收正 `application/repository` 中的接口语义，去掉 Recommendation-specific 命名
2. 收正 `application/dto` 中的读取请求语义，显式支持 Learning engine 自己的过滤条件，而不是 Recommendation 专属读取语义
3. 收正 `query/sqlc/repository`，补足 Replay 所需的最小能力
4. 收正 `target` 控制相关实现边界，避免继续保留错误的 SQL 直写语义
5. 更新 `internal/learningengine/README.md` 中关于当前实现边界的描述，使其能承接后续真实实现

### 涉及边界

本步骤只允许改动以下边界：

1. `application/dto`
2. `application/repository`
3. `application/usecase` 的接口签名
4. `infrastructure/persistence/query`
5. `infrastructure/persistence/repository`
6. `infrastructure/persistence/sqlcgen`
7. `internal/learningengine/README.md`

本步骤不实现 reducer，不实现 usecase 业务流程，不实现 Replay 逻辑。

### 测试方案

本步骤测试应至少包括：

1. 编译通过
2. 受影响 package 的单测通过
3. query / repository 与设计文档约定的语义一致

建议测试命令：

```bash
go test ./internal/learningengine/...
```

若本步骤只改接口和文档，也应至少保证相关包可编译。

### 验收标准

满足以下条件才算本步骤通过：

1. `learningengine` 不再暴露 Recommendation-specific 的读取命名
2. `ResumeTargetUnit` 不再保留“恢复直接变 new”的错误语义
3. Replay 所需的最小 repository/query 能力已经齐备
4. 当前接口、DTO、query、repository 与设计文档语义一致
5. README 已同步反映收正后的边界

### 进度记录

- 状态：ACCEPTED
- 开始时间：2026-04-16
- 完成时间：2026-04-16
- 实际改动摘要：
  - 删除 Recommendation-specific 的 `ListRecommendationStates` 命名
  - 新增 `UserUnitStateFilter`，并把 `ListUserUnitStatesRequest` 与 `UserUnitStateRepository.ListByUser` 收正为统一过滤读取接口
  - 补齐 `DeleteByUser`，为 Replay 提前准备状态清理能力
  - 删除 query/sqlc/repository 中 `SuspendTargetUnit` / `ResumeTargetUnit` 的错误 SQL 直写路径
  - 收缩 `TargetStateCommandRepository` 为纯 control 命令边界
  - 为 `ReplayUserStatesResponse` 增加可观测计数字段
  - 更新 `internal/learningengine/README.md`
- 实际测试命令：
  - `make sqlc-generate`
  - `go test ./internal/learningengine/...`
- 测试结果：
  - `sqlc` 生成成功
  - `go test ./internal/learningengine/...` 通过
- 与设计文档偏差：无偏差

---

## 步骤 2：领域规则与 reducer

### 目标

实现 `learningengine` 的纯领域能力，包括事件校验、强弱事件分类、窗口维护、简化 SM-2、`progress_percent`、`mastery_score`、状态迁移和统一 reducer。

### 当前缺口

当前 `learningengine` 只有：

1. domain model
2. 状态枚举常量
3. 领域目录骨架

当前缺少：

1. 事件合法性校验
2. 强弱事件分类规则
3. 窗口维护逻辑
4. 简化 SM-2 策略
5. `progress_percent` 公式
6. `mastery_score` 公式
7. 状态迁移判定
8. 挂起覆盖规则
9. 单聚合 reducer

### 实施内容

本步骤实施内容固定为：

1. 实现允许的 `event_type`、`status` 与强事件 `quality` 校验
2. 实现弱事件与强事件分类函数
3. 实现 seen 字段更新、计数器更新、窗口追加截断等 helper
4. 按设计文档实现简化 SM-2 成功/失败分支
5. 按设计文档实现 `progress_percent` 公式
6. 按设计文档实现 `mastery_score` 公式
7. 按设计文档实现状态迁移规则
8. 实现挂起覆盖逻辑
9. 实现统一 reducer，使在线写入与 Replay 复用同一套状态推进逻辑
10. 确定在线主路径对晚到强事件的策略：拒绝晚到强事件；弱事件允许更新 seen 相关字段

### 涉及边界

本步骤只允许改动以下边界：

1. `domain/enum`
2. `domain/model`
3. `domain/policy`
4. `domain/aggregate`
5. 相关领域单测

本步骤不实现 repository，不实现 usecase，不实现数据库测试。

### 测试方案

领域单测必须覆盖：

1. 弱事件只更新 seen，不推进状态
2. 首次强事件触发 `new -> learning`
3. 连续成功进入 `reviewing`
4. 达到阈值进入 `mastered`
5. 失败后从 `mastered` 回退
6. recent windows 追加与截断
7. `progress_percent` 在推进时增长、在失败时回落
8. `mastery_score` 在推进时增长、在失败时回落
9. 挂起覆盖规则
10. 晚到强事件拒绝

建议测试命令：

```bash
go test ./internal/learningengine/domain/...
```

### 验收标准

满足以下条件才算本步骤通过：

1. reducer 已可独立驱动 `UserUnitState` 演化
2. SM-2、progress、mastery、状态迁移规则与设计文档一致
3. 在线写入与 Replay 将使用同一个 reducer
4. 领域单测覆盖核心状态演化场景并全部通过

### 进度记录

- 状态：ACCEPTED
- 开始时间：2026-04-16
- 完成时间：2026-04-16
- 实际改动摘要：
  - 新增学习事件类型常量
  - 新增 event 校验、强弱事件分类、recent window 截断、简化 SM-2、progress/mastery 计算、状态迁移与挂起覆盖规则
  - 新增统一 reducer，并暴露 late strong event 错误
  - 先写领域测试，再补实现，完成 reducer 行为收敛
- 实际测试命令：
  - `go test ./internal/learningengine/domain/...`
  - `go test ./internal/learningengine/...`
- 测试结果：
  - 领域测试通过
  - `go test ./internal/learningengine/...` 通过
- 与设计文档偏差：无偏差

---

## 步骤 3：target/control 与读取 usecase

### 目标

实现不依赖主写入链路的 usecase，包括：

1. `EnsureTargetUnits`
2. `SetTargetInactive`
3. `SuspendTargetUnit`
4. `ResumeTargetUnit`
5. `ListUserUnitStates`

### 当前缺口

当前缺少：

1. target/control 相关 usecase 实现
2. `ListUserUnitStates` 的实际读取编排
3. `ResumeTargetUnit` 所需的“重算状态”业务逻辑接线
4. 统一的输入校验与错误语义

### 实施内容

本步骤实施内容固定为：

1. 实现 `EnsureTargetUnitsUsecase`
2. 实现 `SetTargetInactiveUsecase`
3. 实现 `SuspendTargetUnitUsecase`
4. 实现 `ResumeTargetUnitUsecase`
5. 实现 `ListUserUnitStatesUsecase`
6. 为读取 usecase 增加过滤支持，例如：
   - `OnlyTarget`
   - `ExcludeSuspended`
7. 保证 control slice 更新不会重置 progression slice
8. 保证 `ResumeTargetUnit` 通过读取当前状态并重算状态完成恢复，而不是直接写死状态值

### 涉及边界

本步骤允许改动：

1. `application/dto`
2. `application/usecase`
3. `application/service` 或等价 application 实现目录
4. `application/repository`
5. `infrastructure/persistence/repository`
6. 相关单测与模块内连库测试

本步骤不实现 `RecordLearningEvents` 和 `ReplayUserStates`。

### 测试方案

本步骤测试应至少包括：

1. usecase 单测
2. 模块内连库测试

覆盖场景至少包括：

1. `EnsureTargetUnits` 创建默认 `new` 行
2. `EnsureTargetUnits` 不重置已有 progression
3. `SetTargetInactive` 只软失活，不删行
4. `SuspendTargetUnit` 进入挂起态并写入原因
5. `ResumeTargetUnit` 清除挂起并重算状态
6. `ListUserUnitStates` 能按 Learning engine 自己的过滤语义返回状态

建议测试命令：

```bash
go test ./internal/learningengine/...
```

### 验收标准

满足以下条件才算本步骤通过：

1. 五个 usecase 都有真实实现
2. control slice 更新不破坏 progression slice
3. resume 行为符合设计文档
4. 读取 usecase 不再暴露 Recommendation-specific 语义
5. 单测和模块内连库测试通过

### 进度记录

- 状态：ACCEPTED
- 开始时间：2026-04-16
- 完成时间：2026-04-16
- 实际改动摘要：
  - 新增 `application/service`，实现 `EnsureTargetUnits`、`SetTargetInactive`、`SuspendTargetUnit`、`ResumeTargetUnit`、`ListUserUnitStates`
  - 新增轻量事务抽象 `TxManager` / `TransactionalRepositories`
  - 调整基础事务管理器以支持 usecase 在事务内读取和 upsert 状态
  - 新增 usecase 单测
  - 新增最小连库验证和测试数据库 helper
- 实际测试命令：
  - `go test ./internal/learningengine/application/service/...`
  - `go test ./internal/learningengine/...`
- 测试结果：
  - application/service 单测与最小连库测试通过
  - `go test ./internal/learningengine/...` 通过
- 与设计文档偏差：
  - 原计划偏向临时 Postgres 容器；由于当前环境无可用 Docker daemon，最小连库验证改用模块内 embedded Postgres。语义仍为真实 Postgres 连库测试。

---

## 步骤 4：RecordLearningEvents

### 目标

实现 `RecordLearningEvents` 主写入链路，使标准化学习事件可以 append 到事实表，并在同一事务中驱动状态归约和 upsert。

### 当前缺口

当前缺少：

1. `RecordLearningEventsUsecase` 的真实实现
2. 事件分组与排序逻辑
3. 单事务写事件 + 写状态编排
4. 行级锁与并发控制
5. 晚到强事件错误处理

### 实施内容

本步骤实施内容固定为：

1. 实现请求校验
2. 按 `(user_id, coarse_unit_id)` 分组事件
3. 每组内部按 `occurred_at` 排序，并以稳定顺序推进 reducer
4. 在单事务中完成：
   - append `learning.unit_learning_events`
   - 读取当前状态并加锁
   - 逐条 reduce
   - upsert `learning.user_unit_states`
5. 区分错误类型：
   - 参数非法
   - 事件不合法
   - 晚到强事件
   - repository 或事务失败

### 涉及边界

本步骤允许改动：

1. `application/dto`
2. `application/usecase`
3. application 层实现
4. `infrastructure/persistence/repository`
5. `infrastructure/persistence/tx`
6. 相关单测与模块内连库测试

### 测试方案

本步骤测试应至少覆盖：

1. 弱事件只更新 seen
2. 首次强事件 `new -> learning`
3. 连续成功推进到 `reviewing`
4. 连续推进到 `mastered`
5. 失败回退
6. 多 unit 事件分组处理
7. 晚到强事件拒绝
8. 若状态 upsert 失败，则事件 append 一并回滚

建议测试命令：

```bash
go test ./internal/learningengine/...
```

### 验收标准

满足以下条件才算本步骤通过：

1. `RecordLearningEventsUsecase` 有真实实现
2. 写事件与写状态在同一事务中完成
3. 并发安全依赖 `FOR UPDATE` 或等价锁定读取
4. 晚到强事件不会悄悄污染当前状态
5. 单测和模块内连库测试通过

### 进度记录

- 状态：ACCEPTED
- 开始时间：2026-04-16
- 完成时间：2026-04-16
- 实际改动摘要：
  - 新增 `RecordLearningEventsUsecase`
  - 实现事件映射、请求校验、按 unit 分组排序、事务内 append + reduce + batch upsert
  - 将 late strong event 提升为 application 错误
  - 为 `metadata` 补齐默认 `{}`，满足事实表 not-null 约束
  - 新增 `RecordLearningEvents` 的 unit 测试和 real Postgres 连库测试
  - 新增事务回滚测试，验证 state 写失败时 event append 不会落库
- 实际测试命令：
  - `go test ./internal/learningengine/application/service/...`
  - `go test ./internal/learningengine/...`
- 测试结果：
  - `RecordLearningEvents` 单测通过
  - real Postgres 连库测试通过
  - 事务回滚测试通过
  - `go test ./internal/learningengine/...` 通过
- 与设计文档偏差：
  - 连库测试继续使用 embedded Postgres，而非 Docker 容器；原因同步骤 3。

---

## 步骤 5：ReplayUserStates

### 目标

实现 `ReplayUserStates`，范围固定为 `FullUserReplay(user_id)`。

### 当前缺口

当前缺少：

1. Replay 用例实现
2. control slice 快照保留逻辑
3. 删除旧状态并重建状态的事务编排
4. Replay 与在线写入一致性的验证

### 实施内容

本步骤实施内容固定为：

1. 读取当前 `user_unit_states`
2. 提取 control slice 快照：
   - `is_target`
   - `target_source`
   - `target_source_ref_id`
   - `target_priority`
   - `suspended_reason`
3. 读取该用户全部事件并按 `occurred_at, event_id` 排序
4. 删除该用户全部状态
5. 按 unit 从空状态回放 reducer
6. 将重建出的 learning progression 与 control slice 合并
7. 对当前仍处于挂起控制态的状态覆盖 `status='suspended'`
8. batch upsert 重建结果

### 涉及边界

本步骤允许改动：

1. `application/dto`
2. `application/usecase`
3. application 层实现
4. `application/repository`
5. `infrastructure/persistence/repository`
6. `infrastructure/persistence/query`
7. 相关单测与模块内连库测试

本步骤不实现 partial/scoped replay。

### 测试方案

本步骤测试应至少覆盖：

1. 有事件历史的用户可完整重建
2. 只有 target/control、没有事件的状态行不会因 replay 丢失
3. replay 后保留 target/control 字段
4. 挂起状态在 replay 后仍正确保留
5. 在线最终状态与 full replay 结果完全一致

建议测试命令：

```bash
go test ./internal/learningengine/...
```

### 验收标准

满足以下条件才算本步骤通过：

1. `ReplayUserStatesUsecase` 已实现
2. Replay 不会丢失 control slice
3. Replay 与在线 reducer 共享同一套领域规则
4. 在线结果与 replay 结果一致
5. 单测和模块内连库测试通过

### 进度记录

- 状态：ACCEPTED
- 开始时间：2026-04-16
- 完成时间：2026-04-16
- 实际改动摘要：
  - 新增 `ReplayUserStatesUsecase`
  - 实现全量状态读取、control slice 抽取、按事件重放 reducer、delete by user、merge 后 batch upsert
  - 保留无事件 target/control 行
  - 保留挂起控制态
  - 新增 replay 单测和 real Postgres 连库测试
  - 修复无事件状态 upsert 时 `recent_*_window` 为空切片而非 `NULL`
- 实际测试命令：
  - `go test ./internal/learningengine/application/service/...`
  - `go test ./internal/learningengine/...`
- 测试结果：
  - replay 单测通过
  - replay real Postgres 连库测试通过
  - `go test ./internal/learningengine/...` 通过
- 与设计文档偏差：
  - 连库测试继续使用 embedded Postgres，而非 Docker 容器；原因同步骤 3。

---

## 步骤 6：模块内数据库测试

### 目标

建立 `learningengine` 模块自己的数据库测试基础设施和测试集，不进入跨模块测试。

### 当前缺口

当前缺少：

1. 模块内数据库测试基座
2. 临时 Postgres 测试环境
3. 最小外部 stub 初始化
4. repository 连库测试
5. usecase 连库测试

### 实施内容

本步骤实施内容固定为：

1. 为 `learningengine` 建立独立的测试数据库启动方式
2. 启动临时 Postgres
3. 初始化最小外部 stub：
   - `auth.users`
   - `semantic.coarse_unit`
   - `catalog.videos`
4. 应用 `learningengine` 自己的 migration
5. 编写 repository 连库测试
6. 编写事务相关测试
7. 编写 usecase 连库测试

### 涉及边界

本步骤允许改动：

1. `internal/learningengine` 内部测试目录
2. `infrastructure/persistence` 相关测试辅助
3. application/usecase 连库测试

本步骤不进入 `internal/test/e2e`。

### 测试方案

本步骤测试应至少覆盖：

1. migration 可从空库建立
2. repository 可完成基础读写
3. 事务回滚有效
4. `RecordLearningEvents` 在连库环境下满足事务一致性
5. `ReplayUserStates` 在连库环境下满足一致性

建议测试命令：

```bash
go test ./internal/learningengine/...
```

### 验收标准

满足以下条件才算本步骤通过：

1. `learningengine` 有自己的模块内连库测试
2. 连库测试不依赖 live DB
3. 不进入跨模块测试
4. repository、事务、usecase 的关键路径都有数据库级验证

### 进度记录

- 状态：ACCEPTED
- 开始时间：2026-04-16
- 完成时间：2026-04-16
- 实际改动摘要：
  - 新增模块内测试数据库 helper
  - 新增 repository 连库测试：事件表、状态表、target command
  - 新增事务管理器连库测试，验证 error 时回滚
  - 扩展 application/service 的 real Postgres 用例，覆盖 target/control、record、replay
- 实际测试命令：
  - `go test ./internal/learningengine/infrastructure/persistence/repository ./internal/learningengine/infrastructure/persistence/tx`
  - `go test ./internal/learningengine/application/service/...`
  - `go test ./internal/learningengine/...`
- 测试结果：
  - repository 连库测试通过
  - tx 连库测试通过
  - application/service real Postgres 测试通过
  - `go test ./internal/learningengine/...` 通过
- 与设计文档偏差：
  - 模块内数据库测试使用 embedded Postgres，而非 Docker 容器；原因同步骤 3。

---

## 步骤 7：文档同步与最终验收

### 目标

在 `learningengine` 业务实现完成后，完成模块文档同步和最终模块验收。

### 当前缺口

当前缺少：

1. `internal/learningengine/README.md` 的真实实现说明
2. `docs/当前实现现状.md` 中关于 Learning engine 的更新
3. 最终模块级测试与收尾记录

### 实施内容

本步骤实施内容固定为：

1. 更新 `internal/learningengine/README.md`
2. 更新 `docs/当前实现现状.md`
3. 若结构或边界与统一规范存在偏离，在 README 中明确记录偏离原因与维护约束
4. 执行最终模块级测试
5. 回填本文档的最终验收记录和总进度总表

### 涉及边界

本步骤允许改动：

1. `internal/learningengine/README.md`
2. `docs/当前实现现状.md`
3. 本文档

### 测试方案

最终验收命令固定至少包括：

```bash
go test ./internal/learningengine/...
make check
```

### 验收标准

满足以下条件才算本步骤通过：

1. `learningengine` README 已反映真实实现状态
2. 当前实现现状文档已同步
3. `go test ./internal/learningengine/...` 通过
4. `make check` 通过
5. 本文档中 7 个大步骤全部为 `ACCEPTED`

### 进度记录

- 状态：ACCEPTED
- 开始时间：2026-04-16
- 完成时间：2026-04-16
- 实际改动摘要：
  - 更新 `internal/learningengine/README.md`，补齐目录结构、职责边界、主调用链和测试布局
  - 更新 `docs/当前实现现状.md`，把 Learning engine 从“只有基础层”改成已实现业务逻辑和模块内测试
  - 回填本文档全部步骤状态、测试结果和最终验收记录
- 实际测试命令：
  - `go test ./internal/learningengine/...`
  - `make check`
- 测试结果：
  - `go test ./internal/learningengine/...` 通过
  - `make check` 通过
- 与设计文档偏差：
  - 模块内数据库测试继续使用 embedded Postgres，而非 Docker 容器；原因同步骤 3。

---

## 最终验收记录

- 最终状态：ACCEPTED
- `go test ./internal/learningengine/...`：通过
- `make check`：通过
- README 同步状态：已完成
- 当前实现现状文档同步状态：已完成
- 步骤总表最终状态检查：7 个步骤均为 `ACCEPTED`
