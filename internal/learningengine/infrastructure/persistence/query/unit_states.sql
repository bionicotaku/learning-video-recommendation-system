-- 作用：定义状态表相关 SQL，包括计数、单条读取、按用户删除和 upsert。
-- 输入/输出：输入是 sqlc 参数 user_id、coarse_unit_id、state fields；输出是状态行或执行副作用。
-- 谁调用它：sqlc 生成器；运行时通过 user_unit_state_repo.go 间接调用。
-- 它调用谁/传给谁：直接作用于 PostgreSQL；生成的方法会传给 state repository 使用。
-- name: CountUserUnitStates :one
select count(*)::bigint
from learning.user_unit_states;

-- name: GetUserUnitStateByUserAndUnit :one
select *
from learning.user_unit_states
where user_id = sqlc.arg(user_id)
  and coarse_unit_id = sqlc.arg(coarse_unit_id)
limit 1;

-- name: DeleteUserUnitStatesByUser :exec
delete from learning.user_unit_states
where user_id = sqlc.arg(user_id);

-- name: UpsertUserUnitState :exec
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
  created_at,
  updated_at
) values (
  sqlc.arg(user_id),
  sqlc.arg(coarse_unit_id),
  sqlc.arg(is_target),
  sqlc.arg(target_source),
  sqlc.arg(target_source_ref_id),
  sqlc.arg(target_priority),
  sqlc.arg(status),
  sqlc.arg(progress_percent),
  sqlc.arg(mastery_score),
  sqlc.arg(first_seen_at),
  sqlc.arg(last_seen_at),
  sqlc.arg(last_reviewed_at),
  sqlc.arg(seen_count),
  sqlc.arg(strong_event_count),
  sqlc.arg(review_count),
  sqlc.arg(correct_count),
  sqlc.arg(wrong_count),
  sqlc.arg(consecutive_correct),
  sqlc.arg(consecutive_wrong),
  sqlc.arg(last_quality),
  sqlc.arg(recent_quality_window),
  sqlc.arg(recent_correctness_window),
  sqlc.arg(repetition),
  sqlc.arg(interval_days),
  sqlc.arg(ease_factor),
  sqlc.arg(next_review_at),
  sqlc.arg(suspended_reason),
  sqlc.arg(created_at),
  sqlc.arg(updated_at)
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
  created_at = excluded.created_at,
  updated_at = excluded.updated_at;
