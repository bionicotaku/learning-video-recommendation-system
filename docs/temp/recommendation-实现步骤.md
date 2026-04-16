# Recommendation 实现步骤

状态：DRAFT  
更新时间：2026-04-16  
适用范围：`internal/recommendation`  
文档定位：Recommendation 模块后续实现阶段的唯一执行步骤与进度记录文档。

## 1. 文档目的与使用规则

本文档只服务 `internal/recommendation` 的实现推进，不服务 Learning engine，不服务 Catalog，也不服务跨模块 e2e。它的目标不是复述设计文档，而是把 Recommendation 的实现工作拆成可以直接执行的大步骤，并为每个步骤固定：

- 先做什么
- 具体怎么做
- 做完怎么测试
- 怎么验收
- 何时更新进度文档

本轮 Recommendation 实现必须遵守以下边界：

- 只读 `learning.user_unit_states`
- 只读 Catalog 稳定内容契约与 `catalog.video_user_states`
- 只写 `recommendation.*`
- 不回写 Learning engine 或 Catalog owner 对象
- 不做跨模块 e2e
- 可以做 Recommendation 自己的仓储集成测试
- 不新增 API 层、HTTP 层、worker 或额外业务入口

本文档是 Recommendation 后续实现阶段的唯一进度记录源。每完成一个大步骤，必须先更新本文档，再开始下一个步骤。

## 2. 当前 Recommendation 现状

当前 Recommendation 已经完成第一轮骨架与基础设施落地，但还没有任何推荐业务主链路实现。

### 2.1 当前已经有的内容

- 模块根文档：
  - `internal/recommendation/README.md`
  - `internal/recommendation/doc.go`
- application 层骨架：
  - `application/dto/generate_video_recommendations.go`
  - `application/repository/*`
  - `application/usecase/generate_video_recommendations.go`
- domain 层骨架：
  - `domain/assembler`
  - `domain/planner`
  - `domain/candidate/lanes`
  - `domain/resolver`
  - `domain/aggregator`
  - `domain/ranking`
  - `domain/selector`
  - `domain/explain`
  - `domain/policy`
  - `domain/model/*`
- infrastructure 层基础设施：
  - Recommendation owner migration
  - `query/recommendation_reads.sql`
  - `query/recommendation_writes.sql`
  - `sqlc.yaml`
  - `sqlcgen/*`
  - `mapper/*`
  - `repository/*`
  - `tx/manager.go`

### 2.2 当前已经有的契约

- DTO：
  - `GenerateVideoRecommendationsRequest`
  - `GenerateVideoRecommendationsResponse`
- repository port：
  - `LearningStateReader`
  - `RecommendableVideoUnitReader`
  - `UnitInventoryReader`
  - `SemanticSpanReader`
  - `TranscriptSentenceReader`
  - `UnitServingStateRepository`
  - `VideoServingStateRepository`
  - `RecommendationAuditRepository`
- usecase interface：
  - `GenerateVideoRecommendationsUsecase`
- 已有领域模型：
  - `LearningStateSnapshot`
  - `RecommendableVideoUnit`
  - `UnitVideoInventory`
  - `SemanticSpan`
  - `TranscriptSentence`
  - `UserUnitServingState`
  - `UserVideoServingState`
  - `RecommendationRun`
  - `RecommendationItem`

### 2.3 当前明确没有的内容

- `GenerateVideoRecommendations` 的 application/usecase 实现
- Context Assembler 实现
- Demand Planner 实现
- Candidate Generator 实现
- Evidence Resolver 实现
- Video Evidence Aggregator 实现
- Video Ranker 实现
- Video Selector 实现
- Explanation Builder 实现
- Audit Writer 的业务编排实现
- Serving State Manager 的业务编排实现
- Recommendation 自己的单测、scenario/golden 测试、仓储集成测试

因此，当前 Recommendation 的准确状态是：

