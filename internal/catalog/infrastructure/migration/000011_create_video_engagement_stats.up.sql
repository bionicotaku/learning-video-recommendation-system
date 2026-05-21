create table if not exists catalog.video_engagement_stats (
  video_id uuid primary key
    references catalog.videos(video_id) on delete cascade,
  view_count bigint not null default 0,
  like_count bigint not null default 0,
  favorite_count bigint not null default 0,
  completed_count bigint not null default 0,
  total_watch_ms bigint not null default 0,
  updated_at timestamptz not null default now(),

  check (view_count >= 0),
  check (like_count >= 0),
  check (favorite_count >= 0),
  check (completed_count >= 0),
  check (total_watch_ms >= 0)
);

create index if not exists idx_video_engagement_stats_popularity
on catalog.video_engagement_stats (
  view_count desc,
  like_count desc,
  favorite_count desc,
  video_id
);
