create table if not exists catalog.video_user_states (
  user_id uuid not null
    references auth.users(id) on delete cascade,
  video_id uuid not null
    references catalog.videos(video_id) on delete cascade,
  has_liked boolean not null default false,
  has_bookmarked boolean not null default false,
  has_watched boolean not null default false,
  liked_at timestamptz,
  bookmarked_at timestamptz,
  like_state_updated_at timestamptz,
  favorite_state_updated_at timestamptz,
  first_watched_at timestamptz,
  last_watched_at timestamptz,
  watch_count integer not null default 0,
  completed_count integer not null default 0,
  last_position_ms integer not null default 0,
  max_position_ms integer not null default 0,
  total_watch_ms bigint not null default 0,
  updated_at timestamptz not null default now(),

  primary key (user_id, video_id),
  check (watch_count >= 0),
  check (completed_count >= 0),
  check (last_position_ms >= 0),
  check (max_position_ms >= 0),
  check (total_watch_ms >= 0)
);
