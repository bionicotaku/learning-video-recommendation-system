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
  - `Final Item Builder`
- `GenerateVideoRecommendations` 的完整 orchestrator
- Recommendation owner migration
- 物化读视图与 SQL/query/sqlc 基础层
- `v_video_unit_recall_index` 按 `coarse_unit_id -> top-N videos` 提供推荐反向索引
- `video_recommendation_runs` / `video_recommendation_items` 审计写入
- `user_unit_serving_states` / `user_video_serving_states` 短事务原子递增写入
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
  - `DefaultVideoFillService`
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
  - video-level 聚合、expected learning units 构造与覆盖特征
- `domain/ranking`
  - 第一版规则排序与 penalty
- `domain/selector`
  - `normal / low_supply / extreme_sparse` 选择约束
- `application/service/DefaultVideoFillService`
  - 在 selector 后、final item builder / audit 前做轻量 video-level 补全
  - 先补 `mastered_target_fill`，再补 `popular_fill`
  - 只读取小候选池，不走 demand planning、evidence resolver 或 aggregator
- `domain/explain`
  - 从 selected video 构造最终 plan item，并生成 audit 用 `reason_codes`
- `infrastructure/persistence`
  - query/sqlc/repository/tx
- `test`
  - Recommendation 自己的 unit / golden / integration 测试
  - integration 测试使用外部依赖 stub + 真实 Recommendation migration / 物化视图 / refresh 路径
  - `test/fixture` 是模块内 integration 的唯一共享测试基座入口
  - 当前 integration 基座已对齐 `learningengine`：每个 integration 测试包共享一个 embedded Postgres server，template database 只初始化一次，每个测试 case clone 独立数据库
  - embedded Postgres 生命周期和 template clone 通过 `internal/platform/postgres/pgtest` 复用；Recommendation `test/fixture` 只保留 schema plan、seed helper 和 tx helper
  - 当前测试结构已经对齐模块级集中测试规范，不在业务实现目录旁散落 `*_test.go`
  - 日常编码可先运行 `make quick-check`
  - 默认 `make check` 已通过一次 `go test -tags=integration ...` 调用并行调度 Learning engine + Recommendation integration；E2E 仍通过 `make e2e-test` 单独运行

## 当前未实现

- 自动刷新任务
- embedding / vector recall
- ML ranking
- task layer / 非视频 fallback

## Video-level 补全

当正常学习推荐在 selector 后少于 `target_video_count` 时，Recommendation 会在内部追加轻量补全视频，顺序固定为：

```text
normal learning recommendations
-> mastered_target_fill
-> popular_fill
```

`mastered_target_fill` 从当前用户 `is_target=true AND status='mastered'` 的 unit 对应视频中补，保持和当前目标集合相关；`popular_fill` 从全局 active/public/published 视频中补，`catalog.video_engagement_stats` 缺失时按 0 热度处理，不会排除视频。两类补全都只查 video-level 小候选池与 freshness/watch 信号，不查 subtitle/span/sentence evidence，不生成 learning unit，也不触发 end quiz。

## Recall Index

学习候选的第一跳读取是 `recommendation.v_video_unit_recall_index`。该物化视图从 `catalog.video_unit_index`、`catalog.videos`、`catalog.video_transcripts` 派生，过滤 active/public/published 视频，并计算：

- `content_quality_score`：由 `best_evidence_candidate_score`、coverage、mention/sentence count、mapped span ratio 合成的内容质量 prior
- `rank_within_unit`：同一 `coarse_unit_id` 下按内容质量预排序的反向索引 rank

`RecommendableVideoUnitReader` 在线只按 `coarse_unit_id + per_unit_limit` 读取 top-N rows。`learning_units[].evidence` 直接来自 recall row 中的 best evidence sentence/span/start/end，默认 `EvidenceResolver` 不再回查 `catalog.video_semantic_spans` 或 `catalog.video_transcript_sentences`。

## User Recall Queue

在线推荐不再对所有未 mastered target units 读取 recall rows。`DefaultContextAssembler` 先使用 Recommendation-owned `user_unit_recall_queue` projection：

