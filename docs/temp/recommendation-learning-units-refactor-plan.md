# Recommendation 返回 expected learning units 重构方案

日期：2026-05-08

## 1. 背景

当前 Recommendation 主接口的业务语义还是“输入一个 `user_id`，返回一组视频”。代码里虽然已经有 `CoveredUnits`、`CoveredHardReviewUnits`、`CoveredNewNowUnits`、`CoveredSoftReviewUnits`、`CoveredNearFutureUnits` 等字段，但这些字段更接近内部解释和覆盖统计，不适合作为前端学习形态的稳定契约。

前端现在已经可以确定的视频学习形态是：

- feed 给用户分发学习视频；
- 视频字幕 token 可点击；
- 点击 token 后展示上下文释义、字典义；
- 之后可能接入小测、主动反馈、学习模式、即时练习等动作。

这意味着 Recommendation 的最终输出不应只是 video list，而应是 video learning plan：

```text
user_id -> [
  {
    video_id,
    rank,
    score,
    reason_codes,
    learning_units: [
      { coarse_unit_id, role, is_primary, evidence }
    ]
  }
]
```

其中 `learning_units` 是“本轮推荐这个视频时，系统预期用户在该视频里学习或复习哪些 unit”的权威列表。前端基于这份列表决定哪些字幕 token 需要强化展示、哪些词进结尾小测、哪些词进入学习模式的即时练习。

## 2. 当前实现分析

### 2.1 当前接口问题

`internal/recommendation/application/dto/generate_video_recommendations.go` 当前响应结构中，`RecommendationVideo` 暴露：

- `VideoID`
- `Rank`
- `Score`
- `ReasonCodes`
- `CoveredUnits`
- `CoveredHardReviewUnits`
- `CoveredNewNowUnits`
- `CoveredSoftReviewUnits`
- `CoveredNearFutureUnits`
- `BestEvidence`
- `Explanation`

问题是：

1. `Covered*` 字段表达的是“该视频覆盖了哪些内部需求桶”，不是“前端这次应该围绕哪些 unit 组织学习体验”。
2. 前端无法直接知道某个 unit 在本视频中的学习角色，例如 hard review、new now、soft review、near future。
3. 前端无法区分 primary unit 和 incidental unit。
4. evidence 只有 video 级 `BestEvidence`，不是 unit 级 evidence。前端要做字幕高亮、小测定位、弹窗上下文时，真正需要的是 unit 级 evidence。
5. 审计表只存 covered count 和 dominant unit，缺少完整的 learning unit plan，后续无法回放“当时给前端预期学什么”。

### 2.2 当前最佳改造点

当前主链路里最适合生成 `learning_units` 的位置是 `Video Evidence Aggregator`：

```text
Demand Planner
  -> Candidate Generator
  -> Evidence Resolver
  -> Video Evidence Aggregator
  -> Video Ranker
  -> Video Selector
  -> Explanation Builder
  -> Audit Writer / Serving State Manager
```

原因：

- Aggregator 已经按 `video_id` 聚合所有 `VideoUnitCandidate`。
- Aggregator 当前已经能拿到每个 unit 的 `Bucket`，可直接映射为 learning role。
- Aggregator 当前已经选择每个 unit 的 best candidate 和 best evidence。
- Ranker 和 Selector 应继续消费视频级特征，不应负责构造前端学习契约。
- Explanation Builder 只应生成解释，不应重新推导 learning units。

### 2.3 当前 schema 问题

`recommendation.video_recommendation_items` 当前 item 级审计字段包含：

- `dominant_unit_id`
- `covered_hard_review_count`
- `covered_new_now_count`
- `covered_soft_review_count`
- `covered_near_future_count`
- `best_evidence_sentence_index`
- `best_evidence_span_index`
- `best_evidence_start_ms`
- `best_evidence_end_ms`

这些字段可以回答“这个 item 大概覆盖多少内容”，但不能回答：

- 这个视频当时预期用户学习哪些 unit；
- 每个 unit 属于什么 role；
- 哪些 unit 是 primary；
- 每个 unit 对应哪个 evidence；
- 前端最终收到的学习计划是什么。

MVP 阶段既然允许破坏性变更，就不应继续保留 covered count 作为双写字段。count 可以从 `learning_units` 派生，避免 schema 同时保存事实和派生统计导致不一致。

## 3. 重构目标

### 3.1 业务目标

Recommendation 的主接口改为返回 video learning plan。每个推荐视频必须带本轮预期学习 unit 列表，让前端可以组织：