> 已有 owner migration、预计算读模型定义、SQL/query/sqlc 基础层和仓储壳，但还没有视频推荐主链路实现。

## 3. 实现范围与非目标

### 3.1 本轮实现范围

本轮 Recommendation 实现目标是：

- 基于现有骨架实现 Recommendation 主链路
- 对外仍只暴露 `GenerateVideoRecommendations`
- 第一版纳入 `catalog.video_user_states` 的只读支持
- 补齐 Recommendation 自己所需的单测、scenario/golden 测试、仓储集成测试
- 不依赖 Learning engine 或 Catalog 的业务代码已经实现

### 3.2 本轮明确不做

- 跨模块集成测试
- `internal/test/e2e` 测试
- Learning engine reducer / replay / 业务逻辑
- Catalog 导入流程 / 业务逻辑
- HTTP API / gRPC / CLI 业务入口
- 任务层主流程
- 非视频 fallback task
- embedding / semantic recall
- ML ranking

## 4. 总体实施顺序

Recommendation 的实现顺序固定如下，不允许跳步：

1. 补 Recommendation 缺失契约与内部接口
2. 实现 Context Assembler 与 usecase 外壳
3. 实现 Demand Planner
4. 实现 Candidate Generator 四条 lane
5. 实现 Evidence Resolver 与 Video Evidence Aggregator
6. 实现 Ranker / Selector / Explanation Builder
7. 接通 Audit Writer / Serving State Manager / 完整主用例

原因很明确：

- 先补契约，避免后续模块实现时反复改接口
- 先打通上下文装配，再实现 planner
- 先有需求画像，再生成候选
- 先有候选，再做 evidence 解析与 video 聚合
- 先有完整 video candidate，再做排序、选择和解释
- 只在前面读与算都稳定后，最后接短事务写路径

## 5. 步骤总览

| Step | Name | Status | Depends On | Acceptance | Last Updated |
| --- | --- | --- | --- | --- | --- |
| 1 | 补 Recommendation 缺失契约与内部接口 | pending | 无 | `make check` + Recommendation 仓储集成测试 | 2026-04-16 |
| 2 | 实现 Context Assembler 与 usecase 外壳 | pending | Step 1 | `make check` | 2026-04-16 |
| 3 | 实现 Demand Planner | pending | Step 2 | `make check` | 2026-04-16 |
| 4 | 实现 Candidate Generator 四条 lane | pending | Step 3 | `make check` | 2026-04-16 |
| 5 | 实现 Evidence Resolver 与 Video Evidence Aggregator | pending | Step 4 | `make check` | 2026-04-16 |
| 6 | 实现 Ranker / Selector / Explanation Builder | pending | Step 5 | `make check` | 2026-04-16 |
| 7 | 接通 Audit Writer / Serving State Manager / 完整主用例 | pending | Step 6 | `make check` + Recommendation 仓储集成测试 + scenario/golden 验收 | 2026-04-16 |

`Status` 只允许以下三个值：

- `pending`
- `in_progress`
- `accepted`

## 6. 分步骤详细实施说明

## Step 1. 补 Recommendation 缺失契约与内部接口

### 目标

把 Recommendation 现有骨架收口成“后续可以稳定实现主链路”的契约层，避免后续一边写算法一边反复改 repository、model、policy 或 usecase 依赖。

### 实施内容

实施顺序固定为：

1. 先盘点当前已有的 DTO、repository port、domain model，明确哪些契约已足够，哪些仍缺。
2. 补 application/repository 中 Recommendation 主链路真正需要但当前缺失的接口。
3. 补 domain/model 中 Recommendation 主链路真正需要的中间对象。
4. 补 domain/enum 与 domain/policy 中 Recommendation 会稳定引用的枚举和常量。
5. 补 application/service 或对应内部模块接口，用于 orchestrator 依赖注入。
6. 最后才补 `query/sqlc/repository` 对应读写能力，使基础设施与新接口对齐。

