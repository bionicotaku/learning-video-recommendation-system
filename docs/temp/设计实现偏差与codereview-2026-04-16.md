# 设计实现偏差与 Code Review 复查记录（2026-04-16）

## 0. 文档信息

- 评审对象：
  - `docs/全新设计-Catalog-数据库设计.md`
  - `docs/全新设计-学习引擎设计.md`
  - `docs/全新设计-总设计.md`
  - `docs/全新设计-推荐模块设计.md`
  - `internal/catalog`
  - `internal/learningengine`
  - `internal/recommendation`
- 评审方法：
  - 设计对照
  - 代码检查
  - 测试与验收真实性检查
- 当前限制：
  - 本地 `coderabbit` CLI 不可用。执行 `coderabbit --version` 返回 `command not found`，因此本次 CodeRabbit 结果记为 `unavailable`，不伪造外部审阅结果。
- 本次实际执行过的验证命令：
  - `go test ./internal/learningengine/... ./internal/recommendation/...`
  - `go test -tags=integration ./internal/recommendation/test/...`
  - `make check`

## 1. 第一轮已知发现记录

### Finding ID: F-LE-001

- 来源：`round_1_existing`
- Severity：`P1`
- 模块：`learningengine`
- 设计依据：
  - `docs/全新设计-学习引擎设计.md:1684-1693`
  - `docs/全新设计-总设计.md` 中 Replay 与在线一致性要求
- 实现位置：
  - `internal/learningengine/application/service/replay_user_states.go:31-61`
  - `internal/learningengine/infrastructure/persistence/tx/manager.go:24-39`
- 问题描述：
  - `ReplayUserStates` 没有实现“同一用户 Replay 时加用户级互斥”或“Replay 期间禁止同一用户在线写入”的并发保护。
- 影响：
  - Replay 与在线写入并发时，可能出现事件已写入事实层，但 replay 最终状态未包含该批事件的情况。
- 当前结论：
  - 明确违反设计约束，属于正确性问题。

### Finding ID: F-REC-001

- 来源：`round_1_existing`
- Severity：`P1`
- 模块：`recommendation`
- 设计依据：
  - `docs/全新设计-推荐模块设计.md:134-149`
  - `docs/全新设计-推荐模块设计.md:317`
- 实现位置：
  - `internal/recommendation/application/usecase/generate_video_recommendations_impl.go:41-43`
  - `internal/recommendation/application/usecase/generate_video_recommendations_impl.go:85-87`
  - `internal/recommendation/application/usecase/generate_video_recommendations_impl.go:155-167`
  - `internal/recommendation/test/unit/application/usecase/generate_video_recommendations_test.go:14-36`
- 问题描述：
  - `GenerateVideoRecommendations` 仍保留 assembler-only shell constructor，并在依赖未接齐时静默返回空推荐。
- 影响：
  - 主用例可以在流水线未完整装配时“成功”返回空结果，掩盖装配错误。
- 当前结论：
  - 明确违反“唯一主入口必须组织完整流水线”的设计要求。

### Finding ID: F-REC-002

- 来源：`round_1_existing`
- Severity：`P2`
- 模块：`recommendation`
- 设计依据：
  - `docs/全新设计-推荐模块设计.md:242-246`
- 实现位置：
  - `internal/recommendation/domain/planner/default_demand_planner.go:108-119`
  - `internal/recommendation/application/usecase/generate_video_recommendations_impl.go:225-233`
- 问题描述：
  - `ExtremeSparse` 只在四个 bucket 全为空时才触发，而不是在“加上 fallback 后仍不足，或在合理约束下无法凑满 N 条”时触发。
- 影响：
  - `selector_mode` 审计语义失真，稀疏库存降级逻辑与设计不一致。
- 当前结论：
  - 重要设计偏差，可能导致错误运行模式。

### Finding ID: F-REC-003

- 来源：`round_1_existing`
- Severity：`P2`
- 模块：`recommendation`
- 设计依据：
  - 主链路应沿请求上下文传播，设计中不存在脱离请求上下文的热路径读库。