- 字幕 token 高亮；
- token lookup 弹窗；
- lookup 后主动反馈；
- 结尾小测；
- 学习模式下的即时练习；
- 后续 learning event 上报。

### 3.2 模块边界

- Recommendation 只读 `learning.*`，只写 `recommendation.*`。
- Recommendation 不写 `learning.unit_learning_events`，不直接改变学习状态。
- Learning engine 继续负责学习事件和学习状态归约。
- Catalog 继续负责视频、字幕、semantic spans、unit index 等内容事实。
- 前端学习反馈事件最终仍应通过 Learning engine 的 `RecordLearningEvents` 进入学习域。

### 3.3 成功标准

重构完成后必须满足：

1. `GenerateVideoRecommendations` 响应不再暴露 `Covered*` 字段。
2. 每个推荐视频都有 `learning_units`，除非该视频是明确的纯 fallback；fallback 也必须在 reason 或 role 上可解释。
3. 每个 `learning_unit` 至少包含 `coarse_unit_id`、`role`、`is_primary`。
4. 如果 Catalog evidence 可解析，则每个 `learning_unit` 尽量带 unit 级 `evidence`。
5. `recommendation.video_recommendation_items` 保存完整 `learning_units jsonb`，用于回放当次推荐给前端的学习计划。
6. `DefaultServingStateManager` 从 `learning_units` 更新 unit serving state，不再从 `Covered*` 更新。
7. `ExplanationBuilder` 只消费已有 `learning_units` 生成解释，不再生成或合并 covered units。
8. `make quick-check` 通过。
9. 涉及 schema 和 repository 后，`make recommendation-test-integration` 通过。
10. 最终验收 `make check` 通过。

## 4. 优先修改的设计文档

这次应先改设计文档，再改 schema 和代码。建议按下面顺序处理。

### 4.1 `docs/推荐模块设计.md`

这是本次最核心的权威设计文档，必须优先更新。需要修改：

1. 模块目标  
   从“输入用户，输出视频列表”改为“输入用户，输出视频学习计划列表”。

2. 关键原则  
   当前文档里“最终输出是视频列表，不是任务列表/learning cards”的表述需要改成：最终输出仍以视频为主，但每个视频必须携带本轮 expected learning units；Recommendation 不生成学习任务 UI，但生成前端学习体验所需的 unit plan。

3. Domain objects  
   `VideoCandidate`、`FinalRecommendationItem`、DTO 的字段清单要从 `Covered*` 改成 `LearningUnits`。

4. Aggregator 职责  
   明确 Aggregator 是 `VideoUnitCandidate -> video-level candidate + expected learning units` 的构造点。

5. Ranker / Selector 职责  
   明确它们只消费视频级特征和 `LearningUnits` 派生统计，不重新构造 learning plan。

6. Explanation Builder 职责  
   明确它只基于 `LearningUnits`、reason codes、score 生成文案。

7. Audit Writer 字段  
   把 `covered_*_count` 改成 `learning_units jsonb`；`dominant_unit_id` 可保留为索引和排查便利字段，但不能替代完整 plan。

8. 对外接口  
   把 `CoveredUnits`、`CoveredHardReviewUnits` 等响应字段改成 `LearningUnits`。

### 4.2 `docs/视频推荐系统总设计.md`

这是跨模块总设计，需要同步改全局语义：

1. Recommendation 输出从 video list 改成 video learning plan list。
2. 系统链路中增加“前端基于 expected learning units 组织学习反馈”的描述。
3. `video_recommendation_items` 的说明改成保存当次推荐 item 的 `learning_units`。
4. 明确 Learning engine 依然通过事件输入学习反馈，Recommendation 不直接写学习状态。

### 4.3 `internal/recommendation/README.md`

这是代码目录维护文档，需要在代码重构完成后同步更新：

1. 当前已实现列表增加 expected learning units contract。
2. `domain/aggregator` 描述改成“video-level 聚合与 expected learning units 构造”。
3. `domain/explain` 描述改成“基于 learning units 的 reason/explanation”。
4. 维护约束增加：`LearningUnits` 是对外学习计划的唯一事实来源，`Covered*` 不再作为跨层契约。

### 4.4 `internal/recommendation/infrastructure/migration/README.md`

schema 清理后更新：

1. 说明 audit item 保存 `learning_units jsonb`。
2. 说明 covered counts 不再是持久化字段。
3. 保持只定义 recommendation owner 对象。

### 4.5 `docs/当前实现现状.md`

代码和 schema 完成后再更新，不能提前写成已完成。需要同步：

