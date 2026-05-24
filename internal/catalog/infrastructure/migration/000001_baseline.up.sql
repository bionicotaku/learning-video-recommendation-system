create schema if not exists catalog;

create table if not exists catalog.videos (
  video_id uuid primary key default gen_random_uuid(),
  source_clip_key text not null unique,
  parent_video_name text not null,
  parent_video_slug text not null,
  clip_seq integer,
  source_start_ms integer,
  source_end_ms integer,
  source_start_sentence_index integer,
  source_end_sentence_index integer,
  title text not null,
  description text,
  clip_reason text,
  engagement_score jsonb not null default '{}'::jsonb,
  language text not null default 'en',
  duration_ms integer not null,
  video_object_path text not null,
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
  ),
  check (source_start_sentence_index is null or source_start_sentence_index >= 0),
  check (source_end_sentence_index is null or source_end_sentence_index >= 0),
  check (
    source_start_sentence_index is null
    or source_end_sentence_index is null
    or source_end_sentence_index >= source_start_sentence_index
  ),
  check (jsonb_typeof(engagement_score) = 'object')
);

create table if not exists catalog.video_transcripts (
  video_id uuid primary key
    references catalog.videos(video_id) on delete cascade,
  transcript_object_path text not null,
  transcript_checksum text not null,
  transcript_format_version integer not null default 1,
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

create table if not exists catalog.video_transcript_sentences (
  video_id uuid not null
    references catalog.videos(video_id) on delete cascade,
  sentence_index integer not null,
  start_ms integer not null,
  end_ms integer not null,
  text text not null,
  translation text,
  created_at timestamptz not null default now(),

  primary key (video_id, sentence_index),
  check (sentence_index >= 0),
  check (start_ms >= 0),
  check (end_ms > start_ms)
);

create table if not exists catalog.video_semantic_spans (
  video_id uuid not null,
  sentence_index integer not null,
  span_index integer not null,
  start_ms integer not null,
  end_ms integer not null,
  coarse_unit_id bigint,
  surface_text text not null,
  explanation text,
  base_form text,
  translation text,
  dictionary text,
  mapping_reason text,
  created_at timestamptz not null default now(),

  primary key (video_id, sentence_index, span_index),
  foreign key (video_id, sentence_index)
    references catalog.video_transcript_sentences(video_id, sentence_index)
    on delete cascade,
  foreign key (coarse_unit_id)
    references semantic.coarse_unit(id)
    on delete restrict,
  constraint uq_video_semantic_spans_unit_ref
    unique (video_id, coarse_unit_id, sentence_index, span_index),
  check (span_index >= 0),
  check (start_ms >= 0),
  check (end_ms > start_ms)
);

create table if not exists catalog.video_unit_index (
  video_id uuid not null
    references catalog.videos(video_id) on delete cascade,
  coarse_unit_id bigint not null
    references semantic.coarse_unit(id) on delete cascade,
  mention_count integer not null,
  sentence_count integer not null,
  coverage_ms integer not null,
  coverage_ratio numeric(6,5) not null,
  sentence_indexes integer[] not null default '{}',
  best_evidence_sentence_index integer not null,
  best_evidence_span_index integer not null,
  best_evidence_scores jsonb not null default '{}'::jsonb,
  best_evidence_question_reject_reason text,
  best_evidence_selection_reason text,
  best_evidence_candidate_score numeric(8,4),
  best_evidence_target_text text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  best_evidence_start_ms integer,
  best_evidence_end_ms integer,

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
  check (best_evidence_sentence_index >= 0),
  check (best_evidence_span_index >= 0),
  constraint chk_video_unit_index_best_evidence_bounds check (
    best_evidence_start_ms is null
    or best_evidence_end_ms is null
    or (
      best_evidence_start_ms >= 0
      and best_evidence_end_ms > best_evidence_start_ms
    )
  ),
  constraint chk_video_unit_index_best_evidence_scores_object check (jsonb_typeof(best_evidence_scores) = 'object'),
  constraint chk_video_unit_index_best_evidence_question_reject_reason_nonempty check (
    best_evidence_question_reject_reason is null or best_evidence_question_reject_reason <> ''
  ),
  constraint chk_video_unit_index_best_evidence_selection_reason_nonempty check (
    best_evidence_selection_reason is null or best_evidence_selection_reason <> ''
  ),
  constraint chk_video_unit_index_best_evidence_candidate_score_range check (
    best_evidence_candidate_score is null
    or (best_evidence_candidate_score >= 0 and best_evidence_candidate_score <= 10)
  )
);

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
  like_state_updated_at timestamptz,
  favorite_state_updated_at timestamptz,
  first_watched_at timestamptz,
  last_watched_at timestamptz,
  watch_count integer not null default 0,
  completed_count integer not null default 0,
  last_position_ms integer not null default 0,
  max_position_ms integer not null default 0,
  total_watch_ms bigint not null default 0,
  updated_at timestamptz not null default now(),

  primary key (user_id, video_id),
  check (watch_count >= 0),
  check (completed_count >= 0),
  check (last_position_ms >= 0),
  check (max_position_ms >= 0),
  check (total_watch_ms >= 0)
);

