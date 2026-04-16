# 当前数据库 Schema 现状

状态：CURRENT SNAPSHOT  
更新时间：2026-04-16  
数据来源：根据仓库根目录 `.env` 中的 `DATABASE_URL`，对当前 Supabase Postgres 实例做只读探查得到。  
说明：本文描述的是 **live DB 当前真实结构**，不是目标设计稿，也不包含迁移步骤。

## 1. 范围与判定口径

当前 `.env` 只连接到一个 PostgreSQL 数据库：`postgres`。  
因此这里把“需要识别的非 Supabase 默认部分”解释为：

1. 当前数据库中 **非 Supabase 默认的业务 schema**
2. 当前业务链路真实依赖的表、函数、约束与索引

本文不展开 Supabase 系统 schema 的内部结构，只识别其存在并从业务分析中排除。

## 2. 当前库中的 schema 分类

### 2.1 PostgreSQL 系统 schema

- `information_schema`
- `pg_catalog`
- `pg_toast`
- `pg_temp_*`
- `pg_toast_temp_*`

这些都是 PostgreSQL 系统对象，不属于业务 schema。

### 2.2 Supabase 默认/托管 schema

- `auth`
- `extensions`
- `graphql`
- `graphql_public`
- `net`
- `pgbouncer`
- `pgsodium`
- `pgsodium_masks`
- `public`
- `realtime`
- `storage`
- `supabase_functions`
- `supabase_migrations`
- `vault`

### 2.3 当前识别出的业务 schema

以下四个 schema 不是 Supabase 默认业务域，属于项目当前实际使用的业务 schema：

1. `semantic`
2. `catalog`
3. `learning`
4. `recommendation`

## 3. 总体结论

当前 live DB 的业务结构已经收口到四个 schema：

1. `semantic`
2. `catalog`
3. `learning`
4. `recommendation`

`public` 当前没有自定义表、视图、序列或函数。

## 4. 当前业务 schema 现状

### 4.1 `semantic`

用途：语义单元主数据。当前是内容与学习域共享的基础字典层。

当前对象：

- 表：`fine_unit`、`coarse_unit`
- 函数：`coarse_unit_validate_fine_ids()`
- 触发器：`trg_coarse_unit_validate`（`BEFORE INSERT/UPDATE`）
- RLS：`coarse_unit_select_all`，允许 `anon` / `authenticated` 读 `semantic.coarse_unit`

当前数据量：

- `semantic.coarse_unit`: `170730`
- `semantic.fine_unit`: `207429`

#### `semantic.fine_unit`

用途：最细粒度语义单元表。

- 主键：`id`
- 行数：`207429`

列：

- `id bigint not null default nextval(...)`
- `kind text not null`
- `label text not null`
- `lang text not null default 'en'`
- `pos char(1) null`
- `def text null`
- `pattern jsonb null`
- `meta jsonb not null default '{}'::jsonb`
- `status text not null default 'active'`
- `created_at timestamptz null default now()`
- `updated_at timestamptz null default now()`
- `external_key text null`

关键约束：

- `PRIMARY KEY (id)`
- `CHECK kind in ('word_sense', 'phrase_sense', 'grammar_rule')`

索引：

- `fine_unit_pkey`

#### `semantic.coarse_unit`

用途：Recommendation、Catalog、Learning engine 共同依赖的 coarse unit 主表。

- 主键：`id`
- 行数：`170730`

列：

- `id bigint not null default nextval(...)`
- `kind text not null`
- `label text not null`
- `lang text not null default 'en'`
- `pos text null`
- `english_def text null`
- `chinese_def text null`
- `chinese_criteria text null`
- `chinese_label text null`
- `english_label text null`
- `pattern jsonb null`
- `status text not null default 'active'`
- `created_at timestamptz not null default now()`
- `updated_at timestamptz not null default now()`
- `version int not null default 1`
- `fine_unit_ids bigint[] not null`
- `original_defs text[] not null`

关键约束：

- `PRIMARY KEY (id)`
- `CHECK kind in ('word', 'phrase', 'grammar')`

关键行为：

- `trg_coarse_unit_validate` 在写入前执行 `coarse_unit_validate_fine_ids()`，说明 `fine_unit_ids` 会做校验。

索引：

- `coarse_unit_pkey`
- `ix_coarse_kind`
- `ix_coarse_label`
- `coarse_unit_label_lower_idx`
- `coarse_unit_label_lower_trgm_idx`

### 4.2 `catalog`

