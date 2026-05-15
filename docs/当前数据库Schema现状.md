# 当前数据库 Schema 现状

状态：LIVE DB SNAPSHOT
更新时间：2026-05-14
判定口径：基于当前仓库 `.env` 中的 `DATABASE_URL` 做探查，并在本轮执行视频观看状态与全局统计 live DB 临时 SQL 后记录。

## 1. Schema 概览

当前 live DB 中与本项目相关的 schema 状态如下：

| schema | 当前状态 |
| --- | --- |
| `auth` | 存在，Supabase Auth 系统表存在 |
| `semantic` | 存在，包含 `coarse_unit`、`fine_unit` |
| `catalog` | 存在，包含当前 Catalog 内容表、`catalog.questions` 与 `catalog.video_engagement_stats` |
| `analytics` | 存在，包含 `analytics.quiz_events`、`analytics.video_watch_events`、`analytics.learning_interaction_events` |
| `recommendation` | 存在，包含 Recommendation 自有表与物化视图 |
| `learning` | 存在，包含 `learning.unit_learning_events`、`learning.user_unit_states` |

当前 live DB 已有 Catalog、Analytics、Learning Engine、Recommendation 自有表、索引和物化视图。视频观看状态与全局统计已用一次性临时 SQL 对齐，tracking 状态为 `module=analytics current=4 applied=4 pending=0`、`module=catalog current=11 applied=11 pending=0`。

## 2. Catalog Migration 状态

`catalog_schema_migrations` 当前有 11 条记录，对应仓库内 11 个 Catalog migration：

- `000001_create_catalog_schema`
- `000002_create_videos`
- `000003_create_video_transcripts`
- `000004_create_video_transcript_sentences`
- `000005_create_video_semantic_spans`
- `000006_create_video_unit_index`
- `000007_create_video_ingestion_records`
- `000008_create_video_user_states`
- `000009_create_catalog_indexes`
- `000010_create_questions`
- `000011_create_video_engagement_stats`

当前新增的 `catalog.questions` 已存在。只读核对显示该表有 14 个字段，并包含以下索引：

- `questions_pkey`
- `idx_questions_video_unit_active`
- `idx_questions_unit_active`
- `idx_questions_status_created_at`

当前新增的 `catalog.video_engagement_stats` 已存在。只读核对显示该表字段为：`video_id`、`view_count`、`like_count`、`favorite_count`、`completed_count`、`total_watch_ms`、`updated_at`。

当前 `catalog.video_user_states` 已删除 `last_watch_ratio`、`max_watch_ratio`，并新增 `last_position_ms`、`max_position_ms`、`total_watch_ms`。

## 3. Analytics Migration 状态

`analytics_schema_migrations` 当前有 4 条记录，对应仓库内 4 个 Analytics migration：

- `000001_create_analytics_schema`
- `000002_create_quiz_events`
- `000003_create_video_watch_events`
- `000004_create_learning_interaction_events`

当前新增的 `analytics.quiz_events` 已存在。只读核对显示该表有 16 个字段，并包含以下索引：

- `quiz_events_pkey`
- `uq_quiz_events_user_client_event`
- `idx_quiz_events_user_completed_at`
- `idx_quiz_events_question_completed_at`
- `idx_quiz_events_unit_completed_at`
- `idx_quiz_events_video_completed_at`

`analytics.quiz_events` 已包含 `client_context jsonb not null default '{}'::jsonb`，并包含 `quiz_events_client_context_is_object` 约束。

当前新增的 `analytics.video_watch_events` 已存在。只读核对显示该表有 15 个字段，并包含以下索引：

- `video_watch_events_pkey`
- `idx_video_watch_events_user_video_updated_at`
- `idx_video_watch_events_user_updated_at`
- `idx_video_watch_events_video_updated_at`

`analytics.video_watch_events` 已删除旧 `source` 字段，不再保存 `duration_ms` 与 `max_watch_ratio`，并包含 `active_watch_ms bigint not null default 0`、`client_context jsonb not null default '{}'::jsonb` 与 JSON object 约束。

当前新增的 `analytics.learning_interaction_events` 已存在。只读核对显示该表有 24 个字段，并包含以下索引：

- `learning_interaction_events_pkey`
- `uq_learning_interaction_events_user_client_event`
- `idx_learning_interaction_events_user_occurred_at`
- `idx_learning_interaction_events_user_unit_occurred_at`
- `idx_learning_interaction_events_video_occurred_at`
- `idx_learning_interaction_events_watch_session`
- `idx_learning_interaction_events_related_quiz`

