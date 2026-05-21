create schema if not exists auth;
create schema if not exists semantic;
create schema if not exists catalog;
create schema if not exists learning;
create schema if not exists recommendation;

create table if not exists auth.users (
  id uuid primary key,
  email text,
  email_confirmed_at timestamptz
);

create sequence if not exists semantic.coarse_unit_id_seq;

create table if not exists semantic.coarse_unit (
  id bigint primary key default nextval('semantic.coarse_unit_id_seq'::regclass),
  kind text not null,
  label text not null,
  lang text not null default 'en',
  pos text,
  english_def text,
  chinese_def text,
  chinese_criteria text,
  chinese_label text,
  english_label text,
  pattern jsonb,
  status text not null default 'active',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  version integer not null default 1,
  fine_unit_ids bigint[] not null,
  original_defs text[] not null
);

create table if not exists catalog.videos (
  video_id uuid primary key,
  duration_ms integer not null default 0,
  status text not null default 'active',
  visibility_status text not null default 'public',
  publish_at timestamptz
);

create table if not exists catalog.video_transcripts (
  video_id uuid primary key,
  transcript_object_path text,
  transcript_checksum text not null default '',
  transcript_format_version integer not null default 1,
  sentence_count integer not null default 0,
  semantic_span_count integer not null default 0,
  mapped_span_count integer not null default 0,
  unmapped_span_count integer not null default 0,
  mapped_span_ratio numeric(6,5) not null
);

create table if not exists catalog.video_unit_index (
  video_id uuid not null,
  coarse_unit_id bigint not null,
  mention_count integer not null,
  sentence_count integer not null,
  coverage_ms integer not null,
  coverage_ratio numeric(6,5) not null,
  sentence_indexes integer[] not null,
  best_evidence_sentence_index integer not null,
  best_evidence_span_index integer not null,
  best_evidence_start_ms integer,
  best_evidence_end_ms integer,
  best_evidence_scores jsonb not null,
  best_evidence_question_reject_reason text,
  best_evidence_selection_reason text,
  best_evidence_candidate_score numeric(8,4),
  best_evidence_target_text text
);

create table if not exists catalog.video_semantic_spans (
  video_id uuid not null,
  sentence_index integer not null,
  span_index integer not null,
  coarse_unit_id bigint,
  start_ms integer not null,
  end_ms integer not null,
  surface_text text not null,
  explanation text,
  base_form text,
  translation text,
  dictionary text,
  mapping_reason text
);

create table if not exists catalog.video_transcript_sentences (
  video_id uuid not null,
  sentence_index integer not null,
  start_ms integer not null,
  end_ms integer not null,
  text text not null,
  translation text
);

create table if not exists catalog.video_user_states (
  user_id uuid not null,
  video_id uuid not null,
  last_watched_at timestamptz,
  watch_count integer not null default 0,
  completed_count integer not null default 0,
  last_position_ms integer not null default 0,
  max_position_ms integer not null default 0,
  total_watch_ms bigint not null default 0
);

create table if not exists catalog.video_engagement_stats (
  video_id uuid primary key,
  view_count bigint not null default 0,
  like_count bigint not null default 0,
  favorite_count bigint not null default 0,
  completed_count bigint not null default 0,
  total_watch_ms bigint not null default 0,
  updated_at timestamptz not null default now()
);

create table if not exists learning.user_unit_states (
  user_id uuid not null,
  coarse_unit_id bigint not null,
  is_target boolean not null default false,
  target_priority numeric(8,4) not null default 0,
  status text not null default 'new',
  mastery_score numeric(5,4) not null default 0,
  last_progress_quality smallint,
  next_review_at timestamptz,
  updated_at timestamptz not null default now()
);

create materialized view if not exists recommendation.v_video_unit_recall_index as
select
  null::uuid as video_id,
  null::bigint as coarse_unit_id,
  null::integer as mention_count,
  null::integer as sentence_count,
  null::integer as coverage_ms,
  null::numeric(6,5) as coverage_ratio,
  '{}'::integer[] as sentence_indexes,
  null::integer as best_evidence_sentence_index,
  null::integer as best_evidence_span_index,
  null::integer as best_evidence_start_ms,
  null::integer as best_evidence_end_ms,
  '{}'::jsonb as best_evidence_scores,
  null::text as best_evidence_question_reject_reason,
  null::text as best_evidence_selection_reason,
  null::numeric(8,4) as best_evidence_candidate_score,
  null::text as best_evidence_target_text,
  null::integer as duration_ms,
  null::numeric(6,5) as mapped_span_ratio,
  null::text as status,
  null::text as visibility_status,
  null::timestamptz as publish_at,
  null::numeric(10,6) as content_quality_score,
  null::integer as rank_within_unit
where false;

create materialized view if not exists recommendation.v_unit_video_inventory as
select
  null::bigint as coarse_unit_id,
  null::integer as distinct_video_count,
  null::numeric(10,4) as avg_mention_count,
  null::numeric(10,4) as avg_sentence_count,
  null::numeric(12,4) as avg_coverage_ms,
  null::numeric(10,5) as avg_coverage_ratio,
  null::integer as strong_video_count,
  null::text as supply_grade,
  null::timestamptz as updated_at
where false;

create table if not exists recommendation.recall_projection_metadata (
  projection_name text primary key,
  projection_updated_at timestamptz not null default now()
);

create table if not exists recommendation.user_unit_recall_queue (
  user_id uuid not null,
  coarse_unit_id bigint not null,
  status text not null,
  target_priority numeric(8,4) not null default 0,
  mastery_score numeric(5,4) not null default 0,
  last_progress_quality smallint,
  next_review_at timestamptz,
  supply_grade text not null default 'none',
  state_updated_at timestamptz not null,
  source_version text not null,
  rebuilt_at timestamptz not null default now(),
  primary key (user_id, coarse_unit_id)
);

create table if not exists recommendation.user_unit_recall_queue_states (
  user_id uuid primary key,
  source_learning_max_updated_at timestamptz,
  source_projection_updated_at timestamptz not null,
  active_target_unit_count integer not null default 0,
  rebuilt_at timestamptz not null default now()
);

create table if not exists recommendation.user_unit_serving_states (
  user_id uuid not null,
  coarse_unit_id bigint not null,
  last_served_at timestamptz,
  last_run_id uuid,
  served_count integer not null default 0,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  primary key (user_id, coarse_unit_id)
);