必须补齐的契约包括：

- `VideoUserStateReader`
- `UnitServingStateRepository` 的批量读取能力
- `VideoServingStateRepository` 的批量读取能力
- `RecommendationAuditRepository` 的批量写 item 能力
- `RecommendationContext`
- `DemandUnit`
- `DemandBundle`
- `LaneBudget`
- `MixQuota`
- `PlannerFlags`
- `VideoUserState`
- `VideoUnitCandidate`
- `ResolvedEvidenceWindow`
- `VideoCandidate`
- `FinalRecommendationItem`
- `Bucket`
- `Lane`
- `SelectorMode`
- `SessionMode`
- `ReasonCode`

必须补齐的内部模块接口包括：

- `ContextAssembler`
- `DemandPlanner`
- `CandidateGenerator`
- `EvidenceResolver`
- `VideoEvidenceAggregator`
- `VideoRanker`
- `VideoSelector`
- `ExplanationBuilder`
- `ServingStateManager`
- `AuditWriter`

本步骤只补契约和基础设施，不实现 planner/ranker/selector 的业务规则。

### 涉及接口/类型

- `application/repository/*`
- `application/usecase/*`
- `domain/model/*`
- `domain/policy/*`
- `domain/*` 下各能力目录的接口定义
- `infrastructure/persistence/query/*`
- `infrastructure/persistence/sqlcgen/*`
- `infrastructure/persistence/repository/*`

### 测试方式

- 编译级测试：
  - 新增接口后必须保证 Recommendation 模块可编译
  - 新增 repository 实现后必须保证 interface assertion 成立
- Recommendation 仓储集成测试：
  - 读取 `catalog.video_user_states`
  - 读取 unit/video serving states
  - 批量写 recommendation audit items

### 验收标准

- `make check`
- Recommendation 仓储集成测试通过
- 无 import cycle
- 契约层与基础设施层对齐，不存在“接口已定义但无法落到 query/repository”的断裂

### 完成后如何更新文档

完成后必须做以下文档更新：

1. 将 Step 1 状态从 `pending` 改为 `accepted`
2. 在本步骤下写入完成记录
3. 写明本步新增的关键接口/类型
4. 写明执行命令和结果
5. 写明是否与原计划有偏差
6. 只有完成以上更新后，才允许开始 Step 2

### 完成记录

- 日期：
- 实际完成内容：
- 运行命令：
- 验收结果：
- 与计划偏差：
- 下一步入口：

## Step 2. 实现 Context Assembler 与 usecase 外壳

### 目标

先把 Recommendation 的输入装配和主入口时序稳定下来，让主用例具备可执行的 orchestrator 外壳，但先不填复杂推荐算法。

### 实施内容

实施顺序固定为：

1. 先实现 `GenerateVideoRecommendations` 的 application/usecase 外壳。
2. 再实现 `ContextAssembler`，负责统一装配 Recommendation 运行上下文。
3. 给 orchestrator 注入内部模块接口，而不是在 usecase 中直接写 SQL。
4. 先让主用例在“无推荐算法实现”的情况下也能返回合法的空结果结构。

`ContextAssembler` 必须负责：

- 规范化请求默认值：
  - `target_video_count`
  - `preferred_duration_sec`
- 读取 active learning states
- 抽取 unit IDs 并读取 `v_unit_video_inventory`
- 读取 Recommendation 自己的 unit serving states
- 为后续 video 集合读取 `video_user_states` 与 video serving states 预留路径
- 明确 `video_semantic_spans` / `video_transcript_sentences` 延迟到后续 resolver 阶段读取

本步骤不允许实现：

- planner 规则
- candidate 生成
- evidence 解析
- 排序/选择

### 涉及接口/类型

- `GenerateVideoRecommendationsUsecase` 的实现
- `RecommendationContext`
- `ContextAssembler`
- `LearningStateReader`
- `UnitInventoryReader`
- `UnitServingStateRepository`
- `VideoServingStateRepository`
- `VideoUserStateReader`

