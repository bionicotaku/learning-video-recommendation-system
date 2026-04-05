# Learning Engine

`internal/learningengine` 是 Learning engine 模块根目录。

它的职责可以概括成一句话：

> 接收学习事件，维护学习状态，并在需要时从事件历史全量重建状态。

## 1. 模块职责

Learning engine 负责：

- 维护 `learning.unit_learning_events`
- 维护 `learning.user_unit_states`
- 处理标准化学习事件
- 根据领域规则更新学习状态
- 提供 full replay

Learning engine 不负责：

- 生成推荐批次
- 维护 `last_recommended_at`
- 维护推荐运行审计
- 排序视频或学习任务

## 2. 核心数据 owner

这个模块只拥有两张业务表：

1. `learning.unit_learning_events`
2. `learning.user_unit_states`

这两张表的关系是：

- `unit_learning_events` 是事实真相层
- `user_unit_states` 是当前投影层

因此 Learning engine 的核心工作流就是：

1. 先记录事件
2. 再更新状态投影
3. 如有必要，从完整事件历史重建状态

## 3. 当前目录结构

```text
internal/learningengine/
  README.md
  doc.go
  application/
    doc.go
    command/
      doc.go
      record_learning_events.go
      replay_user_states.go
    dto/
      doc.go
      record_learning_events_result.go
      replay_user_states_result.go
    repository/
      doc.go
      tx_manager.go
      unit_learning_event_repository.go
      user_unit_state_repository.go
    service/
      user_state_rebuilder.go
    usecase/
      doc.go
      record_learning_events.go
      replay_user_states.go
  domain/
    doc.go
    aggregate/
      user_unit_reducer.go
    enum/
      doc.go
      event_type.go
      unit_kind.go
      unit_status.go
    model/
      doc.go
      learning_event.go
      coarse_unit_ref.go
      user_unit_state.go
    policy/
      doc.go
      learning_policy.go
    rule/
      doc.go
      state_helpers.go
      weak_event_handler.go
      strong_event_handler.go
    service/
      doc.go
      sm2_updater.go
      status_transitioner.go
      progress_calculator.go
      mastery_calculator.go
  infrastructure/
    doc.go
    config.go
    db.go
    migration/
    persistence/
      mapper/
      query/
      queryctx/
      repository/
      schema/
      sqlcgen/
      tx/
  test/
    doc.go
    unit/
      doc.go
      domain/
        aggregate/
          user_unit_reducer_test.go
        policy/
          learning_policy_test.go
        rule/
          weak_event_handler_test.go
          strong_event_handler_test.go
        service/
          sm2_updater_test.go
          status_transitioner_test.go
          progress_mastery_calculator_test.go
      infrastructure/
        config_test.go
    integration/
      doc.go
      fixture/
        helpers.go
      infrastructure/
        db_integration_test.go
      usecase/
        record_learning_events_usecase_test.go
        replay_user_states_usecase_test.go
```

## 4. 每一层的职责

### `application/`

这里放用例编排，不放领域规则。

主要目录：

- `command/`
  输入命令对象
- `dto/`
  输出结果对象
- `repository/`
  application 依赖的 port interface
- `service/`
  application 编排层的辅助服务
- `usecase/`
  真正的业务入口

当前最关键的入口文件有：

- [record_learning_events.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/learningengine/application/usecase/record_learning_events.go)
- [replay_user_states.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/learningengine/application/usecase/replay_user_states.go)

#### `RecordLearningEvents`

负责：

- 接收标准化事件
- 在事务中写入事件表
- 调用 reducer 更新状态
- upsert `learning.user_unit_states`

#### `ReplayUserStates`

负责：

- 读取某个用户完整事件历史
- 从空状态开始重建所有 unit 状态
- 批量覆盖 `learning.user_unit_states`

MVP 只支持 full replay。

### `domain/`

这里放纯领域规则。它不应该依赖具体 SQL、`sqlc` 生成代码或数据库连接。

#### `aggregate/`

- [user_unit_reducer.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/learningengine/domain/aggregate/user_unit_reducer.go)

这是当前模块最关键的领域入口。它负责把：

- 当前状态
- 新学习事件
- 固定策略

归约成新的 `UserUnitState`。

可以把它理解成整个 Learning engine 的“状态投影核心”。

#### `enum/`

定义稳定枚举：

- `EventType`
- `UnitKind`
- `UnitStatus`

#### `model/`

定义领域模型：

- `LearningEvent`
- `CoarseUnitRef`
- `UserUnitState`

这些类型是整个模块内部沟通的标准数据结构。

#### `policy/`

- [learning_policy.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/learningengine/domain/policy/learning_policy.go)

这里存 Learning engine 的固定策略参数，例如：

- SM-2 相关参数
- 进度和 mastery 的阈值

#### `rule/`

放原子规则处理器：

- [weak_event_handler.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/learningengine/domain/rule/weak_event_handler.go)
- [strong_event_handler.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/learningengine/domain/rule/strong_event_handler.go)
- [state_helpers.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/learningengine/domain/rule/state_helpers.go)

这里关注的是：

- 弱事件更新什么
- 强事件更新什么
- 新状态默认值如何初始化

#### `service/`

放可复用的领域计算器：

- [sm2_updater.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/learningengine/domain/service/sm2_updater.go)
- [status_transitioner.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/learningengine/domain/service/status_transitioner.go)
- [progress_calculator.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/learningengine/domain/service/progress_calculator.go)
- [mastery_calculator.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/learningengine/domain/service/mastery_calculator.go)

