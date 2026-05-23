create index if not exists idx_learning_states_user_target_status_due
on learning.user_unit_states (user_id, is_target, status, next_review_at);

create index if not exists idx_learning_states_user_updated_at
on learning.user_unit_states (user_id, updated_at desc);

create index if not exists idx_learning_events_user_time
on learning.unit_learning_events (user_id, occurred_at, event_id);

create index if not exists idx_learning_events_user_unit_time
on learning.unit_learning_events (user_id, coarse_unit_id, occurred_at, event_id);

create unique index if not exists uq_unit_learning_events_reset_client_event
on learning.unit_learning_events (user_id, source_type, source_ref_id)
where source_type = 'learning_unit_reset';