### 测试方式

- 单测：
  - 请求默认值填充
  - 空用户状态
  - 无 inventory
  - 读取错误透传
  - 只装配上下文、不提前读取 spans/sentences
- usecase 单测：
  - 依赖注入
  - 空结果返回结构
  - orchestration 错误路径

### 验收标准

- `make check`
- Context Assembler 单测通过
- usecase 单测通过
- 主用例可在 mock 依赖下返回合法空响应

### 完成后如何更新文档

完成后必须：

1. 把 Step 2 状态改为 `accepted`
2. 记录新增的 orchestrator 和 assembler 文件/职责
3. 记录测试命令与结果
4. 若请求默认值或上下文字段有收敛，记录到完成记录里
5. 更新完文档后才能开始 Step 3

### 完成记录

- 日期：
- 实际完成内容：
- 运行命令：
- 验收结果：
- 与计划偏差：
- 下一步入口：

## Step 3. 实现 Demand Planner

### 目标

把 Learning state + inventory 转成 Recommendation 可执行的 Demand Bundle，为后续 lane、ranking、selector 提供唯一权威需求画像。

### 实施内容

实施顺序固定为：

1. 先把 planner 依赖的 bucket、session mode、lane budget、mix quota、flags 常量固化到 `domain/policy`
2. 再实现 state -> bucket 的判定
3. 再实现 supply-aware 调整逻辑
4. 最后输出 `DemandBundle` 与 planner snapshot

Planner 规则必须按设计文档直接实现：

- bucket 固定为：
  - `hard_review`
  - `new_now`
  - `soft_review`
  - `near_future`
- precedence 固定为：
  - `hard_review > new_now > soft_review > near_future`
- 一个 unit 在一次 planner run 中只能进入一个主 bucket
- `hard_review` 永不因供给弱而消失
- `new_now` 必须 supply-aware
- `hard_review_low_supply = true` 时，提高 `bundle / soft_future` 预算

`fragility`、`instability` 等只允许作为 Recommendation 内部派生信号，必须由以下稳定字段推导：

- `status`
- `next_review_at`
- `mastery_score`
- `last_quality`
- `recent_quality_window`
- `recent_correctness_window`
- `strong_event_count`
- `review_count`

### 涉及接口/类型

- `DemandPlanner`
- `DemandBundle`
- `DemandUnit`
- `LaneBudget`
- `MixQuota`
- `PlannerFlags`
- `Bucket`
- `SessionMode`
- 相关 policy 常量与阈值

### 测试方式

- 单测：
  - bucket precedence
  - overdue / due now 进入 `hard_review`
  - `new_now` 的 supply-aware 筛选
  - `hard_review_low_supply` 动态 budget 调整
  - `soft_review` 与 `near_future` 分桶
  - 空 inventory / 极稀疏库存
- golden：
  - 固定 planner snapshot
  - 固定 lane budget 与 flags

### 验收标准

- `make check`
- planner 单测通过
- planner golden 测试通过
- planner 输出字段足够驱动后续 candidate/ranker/selector

### 完成后如何更新文档

完成后必须：

1. 把 Step 3 状态改为 `accepted`
2. 记录 planner 的关键 policy 是否与原计划有收敛
3. 记录 golden 样例名称和测试结果
4. 更新完文档后才能开始 Step 4

### 完成记录

- 日期：
- 实际完成内容：
- 运行命令：
- 验收结果：
- 与计划偏差：
- 下一步入口：

## Step 4. 实现 Candidate Generator 四条 lane

### 目标

生成健康的 video-unit 候选池，解决 exact target 候选空间不足的问题。

### 实施内容

实施顺序固定为：

1. 先实现 `exact_core`
2. 再实现 `bundle`
3. 再实现 `soft_future`
4. 最后实现 `quality_fallback`
5. 最终把四条 lane 汇总为统一 `VideoUnitCandidate` 集合，并输出 candidate summary