1. Recommendation 当前主接口响应结构。
2. 当前 migration 基线。
3. 当前测试覆盖与验证命令。
4. 如果 live DB 已被重建，更新 live DB 状态说明。

### 4.6 `docs/当前数据库Schema现状.md`

如果本轮实际连接 live DB 并重建 schema，必须在重建和验证后更新。内容应以真实 DB introspection 为准，不从 migration 手抄。

### 4.7 不建议修改的文档

`docs/学习引擎设计.md` 本轮原则上不需要大改。原因是 Learning engine 的业务边界没有变化：它仍消费 learning events，维护学习状态。最多只在学习事件示例中补一句：`recommendation.learning_units` 可作为前端上报 `source_ref_id` / metadata 的来源之一。

`docs/Catalog-数据库设计.md` 本轮也不应改。Catalog evidence refs 和 semantic spans contract 没变。

## 5. 目标接口设计

### 5.1 DTO

建议最终 DTO 结构为：

```go
type GenerateVideoRecommendationsResponse struct {
    UserID        string               `json:"user_id"`
    RunID         string               `json:"run_id"`
    SelectorMode  string               `json:"selector_mode"`
    Underfilled   bool                 `json:"underfilled"`
    Videos        []RecommendationVideo `json:"videos"`
}

type RecommendationVideo struct {
    VideoID       string                 `json:"video_id"`
    Rank          int                    `json:"rank"`
    Score         float64                `json:"score"`
    ReasonCodes   []string               `json:"reason_codes"`
    LearningUnits []ExpectedLearningUnit `json:"learning_units"`
    Explanation   string                 `json:"explanation"`
}

type ExpectedLearningUnit struct {
    CoarseUnitID int64                 `json:"coarse_unit_id"`
    Role         string                `json:"role"`
    IsPrimary    bool                  `json:"is_primary"`
    Evidence     *LearningUnitEvidence `json:"evidence,omitempty"`
}

type LearningUnitEvidence struct {
    SentenceIndex *int32 `json:"sentence_index,omitempty"`
    SpanIndex     *int32 `json:"span_index,omitempty"`
    StartMs       *int32 `json:"start_ms,omitempty"`
    EndMs         *int32 `json:"end_ms,omitempty"`
}
```

命名说明：

- `LearningUnits` 比 `ExpectedUnits` 更适合直接给前端消费。
- `ExpectedLearningUnit` 保留 expected 语义，避免误解为用户已经学会。
- `Role` 使用字符串，MVP 阶段简单稳定，后续如需 enum 再收敛。
- `Evidence` 是 unit 级，而不是 video 级。

### 5.2 role 取值

MVP 建议固定四类：

```text
hard_review
new_now
soft_review
near_future
```

映射规则：

| 当前 bucket | 新 role | 语义 |
| --- | --- | --- |
| `hard_review` | `hard_review` | 当前最需要复习 |
| `new_now` | `new_now` | 当前适合新学 |
| `soft_review` | `soft_review` | 顺带轻复习 |
| `near_future` | `near_future` | 近期可预热 |

不建议在 MVP 加 `incidental`、`bonus`、`unknown`。如果一个视频完全没有 learning unit，就应通过 fallback reason 解释，而不是塞一个模糊 role。

### 5.3 primary 规则

`is_primary=true` 表示这个 unit 是推荐该视频的主要学习目标。MVP 推荐规则：

1. `hard_review` 和 `new_now` 默认是 primary。
2. `soft_review` 和 `near_future` 默认不是 primary。
3. 如果视频中没有 `hard_review` / `new_now`，则取排序最高的 1 到 2 个 `soft_review` 或 `near_future` 作为 primary，避免前端拿到空主目标。

这个规则应在 Aggregator 中完成。前端不应自己推断 primary。

### 5.4 evidence 规则

每个 `ExpectedLearningUnit` 的 evidence 来自当前 Aggregator 为该 unit 选择的 best candidate：

- `sentence_index`
- `span_index`
- `start_ms`
- `end_ms`

如果 `evidence_span_refs` 无法解析，`evidence` 可以为空，但该 unit 不应丢失。原因是学习计划来自推荐需求和内容索引，evidence 只是定位辅助。

## 6. 目标领域模型

### 6.1 新增或调整领域对象

建议在 `internal/recommendation/domain/model` 下新增或改造：

