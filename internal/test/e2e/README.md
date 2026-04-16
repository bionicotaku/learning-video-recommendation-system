# E2E Tests

这个目录放跨顶层模块、真实数据库链路的端到端测试。

当前这里已经不是空目录，而是一套 **Learning engine × Recommendation** 的真实迁移驱动 E2E：

- 使用一个共享 embedded Postgres
- 直接执行 `learningengine` 与 `recommendation` 的真实 migration `*.up.sql`
- 不通过 `dbtool`
- 不依赖 live DB
- 不把任一侧降级成 stub

## 当前测试基座

共享基座在 [testutil/harness.go](/Users/evan/Downloads/learning-video-recommendation-system/internal/test/e2e/testutil/harness.go)，固定负责：

- 启动和关闭 embedded Postgres
- 顺序执行：
  - `internal/learningengine/infrastructure/persistence/schema/000000_external_refs.sql`
  - `internal/learningengine/infrastructure/migration/*.up.sql`
  - 补齐 Recommendation 需要的外部 `catalog.videos` 字段
  - `internal/recommendation/infrastructure/persistence/schema/000000_external_refs.sql`
  - 删除 external refs 里占位的 Recommendation 物化视图
  - `internal/recommendation/infrastructure/migration/*.up.sql`
- seed 最小外部事实层：
  - `auth.users`
  - `semantic.coarse_unit`
  - `catalog.videos`
  - `catalog.video_transcripts`
  - `catalog.video_unit_index`
  - `catalog.video_semantic_spans`
  - `catalog.video_transcript_sentences`
  - `catalog.video_user_states`
- 刷新：
  - `recommendation.v_recommendable_video_units`
  - `recommendation.v_unit_video_inventory`
- 装配真实 usecase：
  - Learning engine：`EnsureTargetUnits`、`SetTargetInactive`、`SuspendTargetUnit`、`ResumeTargetUnit`、`RecordLearningEvents`、`ReplayUserStates`、`ListUserUnitStates`
  - Recommendation：完整 pipeline + `RecommendationResultWriter`

## 当前场景矩阵

当前已经覆盖：

- `learning_to_recommendation_test.go`
  - `ensure target` 后零事件用户可被 Recommendation 读取
  - `suspend / resume / inactive` 对 Recommendation 输入即时生效
  - `ReplayUserStates` 前后跨模块可观察结果保持一致
- `recommendation_demand_mapping_test.go`
  - `learning.user_unit_states` 的真实字段组合映射到 `hard_review / new_now / soft_review`
  - 零事件但无供给的 target 会形成真实 demand，并在 underfill 时标记 `extreme_sparse`
  - suspended / inactive / 非 target 单元不会进入 Recommendation demand
- `recommendation_supply_modes_test.go`
  - 正常库存下 `selector_mode = normal`
  - `hard_review` 低供给下 `selector_mode = low_supply`
  - 无 active states 时 `selector_mode = normal`，但 `underfilled = true`
  - bundle 视频在真实跨模块链路里可生效
- `recommendation_selector_constraints_test.go`
  - 有真实 demand 且最终 underfill 时 `selector_mode = extreme_sparse`
  - `same_unit_max = 2`
  - `fallback_max = 1`
  - `core_dominant_min` 在真实主链路里生效
  - `low_supply` 下 `future_like_max` 生效
- `recommendation_read_model_visibility_test.go`
  - refresh 前后物化读模型可见性变化
  - `status != active` / `visibility_status != public` / `publish_at > now()` 的视频不会进入推荐
  - `v_unit_video_inventory` 的 `none / weak / ok / strong` 分级契约
- `recommendation_audit_serving_test.go`
  - response `best_evidence_*` 与 audit item 一致
  - `video_recommendation_runs` / `video_recommendation_items` 写入
  - `user_unit_serving_states` / `user_video_serving_states` 写入
  - 第二次推荐时 serving penalty 与 watched penalty 生效
- `recommendation_write_side_test.go`
  - run 元数据与 response 保持一致
  - item rank 连续且稳定
  - 写侧失败时 run / item / serving 整体回滚
  - replay 不会重置 Recommendation own state 的累计计数
- `recommendation_multi_user_isolation_test.go`
  - 多用户共享 Catalog 事实层时，learning / serving / recommendation 输出严格隔离
  - 一个用户 replay 不影响另一个用户的推荐结果

## 稳定断言边界

这些 E2E 只断言跨模块稳定契约，不比较内部私有实现细节。

当前稳定断言包括：

- Learning engine 输出：
  - `status`
  - `is_target`
  - `target_priority`
  - `last_quality`
  - `next_review_at`
  - `strong_event_count`
  - `review_count`
- Recommendation 输出：
  - `selector_mode`
  - `underfilled`
  - selected `video_id` 顺序或关键包含关系
  - `reason_codes` 的稳定子集
  - `covered_*_units`
  - `best_evidence`
- Recommendation 落库对象：
  - run / item 数量
  - run 的 `selector_mode / underfilled / result_count`
  - item 的 `rank / primary_lane / dominant_bucket / dominant_unit_id`
  - serving state `served_count`
  - `v_unit_video_inventory.supply_grade`

当前故意不测：

- 精确浮点分数
- 全部 `reason_codes` 的完整顺序
- planner / candidate 中间 JSON snapshot 的完整字节级内容

## 运行方式

单独运行跨模块 E2E：

```bash
make e2e-test
```

或：

```bash
go test -tags=e2e ./internal/test/e2e/...
```

默认 `make check` **不会**包含这些 E2E，目的是避免常规检查时间过长。
