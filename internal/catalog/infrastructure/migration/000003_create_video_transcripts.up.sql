create table if not exists catalog.video_transcripts (
  video_id uuid primary key
    references catalog.videos(video_id) on delete cascade,
  transcript_object_path text not null,
  transcript_checksum text not null,
  transcript_format_version integer not null default 1,
  full_text text not null,
  sentence_count integer not null,
  semantic_span_count integer not null,
  mapped_span_count integer not null,
  unmapped_span_count integer not null,
  mapped_span_ratio numeric(6,5) not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),

  check (sentence_count >= 0),
  check (semantic_span_count >= 0),
  check (mapped_span_count >= 0),
  check (unmapped_span_count >= 0),
  check (mapped_span_ratio >= 0 and mapped_span_ratio <= 1)
);
