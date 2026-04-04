-- name: CountSchedulerRuns :one
select count(*)::bigint
from learning.scheduler_runs;

-- name: InsertSchedulerRun :exec
insert into learning.scheduler_runs (
  run_id,
  user_id,
  requested_limit,
  generated_at,
  due_review_count,
  selected_review_count,
  selected_new_count,
  context
) values (
  sqlc.arg(run_id),
  sqlc.arg(user_id),
  sqlc.arg(requested_limit),
  sqlc.arg(generated_at),
  sqlc.arg(due_review_count),
  sqlc.arg(selected_review_count),
  sqlc.arg(selected_new_count),
  sqlc.arg(context)
);

-- name: InsertSchedulerRunItem :exec
insert into learning.scheduler_run_items (
  run_id,
  user_id,
  coarse_unit_id,
  recommend_type,
  rank,
  score,
  reason_codes
) values (
  sqlc.arg(run_id),
  sqlc.arg(user_id),
  sqlc.arg(coarse_unit_id),
  sqlc.arg(recommend_type),
  sqlc.arg(rank),
  sqlc.arg(score),
  sqlc.arg(reason_codes)
);
