create table if not exists learning.unit_learning_events (
  event_id uuid primary key default gen_random_uuid(),
  user_id uuid not null references auth.users(id) on delete cascade,
  coarse_unit_id bigint not null references semantic.coarse_unit(id) on delete cascade,
  video_id uuid references catalog.videos(video_id) on delete set null,

  event_type text not null,
  reducer_effect text not null,
  progress_quality smallint,

  source_type text not null,
  source_ref_id text not null,
  is_correct boolean,

  counts_toward_success_streak boolean not null default false,
  consumed_watch_session_ids uuid[] not null default '{}'::uuid[],

  metadata jsonb not null default '{}'::jsonb
    check (jsonb_typeof(metadata) = 'object'),
  occurred_at timestamptz not null,
  created_at timestamptz not null default now(),

  constraint chk_unit_learning_events_event_type
    check (event_type in ('exposure', 'lookup', 'quiz', 'self_mark_mastered', 'reset_unlearned')),
  constraint chk_unit_learning_events_reducer_effect
    check (reducer_effect in ('observe_only', 'affects_progress', 'set_mastered', 'reset_unlearned')),
  constraint chk_unit_learning_events_progress_quality
    check (
      (reducer_effect = 'affects_progress' and progress_quality between 0 and 5)
      or
      (reducer_effect = 'observe_only' and progress_quality is null)
      or
      (reducer_effect = 'set_mastered' and progress_quality is null)
      or
      (reducer_effect = 'reset_unlearned' and progress_quality is null)
    ),
  constraint chk_unit_learning_events_set_mastered_event_type
    check (
      reducer_effect <> 'set_mastered'
      or event_type = 'self_mark_mastered'
    ),
  constraint chk_unit_learning_events_reset_unlearned_event_type
    check (
      reducer_effect <> 'reset_unlearned'
      or event_type = 'reset_unlearned'
    ),
  constraint chk_unit_learning_events_success_streak_effect
    check (
      reducer_effect = 'affects_progress'
      or counts_toward_success_streak = false
    ),
  constraint chk_unit_learning_events_exposure_session3_consumed_sessions
    check (
      (
        source_type = 'exposure_session3_v1'
        and event_type = 'exposure'
        and reducer_effect = 'affects_progress'
        and progress_quality = 4
        and counts_toward_success_streak = false
        and cardinality(consumed_watch_session_ids) = 3
        and array_position(consumed_watch_session_ids, null) is null
      )
      or
      (
        source_type <> 'exposure_session3_v1'
        and cardinality(consumed_watch_session_ids) = 0
      )
    ),
  constraint uq_unit_learning_events_source_unit
    unique (user_id, source_type, source_ref_id, coarse_unit_id)
);
