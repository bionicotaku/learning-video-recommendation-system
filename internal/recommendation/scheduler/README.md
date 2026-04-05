# Recommendation Scheduler

`internal/recommendation/scheduler` 是当前已落地的 Recommendation scheduler 子模块。

它的职责可以概括成一句话：

> 读取 Learning engine 的学习状态，生成一轮推荐批次，并维护 Recommendation 自己的投放状态和推荐审计。

## 1. 子模块职责

scheduler 负责：

- 读取 Learning engine 的学习状态
- 读取 `semantic.coarse_unit`
- 生成推荐批次
- 维护 `recommendation.user_unit_serving_states`
- 维护 `recommendation.scheduler_runs`
- 维护 `recommendation.scheduler_run_items`
- 使用模块内默认调度参数生成推荐

scheduler 不负责：

- 写 `learning.unit_learning_events`
- 写 `learning.user_unit_states`
- replay 学习状态
- 维护用户级调度配置表
- 管理视频内容本体

## 2. MVP 边界

当前 MVP 明确约束：

- Recommendation 不支持用户级调度配置
- `session_default_limit`
- `daily_new_unit_quota`
- `daily_review_soft_limit`
- `daily_review_hard_limit`
- `timezone`

以上参数当前统一使用模块默认值。

这意味着当前 scheduler 的重点是先把以下链路稳定：

- 读取学习状态
- 计算候选
- 分配 quota
- 计算分数
- 组装推荐批次
- 写 serving state
- 写推荐审计

后续如果需要扩展用户级配置，应由 Recommendation 自己新增配置表，而不是回写 Learning engine。

## 3. 核心数据 owner

这个子模块当前会写入 3 张 Recommendation 表：

1. `recommendation.user_unit_serving_states`
2. `recommendation.scheduler_runs`
3. `recommendation.scheduler_run_items`

它只读取，不写入：

- `learning.user_unit_states`
- `semantic.coarse_unit`

因此 scheduler 的真实工作模式是：

1. 从 Learning engine 读取输入
2. 进行推荐决策
3. 把 Recommendation 自己的投放状态和审计结果写回 `recommendation.*`

## 4. 当前目录结构

```text
internal/recommendation/scheduler/
  README.md
  doc.go
  application/
    doc.go
    command/
      doc.go
      generate_recommendations.go
    dto/
      doc.go
      generate_recommendations_result.go
    query/
      doc.go
      candidate.go
    repository/
      doc.go
      scheduler_run_repository.go
      tx_manager.go
      user_unit_serving_state_repository.go
      learning_state_snapshot_read_repository.go
    usecase/
      doc.go
      generate_recommendations.go
  domain/
    doc.go
    enum/
      doc.go
      recommend_type.go
      unit_kind.go
      unit_status.go
    model/
      doc.go
      coarse_unit_ref.go
      recommendation.go
      recommendation_defaults.go
      user_unit_serving_state.go
      learning_state_snapshot.go
    service/
      doc.go
      backlog_calculator.go
      quota_allocator.go
      review_scorer.go
      new_scorer.go
      priority_zero_extractor.go
      recommendation_assembler.go
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
        service/
          quota_allocator_test.go
          scoring_test.go
          recommendation_assembler_test.go
      infrastructure/
        config_test.go
    integration/
      doc.go
      fixture/
        helpers.go
      infrastructure/
        candidate_queries_test.go
        db_integration_test.go
      usecase/
        generate_recommendations_usecase_test.go
    scenario/
      scenarios_test.go
```

## 5. 每一层的职责

### `application/`

这里放用例编排层。

主要目录：

- `command/`
  use case 的输入命令
- `dto/`
  use case 的输出结果
- `query/`
  application 层使用的 candidate 结构
- `repository/`
  use case 依赖的 port interface
- `usecase/`
  业务入口

当前最关键的用例只有一个：

- [generate_recommendations.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/recommendation/scheduler/application/usecase/generate_recommendations.go)