- 实现位置：
  - `internal/recommendation/application/service/default_candidate_generator.go:27`
  - `internal/recommendation/application/service/default_evidence_resolver.go:36-37`
  - `internal/recommendation/application/service/default_evidence_resolver.go:52-53`
- 问题描述：
  - 热路径读库直接使用 `context.Background()`，丢掉了调用方的取消、超时和 trace 上下文。
- 影响：
  - 请求取消和超时无法正确作用于候选读取和证据解析。
- 当前结论：
  - 真实运行时问题。

### Finding ID: F-REC-004

- 来源：`round_1_existing`
- Severity：`P2`
- 模块：`recommendation`
- 设计依据：
  - `docs/全新设计-推荐模块设计.md:389-391`
  - `docs/全新设计-总设计.md` 中关于 `v_recommendable_video_units` / `v_unit_video_inventory` / audit 的 owner 契约
- 实现位置：
  - `internal/recommendation/test/fixture/db.go:71-145`
  - `internal/recommendation/test/integration/repository_integration_test.go:17-237`
- 问题描述：
  - Recommendation integration tests 使用手写最小 schema stub，没有真实应用 Recommendation migration，也没有覆盖两个物化读视图。
- 影响：
  - 测试无法证明 migration head、read model SQL 和真实 owner 契约一致。
- 当前结论：
  - 关键测试缺口。

## 2. 第二轮复查方法与检查面

第二轮 review 不复用第一轮结论作为前提，而是按以下顺序从头重走：

1. 模块边界与 owner
2. 数据库契约与 migration
3. 对外 usecase / DTO / repository port
4. 主链路时序与降级路径
5. 领域规则和公式
6. 事务、上下文、并发与错误语义
7. 测试覆盖与验收真实性

## 3. A. 实现与设计一致项

### A.1 Catalog 结构与边界基本对齐

- `catalog` 仍然只拥有内容事实、Recall-ready 索引、入库审计和视频互动投影，未把 `learning.*` 或 `recommendation.*` 混入 Catalog owner。
- 当前事实链与设计一致：
  - `catalog.videos`
  - `catalog.video_transcripts`
  - `catalog.video_transcript_sentences`
  - `catalog.video_semantic_spans`
  - `catalog.video_unit_index`
- `catalog.video_unit_index` 已采用 `evidence_span_refs jsonb`，没有重新引入被设计明确禁止的旧结构：
  - `catalog.video_segments`
  - `catalog.segment_unit_mappings`
- 证据：
  - 设计：`docs/全新设计-Catalog-数据库设计.md:36`, `150-174`, `250-256`
  - 实现：
    - `internal/catalog/infrastructure/migration/000006_create_video_unit_index.up.sql:1-24`
    - `internal/catalog/infrastructure/migration/000005_create_video_semantic_spans.up.sql:1-24`
    - `internal/catalog/infrastructure/migration/000004_create_video_transcript_sentences.up.sql:1-15`

### A.2 Learning engine 的事件层 / 状态层边界与设计一致

- `learning.unit_learning_events` 作为事实层，`learning.user_unit_states` 作为归约状态层，边界与设计一致。
- 在线写入和 Replay 共用同一个 reducer。
- `SuspendTargetUnit` / `ResumeTargetUnit` 没有保留错误的 SQL 直写恢复语义，恢复时会清理挂起控制字段后重新计算 active status。
- 证据：
  - 设计：`docs/全新设计-学习引擎设计.md:152-158`, `199-211`, `1402-1432`
  - 实现：
    - `internal/learningengine/application/service/record_learning_events.go:63-97`
    - `internal/learningengine/domain/aggregate/reducer.go:14-62`
    - `internal/learningengine/application/service/replay_user_states.go:31-59`
    - `internal/learningengine/application/service/target_unit_commands.go:130-146`

### A.3 Learning engine 的核心公式和状态迁移总体对齐

