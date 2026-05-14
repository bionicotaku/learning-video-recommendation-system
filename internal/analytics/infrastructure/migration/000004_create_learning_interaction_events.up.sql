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
