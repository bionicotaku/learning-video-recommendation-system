create schema if not exists auth;
create schema if not exists semantic;
create schema if not exists analytics;

create table if not exists auth.users (
  id uuid primary key,
  email text,
  email_confirmed_at timestamptz
);

create sequence if not exists semantic.coarse_unit_id_seq;

create table if not exists semantic.coarse_unit (
  id bigint primary key default nextval('semantic.coarse_unit_id_seq'::regclass),
  kind text not null default 'word',
  label text not null default '',
  pos text,
  chinese_def text,
  chinese_label text,
  status text not null default 'active'
);

create table if not exists analytics.video_watch_events (
  watch_session_id uuid primary key,
  user_id uuid not null references auth.users(id) on delete cascade,
  video_id uuid not null,
  started_at timestamptz not null,
  last_seen_at timestamptz not null,
  completed_at timestamptz,
  last_position_ms integer not null default 0,
  max_position_ms integer not null default 0,
  active_watch_ms bigint not null default 0,
  is_completed boolean not null default false,
  progress_report_count integer not null default 0,
  client_context jsonb not null default '{}'::jsonb,
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),

  check (last_position_ms >= 0),
  check (max_position_ms >= 0),
  check (active_watch_ms >= 0),
  check (progress_report_count >= 0),
  check (jsonb_typeof(client_context) = 'object'),
  check (jsonb_typeof(metadata) = 'object')
);
