create index if not exists idx_learning_states_user_target_status_due
on learning.user_unit_states (user_id, is_target, status, next_review_at);

create index if not exists idx_learning_states_user_updated_at
on learning.user_unit_states (user_id, updated_at desc);

create index if not exists idx_user_unit_states_active_unit_ids
on learning.user_unit_states (user_id, coarse_unit_id)
where is_target = true
  and status in ('new', 'learning', 'reviewing');

create index if not exists idx_user_unit_states_unmastered_progress
on learning.user_unit_states (user_id, progress_percent desc, coarse_unit_id)
where is_target = true
  and status in ('new', 'learning', 'reviewing');

create index if not exists idx_user_unit_states_mastered_progress
on learning.user_unit_states (user_id, coarse_unit_id)
where status = 'mastered';

create index if not exists idx_user_unit_states_active_collection_targets
on learning.user_unit_states (user_id, target_source, coarse_unit_id)
where is_target = true;

create unique index if not exists uq_unit_learning_events_ledger_seq
on learning.unit_learning_events (ledger_seq);

create index if not exists idx_learning_events_user_ledger_seq
on learning.unit_learning_events (user_id, ledger_seq);

create index if not exists idx_learning_events_user_unit_ledger_seq
on learning.unit_learning_events (user_id, coarse_unit_id, ledger_seq);

create index if not exists idx_learning_events_reset_boundary
on learning.unit_learning_events (user_id, coarse_unit_id, reset_boundary_at desc, ledger_seq desc)
where source_type = 'learning_unit_reset'
  and event_type = 'reset_unlearned'
  and reducer_effect = 'reset_unlearned'
  and reset_boundary_at is not null;

create unique index if not exists uq_unit_learning_events_reset_client_event
on learning.unit_learning_events (user_id, source_type, source_ref_id)
where source_type = 'learning_unit_reset';
