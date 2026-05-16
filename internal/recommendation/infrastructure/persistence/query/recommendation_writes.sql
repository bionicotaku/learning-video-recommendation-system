-- name: IncrementUserUnitServingStates :exec
insert into recommendation.user_unit_serving_states (
  user_id,
  coarse_unit_id,
  last_served_at,
  last_run_id,
  served_count,
  updated_at
) 
select
  sqlc.arg(user_id),
  input.coarse_unit_id,
  sqlc.narg(last_served_at),
  sqlc.narg(last_run_id),
  1,
  now()
from (
  select distinct unnest(sqlc.arg(coarse_unit_ids)::bigint[]) as coarse_unit_id
) input
where input.coarse_unit_id is not null
on conflict (user_id, coarse_unit_id) do update
set
  last_served_at = excluded.last_served_at,
  last_run_id = excluded.last_run_id,
  served_count = recommendation.user_unit_serving_states.served_count + 1,
  updated_at = now();

-- name: IncrementUserVideoServingStates :exec
insert into recommendation.user_video_serving_states (
  user_id,
  video_id,
  last_served_at,
  last_run_id,
  served_count,
  updated_at
)
select
  sqlc.arg(user_id),
  input.video_id,
  sqlc.narg(last_served_at),
  sqlc.narg(last_run_id),
  1,
  now()
from (
  select distinct unnest(sqlc.arg(video_ids)::uuid[]) as video_id
) input
where input.video_id is not null
on conflict (user_id, video_id) do update
set
  last_served_at = excluded.last_served_at,
  last_run_id = excluded.last_run_id,
  served_count = recommendation.user_video_serving_states.served_count + 1,
  updated_at = now();

-- name: InsertVideoRecommendationRun :exec
insert into recommendation.video_recommendation_runs (
  run_id,
  user_id,
  request_context,
  session_mode,
  selector_mode,
  planner_snapshot,
  lane_budget_snapshot,
  candidate_summary,
  underfilled,
  result_count
) values (
  sqlc.arg(run_id),
  sqlc.arg(user_id),
  sqlc.arg(request_context),
  sqlc.narg(session_mode),
  sqlc.narg(selector_mode),
  sqlc.arg(planner_snapshot),
  sqlc.arg(lane_budget_snapshot),
  sqlc.arg(candidate_summary),
  sqlc.arg(underfilled),
  sqlc.arg(result_count)
);

-- name: InsertVideoRecommendationItem :exec
insert into recommendation.video_recommendation_items (
  run_id,
  rank,
  video_id,
  score,
  primary_lane,
  dominant_role,
  dominant_unit_id,
  reason_codes,
  learning_units
) values (
  sqlc.arg(run_id),
  sqlc.arg(rank),
  sqlc.arg(video_id),
  sqlc.arg(score),
  sqlc.narg(primary_lane),
  sqlc.narg(dominant_role),
  sqlc.narg(dominant_unit_id),
  sqlc.arg(reason_codes),
  sqlc.arg(learning_units)
);

-- name: RefreshRecommendableVideoUnits :exec
refresh materialized view recommendation.v_recommendable_video_units;

-- name: RefreshUnitVideoInventory :exec
refresh materialized view recommendation.v_unit_video_inventory;