固定规则：

- 唯一权威读模型是 `recommendation.v_recommendable_video_units`
- 每条 lane 都必须有独立 cap
- `bundle` 必须按 `video_id` 聚合多 unit 承载能力
- `hard_review_low_supply = true` 时允许 `bundle` 动态放宽
- `quality_fallback` 只在明显无法满足 `target_video_count` 时触发

四条 lane 的实现要求：

- `exact_core`
  - 输入 `hard_review` 与 top-ranked `new_now`
  - 用 `coverage_ratio`、`mention_count`、`sentence_count`、`mapped_span_ratio` 做精排
- `bundle`
  - 默认至少覆盖两个有价值 unit
  - 至少包含一个 `hard_review` 或 `new_now`
  - 低供给时允许受控放宽
- `soft_future`
  - 输入 `soft_review + near_future`
  - 只能补充，不承担主排序目标
- `quality_fallback`
  - 只能补位
  - 不得压过主线候选

### 涉及接口/类型

- `CandidateGenerator`
- `VideoUnitCandidate`
- `Lane`
- `Bucket`
- `DemandBundle`
- `RecommendableVideoUnitReader`

### 测试方式

- lane 级单测：
  - `exact_core` 命中排序
  - `bundle` 至少两个 unit
  - `bundle` 动态放宽
  - `soft_future` 不越权
  - `quality_fallback` 触发与不触发
- scenario：
  - 正常库存
  - `hard_review` 供给弱
  - `near_future` 很多但主线仍优先
- golden：
  - candidate summary
  - distinct candidate videos

### 验收标准

- `make check`
- candidate 单测通过
- candidate scenario/golden 通过
- 候选池 distinct videos 和 lane summary 可用于后续聚合与审计

### 完成后如何更新文档

完成后必须：

1. 把 Step 4 状态改为 `accepted`
2. 记录四条 lane 的实际落地规则
3. 记录是否存在对 bundle 放宽规则的收敛
4. 更新完文档后才能开始 Step 5

### 完成记录

- 日期：
- 实际完成内容：
- 运行命令：
- 验收结果：
- 与计划偏差：
- 下一步入口：

## Step 5. 实现 Evidence Resolver 与 Video Evidence Aggregator

### 目标

把 video-unit candidate 转成可排序、可解释、可审计的 video-level candidate。

### 实施内容

实施顺序固定为：

1. 先实现 `EvidenceResolver`
2. 再实现 `VideoEvidenceAggregator`
3. 最后确认 video-level candidate 已经拥有 rank/select/explain 所需字段

`EvidenceResolver` 固定要求：

- 只按 candidate videos、candidate units、`evidence_span_refs` 定向回查
- 不全量扫 `video_semantic_spans`
- 不全量扫 `video_transcript_sentences`
- 第一版本地 best 选择规则固定为稳定低语义承诺版本：
  - 优先 `evidence_span_refs` 顺序
  - 同句多 ref 时优先更早 `start_ms`

`VideoEvidenceAggregator` 固定要求：

- 同 `video_id + unit_id` 只保留最强证据
- 可选第二条强证据只做弱增量
- 输出四类覆盖集合与计数：
  - `covered_hard_review_units`
  - `covered_new_now_units`
  - `covered_soft_review_units`
  - `covered_near_future_units`
- 输出：
  - `coverage_strength_score`
  - `bundle_value_score`
  - `educational_fit_score`
  - `future_value_score`
  - `best_evidence_*`

### 涉及接口/类型

- `EvidenceResolver`
- `ResolvedEvidenceWindow`
- `VideoEvidenceAggregator`
- `VideoCandidate`
- `SemanticSpanReader`
- `TranscriptSentenceReader`

### 测试方式

- resolver 单测：
  - `evidence_span_refs` 回查
  - span 缺失
  - sentence 缺失
  - 本地 best 选择
