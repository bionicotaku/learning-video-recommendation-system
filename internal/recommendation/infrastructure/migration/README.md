# Recommendation Migrations

这个目录只定义 `recommendation` owner 的数据库对象。

当前基线包括：

- `recommendation` schema
- serving state 表
- video recommendation 审计表
- Recommendation own 物化读视图
- Recommendation 自己的索引

它不定义：

- Catalog owner 的表
- Learning engine owner 的表
- Ranking / Selector / Planner 的业务逻辑
- 自动刷新任务

Tracking table 固定为 `recommendation_schema_migrations`。