它负责：

- 读取 review / new 候选
- 计算 backlog
- 分配 review / new quota
- 对候选打分
- 提取 priority-0
- 组装 `RecommendationBatch`
- 持久化 run / run items
- 更新 `last_recommended_at`

### `domain/`

这里放纯推荐逻辑，不负责具体数据库读写。

#### `enum/`

定义 Recommendation 领域枚举：

- `RecommendType`
- `UnitKind`
- `UnitStatus`

#### `model/`

定义 Recommendation 使用的领域模型：

- `CoarseUnitRef`
- `RecommendationBatch`
- `RecommendationItem`
- `RecommendationDefaults`
- `UserUnitServingState`
- `LearningStateSnapshot`

这里有两点很重要：

1. `LearningStateSnapshot` 在 Recommendation 中是“读模型”，不是 owner 表示
2. `RecommendationDefaults` 是当前 MVP 的固定默认配置，不是用户级配置表映射

#### `service/`

这里放 scheduler 的核心规则服务：

- [backlog_calculator.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/recommendation/scheduler/domain/service/backlog_calculator.go)
- [quota_allocator.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/recommendation/scheduler/domain/service/quota_allocator.go)
- [review_scorer.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/recommendation/scheduler/domain/service/review_scorer.go)
- [new_scorer.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/recommendation/scheduler/domain/service/new_scorer.go)
- [priority_zero_extractor.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/recommendation/scheduler/domain/service/priority_zero_extractor.go)
- [recommendation_assembler.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/recommendation/scheduler/domain/service/recommendation_assembler.go)

这些服务分别负责：

- backlog 计算
- quota 分配
- review 候选评分
- new 候选评分
- P0 提取
- batch 组装

### `infrastructure/`

这里只负责技术落地。

#### `config.go`

读取数据库配置。

#### `db.go`

初始化 `pgx` 连接池。

#### `migration/`

定义 Recommendation 自己的 schema owner：

- `000001_create_recommendation_schema`
- `000002_create_user_unit_serving_states`
- `000003_create_scheduler_runs`
- `000004_create_scheduler_run_items`
- `000005_create_recommendation_indexes`

这里不应该出现 Learning engine 的表。

#### `persistence/query/*.sql`

定义 scheduler 当前使用的 SQL：

- [candidates.sql](/Users/evan/Downloads/learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/query/candidates.sql)
- [serving_states.sql](/Users/evan/Downloads/learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/query/serving_states.sql)
- [scheduler_runs.sql](/Users/evan/Downloads/learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/query/scheduler_runs.sql)

#### `persistence/sqlcgen/`

`sqlc` 生成代码，原则上不手改。

#### `persistence/mapper/`

负责隔离：

- `sqlc` row / param
- Recommendation 领域对象

#### `persistence/repository/`

实现 application 层依赖的 repository interface：

- [learning_state_snapshot_read_repo.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/repository/learning_state_snapshot_read_repo.go)
- [user_unit_serving_state_repo.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/repository/user_unit_serving_state_repo.go)
- [scheduler_run_repo.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/repository/scheduler_run_repo.go)

#### `persistence/tx/`

事务管理器实现：

- [pgx_tx_manager.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/tx/pgx_tx_manager.go)

### `test/`

测试现在按职责分层：

- `test/unit`
  放纯单元测试
- `test/integration`
  放真实数据库和真实用例编排测试
- `test/scenario`
  放更长链路的业务场景测试

当前覆盖重点如下：

- `test/unit/domain/service/*`
  覆盖 backlog、quota、scorer、assembler
- `test/unit/infrastructure/config_test.go`
  覆盖配置校验
- `test/integration/infrastructure/*`
  覆盖数据库连接探针和 candidate query
- `test/integration/usecase/generate_recommendations_usecase_test.go`
  覆盖推荐生成用例和落库
- `test/scenario/scenarios_test.go`
  覆盖典型推荐业务场景

