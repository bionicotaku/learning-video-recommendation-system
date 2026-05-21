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

-- name: InsertVideoRecommendationItems :exec
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
)
select
  input.run_id,
  input.rank,
  input.video_id,
  input.score,
  nullif(input.primary_lane, ''),
  nullif(input.dominant_role, ''),
  input.dominant_unit_id,
  coalesce(input.reason_codes, array[]::text[]),
  coalesce(input.learning_units, '[]'::jsonb)
from (
  select
    (item->>'run_id')::uuid as run_id,
    (item->>'rank')::integer as rank,
    (item->>'video_id')::uuid as video_id,
    (item->>'score')::numeric as score,
    item->>'primary_lane' as primary_lane,
    item->>'dominant_role' as dominant_role,
    case
      when item ? 'dominant_unit_id' and item->>'dominant_unit_id' is not null
      then (item->>'dominant_unit_id')::bigint
      else null
    end as dominant_unit_id,
    (
      select array_agg(value::text order by ordinality)
      from jsonb_array_elements_text(coalesce(item->'reason_codes', '[]'::jsonb)) with ordinality as codes(value, ordinality)
    ) as reason_codes,
    item->'learning_units' as learning_units
  from jsonb_array_elements(sqlc.arg(items)::jsonb) as items(item)
) input;

-- name: RebuildUserUnitRecallQueue :one
with source as (
  select
    states.user_id,
    states.coarse_unit_id,
    states.status,
    states.target_priority,
    states.mastery_score,
    states.last_progress_quality,
    states.next_review_at,
    coalesce(inventory.supply_grade, 'none')::text as supply_grade,
    states.updated_at as state_updated_at,
    extract(epoch from states.updated_at)::text as source_version
  from learning.user_unit_states as states
  left join recommendation.v_unit_video_inventory as inventory
    on inventory.coarse_unit_id = states.coarse_unit_id
  where states.user_id = sqlc.arg(user_id)
    and states.is_target = true
    and states.status in ('new', 'learning', 'reviewing')
),
deleted as (
  delete from recommendation.user_unit_recall_queue
  where user_id = sqlc.arg(user_id)
),
inserted as (
  insert into recommendation.user_unit_recall_queue (
    user_id,
    coarse_unit_id,
    status,
    target_priority,
    mastery_score,
    last_progress_quality,
    next_review_at,
    supply_grade,
    state_updated_at,
    source_version,
    rebuilt_at
  )
  select
    user_id,
    coarse_unit_id,
    status,
    target_priority,
    mastery_score,
    last_progress_quality,
    next_review_at,
    supply_grade,
    state_updated_at,
    source_version,
    now()
  from source
  on conflict (user_id, coarse_unit_id) do update
  set
    status = excluded.status,
    target_priority = excluded.target_priority,
    mastery_score = excluded.mastery_score,
    last_progress_quality = excluded.last_progress_quality,
    next_review_at = excluded.next_review_at,
    supply_grade = excluded.supply_grade,
    state_updated_at = excluded.state_updated_at,
    source_version = excluded.source_version,
    rebuilt_at = excluded.rebuilt_at
  returning state_updated_at
),
summary as (
  select
    count(*)::integer as active_target_unit_count,
    max(state_updated_at) as source_learning_max_updated_at
  from source
)
insert into recommendation.user_unit_recall_queue_states (
  user_id,
  source_learning_max_updated_at,
  source_projection_updated_at,
  active_target_unit_count,
  rebuilt_at
)
select
  sqlc.arg(user_id),
  summary.source_learning_max_updated_at,
  sqlc.arg(source_projection_updated_at),
  summary.active_target_unit_count,
  now()
from summary
on conflict (user_id) do update
set
  source_learning_max_updated_at = excluded.source_learning_max_updated_at,
  source_projection_updated_at = excluded.source_projection_updated_at,
  active_target_unit_count = excluded.active_target_unit_count,
  rebuilt_at = excluded.rebuilt_at
returning
  user_id,
  source_learning_max_updated_at,
  source_projection_updated_at,
  active_target_unit_count,
  rebuilt_at;

-- name: UpsertRecallProjectionMetadata :exec
insert into recommendation.recall_projection_metadata (
  projection_name,
  projection_updated_at
)
values ('video_unit_recall_index', sqlc.arg(projection_updated_at))
on conflict (projection_name) do update
set projection_updated_at = excluded.projection_updated_at;

-- name: RefreshVideoUnitRecallIndex :exec
refresh materialized view recommendation.v_video_unit_recall_index;

-- name: RefreshUnitVideoInventory :exec
refresh materialized view recommendation.v_unit_video_inventory;