用途：内容事实、transcript 读模型、视频级 coarse unit 索引、视频互动投影、单视频入库审计。

当前数据量：

- `catalog.videos`: `72`
- `catalog.video_transcripts`: `72`
- `catalog.video_transcript_sentences`: `3086`
- `catalog.video_semantic_spans`: `15691`
- `catalog.video_unit_index`: `5476`
- `catalog.video_ingestion_records`: `2044`
- `catalog.video_user_states`: `0`

#### `catalog.videos`

用途：视频/clip 主记录。

- 主键：`video_id`
- 唯一键：`source_clip_key`
- 行数：`72`

列：

- `video_id uuid not null default gen_random_uuid()`
- `source_clip_key text not null`
- `parent_video_name text not null`
- `parent_video_slug text not null`
- `clip_seq int null`
- `source_start_ms int null`
- `source_end_ms int null`
- `title text not null`
- `description text null`
- `clip_reason text null`
- `language text not null default 'en'`
- `duration_ms int not null`
- `hls_master_playlist_path text not null`
- `thumbnail_url text null`
- `status text not null default 'active'`
- `visibility_status text not null default 'public'`
- `publish_at timestamptz null`
- `created_at timestamptz not null default now()`
- `updated_at timestamptz not null default now()`

关键约束：

- `CHECK duration_ms > 0`
- `CHECK status in ('active', 'inactive', 'deleted')`
- `CHECK visibility_status in ('public', 'unlisted', 'private')`
- `CHECK source_end_ms > source_start_ms`（当两者都非空）

索引：

- `videos_pkey`
- `videos_source_clip_key_key`
- `idx_videos_status`
- `idx_videos_visibility_publish_at`
- `idx_videos_parent_video_slug_clip_seq`
- `idx_videos_created_at_desc`

#### `catalog.video_transcripts`

用途：视频 transcript 汇总读模型，每个视频一条。

- 主键 / 外键：`video_id -> catalog.videos(video_id)`
- 行数：`72`

列：

- `video_id uuid not null`
- `transcript_object_path text not null`
- `transcript_checksum text not null`
- `transcript_format_version int not null default 1`
- `full_text text not null`
- `sentence_count int not null`
- `semantic_span_count int not null`
- `mapped_span_count int not null`
- `unmapped_span_count int not null`
- `mapped_span_ratio numeric not null`
- `created_at timestamptz not null default now()`
- `updated_at timestamptz not null default now()`

关键约束：

- `sentence_count >= 0`
- `semantic_span_count >= 0`
- `mapped_span_count >= 0`
- `unmapped_span_count >= 0`
- `mapped_span_ratio between 0 and 1`

索引：

- `video_transcripts_pkey`

#### `catalog.video_transcript_sentences`

用途：transcript 句子级读模型。

- 复合主键：`(video_id, sentence_index)`
- 外键：`video_id -> catalog.videos(video_id)`
- 行数：`3086`

列：

- `video_id uuid not null`
- `sentence_index int not null`
- `text text not null`
- `start_ms int not null`
- `end_ms int not null`
- `explanation text null`
- `created_at timestamptz not null default now()`

关键约束：

- `sentence_index >= 0`
- `start_ms >= 0`
- `end_ms > start_ms`

索引：

- `video_transcript_sentences_pkey`
- `idx_video_transcript_sentences_video_start_ms`
- `idx_video_transcript_sentences_video_end_ms`

#### `catalog.video_semantic_spans`

用途：视频 transcript 内的 span 级语义命中事实层。

- 复合主键：`(video_id, sentence_index, span_index)`
- 外键：
  - `(video_id, sentence_index) -> catalog.video_transcript_sentences(video_id, sentence_index)`
  - `coarse_unit_id -> semantic.coarse_unit(id)`，可空
- 行数：`15691`

列：

- `video_id uuid not null`
- `sentence_index int not null`
- `span_index int not null`
- `text text not null`
- `start_ms int not null`
- `end_ms int not null`
- `explanation text null`
- `coarse_unit_id bigint null`
- `base_form text null`
- `dictionary_text text null`
- `created_at timestamptz not null default now()`

关键约束：

- `span_index >= 0`
- `start_ms >= 0`
- `end_ms > start_ms`

索引：

- `video_semantic_spans_pkey`
- `idx_video_semantic_spans_video_sentence`
- `idx_video_semantic_spans_video_start_ms`
- `idx_video_semantic_spans_coarse_unit_video`（`coarse_unit_id is not null`）
- `idx_video_semantic_spans_video_coarse_unit`（`coarse_unit_id is not null`）

