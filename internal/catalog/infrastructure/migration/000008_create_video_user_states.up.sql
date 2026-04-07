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
  first_watched_at timestamptz,
  last_watched_at timestamptz,
  watch_count integer not null default 0,
  completed_count integer not null default 0,
  last_watch_ratio numeric(6,5),
  max_watch_ratio numeric(6,5),
  updated_at timestamptz not null default now(),

  primary key (user_id, video_id),
  check (watch_count >= 0),
  check (completed_count >= 0),
  check (
    last_watch_ratio is null
    or (last_watch_ratio >= 0 and last_watch_ratio <= 1)
  ),
  check (
    max_watch_ratio is null
    or (max_watch_ratio >= 0 and max_watch_ratio <= 1)
  )
);