- queue 缺失、Learning state 版本变化、或 `dbtool refresh recommendation` 更新了 recall projection metadata 时，当前用户 queue 会 lazy rebuild。
- 本轮 `planner_scope` 限制为 `min(max(target_video_count * 12, 64), 200)` 个 units。
- active target 数量变化也会触发 queue rebuild；用户级 rebuild 通过事务锁和 upsert 保持并发幂等。
- `RecallQueueService` 返回显式 `planner_scope` 和 `recall_fetch_scope`：前者进入 Demand Planner，后者读取 `v_video_unit_recall_index`。
- `recall_fetch_scope` 只包含 `supply_grade <> 'none'` 的 units，并按 `min(max(target_video_count * 4, 20), 50)` 读取 top-N recall rows。
- `supply_grade='none'` 的 units 最多少量保留在 `planner_scope` 中供 planner 感知低供给，不做无效 recall row 查询。
- scope 默认 bucket 配额为 `hard_review 40% / new_now 30% / soft_review 20% / near_future 10%`；hard backlog 大时提升 hard_review 配额。
- `candidate_summary` 记录 planner/fetch/no-supply scope 指标和 `pipeline_timing_ms`，用于后续定位推荐主链路瓶颈；`pipeline_timing_ms` 包含 `evidence_resolve`，但不统计 audit / serving state 写入耗时。

补全 item 仍写入 Recommendation audit 和 `user_video_serving_states`，但 `learning_units=[]`，因此不会写 `user_unit_serving_states`。Feed facade 只继续补展示字段，不做补全决策。

## 维护约束

- Recommendation 的审计中心始终是 video recommendation run/item。
- 只读计算不包长事务；只在最终写 audit 和 serving state 时开启短事务。
- Serving state 计数由数据库侧 `served_count = served_count + 1` 原子递增维护，应用层不读取旧计数再覆盖写回。
- `catalog.video_user_states` 只作为轻量 penalty 输入，不承载 Recommendation own 的投放状态。
- Candidate Generator 和 Evidence Resolver 必须沿调用方 `ctx` 传播取消、超时和 trace 上下文。
- Recommendation persistence mappers 通过 `internal/platform/postgres/pgtime` 读写 UTC `time.Time`；serving state 写入前注入的 `now` 也归一化为 UTC。
- UUID、nullable text、numeric 等纯 Postgres 类型转换委托 `internal/platform/postgres/*`；Recommendation 仍保留本地 mapper 函数作为模块边界。
- `selector_mode=extreme_sparse` 由 selection 结果 underfill 后置判定，而不是 planner 预判。
- `GenerateVideoRecommendations` 对外响应是精简 video learning plan：只返回 `run_id`、`items[].video_id`、`items[].duration_ms` 和 `items[].learning_units`。`selector_mode`、`underfilled`、`rank`、`score`、`reason_codes` 只保留在 Recommendation audit 中。
- `learning_units=[]` 只用于 video-level 补全 item。它进入 video serving state，不进入 unit serving state，也不是本轮学习目标。
- `video_recommendation_items` 审计表以 `dominant_role`、`dominant_unit_id` 和 `learning_units jsonb` 保存 item 快照；不再保存 covered count 或 video-level best evidence 字段。
- `LaneSources` 是 video 级所有命中候选 lane 的集合，不是 per-unit winning candidate 的 lane 集合；`primary_lane` 从完整 `LaneSources` 按 lane priority 派生。
- Selector 的 same-unit 硬上限只按 `learning_units` 中 `is_primary=true` 的 unit 计数；非 primary support units 只参与边际覆盖、解释和 serving state。
- Recommendation 的输入装配边界是显式两阶段：
  - `DefaultContextAssembler` 只装配 request-scope / unit-scope 输入：active learning states、unit inventory、unit serving states
  - `DefaultVideoStateEnricher` 负责 candidate-derived video-scope 输入：video serving states、catalog video user states
- `DefaultVideoRanker` 仍计算 `RecentWatchedPenalty` 作为辅助观测值，但 MVP `BaseScore` 不再直接扣这一项，避免与 `FreshnessScore` 重复惩罚。
- `learning_units[].evidence` 只允许从 recall row 的 `best_evidence_*` 派生；Catalog ingest 必须保证这些字段来自同一 `(video_id, coarse_unit_id)` 下的 selected best evidence span。
- Evidence Resolver 不访问数据库；如果未来需要更大的 evidence window，应先扩展 Recommendation read model 或新增明确的非热路径 hydration。
- Audit Writer 批量写入 `video_recommendation_items`；run 仍单条写入，items 不做逐条 roundtrip。
- 当前 Recommendation 的真实验证分两层：
  - 模块内 integration：owner migration、物化视图、refresh、repository 契约
  - 跨模块 E2E：demand mapping、selector constraints、read model visibility、write-side consistency、Replay 交互、多用户隔离
