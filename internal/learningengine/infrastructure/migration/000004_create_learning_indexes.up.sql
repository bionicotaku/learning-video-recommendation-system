create index if not exists idx_user_unit_states_target_status
on learning.user_unit_states (user_id, is_target, status);

create index if not exists idx_user_unit_states_next_review
on learning.user_unit_states (user_id, next_review_at);

create index if not exists idx_unit_learning_events_user_unit_time
on learning.unit_learning_events (user_id, coarse_unit_id, occurred_at desc);

create index if not exists idx_unit_learning_events_user_video_time
on learning.unit_learning_events (user_id, video_id, occurred_at desc);
