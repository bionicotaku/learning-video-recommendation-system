create table if not exists catalog.videos (
  video_id uuid primary key default gen_random_uuid(),
  source_clip_key text not null unique,
  parent_video_name text not null,
  parent_video_slug text not null,
  clip_seq integer,
  source_start_ms integer,
  source_end_ms integer,
  title text not null,
  description text,
  clip_reason text,
  language text not null default 'en',
  duration_ms integer not null,
  hls_master_playlist_path text not null,
  thumbnail_url text,
  status text not null default 'active'
    check (status in ('active', 'inactive', 'deleted')),
  visibility_status text not null default 'public'
    check (visibility_status in ('public', 'unlisted', 'private')),
  publish_at timestamptz,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),

  check (duration_ms > 0),
  check (
    source_end_ms is null
    or source_start_ms is null
    or source_end_ms > source_start_ms
  )
);