`analytics.learning_interaction_events` 已包含 `client_context jsonb not null default '{}'::jsonb` 与 `event_payload jsonb not null default '{}'::jsonb`，并对两者都有 JSON object 约束。

## 4. Recommendation Migration 状态

`recommendation_schema_migrations` 当前有 5 条记录，对应仓库内 5 个 Recommendation migration：

- `000001_create_recommendation_schema`
- `000002_create_serving_state_tables`
- `000003_create_recommendation_audit_tables`
- `000004_create_materialized_views`
- `000005_create_recommendation_indexes`

Recommendation 本轮没有重新执行 migrate 或 refresh。

## 4.1 Learning Engine Migration 状态

`learningengine_schema_migrations` 当前有 5 条记录，对应仓库内 5 个 Learning Engine migration：

- `000001_create_learning_schema`
- `000002_create_user_unit_states`
- `000003_create_unit_learning_events`
- `000004_create_learning_indexes`
- `000005_add_set_mastered_reducer_effect`

当前 `learning.unit_learning_events` 已是 normalized Learning Engine event ledger。只读核对显示该表有 13 个字段，并包含以下关键约束与索引：

- `unit_learning_events_pkey`
- `uq_unit_learning_events_source_unit`
- `idx_learning_events_user_time`
- `idx_learning_events_user_unit_time`
- `event_type in ('exposure', 'lookup', 'quiz', 'self_mark_mastered')`
- `reducer_effect in ('observe_only', 'affects_progress', 'set_mastered')`
- `progress_quality` 仅在 `affects_progress` 时必填，范围 `0..5`；`observe_only` 和 `set_mastered` 必须为空
- `set_mastered` 只能与 `event_type = 'self_mark_mastered'` 一起使用
- `metadata` 必须为 JSON object

当前 `learning.user_unit_states` 已是 progress / schedule 语义的状态投影表。只读核对显示该表有 28 个字段，并包含以下索引：

- `user_unit_states_pkey`
- `idx_learning_states_user_target_status_due`
- `idx_learning_states_user_updated_at`

该表当前已使用 `first_observed_at`、`observation_count`、`progress_event_count`、`last_progress_quality`、`recent_progress_qualities`、`recent_progress_passes`、`schedule_repetition`、`schedule_interval_days`、`schedule_ease_factor` 等新字段，不再包含旧 `strong_event_count`、`review_count`、`last_quality`。

## 5. Recommendation 表与视图

当前 `recommendation` schema 包含：

- `recommendation.user_unit_serving_states`
- `recommendation.user_video_serving_states`
- `recommendation.video_recommendation_runs`
- `recommendation.video_recommendation_items`
- `recommendation.v_recommendable_video_units`
- `recommendation.v_unit_video_inventory`

其中两个物化视图已刷新：

- `recommendation.v_recommendable_video_units`
- `recommendation.v_unit_video_inventory`

## 6. `video_recommendation_items`

当前审计 item 表结构已经切换为 video learning plan 契约：

| column | type | nullable | default |
| --- | --- | --- | --- |
| `run_id` | `uuid` | no | |
| `rank` | `integer` | no | |
| `video_id` | `uuid` | no | |
| `score` | `numeric` | no | `0` |
| `primary_lane` | `text` | yes | |
| `dominant_role` | `text` | yes | |
| `dominant_unit_id` | `bigint` | yes | |
| `reason_codes` | `text[]` | no | `'{}'::text[]` |
| `learning_units` | `jsonb` | no | `'[]'::jsonb` |
| `created_at` | `timestamptz` | no | `now()` |

关键约束与语义：

- 主键是 `(run_id, rank)`。
- `run_id` 级联引用 `recommendation.video_recommendation_runs(run_id)`。
- `video_id` 级联引用 `catalog.videos(video_id)`。
- `dominant_unit_id` 引用 `semantic.coarse_unit(id)`，删除 coarse unit 时置空。
- `learning_units` 必须是 JSON array。
- 旧的 covered count 字段和 video-level best evidence 字段已经不再存在。

## 7. Recommendation 索引

当前 Recommendation owner 索引包括：

- `idx_recommendation_unit_serving_states_last_served_at`
- `idx_recommendation_video_serving_states_last_served_at`
- `idx_video_recommendation_runs_user_created_at`
- `idx_video_recommendation_items_video_id`
- `idx_video_recommendation_items_dominant_unit`
- `idx_v_recommendable_video_units_unit_video`
- `idx_v_recommendable_video_units_video_id`
- `idx_v_unit_video_inventory_unit`
- `idx_v_unit_video_inventory_supply_grade`

MVP 阶段未给 `learning_units` 增加 GIN 索引；它目前是审计快照字段，不承担高频查询入口。