- 弱/强事件划分与设计一致。
- 简化 SM-2 规则与设计一致：
  - 成功：`1 / 3 / 6 / round(interval_days * ease_factor)`
  - 失败：`repetition = 0`, `interval_days = 1`
  - `ease_factor` 下限 `1.3`
- `progress_percent` 与 `mastery_score` 的公式与文档一致。
- `mastered` 的 21 天阈值已实现。
- 证据：
  - 设计：
    - `docs/全新设计-学习引擎设计.md:233-253`
    - `docs/全新设计-学习引擎设计.md:1168-1255`
  - 实现：
    - `internal/learningengine/domain/aggregate/reducer.go:25-60`
    - `internal/learningengine/domain/policy/progression.go:19-109`

### A.4 Recommendation 的 owner 边界总体对齐

- Recommendation 只读 `learning.user_unit_states`、Catalog 内容事实 / 读模型、`catalog.video_user_states`。
- Recommendation 只写 `recommendation.*`。
- 没有回写 Learning engine 或 Catalog owner 对象。
- 证据：
  - 设计：`docs/全新设计-推荐模块设计.md:12-26`, `38-42`
  - 实现：
    - `internal/recommendation/infrastructure/persistence/query/recommendation_reads.sql:1-68`
    - `internal/recommendation/infrastructure/migration/000003_create_recommendation_audit_tables.up.sql`
    - `internal/recommendation/infrastructure/migration/000004_create_materialized_views.up.sql:1-24`
    - `internal/recommendation/infrastructure/migration/000005_create_recommendation_indexes.up.sql:1-62`

### A.5 Recommendation 的主链路模块基本齐全

- 已有实现模块：
  - Context Assembler
  - Demand Planner
  - Candidate Generator
  - Evidence Resolver
  - Video Evidence Aggregator
  - Video Ranker
  - Video Selector
  - Explanation Builder
  - Audit / Serving State 写入
- 这说明 Recommendation 已经不是骨架态，而是完整的一条视频推荐流水线实现。
- 证据：
  - `internal/recommendation/application/usecase/generate_video_recommendations_impl.go:25-152`
  - `internal/recommendation/application/service/context_assembler.go:19-84`
  - `internal/recommendation/application/service/default_candidate_generator.go:14-277`
  - `internal/recommendation/application/service/default_evidence_resolver.go:13-76`
  - `internal/recommendation/domain/aggregator/default_video_evidence_aggregator.go:19-128`
  - `internal/recommendation/domain/ranking/default_video_ranker.go:20-67`
  - `internal/recommendation/domain/selector/default_video_selector.go:18-99`
  - `internal/recommendation/domain/explain/default_explanation_builder.go:19-42`

## 4. B. 实现偏差但暂不构成明确 bug 的项

### B.1 Recommendation 返回结构把 `best_evidence` 扁平化了（已修复）

- 当前状态：
  - 对外 DTO 已改为 `BestEvidence *BestEvidence`
  - 当 4 个 evidence 值全为空时返回 `nil`，而不是空对象
  - audit 存储层仍保持扁平字段，这部分属于持久化契约，不在本轮变更范围
- 证据：
  - 设计：`docs/全新设计-推荐模块设计.md:128`, `317`
  - 实现：`internal/recommendation/application/dto/generate_video_recommendations.go`

### B.2 Context Assembler 的边界比设计更窄（已收口为显式两阶段边界）

- 当前状态：
  - `DefaultContextAssembler` 明确只装配 request-scope / unit-scope 输入：active states、inventory、unit serving states
  - `DefaultVideoStateEnricher` 显式负责 candidate-derived video-scope 输入：video serving states、video user states
  - Recommendation usecase 不再内联 repository 读取视频态状态
- 证据：
  - 设计：`docs/全新设计-推荐模块设计.md:160-162`
  - 实现：
    - `internal/recommendation/application/service/context_assembler.go`
    - `internal/recommendation/application/service/default_video_state_enricher.go`
    - `internal/recommendation/application/usecase/generate_video_recommendations_impl.go`

