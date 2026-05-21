create index if not exists idx_user_unit_states_active_target_priority
on learning.user_unit_states (
  user_id,
  target_priority desc,
  coarse_unit_id
)
where is_target = true
  and status in ('new', 'learning', 'reviewing');

create index if not exists idx_user_unit_states_mastered_target
on learning.user_unit_states (
  user_id,
  coarse_unit_id
)
where is_target = true
  and status = 'mastered';
