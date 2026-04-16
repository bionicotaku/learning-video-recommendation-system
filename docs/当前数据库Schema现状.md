# 当前数据库 Schema 现状

状态：CURRENT SNAPSHOT  
更新时间：2026-04-16  
数据来源：根据仓库根目录 `.env` 中的 `DATABASE_URL`，对当前 Supabase Postgres 实例做只读探查得到。  
说明：本文描述的是 **live DB 当前真实结构**，不是目标设计稿，也不包含迁移步骤。

## 1. 范围与判定口径

当前 `.env` 只连接到一个 PostgreSQL 数据库：`postgres`。  
本文只描述当前实例里仍然存在、且与项目业务直接相关的 schema、表、约束与索引。

不展开：

- PostgreSQL 系统 schema
- Supabase 托管 schema 的内部结构
- 已经删除的历史 schema 或历史表

## 2. 当前 schema 分类

### 2.1 PostgreSQL 系统 schema

- `information_schema`
- `pg_catalog`
- `pg_toast`
- `pg_temp_*`
- `pg_toast_temp_*`

### 2.2 Supabase 默认 / 托管 schema

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

### 2.3 当前业务 schema

当前实例里仍然存在的业务 schema 只有两个：

1. `semantic`
2. `catalog`

### 2.4 已清空 / 已移除

截至本次更新：

- `public` 中没有自定义表、视图、序列或函数
- `learning` schema 已删除
- `recommendation` schema 已删除

## 3. 总体结论

当前 live DB 已收口到：

1. `semantic`
   - 语义主数据
2. `catalog`
   - 内容事实、transcript 读模型、视频级 coarse unit 索引、入库审计、用户视频互动投影

也就是说，当前数据库已经不再承载 Learning engine 和 Recommendation 的业务表。

## 4. 当前业务 schema 现状

### 4.1 `semantic`

用途：语义单元主数据。当前是内容域共享的基础字典层。

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

用途：Catalog 当前依赖的 coarse unit 主表。

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

- `trg_coarse_unit_validate` 在写入前执行 `coarse_unit_validate_fine_ids()`，说明 `fine_unit_ids` 会做校验

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

用途：视频 / clip 主记录。

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
- `idx_videos_recommendable`

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
- `mapped_span_ratio numeric(6,5) not null`
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
- `idx_video_semantic_spans_unit_video_start`（`coarse_unit_id is not null`）

#### `catalog.video_unit_index`

用途：视频级 coarse unit 索引。

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
- `coverage_ratio numeric(6,5) not null`
- `sentence_indexes int[] not null default '{}'`
- `evidence_span_refs jsonb not null default '[]'::jsonb`
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
- `idx_video_unit_index_unit_video`
- `idx_video_unit_index_video_id`

注意：

- `evidence_span_refs` 是最终 evidence 表达
- 每个元素至少包含 `sentence_index` 与 `span_index`
- Catalog 不在这一层固化 `best_evidence_*`

#### `catalog.video_ingestion_records`

用途：单视频入库 / 处理审计。

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
- `last_watch_ratio numeric(6,5) null`
- `max_watch_ratio numeric(6,5) null`
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

## 5. `public` schema 当前状态

`public` 是 Supabase 默认 schema。  
截至本次探查，`public` 中当前没有自定义：

- 表
- 视图
- 物化视图
- 序列
- 函数

因此当前业务现状文档不把 `public` 作为业务结构展开。

## 6. 当前 cross-schema 依赖

### 6.1 主链路

当前真实依赖链是：

1. `semantic.coarse_unit`
2. `catalog.videos`
3. `catalog.video_transcripts`
4. `catalog.video_transcript_sentences`
5. `catalog.video_semantic_spans`
6. `catalog.video_unit_index`

### 6.2 主要外键关系

- `catalog.video_transcripts.video_id -> catalog.videos.video_id`
- `catalog.video_transcript_sentences.video_id -> catalog.videos.video_id`
- `catalog.video_semantic_spans.(video_id, sentence_index) -> catalog.video_transcript_sentences`
- `catalog.video_semantic_spans.coarse_unit_id -> semantic.coarse_unit.id`
- `catalog.video_unit_index.video_id -> catalog.videos.video_id`
- `catalog.video_unit_index.coarse_unit_id -> semantic.coarse_unit.id`
- `catalog.video_user_states.user_id -> auth.users.id`
- `catalog.video_user_states.video_id -> catalog.videos.video_id`

## 7. 当前现状与最新版设计的对齐情况

### 7.1 Catalog schema 已对齐当前最终设计

截至本次探查，`catalog` 已完成当前最终设计要求的关键对齐：

- `catalog.video_unit_index` 已使用 `evidence_span_refs`
- 旧 evidence 两列已删除
- `catalog.video_unit_index (coarse_unit_id, video_id)` 索引已存在
- `catalog.video_semantic_spans (coarse_unit_id, video_id, start_ms)` partial index 已存在
- `catalog.videos` 的 recommendable partial index 已存在

因此当前实例里的 `catalog` schema 已与 migration head 一致，也与《Catalog-数据库设计.md》的最终设计一致。

### 7.2 Learning / Recommendation schema 已不存在

当前实例里已经没有：

- `learning` schema
- `recommendation` schema

因此 live DB 当前只承载语义与内容层，不承载 Learning engine 或 Recommendation 的业务表。

## 8. 结论

如果只看 live DB，当前数据库可以概括为：

1. `semantic` 语义主数据已大量落库
2. `catalog` 内容事实链已经有真实数据
3. `public` 当前为空，不承载业务对象
4. `learning` 和 `recommendation` schema 已被清理
5. 当前 `catalog` schema 已与最终设计对齐

因此，当前库最准确的描述是：

> 当前实例已经收口为 `semantic + catalog` 两层结构；学习与推荐层已从 live DB 中移除，而 `catalog` 已经对齐当前最终设计。