#### `catalog.video_unit_index`

用途：视频级 coarse unit 索引，供 recall / recommendation 使用。

- 复合主键：`(video_id, coarse_unit_id)`
- 外键：
  - `video_id -> catalog.videos(video_id)`
  - `coarse_unit_id -> semantic.coarse_unit(id)`
- 行数：`5476`

列：

- `video_id uuid not null`
- `coarse_unit_id bigint not null`
- `mention_count int not null`
- `sentence_count int not null`
- `first_start_ms int not null`
- `last_end_ms int not null`
- `coverage_ms int not null`
- `coverage_ratio numeric not null`
- `sentence_indexes int[] not null default '{}'`
- `evidence_sentence_indexes int[] not null default '{}'`
- `evidence_span_indexes int[] not null default '{}'`
- `sample_surface_forms text[] not null default '{}'`
- `created_at timestamptz not null default now()`
- `updated_at timestamptz not null default now()`

关键约束：

- `mention_count > 0`
- `sentence_count > 0`
- `coverage_ms > 0`
- `coverage_ratio between 0 and 1`
- `last_end_ms > first_start_ms`

索引：

- `video_unit_index_pkey`
- `idx_video_unit_index_coarse_unit_mention_coverage`
- `idx_video_unit_index_video_id`

注意：

- live DB 里仍然是 `evidence_sentence_indexes + evidence_span_indexes` 组合。
- 当前库里 **还没有** `evidence_span_refs`。

#### `catalog.video_ingestion_records`

用途：单视频入库/处理审计。

- 主键：`ingestion_record_id`
- 外键：`video_id -> catalog.videos(video_id)`，`ON DELETE SET NULL`
- 行数：`2044`

列：

- `ingestion_record_id uuid not null`
- `source_clip_key text not null`
- `video_id uuid null`
- `source_name text null`
- `status text not null`
- `warning_codes text[] not null default '{}'`
- `error_code text null`
- `error_message text null`
- `context jsonb not null default '{}'::jsonb`
- `started_at timestamptz not null`
- `finished_at timestamptz null`
- `created_at timestamptz not null default now()`

关键约束：

- `CHECK status in ('running', 'succeeded', 'failed', 'skipped')`

索引：

- `video_ingestion_records_pkey`
- `idx_video_ingestion_records_source_clip_key_started_at`
- `idx_video_ingestion_records_status_started_at`
- `idx_video_ingestion_records_video_id`

#### `catalog.video_user_states`

用途：用户对视频的互动状态投影。

- 复合主键：`(user_id, video_id)`
- 外键：
  - `user_id -> auth.users(id)`
  - `video_id -> catalog.videos(video_id)`
- 行数：`0`

列：

- `user_id uuid not null`
- `video_id uuid not null`
- `has_liked bool not null default false`
- `has_bookmarked bool not null default false`
- `has_watched bool not null default false`
- `liked_at timestamptz null`
- `bookmarked_at timestamptz null`
- `first_watched_at timestamptz null`
- `last_watched_at timestamptz null`
- `watch_count int not null default 0`
- `completed_count int not null default 0`
- `last_watch_ratio numeric null`
- `max_watch_ratio numeric null`
- `updated_at timestamptz not null default now()`

关键约束：

- `watch_count >= 0`
- `completed_count >= 0`
- `last_watch_ratio is null or between 0 and 1`
- `max_watch_ratio is null or between 0 and 1`

索引：

- `video_user_states_pkey`
- `idx_video_user_states_user_last_watched_at`
- `idx_video_user_states_video_id`

### 4.3 `learning`

用途：Learning engine 的事件真相层和状态投影层。

当前数据量：

- `learning.unit_learning_events`: `0`
- `learning.user_unit_states`: `0`

#### `learning.unit_learning_events`

用途：append-only 学习事件表。

- 主键：`event_id`
- 外键：
  - `user_id -> auth.users(id)`
  - `coarse_unit_id -> semantic.coarse_unit(id)`
  - `video_id -> catalog.videos(video_id)`，`ON DELETE SET NULL`
- 行数：`0`

列：

- `event_id bigint not null default nextval(...)`
- `user_id uuid not null`
- `coarse_unit_id bigint not null`
- `video_id uuid null`
- `event_type text not null`
- `source_type text not null`
- `source_ref_id text null`
- `is_correct bool null`
- `quality smallint null`
- `response_time_ms int null`
- `metadata jsonb not null default '{}'::jsonb`
- `occurred_at timestamptz not null`
- `created_at timestamptz not null default now()`

