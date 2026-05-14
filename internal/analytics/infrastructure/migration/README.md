# Analytics Migrations

这个目录只定义 `analytics` owner 的数据库对象。

当前基线包括：

- `analytics` schema
- `analytics.quiz_events`
- `analytics.video_watch_events`
- `analytics.learning_interaction_events`
- raw event 表统一的 `client_context` JSONB 上下文字段
- Analytics 自己的索引

`analytics.video_watch_events` 保存观看 session 的位置、完成状态和 `active_watch_ms`。视频时长与观看比例不在 Analytics raw fact 表中重复保存，统一从 `catalog.videos.duration_ms` 派生。

它不定义：

- Catalog owner 的表
- Learning engine owner 的表
- Recommendation owner 的表
- normalizer / reducer 的业务逻辑

Tracking table 固定为 `analytics_schema_migrations`。
