-- name: AppendLearningEvent :one
insert into learning.unit_learning_events (
  user_id,
  coarse_unit_id,
  video_id,
  event_type,
  source_type,
  source_ref_id,
  is_correct,
  quality,
  response_time_ms,
  metadata,
  occurred_at
) values (
  sqlc.arg(user_id),
  sqlc.arg(coarse_unit_id),
  sqlc.narg(video_id),
  sqlc.arg(event_type),
  sqlc.arg(source_type),
  sqlc.narg(source_ref_id),
  sqlc.narg(is_correct),
  sqlc.narg(quality),
  sqlc.narg(response_time_ms),
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
