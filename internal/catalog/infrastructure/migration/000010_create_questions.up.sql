create table if not exists catalog.questions (
  question_id uuid primary key default gen_random_uuid(),

  scope_type text not null
    check (scope_type in ('unit', 'video_unit')),

  question_type text not null
    check (question_type in (
      'context_meaning_choice',
      'unit_meaning_choice',
      'context_cloze_choice',
      'reverse_identification_choice'
    )),

  coarse_unit_id bigint not null
    references semantic.coarse_unit(id) on delete restrict,

  target_text text not null,

  video_id uuid
    references catalog.videos(video_id) on delete cascade,

  context_sentence_index integer,
  context_span_index integer,
  context_start_ms integer,
  context_end_ms integer,

  content_payload jsonb not null,

  status text not null default 'active'
    check (status in ('draft', 'active', 'retired', 'rejected')),

  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),

  check (jsonb_typeof(content_payload) = 'object'),
  check (
    (scope_type = 'unit' and video_id is null)
    or
    (scope_type = 'video_unit' and video_id is not null)
  ),
  check (
    context_start_ms is null
    or context_end_ms is null
    or context_end_ms > context_start_ms
  )
);

create index if not exists idx_questions_video_unit_active
on catalog.questions (video_id, coarse_unit_id, question_type, created_at desc)
where scope_type = 'video_unit' and status = 'active';

create index if not exists idx_questions_unit_active
on catalog.questions (coarse_unit_id, question_type, created_at desc)
where scope_type = 'unit' and status = 'active';

create index if not exists idx_questions_status_created_at
on catalog.questions (status, created_at desc);
