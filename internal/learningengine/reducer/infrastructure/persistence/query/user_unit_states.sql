-- name: GetUserUnitStateForUpdate :one
select *
from learning.user_unit_states
where user_id = sqlc.arg(user_id)
  and coarse_unit_id = sqlc.arg(coarse_unit_id)
for update;

-- name: UpsertUserUnitState :one
insert into learning.user_unit_states (
  user_id,
  coarse_unit_id,
  is_target,
  target_source,
  target_source_ref_id,
  target_priority,
  status,
  progress_percent,
  mastery_score,
  first_observed_at,
  last_observed_at,
  observation_count,
  progress_event_count,
  last_progress_at,
  last_progress_quality,
  recent_progress_qualities,
  recent_progress_passes,
  progress_success_count,
  progress_failure_count,
  consecutive_success_count,
  consecutive_failure_count,
  schedule_repetition,
  schedule_interval_days,
  schedule_ease_factor,
  next_review_at,
  suspended_reason,
  updated_at
) values (
  sqlc.arg(user_id),
  sqlc.arg(coarse_unit_id),
  sqlc.arg(is_target),
  sqlc.narg(target_source),
  sqlc.narg(target_source_ref_id),
  sqlc.arg(target_priority),
  sqlc.arg(status),
  sqlc.arg(progress_percent),
  sqlc.arg(mastery_score),
  sqlc.narg(first_observed_at),
  sqlc.narg(last_observed_at),
  sqlc.arg(observation_count),
  sqlc.arg(progress_event_count),
  sqlc.narg(last_progress_at),
  sqlc.narg(last_progress_quality),
  sqlc.arg(recent_progress_qualities),
  sqlc.arg(recent_progress_passes),
  sqlc.arg(progress_success_count),
  sqlc.arg(progress_failure_count),
  sqlc.arg(consecutive_success_count),
  sqlc.arg(consecutive_failure_count),
  sqlc.arg(schedule_repetition),
  sqlc.arg(schedule_interval_days),
  sqlc.arg(schedule_ease_factor),
  sqlc.narg(next_review_at),
  sqlc.narg(suspended_reason),
  now()
)
on conflict (user_id, coarse_unit_id) do update
set
  is_target = excluded.is_target,
  target_source = excluded.target_source,
  target_source_ref_id = excluded.target_source_ref_id,
  target_priority = excluded.target_priority,
  status = excluded.status,
  progress_percent = excluded.progress_percent,
  mastery_score = excluded.mastery_score,
  first_observed_at = excluded.first_observed_at,
  last_observed_at = excluded.last_observed_at,
  observation_count = excluded.observation_count,
  progress_event_count = excluded.progress_event_count,
  last_progress_at = excluded.last_progress_at,
  last_progress_quality = excluded.last_progress_quality,
  recent_progress_qualities = excluded.recent_progress_qualities,
  recent_progress_passes = excluded.recent_progress_passes,
  progress_success_count = excluded.progress_success_count,
  progress_failure_count = excluded.progress_failure_count,
  consecutive_success_count = excluded.consecutive_success_count,
  consecutive_failure_count = excluded.consecutive_failure_count,
  schedule_repetition = excluded.schedule_repetition,
  schedule_interval_days = excluded.schedule_interval_days,
  schedule_ease_factor = excluded.schedule_ease_factor,
  next_review_at = excluded.next_review_at,
  suspended_reason = excluded.suspended_reason,
  updated_at = now()
returning *;

-- name: EnsureTargetUnit :exec
insert into learning.user_unit_states (
  user_id,
  coarse_unit_id,
  is_target,
  target_source,
  target_source_ref_id,
  target_priority
) values (
  sqlc.arg(user_id),
  sqlc.arg(coarse_unit_id),
  true,
  sqlc.narg(target_source),
  sqlc.narg(target_source_ref_id),
  sqlc.arg(target_priority)
)
on conflict (user_id, coarse_unit_id) do update
set
  is_target = true,
  target_source = excluded.target_source,
  target_source_ref_id = excluded.target_source_ref_id,
  target_priority = excluded.target_priority,
  updated_at = now();

-- name: SetTargetInactive :exec
update learning.user_unit_states
set
  is_target = false,
  updated_at = now()
where user_id = sqlc.arg(user_id)
  and coarse_unit_id = sqlc.arg(coarse_unit_id);

-- name: DeleteUserUnitStatesByUser :exec
delete from learning.user_unit_states
where user_id = sqlc.arg(user_id);

-- name: ListUserUnitStates :many
select *
from learning.user_unit_states
where user_id = sqlc.arg(user_id)
  and (not sqlc.arg(only_target)::boolean or is_target = true)
  and (not sqlc.arg(exclude_suspended)::boolean or status <> 'suspended')
order by coarse_unit_id asc;
