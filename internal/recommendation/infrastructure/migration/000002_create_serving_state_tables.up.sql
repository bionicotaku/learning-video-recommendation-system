create table if not exists recommendation.user_unit_serving_states (
  user_id uuid not null references auth.users(id) on delete cascade,
  coarse_unit_id bigint not null references semantic.coarse_unit(id) on delete cascade,
  last_served_at timestamptz,
  last_run_id uuid,
  served_count integer not null default 0,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),

  primary key (user_id, coarse_unit_id),
  check (served_count >= 0)
);

create table if not exists recommendation.user_video_serving_states (
  user_id uuid not null references auth.users(id) on delete cascade,
  video_id uuid not null references catalog.videos(video_id) on delete cascade,
  last_served_at timestamptz,
  last_run_id uuid,
  served_count integer not null default 0,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),

  primary key (user_id, video_id),
  check (served_count >= 0)
);