### B.3 Recommendation 的排序公式额外引入了 `recent_watched_penalty`（已修复）

- 当前状态：
  - `RecentWatchedPenalty` 仍保留为辅助观测值
  - `FreshnessScore` 仍可综合 watched 信号
  - MVP `BaseScore` 已删除 `- 0.02 * RecentWatchedPenalty` 这一项，避免重复惩罚
- 证据：
  - 设计：
    - `docs/全新设计-推荐模块设计.md:214-222`
    - `docs/全新设计-总设计.md:810-825`
  - 实现：`internal/recommendation/domain/ranking/default_video_ranker.go`

## 5. C. 明确的 Code Review Findings

### C.1 Finding ID: F-LE-001

- 来源：`round_2_new_confirmed`
- Severity：`P1`
- 模块：`learningengine`
- 设计依据：
  - `docs/全新设计-学习引擎设计.md:1684-1693`
  - `docs/全新设计-学习引擎设计.md:1450-1455`
- 实现位置：
  - `internal/learningengine/application/service/replay_user_states.go:31-61`
  - `internal/learningengine/infrastructure/persistence/tx/manager.go:24-39`
- 问题描述：
  - Replay 没有用户级互斥或等价机制来阻止同一用户在线写入并发进入。
- 影响：
  - 会破坏“在线逐条写入最终状态 == full replay 最终状态”的关键设计保证。
- Why This Violates The Design：
  - 文档已明确要求 Replay 期间禁止同一用户在线写入，或使用用户级互斥。
- Fix Direction：
  - 在 Learning engine 内增加同一用户 Replay / online write 的互斥机制，至少保证 `RecordLearningEvents` 与 `ReplayUserStates` 不会并发处理同一用户。

### C.2 Finding ID: F-REC-001

- 来源：`round_2_new_confirmed`
- Severity：`P1`
- 模块：`recommendation`
- 设计依据：
  - `docs/全新设计-推荐模块设计.md:134-149`
  - `docs/全新设计-推荐模块设计.md:317`
- 实现位置：
  - `internal/recommendation/application/usecase/generate_video_recommendations_impl.go:41-43`
  - `internal/recommendation/application/usecase/generate_video_recommendations_impl.go:85-87`
  - `internal/recommendation/application/usecase/generate_video_recommendations_impl.go:155-167`
  - `internal/recommendation/README.md:45-48`
- 问题描述：
  - Recommendation 主用例仍支持 assembler-only shell 降级路径。
- 影响：
  - 主链路未完整装配时不会报错，而是返回空推荐，导致错误被隐藏。
- Why This Violates The Design：
  - 设计要求 `GenerateVideoRecommendations` 是唯一主入口，并且组织完整流水线，而不是可降级壳。
- Fix Direction：
  - 删除 shell constructor / shell response 路径；未完整装配时直接返回显式错误。

### C.3 Finding ID: F-REC-002

- 来源：`round_2_new_confirmed`
- Severity：`P2`
- 模块：`recommendation`
- 设计依据：
  - `docs/全新设计-推荐模块设计.md:242-246`
- 实现位置：
  - `internal/recommendation/domain/planner/default_demand_planner.go:108-119`
  - `internal/recommendation/application/usecase/generate_video_recommendations_impl.go:225-233`
- 问题描述：
  - `extreme_sparse` 的判断条件实现成了“当前所有 bucket 都为空”，而不是“在合理约束下仍无法满足 target_video_count”。
- 影响：
  - selector mode、under-fill 语义和审计结果会与设计预期错位。
- Why This Violates The Design：
  - 设计把 `extreme_sparse` 定义成一种供给不足下的运行模式，不是“无需求”的特殊值。
- Fix Direction：
  - 把 `extreme_sparse` 的判定移到 candidate/selector 之后，基于候选不足或约束后仍无法满足 `target_video_count` 来决定。