```go
type LearningRole string

const (
    LearningRoleHardReview LearningRole = "hard_review"
    LearningRoleNewNow     LearningRole = "new_now"
    LearningRoleSoftReview LearningRole = "soft_review"
    LearningRoleNearFuture LearningRole = "near_future"
)

type LearningUnitEvidence struct {
    SentenceIndex *int32
    SpanIndex     *int32
    StartMs       *int32
    EndMs         *int32
}

type ExpectedLearningUnit struct {
    CoarseUnitID int64
    Role         LearningRole
    IsPrimary    bool
    Evidence     *LearningUnitEvidence
}
```

### 6.2 `VideoCandidate`

`VideoCandidate` 应以 `LearningUnits` 作为 canonical unit plan：

```go
type VideoCandidate struct {
    VideoID       string
    LearningUnits []ExpectedLearningUnit

    DominantRole   LearningRole
    DominantUnitID int64

    HardReviewCover float64
    NewNowCover     float64
    SoftReviewCover float64
    NearFutureCover float64

    LaneSources []string
    BaseScore   float64
    Score       float64
    ReasonCodes []string
}
```

不再保留：

- `CoveredUnits`
- `CoveredHardReviewUnits`
- `CoveredNewNowUnits`
- `CoveredSoftReviewUnits`
- `CoveredNearFutureUnits`

如果 Ranker / Selector 需要按 role 计数，增加小型 helper，从 `LearningUnits` 派生。

### 6.3 `FinalRecommendationItem`

最终 item 也只保留：

```go
type FinalRecommendationItem struct {
    VideoID       string
    Rank          int
    Score         float64
    ReasonCodes   []string
    LearningUnits []ExpectedLearningUnit
    Explanation   string
}
```

如果 audit 仍需要 `PrimaryLane`、`DominantBucket` 之类排查字段，可以保留在内部 audit model，但不进入前端 DTO。

### 6.4 `RecommendationItem` audit model

audit model 应保存完整 learning plan：

```go
type RecommendationItem struct {
    RunID         string
    Rank          int
    VideoID       string
    Score         float64
    PrimaryLane   string
    DominantRole  string
    DominantUnitID *int64
    ReasonCodes   []string
    LearningUnits []ExpectedLearningUnit
    CreatedAt     time.Time
}
```

`DominantUnitID` 和 `DominantRole` 只用于排查、索引和粗粒度统计，不是前端学习契约。

## 7. schema 和 migration 清理方案

### 7.1 原则

因为当前是 MVP，且明确不需要向后兼容，本轮不新增兼容迁移，不做双写，不保留旧 covered count 字段。

推荐做法：

1. 先回滚并删除当前 `recommendation` schema。
2. 修改现有 Recommendation migration baseline。
3. 重新 migrate up。
4. refresh materialized views。
5. 跑 integration 和 E2E 验证。

### 7.2 清理顺序

先确认目标数据库，避免误删：

```bash
make recommendation-migrate-status
```

如果确认当前 DB 可以破坏性重建，按已应用版本数一次性回滚到 0。当前 Recommendation 有 5 个 migration，因此命令是：

```bash
go run ./cmd/dbtool migrate down --module=recommendation --steps=5
```

注意：`make recommendation-migrate-down` 默认只回滚 1 个 migration，不足以删除整个 schema。真正删除 schema 的文件是：

```text
internal/recommendation/infrastructure/migration/000001_create_recommendation_schema.down.sql
```

其内容是：

```sql
drop schema if exists recommendation cascade;
```

如果未来 migration 数量变化，应以 `make recommendation-migrate-status` 输出的 applied count 为准，而不是写死 5。

### 7.3 migration baseline 修改

重点修改：

```text
internal/recommendation/infrastructure/migration/000003_create_recommendation_audit_tables.up.sql
internal/recommendation/infrastructure/migration/000003_create_recommendation_audit_tables.down.sql
```

`video_recommendation_items` 建议改为：

```sql
create table if not exists recommendation.video_recommendation_items (
  run_id uuid not null references recommendation.video_recommendation_runs(run_id) on delete cascade,
  rank integer not null check (rank > 0),
  video_id text not null,
  score numeric(10, 4) not null,
  primary_lane text not null,
  dominant_role text,
  dominant_unit_id bigint references learning.coarse_units(coarse_unit_id),
  reason_codes text[] not null default '{}',
  learning_units jsonb not null default '[]'::jsonb,
  created_at timestamptz not null default now(),
  primary key (run_id, rank),
  check (jsonb_typeof(learning_units) = 'array')
);
```

删除：

- `dominant_bucket`
- `covered_hard_review_count`
- `covered_new_now_count`
- `covered_soft_review_count`
- `covered_near_future_count`
- `best_evidence_sentence_index`
- `best_evidence_span_index`
- `best_evidence_start_ms`
- `best_evidence_end_ms`

