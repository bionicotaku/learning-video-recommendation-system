-- name: CountSchedulerRuns :one
select count(*)::bigint
from recommendation.scheduler_runs;

-- name: UpsertSchedulerRun :exec
insert into recommendation.scheduler_runs (
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
)
on conflict (run_id) do update
set
  user_id = excluded.user_id,
  requested_limit = excluded.requested_limit,
  generated_at = excluded.generated_at,
  due_review_count = excluded.due_review_count,
  selected_review_count = excluded.selected_review_count,
  selected_new_count = excluded.selected_new_count,
  context = excluded.context;

-- name: UpsertSchedulerRunItem :exec
insert into recommendation.scheduler_run_items (
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
)
on conflict (run_id, coarse_unit_id) do update
set
  user_id = excluded.user_id,
  recommend_type = excluded.recommend_type,
  rank = excluded.rank,
  score = excluded.score,
  reason_codes = excluded.reason_codes;
