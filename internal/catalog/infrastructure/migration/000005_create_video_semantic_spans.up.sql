create table if not exists catalog.video_semantic_spans (
  video_id uuid not null,
  sentence_index integer not null,
  span_index integer not null,
  text text not null,
  start_ms integer not null,
  end_ms integer not null,
  explanation text,
  coarse_unit_id bigint,
  base_form text,
  dictionary_text text,
  created_at timestamptz not null default now(),

  primary key (video_id, sentence_index, span_index),
  foreign key (video_id, sentence_index)
    references catalog.video_transcript_sentences(video_id, sentence_index)
    on delete cascade,
  foreign key (coarse_unit_id)
    references semantic.coarse_unit(id)
    on delete restrict,
  check (span_index >= 0),
  check (start_ms >= 0),
  check (end_ms > start_ms)
);
