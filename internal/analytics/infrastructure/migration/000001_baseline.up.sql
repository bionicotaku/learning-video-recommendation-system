create schema if not exists analytics;

create table if not exists analytics.quiz_events (
  event_id uuid primary key default gen_random_uuid(),

  client_event_id text not null,

  user_id uuid not null
    references auth.users(id) on delete cascade,

  client_context jsonb not null default '{}'::jsonb,

  question_id uuid not null
    references catalog.questions(question_id) on delete restrict,

  coarse_unit_id bigint not null
    references semantic.coarse_unit(id) on delete restrict,

  video_id uuid
    references catalog.videos(video_id) on delete set null,

  recommendation_run_id uuid,

  trigger_type text not null
    check (trigger_type in (
      'video_end',
      'lookup_practice',
      'feed_review',
      'mid_video',
      'manual'
    )),

  selected_option_ids text[] not null,
  selection_interval_ms integer[] not null,

  is_first_try_correct boolean not null,
  total_elapsed_ms integer not null,

  shown_at timestamptz not null,
  completed_at timestamptz not null,

  created_at timestamptz not null default now(),

  check (cardinality(selected_option_ids) >= 1),
  check (cardinality(selected_option_ids) = cardinality(selection_interval_ms)),
  check (selected_option_ids[cardinality(selected_option_ids)] = 'correct'),
  check (is_first_try_correct = (selected_option_ids[1] = 'correct')),
  check (total_elapsed_ms >= 0),
  check (completed_at >= shown_at),
  check (jsonb_typeof(client_context) = 'object')
);

create unique index if not exists uq_quiz_events_user_client_event
on analytics.quiz_events (user_id, client_event_id);

create index if not exists idx_quiz_events_user_completed_at
on analytics.quiz_events (user_id, completed_at desc);

create index if not exists idx_quiz_events_question_completed_at
on analytics.quiz_events (question_id, completed_at desc);

create index if not exists idx_quiz_events_unit_completed_at
on analytics.quiz_events (coarse_unit_id, completed_at desc);

create index if not exists idx_quiz_events_video_completed_at
on analytics.quiz_events (video_id, completed_at desc)
where video_id is not null;

create table if not exists analytics.video_watch_events (
  watch_session_id uuid primary key,

  user_id uuid not null references auth.users(id) on delete cascade,
  video_id uuid not null references catalog.videos(video_id) on delete cascade,

  started_at timestamptz not null,
  last_seen_at timestamptz not null,
  completed_at timestamptz,

  last_position_ms integer not null default 0,
  max_position_ms integer not null default 0,
  active_watch_ms bigint not null default 0,
  is_completed boolean not null default false,

  progress_report_count integer not null default 0,
  client_context jsonb not null default '{}'::jsonb,
  metadata jsonb not null default '{}'::jsonb,

  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),

  check (last_position_ms >= 0),
  check (max_position_ms >= 0),
  check (active_watch_ms >= 0),
  check (progress_report_count >= 0),
  check (jsonb_typeof(client_context) = 'object'),
  check (jsonb_typeof(metadata) = 'object')
);

create index if not exists idx_video_watch_events_user_video_updated_at
on analytics.video_watch_events (user_id, video_id, updated_at desc);

create index if not exists idx_video_watch_events_user_updated_at
on analytics.video_watch_events (user_id, updated_at desc);

create index if not exists idx_video_watch_events_video_updated_at
on analytics.video_watch_events (video_id, updated_at desc);

create table if not exists analytics.learning_interaction_events (
  event_id uuid primary key default gen_random_uuid(),

  client_event_id text not null,

  user_id uuid not null
    references auth.users(id) on delete cascade,

  client_context jsonb not null default '{}'::jsonb,

  event_type text not null
    check (event_type in ('exposure', 'lookup', 'self_mark_mastered')),

  source_surface text not null,

  video_id uuid
    references catalog.videos(video_id) on delete set null,

  watch_session_id uuid
    references analytics.video_watch_events(watch_session_id) on delete set null,

  recommendation_run_id uuid,

  related_quiz_event_id uuid
    references analytics.quiz_events(event_id) on delete set null,

  coarse_unit_id bigint
    references semantic.coarse_unit(id) on delete set null,

  token_text text,
  sentence_index integer,
  span_index integer,

  occurred_at timestamptz not null,

  exposure_start_ms integer,
  exposure_end_ms integer,
  exposure_count integer,

  lookup_visible_ms integer,
  lookup_sentence_audio_replay_count integer not null default 0,
  lookup_word_audio_play_count integer not null default 0,
  lookup_practice_now_clicked boolean not null default false,

  event_payload jsonb not null default '{}'::jsonb,

  created_at timestamptz not null default now(),

  check (jsonb_typeof(client_context) = 'object'),
  check (jsonb_typeof(event_payload) = 'object'),
  check (exposure_start_ms is null or exposure_start_ms >= 0),
  check (exposure_end_ms is null or exposure_end_ms >= 0),
  check (
    exposure_start_ms is null
    or exposure_end_ms is null
    or exposure_end_ms >= exposure_start_ms
  ),
  check (exposure_count is null or exposure_count >= 1),
  check (lookup_visible_ms is null or lookup_visible_ms >= 0),
  check (lookup_sentence_audio_replay_count >= 0),
  check (lookup_word_audio_play_count >= 0)
);

create unique index if not exists uq_learning_interaction_events_user_client_event
on analytics.learning_interaction_events (user_id, client_event_id);

create index if not exists idx_learning_interaction_events_user_occurred_at
on analytics.learning_interaction_events (user_id, occurred_at desc);

create index if not exists idx_learning_interaction_events_user_unit_occurred_at
on analytics.learning_interaction_events (user_id, coarse_unit_id, occurred_at desc)
where coarse_unit_id is not null;

create index if not exists idx_learning_interaction_events_video_occurred_at
on analytics.learning_interaction_events (video_id, occurred_at desc)
where video_id is not null;

create index if not exists idx_learning_interaction_events_watch_session
on analytics.learning_interaction_events (watch_session_id, occurred_at asc)
where watch_session_id is not null;

create index if not exists idx_learning_interaction_events_related_quiz
on analytics.learning_interaction_events (related_quiz_event_id)
where related_quiz_event_id is not null;

create index if not exists idx_quiz_events_completed_at_event_id
  on analytics.quiz_events (completed_at, event_id);

create index if not exists idx_learning_interaction_events_pending_normalizer
  on analytics.learning_interaction_events (occurred_at, event_id)
  where coarse_unit_id is not null
    and event_type in ('exposure', 'lookup', 'self_mark_mastered');

create index if not exists idx_learning_interaction_events_exposure_session
on analytics.learning_interaction_events (
  user_id,
  coarse_unit_id,
  watch_session_id,
  occurred_at,
  event_id
)
where event_type = 'exposure'
  and coarse_unit_id is not null
  and watch_session_id is not null;

create index if not exists idx_learning_interaction_events_lookup_unit_time
on analytics.learning_interaction_events (
  user_id,
  coarse_unit_id,
  occurred_at desc,
  event_id
)
where event_type = 'lookup'
  and coarse_unit_id is not null;