### C.4 Finding ID: F-REC-003

- 来源：`round_2_new_confirmed`
- Severity：`P2`
- 模块：`recommendation`
- 设计依据：
  - 主链路热路径应沿请求上下文传播，支持取消和超时。
- 实现位置：
  - `internal/recommendation/application/service/default_candidate_generator.go:27`
  - `internal/recommendation/application/service/default_evidence_resolver.go:36-37`
  - `internal/recommendation/application/service/default_evidence_resolver.go:52-53`
- 问题描述：
  - Candidate Generator 和 Evidence Resolver 读库时使用 `context.Background()`。
- 影响：
  - 请求取消、超时、trace 传播在这两段热点路径上失效。
- Why This Violates The Design：
  - 设计要求的是一条完整、可审计、可控的推荐流水线，而不是脱离请求上下文执行的局部读操作。
- Fix Direction：
  - 将 domain/service 接口收敛到显式接收 `ctx context.Context`，并沿主链路把调用上下文传到底层 repository。

### C.5 Finding ID: F-REC-004

- 来源：`round_2_new_confirmed`
- Severity：`P2`
- 模块：`recommendation`
- 设计依据：
  - `docs/全新设计-推荐模块设计.md:389-391`
  - `docs/全新设计-总设计.md:1165-1176`
- 实现位置：
  - `internal/recommendation/test/fixture/db.go:71-145`
  - `internal/recommendation/test/integration/repository_integration_test.go:17-237`
- 问题描述：
  - Integration tests 没有真实跑 Recommendation migration，也没有覆盖 `v_recommendable_video_units` / `v_unit_video_inventory`。
- 影响：
  - 目前无法证明 migration head 与 repository/query/read model 真正对齐。
- Why This Violates The Design：
  - 文档把这些物化视图和审计表定义成 Recommendation 的核心 owner 契约，但测试没有验证它们。
- Fix Direction：
  - 让 integration fixture 真实应用 Recommendation migration head，并补对物化视图与 refresh 路径的验证。

## 6. API / 契约偏差

### 6.1 `GenerateVideoRecommendations` 是否仍是唯一对外主入口

- 结论：
  - 部分符合。
- 原因：
  - Recommendation 仍以 `GenerateVideoRecommendationsService` 为唯一对外 usecase 类型。
  - 但由于保留 assembler-only shell constructor，主入口的运行语义被分裂成了“完整流水线”与“空壳降级”两种模式。
- 证据：
  - `internal/recommendation/application/usecase/generate_video_recommendations_impl.go:25-71`
  - `internal/recommendation/README.md:45-48`

### 6.2 Learning engine 是否只暴露 `learning.user_unit_states` 稳定契约

- 结论：
  - 符合。
- 原因：
  - Recommendation 读取 `learning.user_unit_states` 的字段集合与设计一致，没有要求 Learning engine 暴露 reducer 内部实现。
- 证据：
  - `internal/recommendation/infrastructure/persistence/query/recommendation_reads.sql:1-21`
  - `docs/全新设计-推荐模块设计.md:38`

### 6.3 Recommendation 是否反向要求 Learning engine 持久化 `fragility` / `instability`

- 结论：
  - 符合。
- 原因：
  - Recommendation 当前 planner 直接基于 `status`、`next_review_at`、`mastery_score`、`last_quality` 等底层状态推导软信号，没有要求 Learning engine 增加额外持久化字段。
- 证据：
  - `internal/recommendation/domain/planner/default_demand_planner.go:60-106`
  - `docs/全新设计-推荐模块设计.md:166`
  - `docs/全新设计-总设计.md:965`

### 6.4 DTO 是否与设计的最小返回结构一致

- 结论：
  - 大体符合，但 `best_evidence` 的表达方式有偏差。
- 原因：
  - `run_id`、`selector_mode`、`underfilled`、`videos[]` 都已具备。
  - `best_evidence` 未抽象为对象，而是拆成 4 个扁平字段。