create index if not exists idx_videos_parent_video_slug_clip_seq
on catalog.videos (parent_video_slug, clip_seq);

create index if not exists idx_videos_status
on catalog.videos (status);

create index if not exists idx_videos_visibility_publish_at
on catalog.videos (visibility_status, publish_at);

create index if not exists idx_videos_created_at_desc
on catalog.videos (created_at desc);

create index if not exists idx_video_transcript_sentences_video_start_ms
on catalog.video_transcript_sentences (video_id, start_ms);

create index if not exists idx_video_transcript_sentences_video_end_ms
on catalog.video_transcript_sentences (video_id, end_ms);

create index if not exists idx_video_semantic_spans_video_sentence
on catalog.video_semantic_spans (video_id, sentence_index);

create index if not exists idx_video_semantic_spans_video_start_ms
on catalog.video_semantic_spans (video_id, start_ms);

create index if not exists idx_video_semantic_spans_coarse_unit_video
on catalog.video_semantic_spans (coarse_unit_id, video_id)
where coarse_unit_id is not null;

create index if not exists idx_video_semantic_spans_video_coarse_unit
on catalog.video_semantic_spans (video_id, coarse_unit_id)
where coarse_unit_id is not null;

create index if not exists idx_video_semantic_spans_unit_video_start
on catalog.video_semantic_spans (coarse_unit_id, video_id, start_ms)
where coarse_unit_id is not null;

create index if not exists idx_video_unit_index_coarse_unit_mention_coverage
on catalog.video_unit_index (coarse_unit_id, mention_count desc, coverage_ratio desc);

create index if not exists idx_video_unit_index_unit_video
on catalog.video_unit_index (coarse_unit_id, video_id);

create index if not exists idx_video_unit_index_video_id
on catalog.video_unit_index (video_id);

create index if not exists idx_videos_recommendable
on catalog.videos (publish_at desc, duration_ms)
where status = 'active'
  and visibility_status = 'public';

create index if not exists idx_video_ingestion_records_source_clip_key_started_at
on catalog.video_ingestion_records (source_clip_key, started_at desc);

create index if not exists idx_video_ingestion_records_video_id
on catalog.video_ingestion_records (video_id);

create index if not exists idx_video_ingestion_records_status_started_at
on catalog.video_ingestion_records (status, started_at desc);

create index if not exists idx_video_user_states_video_id
on catalog.video_user_states (video_id);

create index if not exists idx_video_user_states_favorites_page
on catalog.video_user_states (user_id, bookmarked_at desc, video_id asc)
where has_bookmarked = true and bookmarked_at is not null;

create index if not exists idx_video_user_states_history_page
on catalog.video_user_states (user_id, last_watched_at desc, video_id asc)
where has_watched = true and last_watched_at is not null;

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

create table if not exists catalog.word_favorites (
  favorite_id uuid primary key default gen_random_uuid(),
  user_id uuid not null
    references auth.users(id) on delete cascade,
  favorite_key_type text not null
    check (favorite_key_type in ('coarse_unit', 'video_token')),
  coarse_unit_id bigint,
  source text not null
    check (source in ('word_list', 'video_transcript')),
  video_id uuid,
  sentence_index integer,
  token_index integer,
  is_favorited boolean not null default false,
  favorited_at timestamptz,
  state_updated_at timestamptz not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),

  check (
    (
      favorite_key_type = 'coarse_unit'
      and coarse_unit_id is not null
    )
    or
    (
      favorite_key_type = 'video_token'
      and coarse_unit_id is null
      and video_id is not null
      and sentence_index is not null
      and token_index is not null
    )
  ),
  check (
    source = 'word_list'
    or
    (
      source = 'video_transcript'
      and video_id is not null
      and sentence_index is not null
      and token_index is not null
    )
  ),
  check (sentence_index is null or sentence_index >= 0),
  check (token_index is null or token_index >= 0),
  check (
    (
      is_favorited = true
      and favorited_at is not null
    )
    or
    (
      is_favorited = false
      and favorited_at is null
    )
  )
);

create unique index if not exists uq_word_favorites_coarse_unit
on catalog.word_favorites (user_id, coarse_unit_id)
where favorite_key_type = 'coarse_unit';

create unique index if not exists uq_word_favorites_video_token
on catalog.word_favorites (user_id, video_id, sentence_index, token_index)
where favorite_key_type = 'video_token';

create index if not exists idx_word_favorites_page
on catalog.word_favorites (user_id, favorited_at desc, favorite_id asc)
where is_favorited = true and favorited_at is not null;
