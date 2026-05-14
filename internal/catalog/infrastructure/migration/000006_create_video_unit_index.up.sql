create table if not exists catalog.video_unit_index (
  video_id uuid not null
    references catalog.videos(video_id) on delete cascade,
  coarse_unit_id bigint not null
    references semantic.coarse_unit(id) on delete cascade,
  mention_count integer not null,
  sentence_count integer not null,
  first_start_ms integer not null,
  last_end_ms integer not null,
  coverage_ms integer not null,
  coverage_ratio numeric(6,5) not null,
  sentence_indexes integer[] not null default '{}',
  best_evidence_sentence_index integer not null,
  best_evidence_span_index integer not null,
  best_evidence_source text not null,
  best_evidence_model text,
  best_evidence_version integer not null default 1,
  best_evidence_metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),

  primary key (video_id, coarse_unit_id),
  foreign key (
    video_id,
    coarse_unit_id,
    best_evidence_sentence_index,
    best_evidence_span_index
  )
    references catalog.video_semantic_spans (
      video_id,
      coarse_unit_id,
      sentence_index,
      span_index
    )
    on delete cascade,
  check (mention_count > 0),
  check (sentence_count > 0),
  check (coverage_ms > 0),
  check (coverage_ratio >= 0 and coverage_ratio <= 1),
  check (last_end_ms > first_start_ms),
  check (best_evidence_sentence_index >= 0),
  check (best_evidence_span_index >= 0),
  check (best_evidence_source <> ''),
  check (best_evidence_version > 0)
);
