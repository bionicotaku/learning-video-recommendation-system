create table if not exists analytics.video_watch_events (
  watch_session_id uuid primary key,

  user_id uuid not null references auth.users(id) on delete cascade,
  video_id uuid not null references catalog.videos(video_id) on delete cascade,

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

create index if not exists idx_video_watch_events_user_video_updated_at
on analytics.video_watch_events (user_id, video_id, updated_at desc);

create index if not exists idx_video_watch_events_user_updated_at
on analytics.video_watch_events (user_id, updated_at desc);

create index if not exists idx_video_watch_events_video_updated_at
on analytics.video_watch_events (video_id, updated_at desc);
