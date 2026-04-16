# Learning Engine Migrations

这个目录只定义 `learningengine` owner 的数据库对象。

它只保留当前最终基线：

- `learning` schema
- `learning.user_unit_states`
- `learning.unit_learning_events`
- Learning engine 自己的索引

它不定义：

- Recommendation 的表或视图
- Catalog 的表
- 历史兼容 patch migration
- reducer / replay 的 SQL 逻辑

Tracking table 固定为 `learningengine_schema_migrations`。
