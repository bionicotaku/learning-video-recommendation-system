-- name: AppendLearningEvent :one
insert into learning.unit_learning_events (
  user_id,
  coarse_unit_id,
  video_id,
  event_type,
  reducer_effect,
  progress_quality,
  source_type,
  source_ref_id,
  is_correct,
  metadata,
  occurred_at
) values (
  sqlc.arg(user_id),
  sqlc.arg(coarse_unit_id),
  sqlc.narg(video_id),
  sqlc.arg(event_type),
  sqlc.arg(reducer_effect),
  sqlc.narg(progress_quality),
  sqlc.arg(source_type),
  sqlc.arg(source_ref_id),
  sqlc.narg(is_correct),
  sqlc.arg(metadata),
  sqlc.arg(occurred_at)
)
returning *;

-- name: ListLearningEventsByUserOrdered :many
select *
from learning.unit_learning_events
where user_id = sqlc.arg(user_id)
order by occurred_at asc, event_id asc;

-- name: ListLearningEventsByUserUnitOrdered :many
select *
from learning.unit_learning_events
where user_id = sqlc.arg(user_id)
  and coarse_unit_id = sqlc.arg(coarse_unit_id)
order by occurred_at asc, event_id asc;
