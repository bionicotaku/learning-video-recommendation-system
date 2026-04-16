# Recommendation

`recommendation` 负责视频推荐主链路、serving state、预计算读模型和推荐审计。

当前首轮落地只补：

- 模块标准骨架
- Recommendation owner 的 migration
- 物化读视图与 SQL/query/sqlc 基础层
- DTO、repository port 和 usecase 接口

当前不补：

- planner / candidate / resolver / aggregator / ranker / selector / explain 业务逻辑
- 自动刷新任务
- 任何回写 Learning engine 或 Catalog owner 对象的逻辑
