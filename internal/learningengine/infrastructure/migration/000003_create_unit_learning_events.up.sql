create table if not exists learning.unit_learning_events (
  event_id bigint generated always as identity primary key,
  user_id uuid not null references auth.users(id) on delete cascade,
  coarse_unit_id bigint not null references semantic.coarse_unit(id) on delete cascade,
  video_id uuid references catalog.videos(video_id) on delete set null,

  event_type text not null,
  source_type text not null,
  source_ref_id text,

  is_correct boolean,
  quality smallint check (quality between 0 and 5),
  response_time_ms integer,

  metadata jsonb not null default '{}'::jsonb,
  occurred_at timestamptz not null,
  created_at timestamptz not null default now()
);
