# Recommendation Migrations

这个目录只定义 `recommendation` owner 的数据库对象。

当前基线包括：

- `recommendation` schema
- serving state 表
- video recommendation 审计表，其中 `video_recommendation_items` 保存 `dominant_role`、`dominant_unit_id` 和 `learning_units jsonb`
- Recommendation own 物化读视图
- Recommendation 自己的索引

它不定义：

- Catalog owner 的表
- Learning engine owner 的表
- Ranking / Selector / Planner 的业务逻辑
- 自动刷新任务
- 拆分 item-unit 明细子表或 `learning_units` 的 GIN 索引；MVP 只保存 JSONB 审计快照与常用 video/unit 索引

Tracking table 固定为 `recommendation_schema_migrations`。