- aggregator 单测：
  - 同 unit 去重
  - bucket 覆盖统计
  - best evidence 选择
  - 长视频重复命中同 unit 时避免高估
- scenario：
  - `evidence_span_refs` 与 span 表不一致

### 验收标准

- `make check`
- resolver 单测通过
- aggregator 单测通过
- scenario 测试通过
- video candidate 输出字段足够驱动 rank/select/explain/audit

### 完成后如何更新文档

完成后必须：

1. 把 Step 5 状态改为 `accepted`
2. 记录 best evidence 选择规则
3. 记录 evidence 容错行为
4. 更新完文档后才能开始 Step 6

### 完成记录

- 日期：
- 实际完成内容：
- 运行命令：
- 验收结果：
- 与计划偏差：
- 下一步入口：

## Step 6. 实现 Ranker / Selector / Explanation Builder

### 目标

完成 Recommendation 的“排序、最终选择、解释生成”主逻辑，产出最终视频列表。

### 实施内容

实施顺序固定为：

1. 先实现 `VideoRanker`
2. 再实现 `VideoSelector`
3. 最后实现 `ExplanationBuilder`

`VideoRanker` 固定按设计文档第一版公式实现：

```text
base_score =
  0.40 * demand_coverage
+ 0.18 * coverage_strength_score
+ 0.15 * bundle_value_score
+ 0.15 * educational_fit_score
+ 0.05 * future_value_score
+ 0.05 * freshness_score
- 0.03 * recent_served_penalty
- 0.02 * overload_penalty
```

其中：

- `demand_coverage` 采用四类 bucket 覆盖权重
- `catalog.video_user_states` 只允许提供轻量 penalty
- `recent_served_penalty` 由 Recommendation 自己的 serving state 提供

`VideoSelector` 固定三种模式：

- `normal`
- `low_supply`
- `extreme_sparse`

固定约束：

- 主线覆盖优先
- future 占比受控
- fallback 占比受控
- `same_dominant_unit_max` 受控
- 宁可 under-fill，也不要乱推

`ExplanationBuilder` 固定要求：

- 输出 `reason_codes`
- 输出模板化 `explanation`
- 不使用 LLM
- 解释必须基于真实覆盖、真实 evidence、真实 penalty 状态

### 涉及接口/类型

- `VideoRanker`
- `VideoSelector`
- `ExplanationBuilder`
- `SelectorMode`
- `ReasonCode`
- `FinalRecommendationItem`
- `VideoUserState`

### 测试方式

- ranker 单测：
  - 公式
  - 覆盖权重
  - `recent_served_penalty`
  - `video_user_states` penalty
  - `overload_penalty`
- selector 单测：
  - 三种 mode
  - 主线覆盖
  - future/fallback 比例
  - dominant unit 重复控制
  - under-fill
- explain 单测：
  - reason code 生成
  - explanation 模板拼接
  - best evidence 文案
- golden：
  - 最终 video 排序
  - `reason_codes`
  - `explanation`

### 验收标准

- `make check`
- ranker/selector/explain 单测通过
- golden 测试通过
- 输出结果结构已经能直接映射到 `GenerateVideoRecommendationsResponse`

### 完成后如何更新文档

完成后必须：

1. 把 Step 6 状态改为 `accepted`
2. 记录 ranker 公式是否有实现层面的轻微收敛
3. 记录 selector 配额实现
4. 记录 explanation reason code 集合
5. 更新完文档后才能开始 Step 7

### 完成记录

- 日期：
- 实际完成内容：
- 运行命令：
- 验收结果：
- 与计划偏差：
- 下一步入口：

## Step 7. 接通 Audit Writer / Serving State Manager / 完整主用例

### 目标

在前面所有只读与纯计算能力稳定之后，接通 Recommendation 的最终短事务写路径，形成完整主用例闭环。

### 实施内容

实施顺序固定为：

