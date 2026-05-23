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