关键约束：

- `CHECK event_type in ('exposure', 'lookup', 'new_learn', 'review', 'quiz')`
- `CHECK quality between 0 and 5`

索引：

- `unit_learning_events_pkey`
- `idx_unit_learning_events_user_unit_time`
- `idx_unit_learning_events_user_video_time`

#### `learning.user_unit_states`

用途：用户 x coarse unit 当前状态投影。

- 复合主键：`(user_id, coarse_unit_id)`
- 外键：
  - `user_id -> auth.users(id)`
  - `coarse_unit_id -> semantic.coarse_unit(id)`
- 行数：`0`

列：

- `user_id uuid not null`
- `coarse_unit_id bigint not null`
- `is_target bool not null default true`
- `target_source text null`
- `target_source_ref_id text null`
- `target_priority numeric not null default 0.5`
- `status text not null default 'new'`
- `progress_percent numeric not null default 0`
- `mastery_score numeric not null default 0`
- `first_seen_at timestamptz null`
- `last_seen_at timestamptz null`
- `last_reviewed_at timestamptz null`
- `seen_count int not null default 0`
- `strong_event_count int not null default 0`
- `review_count int not null default 0`
- `correct_count int not null default 0`
- `wrong_count int not null default 0`
- `consecutive_correct int not null default 0`
- `consecutive_wrong int not null default 0`
- `last_quality smallint null`
- `recent_quality_window smallint[] not null default '{}'`
- `recent_correctness_window boolean[] not null default '{}'`
- `repetition int not null default 0`
- `interval_days numeric not null default 0`
- `ease_factor numeric not null default 2.5`
- `next_review_at timestamptz null`
- `suspended_reason text null`
- `created_at timestamptz not null default now()`
- `updated_at timestamptz not null default now()`

关键约束：

- `CHECK status in ('new', 'learning', 'reviewing', 'mastered', 'suspended')`
- `CHECK progress_percent between 0 and 100`
- `CHECK mastery_score between 0 and 1`
- `CHECK last_quality between 0 and 5`（允许 null）

索引：

- `user_unit_states_pkey`
- `idx_user_unit_states_target_status`
- `idx_user_unit_states_next_review`

### 4.4 `recommendation`

用途：当前 live DB 中 Recommendation 已落地的仍是旧 `scheduler` 结构。

当前数据量：

- `recommendation.scheduler_runs`: `0`
- `recommendation.scheduler_run_items`: `0`
- `recommendation.user_unit_serving_states`: `0`

#### `recommendation.scheduler_runs`

用途：单次 scheduler 运行头记录。

- 主键：`run_id`
- 外键：`user_id -> auth.users(id)`
- 行数：`0`

列：

- `run_id uuid not null`
- `user_id uuid not null`
- `requested_limit int not null`
- `generated_at timestamptz not null`
- `due_review_count int not null default 0`
- `selected_review_count int not null default 0`
- `selected_new_count int not null default 0`
- `context jsonb not null default '{}'::jsonb`

索引：

- `scheduler_runs_pkey`

#### `recommendation.scheduler_run_items`

用途：单次 scheduler 输出的 coarse unit 列表。

- 复合主键：`(run_id, coarse_unit_id)`
- 外键：
  - `run_id -> recommendation.scheduler_runs(run_id)`
  - `user_id -> auth.users(id)`
  - `coarse_unit_id -> semantic.coarse_unit(id)`
- 行数：`0`

列：

- `run_id uuid not null`
- `user_id uuid not null`
- `coarse_unit_id bigint not null`
- `recommend_type text not null`
- `rank int not null`
- `score numeric not null`
- `reason_codes text[] not null default '{}'`

关键约束：

- `CHECK recommend_type in ('review', 'new')`

索引：

- `scheduler_run_items_pkey`

#### `recommendation.user_unit_serving_states`

用途：Recommendation 对用户 x coarse unit 的 serving 冷却状态。

- 复合主键：`(user_id, coarse_unit_id)`
- 外键：
  - `user_id -> auth.users(id)`
  - `coarse_unit_id -> semantic.coarse_unit(id)`
- 行数：`0`

列：

- `user_id uuid not null`
- `coarse_unit_id bigint not null`
- `last_recommended_at timestamptz null`
- `last_recommendation_run_id uuid null`
- `created_at timestamptz not null default now()`
- `updated_at timestamptz not null default now()`

