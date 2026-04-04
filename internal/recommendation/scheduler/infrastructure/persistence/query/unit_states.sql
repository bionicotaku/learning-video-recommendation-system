-- name: CountUserUnitStates :one
select count(*)::bigint
from learning.user_unit_states;

-- name: GetUserUnitStateByUserAndUnit :one
select *
from learning.user_unit_states
where user_id = sqlc.arg(user_id)
  and coarse_unit_id = sqlc.arg(coarse_unit_id)
limit 1;

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
  last_recommended_at,
  seen_count,
  strong_event_count,
  review_count,
  correct_count,
  wrong_count,
  consecutive_correct,
  consecutive_wrong,
  last_quality,
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
  sqlc.arg(last_recommended_at),
  sqlc.arg(seen_count),
  sqlc.arg(strong_event_count),
  sqlc.arg(review_count),
  sqlc.arg(correct_count),
  sqlc.arg(wrong_count),
  sqlc.arg(consecutive_correct),
  sqlc.arg(consecutive_wrong),
  sqlc.arg(last_quality),
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
  last_recommended_at = excluded.last_recommended_at,
  seen_count = excluded.seen_count,
  strong_event_count = excluded.strong_event_count,
  review_count = excluded.review_count,
  correct_count = excluded.correct_count,
  wrong_count = excluded.wrong_count,
  consecutive_correct = excluded.consecutive_correct,
  consecutive_wrong = excluded.consecutive_wrong,
  last_quality = excluded.last_quality,
  repetition = excluded.repetition,
  interval_days = excluded.interval_days,
  ease_factor = excluded.ease_factor,
  next_review_at = excluded.next_review_at,
  suspended_reason = excluded.suspended_reason,
  created_at = excluded.created_at,
  updated_at = excluded.updated_at;

-- name: FindDueReviewCandidates :many
select
  s.user_id,
  s.coarse_unit_id,
  s.is_target,
  s.target_source,
  s.target_source_ref_id,
  s.target_priority,
  s.status,
  s.progress_percent,
  s.mastery_score,
  s.first_seen_at,
  s.last_seen_at,
  s.last_reviewed_at,
  s.last_recommended_at,
  s.seen_count,
  s.strong_event_count,
  s.review_count,
  s.correct_count,
  s.wrong_count,
  s.consecutive_correct,
  s.consecutive_wrong,
  s.last_quality,
  s.repetition,
  s.interval_days,
  s.ease_factor,
  s.next_review_at,
  s.suspended_reason,
  s.created_at,
  s.updated_at,
  u.kind as unit_kind,
  u.label as unit_label,
  u.pos as unit_pos,
  u.english_def as unit_english_def,
  u.chinese_def as unit_chinese_def
from learning.user_unit_states s
join semantic.coarse_unit u on u.id = s.coarse_unit_id
where s.user_id = sqlc.arg(user_id)
  and s.is_target = true
  and s.status in ('learning', 'reviewing', 'mastered')
  and s.next_review_at <= sqlc.arg(now)
order by s.next_review_at asc, s.coarse_unit_id asc;

-- name: FindNewCandidates :many
select
  s.user_id,
  s.coarse_unit_id,
  s.is_target,
  s.target_source,
  s.target_source_ref_id,
  s.target_priority,
  s.status,
  s.progress_percent,
  s.mastery_score,
  s.first_seen_at,
  s.last_seen_at,
  s.last_reviewed_at,
  s.last_recommended_at,
  s.seen_count,
  s.strong_event_count,
  s.review_count,
  s.correct_count,
  s.wrong_count,
  s.consecutive_correct,
  s.consecutive_wrong,
  s.last_quality,
  s.repetition,
  s.interval_days,
  s.ease_factor,
  s.next_review_at,
  s.suspended_reason,
  s.created_at,
  s.updated_at,
  u.kind as unit_kind,
  u.label as unit_label,
  u.pos as unit_pos,
  u.english_def as unit_english_def,
  u.chinese_def as unit_chinese_def
from learning.user_unit_states s
join semantic.coarse_unit u on u.id = s.coarse_unit_id
where s.user_id = sqlc.arg(user_id)
  and s.is_target = true
  and s.status = 'new'
order by s.target_priority desc, s.coarse_unit_id asc;
