create schema if not exists auth;
create schema if not exists semantic;
create schema if not exists catalog;
create schema if not exists learning;
create schema if not exists recommendation;

create table if not exists auth.users (
  id uuid primary key
);

create table if not exists semantic.coarse_unit (
  id bigint primary key
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
  mapped_span_ratio numeric(6,5) not null
);

create table if not exists catalog.video_unit_index (
  video_id uuid not null,
  coarse_unit_id bigint not null,
  mention_count integer not null,
  sentence_count integer not null,
  first_start_ms integer not null,
  last_end_ms integer not null,
  coverage_ms integer not null,
  coverage_ratio numeric(6,5) not null,
  sentence_indexes integer[] not null,
  evidence_span_refs jsonb not null,
  sample_surface_forms text[] not null
);

create table if not exists catalog.video_semantic_spans (
  video_id uuid not null,
  sentence_index integer not null,
  span_index integer not null,
  coarse_unit_id bigint,
  start_ms integer not null,
  end_ms integer not null,
  text text not null,
  explanation text
);

create table if not exists catalog.video_transcript_sentences (
  video_id uuid not null,
  sentence_index integer not null,
  text text not null,
  start_ms integer not null,
  end_ms integer not null,
  explanation text
);

create table if not exists catalog.video_user_states (
  user_id uuid not null,
  video_id uuid not null,
  last_watched_at timestamptz,
  watch_count integer not null default 0,
  completed_count integer not null default 0,
  last_watch_ratio numeric(6,5),
  max_watch_ratio numeric(6,5)
);

create table if not exists learning.user_unit_states (
  user_id uuid not null,
  coarse_unit_id bigint not null,
  is_target boolean not null default true,
  target_priority numeric(5,4) not null default 0.5,
  status text not null default 'new',
  progress_percent numeric(5,2) not null default 0,
  mastery_score numeric(5,4) not null default 0,
  last_quality smallint,
  next_review_at timestamptz,
  recent_quality_window smallint[] not null default '{}',
  recent_correctness_window boolean[] not null default '{}',
  strong_event_count integer not null default 0,
  review_count integer not null default 0,
  updated_at timestamptz not null default now()
);

create materialized view if not exists recommendation.v_recommendable_video_units as
select
  null::uuid as video_id,
  null::bigint as coarse_unit_id,
  null::integer as mention_count,
  null::integer as sentence_count,
  null::integer as first_start_ms,
  null::integer as last_end_ms,
  null::integer as coverage_ms,
  null::numeric(6,5) as coverage_ratio,
  '{}'::integer[] as sentence_indexes,
  '[]'::jsonb as evidence_span_refs,
  '{}'::text[] as sample_surface_forms,
  null::integer as duration_ms,
  null::numeric(6,5) as mapped_span_ratio,
  null::text as status,
  null::text as visibility_status,
  null::timestamptz as publish_at
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
