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
- 基于真实 Learning engine + Catalog fixture + Recommendation 主链路的跨模块 E2E

## 当前边界

- 只读 `learning.user_unit_states`
- 只读 Catalog 稳定内容契约与 `catalog.video_user_states`
- 只写 `recommendation.*`
- 不回写 Learning engine 或 Catalog owner 对象
- 不包含 HTTP API、worker、自动刷新任务

补充说明：

- 跨模块 E2E 不属于 Recommendation 模块自身 owner 范围，但当前仓库已经建立并持续扩展了真实 `learningengine × recommendation` 端到端回归，用来验证 Recommendation 在真实上游输入下的稳定行为

## 目录职责

- `application/dto`
  - 对外请求/响应结构
- `application/repository`
  - Recommendation 主链路依赖的读写 port
- `application/service`
  - 依赖 repository 的实现：
    - `DefaultContextAssembler`
    - `DefaultVideoStateEnricher`
    - `DefaultCandidateGenerator`
    - `DefaultEvidenceResolver`
    - `DefaultAuditWriter`
    - `DefaultServingStateManager`
    - `DefaultRecommendationResultWriter`
- `application/usecase`
  - `GenerateVideoRecommendationsService`
  - 只通过完整 pipeline constructor 构造
  - constructor 会校验主链路依赖必须全部就绪
  - 不再保留 assembler-only shell 或空响应降级路径
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
  - integration 测试使用外部依赖 stub + 真实 Recommendation migration / 物化视图 / refresh 路径
  - `test/fixture` 是模块内 integration 的唯一共享测试基座入口
  - 当前 integration 基座已对齐 `learningengine`：每个 integration 测试包共享一个 embedded Postgres server，template database 只初始化一次，每个测试 case clone 独立数据库
  - 当前测试结构已经对齐模块级集中测试规范，不在业务实现目录旁散落 `*_test.go`
  - 日常编码可先运行 `make quick-check`
  - 默认 `make check` 已通过一次 `go test -tags=integration ...` 调用并行调度 Learning engine + Recommendation integration；E2E 仍通过 `make e2e-test` 单独运行

## 当前未实现

- 自动刷新任务
- embedding / vector recall
- ML ranking
- task layer / 非视频 fallback

## 维护约束

- Recommendation 的审计中心始终是 video recommendation run/item。
- 只读计算不包长事务；只在最终写 audit 和 serving state 时开启短事务。
- `catalog.video_user_states` 只作为轻量 penalty 输入，不承载 Recommendation own 的投放状态。
- Candidate Generator 和 Evidence Resolver 必须沿调用方 `ctx` 传播取消、超时和 trace 上下文。
- `selector_mode=extreme_sparse` 由 selection 结果 underfill 后置判定，而不是 planner 预判。
- `GenerateVideoRecommendations` 对外响应使用 `best_evidence` 对象，而不是 4 个扁平字段；audit 表仍保留扁平存储字段。
- Recommendation 的输入装配边界是显式两阶段：
  - `DefaultContextAssembler` 只装配 request-scope / unit-scope 输入：active learning states、unit inventory、unit serving states
  - `DefaultVideoStateEnricher` 负责 candidate-derived video-scope 输入：video serving states、catalog video user states
- `DefaultVideoRanker` 仍计算 `RecentWatchedPenalty` 作为辅助观测值，但 MVP `BaseScore` 不再直接扣这一项，避免与 `FreshnessScore` 重复惩罚。
- `best_evidence` 只允许从 `evidence_span_refs` 命中结果中派生；如果 refs 无法命中 `catalog.video_semantic_spans`，当前实现会视为 Catalog 证据不一致并返回空 `best_evidence`，不会再兜底选“最早 span”。
- 当前 Recommendation 的真实验证分两层：
  - 模块内 integration：owner migration、物化视图、refresh、repository 契约
  - 跨模块 E2E：demand mapping、selector constraints、read model visibility、write-side consistency、Replay 交互、多用户隔离
