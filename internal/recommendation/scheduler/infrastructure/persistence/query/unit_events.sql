-- name: CountUnitLearningEvents :one
select count(*)::bigint
from learning.unit_learning_events;

-- name: InsertUnitLearningEvent :exec
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
  occurred_at,
  created_at
) values (
  sqlc.arg(user_id),
  sqlc.arg(coarse_unit_id),
  sqlc.arg(video_id),
  sqlc.arg(event_type),
  sqlc.arg(source_type),
  sqlc.arg(source_ref_id),
  sqlc.arg(is_correct),
  sqlc.arg(quality),
  sqlc.arg(response_time_ms),
  sqlc.arg(metadata),
  sqlc.arg(occurred_at),
  sqlc.arg(created_at)
);

-- name: FindUnitLearningEventsForReplay :many
select *
from learning.unit_learning_events
where user_id = sqlc.arg(user_id)
  and (sqlc.narg(coarse_unit_id)::bigint is null or coarse_unit_id = sqlc.narg(coarse_unit_id)::bigint)
  and (sqlc.narg(from_occurred_at)::timestamptz is null or occurred_at >= sqlc.narg(from_occurred_at)::timestamptz)
order by occurred_at asc, event_id asc;