原因：

- `dominant_bucket` 改名为 `dominant_role`，和对外 `role` 对齐。
- covered counts 可从 `learning_units` 派生，不应持久化双写。
- video 级 best evidence 不再是核心契约，unit 级 evidence 已在 `learning_units` 内保存。

可选索引：

```sql
create index if not exists idx_video_recommendation_items_learning_units_gin
  on recommendation.video_recommendation_items
  using gin (learning_units);
```

MVP 可先不加 GIN，除非已经有按 JSONB 查询 audit 的明确需求。更推荐只保留：

```sql
create index if not exists idx_video_recommendation_items_video_id
  on recommendation.video_recommendation_items(video_id);

create index if not exists idx_video_recommendation_items_dominant_unit
  on recommendation.video_recommendation_items(dominant_unit_id)
  where dominant_unit_id is not null;
```

### 7.4 query/sqlc 更新

需要更新：

```text
internal/recommendation/infrastructure/persistence/query/recommendation_writes.sql
internal/recommendation/infrastructure/persistence/sqlc.yaml
internal/recommendation/infrastructure/persistence/sqlcgen/*
```

预期动作：

1. `InsertVideoRecommendationItem` 参数删除 covered count 和 best evidence 字段。
2. 新增 `learning_units` JSONB 参数。
3. sqlc 生成后，Go 参数类型大概率是 `[]byte` 或 `pgtype`，具体以当前 sqlc 生成结果为准。
4. mapper 层负责把 `[]ExpectedLearningUnit` marshal 为 JSONB。

生成命令：

```bash
make sqlc-generate
```

### 7.5 重新创建 schema

修改 migration 和 sqlc query 后：

```bash
make recommendation-migrate-up
make recommendation-refresh
make recommendation-migrate-status
```

如果本轮是在 embedded integration DB 中验证，则由 integration harness 应用 migration，不一定要手动操作 live DB。但如果用户明确要求重建当前 `.env` 指向的 DB，就必须执行上面的 live DB 命令，并更新 `docs/当前数据库Schema现状.md`。

## 8. 代码重构步骤

### Step 1：更新权威设计文档

先改：

```text
docs/推荐模块设计.md
docs/视频推荐系统总设计.md
```

代码完成后再改：

```text
internal/recommendation/README.md
internal/recommendation/infrastructure/migration/README.md
docs/当前实现现状.md
docs/当前数据库Schema现状.md
```

验收：

- 文档中不再把 Recommendation 主输出描述成纯 video list。
- 文档中 `Covered*` 不再作为对外契约出现。
- 文档中明确 `learning_units` 是前端学习计划来源。

### Step 2：调整 domain model

修改：

```text
internal/recommendation/domain/model/candidate_models.go
internal/recommendation/domain/model/final_recommendation_item.go
internal/recommendation/domain/model/recommendation_audit.go
internal/recommendation/domain/model/evidence_models.go
```

动作：

1. 新增 `LearningRole`、`ExpectedLearningUnit`、`LearningUnitEvidence`。
2. `VideoCandidate` 删除 `Covered*` 字段，新增 `LearningUnits`。
3. `FinalRecommendationItem` 删除 `Covered*` 字段，新增 `LearningUnits`。
4. `RecommendationItem` 删除 covered count 和 best evidence 扁平字段，新增 `LearningUnits`。
5. 增加 helper：
   - `LearningUnitIDs(units []ExpectedLearningUnit) []int64`
   - `LearningUnitIDsByRole(units []ExpectedLearningUnit, role LearningRole) []int64`
   - `CountLearningUnitsByRole(units []ExpectedLearningUnit, role LearningRole) int`
   - `DominantLearningUnit(units []ExpectedLearningUnit) (role, unitID, ok)`

验收：

```bash
go test ./internal/recommendation/domain/model/...
```

如果没有单独 model 测试包，可先跑：

```bash
go test ./internal/recommendation/test/unit/domain/...
```

### Step 3：重构 Aggregator

修改：

```text
internal/recommendation/domain/aggregator/default_video_evidence_aggregator.go
internal/recommendation/domain/aggregator/video_evidence_aggregator.go
internal/recommendation/test/unit/domain/aggregator/video_evidence_aggregator_test.go
```

动作：

