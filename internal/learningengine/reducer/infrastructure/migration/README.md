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

当前 baseline 约束：

- `000002_create_user_unit_states` 定义 progress / schedule 语义的 `learning.user_unit_states`。
- `000003_create_unit_learning_events` 定义 normalized event ledger，不定义 analytics raw log。
- `000004_create_learning_indexes` 只放当前最终索引。
- 迁移历史已压缩到干净 head `4`；不要重新引入中间过程 migration。

Tracking table 固定为 `learningengine_schema_migrations`。