- 证据：
  - `internal/recommendation/application/dto/generate_video_recommendations.go:3-32`

## 7. 测试覆盖与验收真实性

### 7.1 已实际执行并通过的验证

- `go test ./internal/learningengine/... ./internal/recommendation/...`
- `go test -tags=integration ./internal/recommendation/test/...`
- `make check`

### 7.2 Learning engine 的测试真实性评价

- 优点：
  - 已覆盖在线写入和 Replay 的主路径。
  - 已验证 full replay 与在线结果一致。
  - 已覆盖事务回滚。
- 证据：
  - 单测：`internal/learningengine/application/service/usecases_test.go:300-380`
  - 连库测试：`internal/learningengine/application/service/usecases_integration_test.go:167-250`
- 当前缺口：
  - 没有验证“Replay 与在线写入并发冲突”这一设计中明确提出的约束。

### 7.3 Recommendation 的测试真实性评价

- 优点：
  - 已覆盖 planner、ranker、selector、aggregator、explanation 的单测。
  - 已覆盖完整 usecase pipeline 的 golden 测试。
  - 已覆盖 build-tag integration tests。
- 证据：
  - pipeline：`internal/recommendation/test/unit/application/usecase/generate_video_recommendations_pipeline_test.go:26-140`
  - selector：`internal/recommendation/test/unit/domain/selector/video_selector_test.go:11-101`
- 当前缺口：
  - 上述 Recommendation 测试缺口已在后续修复中补齐：
    - integration tests 已切到真实 migration / 物化视图
    - 默认 `make check` 已包含 Recommendation integration
    - pipeline / E2E 已覆盖 `extreme_sparse` 的当前设计定义

## 8. 测试覆盖缺口

### Gap ID: G-REC-001

- 缺什么：
  - Recommendation migration head 和两个物化视图的真实连库验证。
- 为什么重要：
  - 这是 Recommendation 最关键的 owner 契约之一。
- 当前状态：
  - 已修复。integration fixture 现已执行外部依赖 stub SQL + Recommendation 真实 migration head，并覆盖两个物化视图与 refresh 路径。

### Gap ID: G-REC-002

- 缺什么：
  - `extreme_sparse` 判定从 planner 到 selector_mode/audit 的端到端验证。
- 为什么重要：
  - 当前实现最容易在稀疏供给场景下偏离设计。
- 当前测试为什么不足：
  - selector 单测只验证 flag 输入后的选择行为，不验证 flag 的生成逻辑。

### Gap ID: G-LE-001

- 缺什么：
  - Replay 与在线写入并发冲突保护测试。
- 为什么重要：
  - 这是 Learning engine 状态一致性的关键边界。
- 当前状态：
  - 已修复。`WithinUserTx(...)` 的 advisory xact lock 串行化了同用户写入，并补了 manager/usecase 级并发验证。

### Gap ID: G-REC-003

- 缺什么：
  - Context 取消 / 超时是否能贯穿 Candidate Generator 与 Evidence Resolver。
- 为什么重要：
  - 这是热路径的运行时稳定性要求。
- 当前状态：
  - 已修复。Candidate Generator / Evidence Resolver 已透传调用方 `ctx`，并补了 context 传播单测。

## 9. 总体结论

### 9.1 哪一部分最贴近设计

- `catalog` 最贴近设计，几乎没有看到结构级偏差。
- `learningengine` 的核心事件 / 状态 / reducer / Replay 模型也已经高度收敛到设计文档。

### 9.2 哪一部分偏差最大

- `recommendation` 的偏差最大，主要集中在：
  - 主入口仍允许降级为空壳
  - 稀疏供给运行模式判定错位
  - 热路径上下文传播断裂
  - integration tests 没有验证真实 migration 契约

### 9.3 第二轮复查是否新增更高严重度 finding

- 第二轮复查未新增比第一轮更高严重度的 finding。
- 但第二轮把这些问题进一步确认成了更强的结论，并补齐了 API 偏差、测试盲区和设计一致项。

