# 当前数据库 Schema 现状

状态：LIVE DB SNAPSHOT
更新时间：2026-05-08
判定口径：基于当前仓库 `.env` 中的 `DATABASE_URL` 做只读探查，并在本轮执行 `make recommendation-migrate-up` 与 `make recommendation-refresh` 后记录。

## 1. Schema 概览

当前 live DB 中与本项目相关的 schema 状态如下：

| schema | 当前状态 |
| --- | --- |
| `auth` | 存在，Supabase Auth 系统表存在 |
| `semantic` | 存在，包含 `coarse_unit`、`fine_unit` |
| `catalog` | 存在，包含当前 Catalog 内容表 |
| `recommendation` | 存在，本轮按当前 migration 干净创建 |
| `learning` | 不存在 |

这意味着当前 live DB 已有 Recommendation 自有表、索引和物化视图，但还没有 Learning engine 的 `learning.*` 表。需要完整线上闭环时，仍必须另行应用 Learning engine migration。

## 2. Recommendation Migration 状态

`recommendation_schema_migrations` 当前有 5 条记录，对应仓库内 5 个 Recommendation migration：

- `000001_create_recommendation_schema`
- `000002_create_serving_state_tables`
- `000003_create_recommendation_audit_tables`
- `000004_create_materialized_views`
- `000005_create_recommendation_indexes`

本轮 preflight 查询 `to_regclass('recommendation_schema_migrations')` 返回为空，因此没有执行 down；随后执行了 `make recommendation-migrate-up` 和 `make recommendation-refresh`。

## 3. Recommendation 表与视图

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

## 4. `video_recommendation_items`

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

## 5. Recommendation 索引

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