1. 在按 video 聚合时，为每个 unit 构造 `ExpectedLearningUnit`。
2. 从 candidate bucket 映射 role。
3. 从当前 best evidence 构造 unit 级 `LearningUnitEvidence`。
4. 统一决定 `is_primary`。
5. 保留现有 cover ratio / score 计算，但 unit list 改从 `LearningUnits` 派生。
6. 删除 `coveredHard`、`coveredNew`、`coveredSoft`、`coveredFuture` 等数组字段的向外传递。

验收：

```bash
go test ./internal/recommendation/test/unit/domain/aggregator -run VideoEvidenceAggregator
```

### Step 4：重构 Ranker / Selector

修改：

```text
internal/recommendation/domain/ranking/default_video_ranker.go
internal/recommendation/domain/selector/default_video_selector.go
internal/recommendation/test/unit/domain/ranking/video_ranker_test.go
internal/recommendation/test/unit/domain/selector/video_selector_test.go
```

动作：

1. 所有按 covered arrays 的判断改为按 `LearningUnits` helper 派生。
2. Selector 的 dominant/core/fallback/future-like 判断改为基于 `DominantRole` 或 `LearningUnits`。
3. 不在 Ranker / Selector 中修改 `LearningUnits` 内容，除非只是在排序后保留原对象。

验收：

```bash
go test ./internal/recommendation/test/unit/domain/ranking -run VideoRanker
go test ./internal/recommendation/test/unit/domain/selector -run VideoSelector
```

### Step 5：重构 Explanation Builder

修改：

```text
internal/recommendation/domain/explain/default_explanation_builder.go
internal/recommendation/domain/explain/explanation_builder.go
internal/recommendation/test/unit/domain/explain/explanation_builder_test.go
```

动作：

1. `Build` 输出 `FinalRecommendationItem.LearningUnits`。
2. 删除 `mergeCoveredUnits`。
3. explanation 文案从 `LearningUnits` 的 role/count 派生。
4. reason code 可以继续保留现有规则，但名称里如有 covered 语义应改名。

验收：

```bash
go test ./internal/recommendation/test/unit/domain/explain -run ExplanationBuilder
```

### Step 6：重构 application DTO 和 usecase mapper

修改：

```text
internal/recommendation/application/dto/generate_video_recommendations.go
internal/recommendation/application/usecase/generate_video_recommendations_impl.go
internal/recommendation/test/unit/application/usecase/generate_video_recommendations_test.go
internal/recommendation/test/unit/application/usecase/generate_video_recommendations_pipeline_test.go
internal/recommendation/test/golden/usecase_pipeline_response.json
```

动作：

1. DTO 删除 `Covered*`。
2. DTO 新增 `LearningUnits` 和 unit 级 evidence。
3. usecase mapper 从 `FinalRecommendationItem.LearningUnits` 映射到 DTO。
4. golden 文件改成新的 JSON shape。
5. 如现有测试直接断言 `Covered*`，全部切换为 `learning_units`。

验收：

```bash
go test ./internal/recommendation/test/unit/application/usecase -run GenerateVideoRecommendations
```

### Step 7：重构 Audit Writer 和 persistence

修改：

```text
internal/recommendation/application/service/default_audit_writer.go
internal/recommendation/application/repository/recommendation_audit_repository.go
internal/recommendation/infrastructure/persistence/repository/recommendation_audit_repository.go
internal/recommendation/infrastructure/persistence/mapper/models.go
internal/recommendation/infrastructure/persistence/mapper/pgtypes.go
internal/recommendation/infrastructure/persistence/query/recommendation_writes.sql
internal/recommendation/infrastructure/persistence/sqlcgen/*
internal/recommendation/test/integration/repository_integration_test.go
```

动作：

1. `DefaultAuditWriter` 写入 `LearningUnits`。
2. repository interface 参数改成新的 audit item。
3. persistence mapper 将 `[]ExpectedLearningUnit` marshal 为 JSONB。
4. integration test 验证 DB 中 `learning_units` JSONB 内容。
5. 删除 covered count 和 best evidence 扁平字段相关测试。

验收：

```bash
make sqlc-generate
make recommendation-test-integration
```

### Step 8：重构 Serving State Manager

修改：

```text
internal/recommendation/application/service/default_serving_state_manager.go
internal/recommendation/test/unit/application/service/*
internal/test/e2e/*
```

动作：

1. `ApplySelection` 从 `FinalRecommendationItem.LearningUnits` 收集 unit IDs。
2. 去重逻辑保留。
3. video serving state 逻辑不变。
4. E2E 断言从 covered units 切到 learning units。

验收：

```bash
go test ./internal/recommendation/test/unit/application/service/...
make e2e-test
```

### Step 9：删除旧字段和旧 helper

