create table if not exists catalog.video_ingestion_records (
  ingestion_record_id uuid primary key,
  source_clip_key text not null,
  video_id uuid references catalog.videos(video_id) on delete set null,
  source_name text,
  status text not null
    check (status in ('running', 'succeeded', 'failed', 'skipped')),
  warning_codes text[] not null default '{}',
  error_code text,
  error_message text,
  context jsonb not null default '{}'::jsonb,
  started_at timestamptz not null,
  finished_at timestamptz,
  created_at timestamptz not null default now()
);
