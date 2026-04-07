create table if not exists catalog.video_transcript_sentences (
  video_id uuid not null
    references catalog.videos(video_id) on delete cascade,
  sentence_index integer not null,
  text text not null,
  start_ms integer not null,
  end_ms integer not null,
  explanation text,
  created_at timestamptz not null default now(),

  primary key (video_id, sentence_index),
  check (sentence_index >= 0),
  check (start_ms >= 0),
  check (end_ms > start_ms)
);
