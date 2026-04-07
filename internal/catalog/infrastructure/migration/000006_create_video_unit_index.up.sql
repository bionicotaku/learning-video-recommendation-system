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
  evidence_sentence_indexes integer[] not null default '{}',
  evidence_span_indexes integer[] not null default '{}',
  sample_surface_forms text[] not null default '{}',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),

  primary key (video_id, coarse_unit_id),
  check (mention_count > 0),
  check (sentence_count > 0),
  check (coverage_ms > 0),
  check (coverage_ratio >= 0 and coverage_ratio <= 1),
  check (last_end_ms > first_start_ms)
);
