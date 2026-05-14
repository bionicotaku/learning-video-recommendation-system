# Analytics Migrations

这个目录只定义 `analytics` owner 的数据库对象。

当前基线包括：

- `analytics` schema
- `analytics.quiz_events`
- Analytics 自己的索引

它不定义：

- Catalog owner 的表
- Learning engine owner 的表
- Recommendation owner 的表
- normalizer / reducer 的业务逻辑

Tracking table 固定为 `analytics_schema_migrations`。
