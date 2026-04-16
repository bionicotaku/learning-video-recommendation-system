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
  first_seen_at,
  last_seen_at,
  last_reviewed_at,
  seen_count,
  strong_event_count,
  review_count,
  correct_count,
  wrong_count,
  consecutive_correct,
  consecutive_wrong,
  last_quality,
  recent_quality_window,
  recent_correctness_window,
  repetition,
  interval_days,
  ease_factor,
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
  sqlc.narg(first_seen_at),
  sqlc.narg(last_seen_at),
  sqlc.narg(last_reviewed_at),
  sqlc.arg(seen_count),
  sqlc.arg(strong_event_count),
  sqlc.arg(review_count),
  sqlc.arg(correct_count),
  sqlc.arg(wrong_count),
  sqlc.arg(consecutive_correct),
  sqlc.arg(consecutive_wrong),
  sqlc.narg(last_quality),
  sqlc.arg(recent_quality_window),
  sqlc.arg(recent_correctness_window),
  sqlc.arg(repetition),
  sqlc.arg(interval_days),
  sqlc.arg(ease_factor),
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
  first_seen_at = excluded.first_seen_at,
  last_seen_at = excluded.last_seen_at,
  last_reviewed_at = excluded.last_reviewed_at,
  seen_count = excluded.seen_count,
  strong_event_count = excluded.strong_event_count,
  review_count = excluded.review_count,
  correct_count = excluded.correct_count,
  wrong_count = excluded.wrong_count,
  consecutive_correct = excluded.consecutive_correct,
  consecutive_wrong = excluded.consecutive_wrong,
  last_quality = excluded.last_quality,
  recent_quality_window = excluded.recent_quality_window,
  recent_correctness_window = excluded.recent_correctness_window,
  repetition = excluded.repetition,
  interval_days = excluded.interval_days,
  ease_factor = excluded.ease_factor,
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

-- name: SuspendTargetUnit :exec
update learning.user_unit_states
set
  status = 'suspended',
  suspended_reason = sqlc.narg(suspended_reason),
  updated_at = now()
where user_id = sqlc.arg(user_id)
  and coarse_unit_id = sqlc.arg(coarse_unit_id);

-- name: ResumeTargetUnit :exec
update learning.user_unit_states
set
  status = case when status = 'suspended' then 'new' else status end,
  suspended_reason = null,
  updated_at = now()
where user_id = sqlc.arg(user_id)
  and coarse_unit_id = sqlc.arg(coarse_unit_id);

-- name: ListUserUnitStates :many
select *
from learning.user_unit_states
where user_id = sqlc.arg(user_id)
order by coarse_unit_id asc;

-- name: ListRecommendationUnitStates :many
select *
from learning.user_unit_states
where user_id = sqlc.arg(user_id)
  and is_target = true
  and status <> 'suspended'
order by target_priority desc, coarse_unit_id asc;