这些 service 不直接碰数据库，只做纯规则计算。

### `infrastructure/`

这里只负责技术落地，不负责决定业务规则。

#### `config.go`

读取数据库连接等基础配置。

#### `db.go`

初始化 `pgx` 连接池。

#### `migration/`

定义 Learning engine 自己的 schema owner：

- `000001_create_learning_schema`
- `000002_create_user_unit_states`
- `000003_create_unit_learning_events`
- `000004_create_learning_indexes`

这里不应出现 Recommendation 的表。

#### `persistence/query/*.sql`

定义 SQL：

- [unit_events.sql](/Users/evan/Downloads/learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/query/unit_events.sql)
- [unit_states.sql](/Users/evan/Downloads/learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/query/unit_states.sql)

#### `persistence/sqlcgen/`

`sqlc` 生成代码。原则上不手改。

#### `persistence/mapper/`

负责隔离：

- `sqlc` 生成类型
- 领域模型

#### `persistence/repository/`

实现 application 层声明的 repository interface：

- [unit_learning_event_repo.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/repository/unit_learning_event_repo.go)
- [user_unit_state_repo.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/repository/user_unit_state_repo.go)

#### `persistence/tx/`

实现事务管理器：

- [pgx_tx_manager.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/tx/pgx_tx_manager.go)

### `test/`

测试已按职责拆成两层：

- `test/unit`
  放纯单元测试
- `test/integration`
  放真实数据库、真实事务、真实用例编排测试

当前 Learning engine 的测试重点是：

- `test/unit/domain/*`
  覆盖 reducer、rule、policy、service
- `test/unit/infrastructure/config_test.go`
  覆盖配置校验
- `test/integration/infrastructure/db_integration_test.go`
  覆盖真实数据库连接探针
- `test/integration/usecase/*`
  覆盖记录事件与 full replay

`test/integration/fixture/helpers.go` 提供跨集成测试共享的数据库、用户、coarse unit 与 use case 构造辅助。

跨模块端到端测试不放在这里，而统一放在：

- [internal/test/e2e](/Users/evan/Downloads/learning-video-recommendation-system/internal/test/e2e)

那里负责验证：

- Learning engine 产出的状态是否能被 Recommendation 正确消费
- full replay 后 Recommendation 输入是否仍稳定

## 5. 关键调用关系

### 学习事件写入链路

```text
RecordLearningEventsUseCase
  -> UnitLearningEventRepository.Append
  -> UserUnitStateRepository.GetByUserAndUnit
  -> UserUnitReducer.Reduce
  -> UserUnitStateRepository.Upsert
```

### full replay 链路

```text
ReplayUserStatesUseCase
  -> UnitLearningEventRepository.ListByUserOrdered
  -> UserStateRebuilder.Rebuild
  -> UserUnitStateRepository.DeleteByUser
  -> UserUnitStateRepository.BatchUpsert
```

## 6. 与 Recommendation 的边界

Learning engine 与 Recommendation 的边界必须保持清楚：

- Learning engine 不关心推荐分数
- Learning engine 不关心 `last_recommended_at`
- Learning engine 不关心 `scheduler_runs`
- Learning engine 不关心候选视频

Recommendation 只能读取 Learning engine 的结果，不能把推荐域字段重新塞回 `learning.user_unit_states`。

## 7. 新人最应该先看的文件

如果你要快速理解这个模块，建议按下面顺序读：

1. [record_learning_events.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/learningengine/application/usecase/record_learning_events.go)
2. [replay_user_states.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/learningengine/application/usecase/replay_user_states.go)
3. [user_unit_reducer.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/learningengine/domain/aggregate/user_unit_reducer.go)
4. [learning_event.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/learningengine/domain/model/learning_event.go)
5. [user_unit_state.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/learningengine/domain/model/user_unit_state.go)
6. [unit_events.sql](/Users/evan/Downloads/learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/query/unit_events.sql)
7. [unit_states.sql](/Users/evan/Downloads/learning-video-recommendation-system/internal/learningengine/infrastructure/persistence/query/unit_states.sql)
8. [record_learning_events_usecase_test.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/learningengine/test/integration/usecase/record_learning_events_usecase_test.go)

## 8. 常见改动应该落在哪里

### 新增一种学习事件

通常要同时看：

- `domain/enum/event_type.go`
- `domain/model/learning_event.go`
- `domain/rule/*`
- `domain/aggregate/user_unit_reducer.go`
- `application/command/*`
- 相关测试

### 调整状态迁移规则

优先看：

- `domain/service/status_transitioner.go`
- `domain/aggregate/user_unit_reducer.go`
- 对应单测

### 调整复习间隔规则

优先看：

- `domain/service/sm2_updater.go`
- `domain/policy/learning_policy.go`
- 对应单测

### 调整状态表字段

通常要同时改：

- `domain/model/user_unit_state.go`
- migration
- `persistence/query/unit_states.sql`
- mapper
- `sqlc` 生成代码
- repository
- 集成测试

## 9. 不该做什么

以后继续维护时，不要做这些事：

- 不要在 SQL 中实现 Learning engine 领域规则
- 不要让 repository 决定状态迁移
- 不要把 Recommendation 字段写进 `learning.user_unit_states`
- 不要把 full replay 扩成局部 replay 却不先明确文档边界
- 不要在 `sqlcgen/` 目录里手改生成代码