## 10. 建议修复顺序

1. 先修 `P1`：
   - `learningengine` 的 Replay / 在线写入用户级互斥
   - `recommendation` 删除 assembler-only shell 路径
2. 再修关键 `P2`：
   - `extreme_sparse` 判定逻辑
   - Candidate Generator / Evidence Resolver 的上下文传播
   - Recommendation integration tests 改为真实 migration 驱动
3. 最后修 `P3` 和文档/API 收口：
   - `best_evidence` 返回结构表达
   - Context Assembler 边界是否要与文档重新对齐
   - 排序公式与 README / 文档同步

## 11. 本次评审使用的主要证据文件

- 设计文档：
  - `docs/全新设计-Catalog-数据库设计.md`
  - `docs/全新设计-学习引擎设计.md`
  - `docs/全新设计-总设计.md`
  - `docs/全新设计-推荐模块设计.md`
- 代码实现：
  - `internal/catalog/infrastructure/migration/*`
  - `internal/learningengine/application/service/*`
  - `internal/learningengine/domain/*`
  - `internal/recommendation/application/usecase/generate_video_recommendations_impl.go`
  - `internal/recommendation/application/service/*`
  - `internal/recommendation/domain/*`
  - `internal/recommendation/test/*`

## 12. 修复状态更新（本轮已完成）

### F-LE-001

- 状态：`已修复`
- 修复内容：
  - `learningengine` 的所有同用户状态写入 usecase 统一改为走 `TxManager.WithinUserTx(...)`
  - `WithinUserTx(...)` 在事务开始后通过 `pg_advisory_xact_lock(hashtextextended('learningengine:user:' || $1, 0))` 获取用户级事务锁
  - `EnsureTargetUnits` / `SetTargetInactive` 也已纳入同一用户锁范围，不再只保护 `RecordLearningEvents` / `ReplayUserStates`
- 主要代码位置：
  - `internal/learningengine/application/service/tx.go`
  - `internal/learningengine/infrastructure/persistence/tx/manager.go`
  - `internal/learningengine/application/service/target_unit_commands.go`
  - `internal/learningengine/application/service/record_learning_events.go`
  - `internal/learningengine/application/service/replay_user_states.go`
- 新增验证：
  - `internal/learningengine/infrastructure/persistence/tx/manager_integration_test.go`
  - `internal/learningengine/application/service/usecases_integration_test.go`

### F-REC-001

- 状态：`已修复`
- 修复内容：
  - 删除 assembler-only shell 路径
  - Recommendation 只保留完整 pipeline constructor
  - constructor 在装配阶段对 assembler/planner/candidate/resolver/aggregator/ranker/selector/explainer 做完整依赖校验，缺失时返回显式错误 `ErrIncompletePipeline`
- 主要代码位置：
  - `internal/recommendation/application/usecase/generate_video_recommendations_impl.go`
  - `internal/recommendation/test/unit/application/usecase/generate_video_recommendations_test.go`
  - `internal/recommendation/README.md`

### F-REC-002

- 状态：`已修复`
- 修复内容：
  - planner 不再预判 `ExtremeSparse`
  - `selector_mode=extreme_sparse` 改为在 selection 完成后，根据“存在实际 demand 且最终 underfill”统一后置判定
  - selector 不再持有基于 `ExtremeSparse` 的专用分支
- 主要代码位置：
  - `internal/recommendation/domain/planner/default_demand_planner.go`
  - `internal/recommendation/domain/selector/default_video_selector.go`
  - `internal/recommendation/application/usecase/generate_video_recommendations_impl.go`
  - `internal/recommendation/test/unit/application/usecase/generate_video_recommendations_pipeline_test.go`

### F-REC-003

- 状态：`已修复`
- 修复内容：
  - `CandidateGenerator.Generate(...)` 与 `EvidenceResolver.Resolve(...)` 都改为显式接收 `ctx`
  - `GenerateVideoRecommendations.Execute(...)` 透传调用方上下文到底层读库
  - 已删除热路径读库中的 `context.Background()` 使用
