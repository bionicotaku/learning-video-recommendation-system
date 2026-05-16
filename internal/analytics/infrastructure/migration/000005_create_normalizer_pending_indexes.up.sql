create index if not exists idx_quiz_events_completed_at_event_id
  on analytics.quiz_events (completed_at, event_id);

create index if not exists idx_learning_interaction_events_pending_normalizer
  on analytics.learning_interaction_events (occurred_at, event_id)
  where coarse_unit_id is not null
    and event_type in ('exposure', 'lookup', 'self_mark_mastered');
