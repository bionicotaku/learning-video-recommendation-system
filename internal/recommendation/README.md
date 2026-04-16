# Recommendation

`recommendation` 负责视频推荐主链路、serving state、Recommendation own 读模型和 video recommendation 审计。

## 当前已实现

- 完整主链路：
  - `Context Assembler`
  - `Demand Planner`
  - `Candidate Generator`
  - `Evidence Resolver`
  - `Video Evidence Aggregator`
  - `Video Ranker`
  - `Video Selector`
  - `Explanation Builder`
- `GenerateVideoRecommendations` 的完整 orchestrator
- Recommendation owner migration
- 物化读视图与 SQL/query/sqlc 基础层
- `video_recommendation_runs` / `video_recommendation_items` 审计写入
- `user_unit_serving_states` / `user_video_serving_states` 短事务写入
- 单测、scenario/golden 测试、Recommendation integration 测试

## 当前边界

- 只读 `learning.user_unit_states`
- 只读 Catalog 稳定内容契约与 `catalog.video_user_states`
- 只写 `recommendation.*`
- 不回写 Learning engine 或 Catalog owner 对象
- 不包含跨模块 e2e、HTTP API、worker、自动刷新任务

## 目录职责

- `application/dto`
  - 对外请求/响应结构
- `application/repository`
  - Recommendation 主链路依赖的读写 port
- `application/service`
  - 依赖 repository 的实现：
    - `DefaultContextAssembler`
    - `DefaultCandidateGenerator`
    - `DefaultEvidenceResolver`
    - `DefaultAuditWriter`
    - `DefaultServingStateManager`
    - `DefaultRecommendationResultWriter`
- `application/usecase`
  - `GenerateVideoRecommendationsService`
  - 兼容保留 assembler-only shell constructor
  - 完整主链路通过 pipeline constructor 启用
- `domain/planner`
  - 需求分桶、lane budget、mix quota
- `domain/aggregator`
  - video-level 聚合与覆盖特征
- `domain/ranking`
  - 第一版规则排序与 penalty
- `domain/selector`
  - `normal / low_supply / extreme_sparse` 选择约束
- `domain/explain`
  - 模板化 `reason_codes + explanation`
- `infrastructure/persistence`
  - query/sqlc/repository/tx
- `test`
  - Recommendation 自己的 unit / golden / integration 测试

## 当前未实现

- 跨模块 e2e
- 自动刷新任务
- embedding / vector recall
- ML ranking
- task layer / 非视频 fallback

## 维护约束

- Recommendation 的审计中心始终是 video recommendation run/item。
- 只读计算不包长事务；只在最终写 audit 和 serving state 时开启短事务。
- `catalog.video_user_states` 只作为轻量 penalty 输入，不承载 Recommendation own 的投放状态。