1. 先实现 `AuditWriter`
2. 再实现 `ServingStateManager`
3. 最后把二者接入 `GenerateVideoRecommendations` 的完整 orchestrator

最终主用例时序固定为：

1. assemble
2. plan
3. generate candidates
4. resolve evidence
5. aggregate
6. rank
7. select
8. build explanation
9. 开启短事务
10. 写 `video_recommendation_runs`
11. 写 `video_recommendation_items`
12. upsert `user_unit_serving_states`
13. upsert `user_video_serving_states`
14. 提交事务

固定要求：

- 只读计算过程不包长事务
- 只在最终写 recommendation 审计与 serving state 时开启短事务
- 审计写入对象必须是 video recommendation run/item
- serving state 必须使用幂等 upsert

### 涉及接口/类型

- `AuditWriter`
- `ServingStateManager`
- `RecommendationAuditRepository`
- `UnitServingStateRepository`
- `VideoServingStateRepository`
- `GenerateVideoRecommendations` 完整实现

### 测试方式

- 单测：
  - audit snapshot 组装
  - serving state 更新映射
- Recommendation 仓储集成测试：
  - `video_recommendation_runs`
  - `video_recommendation_items`
  - unit/video serving state upsert
  - 短事务一致性
- scenario/golden：
  - 完整主用例输入输出快照

仓储集成测试允许使用 Recommendation 自己的最小外部契约夹具，不要求另一个模块已有业务实现。

### 验收标准

- `make check`
- Recommendation 仓储集成测试通过
- scenario/golden 验收通过
- `GenerateVideoRecommendations` 主用例完整返回结果并写 recommendation 自有状态

### 完成后如何更新文档

完成后必须：

1. 把 Step 7 状态改为 `accepted`
2. 写完整主用例实际写入路径
3. 写仓储集成测试命令与结果
4. 写 scenario/golden 验收结果
5. 完成后进入“收尾要求”阶段

### 完成记录

- 日期：
- 实际完成内容：
- 运行命令：
- 验收结果：
- 与计划偏差：
- 下一步入口：

## 7. 进度记录规则

以下规则是强制规则，不是建议：

1. 每个大步骤开始前，先把对应步骤状态改成 `in_progress`
2. 每个大步骤完成后，必须先更新本文档
3. 更新内容至少包括：
   - 完成记录
   - 执行命令
   - 验收结果
   - 偏差说明
   - 下一步入口
4. 只有当完成记录写完、步骤状态改成 `accepted` 后，才允许开始下一步
5. 如果某一步中途发现方案必须调整，先更新本文档再改实现
6. 不允许出现“代码已经做完，但进度文档还没补”的情况

## 8. 每步完成记录模板

每个步骤完成后，统一按以下模板填写：

```text
日期：
实际完成内容：
运行命令：
验收结果：
与计划偏差：
下一步入口：
```

如果某一步没有偏差，`与计划偏差` 明确写：

```text
无
```

如果某一步验收失败，必须先在该步骤下补充失败原因和调整方案，再继续迭代，不得直接跳到下一步。

## 9. 收尾要求

Recommendation 主链路全部实现完成后，必须额外完成以下收尾工作：

1. 更新 `internal/recommendation/README.md`
   - 写清当前 Recommendation 已实现能力
   - 写清未实现能力
   - 写清测试边界
   - 写清对外依赖契约
2. 如果 Recommendation 的稳定对外契约在实现过程中发生收敛，需要同步更新对应设计映射文档
3. 本文档保留在 `docs/temp/` 作为实施记录，不替代正式 README

## 10. 当前结论

Recommendation 的下一阶段实现必须严格按本文档推进。

当前第一步不是直接写 planner 或 ranker，而是：

> **先补 Recommendation 缺失契约与内部接口，并同步补齐其仓储基础设施与最小集成测试能力。**

在 Step 1 被验收并记录完成之前，不应进入 Context Assembler 或任何推荐算法实现。