- 主要代码位置：
  - `internal/recommendation/domain/candidate/candidate_generator.go`
  - `internal/recommendation/domain/resolver/evidence_resolver.go`
  - `internal/recommendation/application/service/default_candidate_generator.go`
  - `internal/recommendation/application/service/default_evidence_resolver.go`
  - `internal/recommendation/test/unit/application/service/candidate_generator_test.go`
  - `internal/recommendation/test/unit/application/service/evidence_resolver_test.go`

### F-REC-004

- 状态：`已修复`
- 修复内容：
  - Recommendation integration fixture 不再手写 recommendation schema
  - 先执行 `internal/recommendation/infrastructure/persistence/schema/000000_external_refs.sql`
  - 再顺序执行 Recommendation migration head
  - integration tests 已覆盖真实 `v_recommendable_video_units`、`v_unit_video_inventory` 和 refresh 路径
- 主要代码位置：
  - `internal/recommendation/test/fixture/db.go`
  - `internal/recommendation/test/integration/repository_integration_test.go`

### 本轮实际验证命令

- `go test ./internal/recommendation/...`
- `go test ./internal/learningengine/...`
- `go test -tags=integration ./internal/recommendation/test/...`

## 13. 修复状态更新（本轮剩余收口项）

### B.1 / API 契约表达

- 状态：`已修复`
- 修复内容：
  - `RecommendationVideo` 对外响应已改为 `BestEvidence *BestEvidence`
  - usecase 映射层在 4 个 evidence 值全空时返回 `nil`
  - golden 与 pipeline 单测已同步
- 主要代码位置：
  - `internal/recommendation/application/dto/generate_video_recommendations.go`
  - `internal/recommendation/application/usecase/generate_video_recommendations_impl.go`
  - `internal/recommendation/test/golden/usecase_pipeline_response.json`
  - `internal/recommendation/test/unit/application/usecase/generate_video_recommendations_pipeline_test.go`

### B.2 / 两阶段输入边界

- 状态：`已修复`
- 修复内容：
  - `DefaultContextAssembler` 不再持有 video-scope repository
  - 新增 `DefaultVideoStateEnricher`，显式负责候选视频派生后的 `video serving states` / `video user states` 加载
  - `GenerateVideoRecommendationsService` 不再内联视频态 enrichment 读取逻辑
- 主要代码位置：
  - `internal/recommendation/application/service/context_assembler.go`
  - `internal/recommendation/application/service/default_video_state_enricher.go`
  - `internal/recommendation/application/service/side_effects.go`
  - `internal/recommendation/application/usecase/generate_video_recommendations_impl.go`
  - `internal/recommendation/test/unit/application/service/video_state_enricher_test.go`

### B.3 / 排序公式收口

- 状态：`已修复`
- 修复内容：
  - `DefaultVideoRanker` 保留 `RecentWatchedPenalty` 作为辅助指标
  - MVP `BaseScore` 已删除对 `RecentWatchedPenalty` 的直接扣分
  - ranking 单测已改为校验“有 watched penalty 指标，但不直接参与 BaseScore”
- 主要代码位置：
  - `internal/recommendation/domain/ranking/default_video_ranker.go`
  - `internal/recommendation/test/unit/domain/ranking/video_ranker_test.go`

### 默认验收链路

- 状态：`已修复`
- 修复内容：
  - `make check` 已纳入 `go test -tags=integration ./internal/recommendation/test/...`
  - E2E 仍保持显式 `make e2e-test`
- 主要代码位置：
  - `Makefile`
  - `docs/当前实现现状.md`
  - `internal/recommendation/README.md`

### 本轮新增验证命令

- `go test ./internal/recommendation/test/unit/...`
- `go test ./internal/recommendation/...`
- `go test ./internal/learningengine/... ./internal/recommendation/...`
- `go test -tags=integration ./internal/recommendation/test/...`
- `make check`