全仓搜索：

```bash
rg "CoveredUnits|CoveredHardReviewUnits|CoveredNewNowUnits|CoveredSoftReviewUnits|CoveredNearFutureUnits|covered_.*_count|dominant_bucket|BestEvidence" internal docs
```

处理规则：

- DTO、domain、audit schema 中不应再有 `Covered*`。
- `best_evidence` 如果只作为 video 级对外字段，应删除。
- 如果 `BestEvidence` 在内部仍用于 evidence 解析，可改名为 unit evidence 或限制在局部变量，不要进入对外 response。
- 设计文档中可以保留历史描述，但必须标注为已删除或旧实现，不应出现在当前权威路径。

验收：

```bash
rg "CoveredUnits|CoveredHardReviewUnits|CoveredNewNowUnits|CoveredSoftReviewUnits|CoveredNearFutureUnits" internal/recommendation
```

该命令应没有结果，除非是在迁移说明或测试 fixture 的旧历史文本中。

### Step 10：重建 DB 并验证

如果本轮需要操作当前 `.env` 指向的 DB：

```bash
make recommendation-migrate-status
go run ./cmd/dbtool migrate down --module=recommendation --steps=5
make recommendation-migrate-up
make recommendation-refresh
make recommendation-migrate-status
```

之后用 introspection 更新：

```text
docs/当前数据库Schema现状.md
```

如果只在 embedded DB 验证，不更新 live DB 文档。

### Step 11：最终回归

代码重构完成后按顺序跑：

```bash
make sqlc-generate
go test ./internal/recommendation/...
make recommendation-test-integration
make e2e-test
make quick-check
make check
```

如果改动涉及 graphify 需要更新：

```bash
graphify update .
```

## 9. 前端事件契约建议

Recommendation 本轮只负责输出 `learning_units`。前端学习反馈仍建议通过 Learning engine 事件表达。

### 9.1 从 recommendation 到前端

前端拿到：

```json
{
  "video_id": "video-1",
  "learning_units": [
    {
      "coarse_unit_id": 101,
      "role": "new_now",
      "is_primary": true,
      "evidence": {
        "sentence_index": 12,
        "span_index": 3,
        "start_ms": 35120,
        "end_ms": 36340
      }
    }
  ]
}
```

前端可以据此：

- 标记该 video 的预期学习词；
- 在字幕中高亮相关 token；
- lookup 弹窗中突出 expected unit；
- 结尾小测优先覆盖 primary unit；
- 学习模式下对 primary unit 触发即时练习。

### 9.2 从前端到 Learning engine

建议事件仍分弱信号和强信号：

| 前端动作 | Learning event type | 说明 |
| --- | --- | --- |
| 预期 unit 对应字幕自然播放到 | `exposure` | 弱信号，谨慎上报，可聚合后上报 |
| 点击字幕 token 查义 | `lookup` | 弱信号，但价值高于纯 exposure |
| 弹窗选择认识 | `review` 或 `quiz` + quality | 强信号，取决于是否有题目 |
| 弹窗选择模糊 | `review` + 中低 quality | 强信号 |
| 弹窗选择不认识 | `review` + 低 quality，或 `new_learn` | 强信号 |
| 结尾小测答题 | `quiz` | 强信号 |
| 学习模式即时练习 | `quiz` 或 `review` | 强信号 |

本轮 Recommendation 重构不直接改 Learning engine schema，但 `learning_units` 会让前端能正确构造 `coarse_unit_id`、`video_id`、`source_type`、`source_ref_id`、`metadata`。

## 10. 关键取舍

### 10.1 为什么不用 `Covered*`

`Covered*` 的问题不是字段名不好，而是抽象层级不对。它表达推荐算法内部“覆盖了哪些桶”，但前端需要的是“本轮学习计划”。两者有交集，但不是同一个契约。

### 10.2 为什么用 JSONB 保存 `learning_units`

MVP 阶段 `learning_units` 是 recommendation item 的审计快照，不是高频 join 事实表。JSONB 的好处是：

- schema 简单；
- 能完整保存前端当时收到的契约；
- role / evidence 扩字段成本低；
- 不需要额外 item-unit 子表和复杂 migration。

暂不建议新增：

```text
recommendation.video_recommendation_item_units
```

除非后续明确需要对 audit 中的 unit 做大量 SQL 统计、筛选、报表。

### 10.3 为什么删除 covered counts

covered counts 是 `learning_units` 的派生值。持久化它会带来两类问题：