索引：

- `user_unit_serving_states_pkey`
- `idx_user_unit_serving_states_user_last_recommended`

## 5. `public` schema 当前状态

`public` 是 Supabase 默认 schema。  
截至本次探查，`public` 中当前没有自定义：

- 表
- 视图
- 物化视图
- 序列
- 函数

因此当前业务现状文档不再把 `public` 作为业务结构的一部分展开。

## 6. 当前 cross-schema 依赖

### 6.1 主链路

当前真实依赖链是：

1. `semantic.coarse_unit`
2. `catalog.videos`
3. `catalog.video_transcripts`
4. `catalog.video_transcript_sentences`
5. `catalog.video_semantic_spans`
6. `catalog.video_unit_index`
7. `learning.*`
8. `recommendation.*`

### 6.2 主要外键关系

- `catalog.video_transcripts.video_id -> catalog.videos.video_id`
- `catalog.video_transcript_sentences.video_id -> catalog.videos.video_id`
- `catalog.video_semantic_spans.(video_id, sentence_index) -> catalog.video_transcript_sentences`
- `catalog.video_semantic_spans.coarse_unit_id -> semantic.coarse_unit.id`
- `catalog.video_unit_index.video_id -> catalog.videos.video_id`
- `catalog.video_unit_index.coarse_unit_id -> semantic.coarse_unit.id`
- `catalog.video_user_states.user_id -> auth.users.id`
- `catalog.video_user_states.video_id -> catalog.videos.video_id`
- `learning.unit_learning_events.user_id -> auth.users.id`
- `learning.unit_learning_events.coarse_unit_id -> semantic.coarse_unit.id`
- `learning.unit_learning_events.video_id -> catalog.videos.video_id`
- `learning.user_unit_states.user_id -> auth.users.id`
- `learning.user_unit_states.coarse_unit_id -> semantic.coarse_unit.id`
- `recommendation.scheduler_runs.user_id -> auth.users.id`
- `recommendation.scheduler_run_items.run_id -> recommendation.scheduler_runs.run_id`
- `recommendation.scheduler_run_items.user_id -> auth.users.id`
- `recommendation.scheduler_run_items.coarse_unit_id -> semantic.coarse_unit.id`
- `recommendation.user_unit_serving_states.user_id -> auth.users.id`
- `recommendation.user_unit_serving_states.coarse_unit_id -> semantic.coarse_unit.id`

## 7. 当前现状与最新版设计文档的明显差异

这部分不是目标设计，而是 live DB 与当前最新版设计文档之间已经能直接观察到的差异。

### 7.1 `catalog.video_unit_index` 仍是旧 evidence 结构

当前 live DB：

- 有 `evidence_sentence_indexes`
- 有 `evidence_span_indexes`
- 没有 `evidence_span_refs`

这说明 Catalog 仍停留在旧 evidence 表达。

### 7.2 `recommendation` 仍是旧 `scheduler_*` 结构

当前 live DB 只有：

- `scheduler_runs`
- `scheduler_run_items`
- `user_unit_serving_states`

还没有最新版设计里更完整的视频推荐 run/item 结构，也没有 `v_recommendable_video_units`、`v_unit_video_inventory` 这类读模型对象。

### 7.3 Learning / Recommendation 业务数据尚未启动

虽然 schema 已建好，但当前：

- `learning.unit_learning_events = 0`
- `learning.user_unit_states = 0`
- `recommendation.scheduler_runs = 0`
- `recommendation.scheduler_run_items = 0`
- `recommendation.user_unit_serving_states = 0`

说明学习与推荐主链路在当前实例里还没有真实业务数据。

## 8. 结论

如果只看 live DB，当前数据库可以概括为：

1. **语义主数据已大量落库**
   - `semantic.coarse_unit`
   - `semantic.fine_unit`
2. **Catalog 新结构已经有真实内容数据**
   - `videos / transcripts / sentences / spans / unit_index`
3. **Learning / Recommendation schema 已建，但还没有业务数据**
4. **当前 DB 现实仍早于最新版设计文档**
   - 特别是 `evidence_span_refs`
   - 以及 Recommendation 的最终 run/item / read-model 结构

因此，当前库最准确的描述不是“已经完全对齐最新版设计”，而是：

> 四个业务 schema 已经落库并承载当前业务主链路，但部分最新版设计对象还没有真正落到 live DB。
