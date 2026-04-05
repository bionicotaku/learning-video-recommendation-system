-- name: UpsertUserUnitServingState :exec
insert into recommendation.user_unit_serving_states (
  user_id,
  coarse_unit_id,
  last_recommended_at,
  last_recommendation_run_id,
  created_at,
  updated_at
) values (
  sqlc.arg(user_id),
  sqlc.arg(coarse_unit_id),
  sqlc.arg(last_recommended_at),
  sqlc.arg(last_recommendation_run_id),
  sqlc.arg(created_at),
  sqlc.arg(updated_at)
)
on conflict (user_id, coarse_unit_id) do update
set
  last_recommended_at = excluded.last_recommended_at,
  last_recommendation_run_id = excluded.last_recommendation_run_id,
  updated_at = excluded.updated_at;