`test/integration/fixture/helpers.go` 负责共享数据库、测试用户、coarse unit、状态插入和 use case 构造辅助。

## 6. 关键调用关系

当前推荐主链路可以简化成：

```text
GenerateLearningUnitRecommendationsUseCase
  -> LearningStateSnapshotReadRepository.FindDueReviewCandidates
  -> LearningStateSnapshotReadRepository.FindNewCandidates
  -> BacklogCalculator.Compute
  -> QuotaAllocator.Allocate
  -> ReviewScorer.Score
  -> NewScorer.Score
  -> PriorityZeroExtractor.Extract
  -> RecommendationAssembler.Assemble
  -> SchedulerRunRepository.SaveRun
  -> SchedulerRunRepository.SaveRunItems
  -> UserUnitServingStateRepository.TouchRecommendedAt
```

这里的关键点是：

- scheduler 先读 Learning engine 的状态
- 再做纯推荐决策
- 最后只写 Recommendation 自己的表

## 7. 与 Learning engine 的边界

scheduler 读取 Learning engine 的字段，但不拥有它们。

当前重要输入来源如下：

### 来自 `learning.user_unit_states`

- `is_target`
- `target_source`
- `target_source_ref_id`
- `target_priority`
- `status`
- `progress_percent`
- `mastery_score`
- `last_quality`
- `next_review_at`
- `consecutive_wrong`

### 来自 `recommendation.user_unit_serving_states`

- `last_recommended_at`

这条边界非常关键：

- `target_*` 仍属于 Learning engine 聚合结果
- `last_recommended_at` 已迁到 Recommendation 自己的 serving state

## 8. 新人最应该先看的文件

如果你要快速理解这个子模块，建议按下面顺序读：

1. [generate_recommendations.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/recommendation/scheduler/application/usecase/generate_recommendations.go)
2. [candidate.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/recommendation/scheduler/application/query/candidate.go)
3. [recommendation.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/recommendation/scheduler/domain/model/recommendation.go)
4. [recommendation_defaults.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/recommendation/scheduler/domain/model/recommendation_defaults.go)
5. [quota_allocator.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/recommendation/scheduler/domain/service/quota_allocator.go)
6. [review_scorer.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/recommendation/scheduler/domain/service/review_scorer.go)
7. [new_scorer.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/recommendation/scheduler/domain/service/new_scorer.go)
8. [recommendation_assembler.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/recommendation/scheduler/domain/service/recommendation_assembler.go)
9. [candidates.sql](/Users/evan/Downloads/learning-video-recommendation-system/internal/recommendation/scheduler/infrastructure/persistence/query/candidates.sql)
10. [generate_recommendations_usecase_test.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/recommendation/scheduler/test/integration/usecase/generate_recommendations_usecase_test.go)

## 9. 常见改动应该落在哪里

### 调整 quota 规则

优先看：

- `domain/service/quota_allocator.go`
- `domain/model/recommendation_defaults.go`
- 对应测试

### 调整 review / new 排序公式

优先看：

- `domain/service/review_scorer.go`
- `domain/service/new_scorer.go`
- `test/unit/domain/service/scoring_test.go`

### 调整推荐输出结构

通常要同时改：

- `domain/model/recommendation.go`
- assembler
- run mapper
- `scheduler_runs.sql`
- 相关测试

### 调整候选 SQL

通常要同时改：

- `application/query/candidate.go`
- `infrastructure/persistence/query/candidates.sql`
- mapper
- `sqlc` 生成代码
- `test/integration/infrastructure/candidate_queries_test.go`

## 10. 不该做什么

以后继续维护时，不要做这些事：

- 不要重新给 Recommendation 加 `learning.*` 写权限
- 不要把 `last_recommended_at` 塞回 `learning.user_unit_states`
- 不要在 SQL 中实现评分公式
- 不要把用户级调度配置表偷偷补回去却不更新文档边界
- 不要在 `sqlcgen/` 目录里手改生成代码