- 写入时可能和 `learning_units` 不一致；
- 后续 role 规则变化时历史 count 解释困难。

MVP 先保留单一事实来源，统计需要时从 JSONB 派生或离线处理。

### 10.4 为什么 Aggregator 负责 primary

primary 是推荐算法对“这个视频主要用来学什么”的判断，不是前端展示层判断。Aggregator 同时拥有视频内候选、unit role、best evidence 和强弱排序信息，因此它是最合适的位置。

### 10.5 为什么不让 Learning engine 返回 expected units

Learning engine 只知道用户学习状态，不知道本次要推哪个视频，也不知道某个视频承载哪些 unit 最合适。因此 expected units 必须由 Recommendation 在 video selection 时生成。

## 11. 预期文件改动清单

设计文档：

```text
docs/推荐模块设计.md
docs/视频推荐系统总设计.md
docs/当前实现现状.md
docs/当前数据库Schema现状.md
internal/recommendation/README.md
internal/recommendation/infrastructure/migration/README.md
```

schema / persistence：

```text
internal/recommendation/infrastructure/migration/000003_create_recommendation_audit_tables.up.sql
internal/recommendation/infrastructure/migration/000003_create_recommendation_audit_tables.down.sql
internal/recommendation/infrastructure/persistence/query/recommendation_writes.sql
internal/recommendation/infrastructure/persistence/sqlcgen/*
internal/recommendation/infrastructure/persistence/mapper/models.go
internal/recommendation/infrastructure/persistence/mapper/pgtypes.go
internal/recommendation/infrastructure/persistence/repository/recommendation_audit_repository.go
```

domain：

```text
internal/recommendation/domain/model/*
internal/recommendation/domain/aggregator/*
internal/recommendation/domain/ranking/*
internal/recommendation/domain/selector/*
internal/recommendation/domain/explain/*
```

application：

```text
internal/recommendation/application/dto/generate_video_recommendations.go
internal/recommendation/application/usecase/generate_video_recommendations_impl.go
internal/recommendation/application/service/default_audit_writer.go
internal/recommendation/application/service/default_serving_state_manager.go
internal/recommendation/application/repository/recommendation_audit_repository.go
```

tests：

```text
internal/recommendation/test/unit/domain/aggregator/video_evidence_aggregator_test.go
internal/recommendation/test/unit/domain/ranking/video_ranker_test.go
internal/recommendation/test/unit/domain/selector/video_selector_test.go
internal/recommendation/test/unit/domain/explain/explanation_builder_test.go
internal/recommendation/test/unit/application/usecase/*
internal/recommendation/test/unit/application/service/*
internal/recommendation/test/integration/*
internal/recommendation/test/golden/usecase_pipeline_response.json
internal/test/e2e/*
```

## 12. 推荐执行顺序

最终实施建议严格按这个顺序：

1. 更新 `docs/推荐模块设计.md` 和 `docs/视频推荐系统总设计.md`。
2. 修改 Recommendation audit migration baseline。
3. 修改 persistence query 并运行 `make sqlc-generate`。
4. 修改 domain model，建立 `LearningUnits` canonical model。
5. 重构 Aggregator 输出 `LearningUnits`。
6. 重构 Ranker / Selector 使用 helper 派生统计。
7. 重构 Explanation Builder 透传并解释 `LearningUnits`。
8. 重构 DTO 和 usecase mapper。
9. 重构 Audit Writer 保存 `learning_units jsonb`。
10. 重构 Serving State Manager 从 `LearningUnits` 更新 unit serving state。
11. 更新 unit / golden / integration / E2E 测试。
12. 删除旧 `Covered*` 和 video-level `BestEvidence` 对外痕迹。
13. 重建 recommendation schema。
14. 更新 `internal/recommendation/README.md`、migration README、当前实现现状、当前 DB schema 现状。
15. 运行 `make quick-check`、`make recommendation-test-integration`、`make e2e-test`、`make check`。
16. 如果改了代码，运行 `graphify update .`。

## 13. 最小 MVP 范围

本轮不要做：

- HTTP API；
- 前端 UI；
- Learning engine event schema 重构；
- audit item-unit 子表；
- ML ranker；
- vector recall；
- 个性化练习题生成；
- 复杂 exposure 自动上报策略。

本轮只做一件事：

```text
把 Recommendation 的最终输出和审计事实，从 video list / covered units，重构为 video learning plan / learning_units。
```

这个范围足够支撑前端后续的字幕 lookup、小测、学习模式和学习反馈闭环，同时不会把 MVP 复杂度扩散到 Learning engine 或 Catalog。
