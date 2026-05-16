-- name: GetUserUnitStateForUpdate :one
select *
from learning.user_unit_states
where user_id = sqlc.arg(user_id)
  and coarse_unit_id = sqlc.arg(coarse_unit_id)
for update;

-- name: ListUserUnitStatesForUpdateByUnitIDs :many
select *
from learning.user_unit_states
where user_id = sqlc.arg(user_id)
  and coarse_unit_id = any(sqlc.arg(coarse_unit_ids)::bigint[])
order by coarse_unit_id asc
for update;

-- name: UpsertUserUnitState :one
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
  first_observed_at,
  last_observed_at,
  observation_count,
  progress_event_count,
  last_progress_at,
  last_progress_quality,
  recent_progress_qualities,
  recent_progress_passes,
  progress_success_count,
  progress_failure_count,
  consecutive_success_count,
  consecutive_failure_count,
  schedule_repetition,
  schedule_interval_days,
  schedule_ease_factor,
  next_review_at,
  suspended_reason,
  updated_at
) values (
  sqlc.arg(user_id),
  sqlc.arg(coarse_unit_id),
  sqlc.arg(is_target),
  sqlc.narg(target_source),
  sqlc.narg(target_source_ref_id),
  sqlc.arg(target_priority),
  sqlc.arg(status),
  sqlc.arg(progress_percent),
  sqlc.arg(mastery_score),
  sqlc.narg(first_observed_at),
  sqlc.narg(last_observed_at),
  sqlc.arg(observation_count),
  sqlc.arg(progress_event_count),
  sqlc.narg(last_progress_at),
  sqlc.narg(last_progress_quality),
  sqlc.arg(recent_progress_qualities),
  sqlc.arg(recent_progress_passes),
  sqlc.arg(progress_success_count),
  sqlc.arg(progress_failure_count),
  sqlc.arg(consecutive_success_count),
  sqlc.arg(consecutive_failure_count),
  sqlc.arg(schedule_repetition),
  sqlc.arg(schedule_interval_days),
  sqlc.arg(schedule_ease_factor),
  sqlc.narg(next_review_at),
  sqlc.narg(suspended_reason),
  now()
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
  first_observed_at = excluded.first_observed_at,
  last_observed_at = excluded.last_observed_at,
  observation_count = excluded.observation_count,
  progress_event_count = excluded.progress_event_count,
  last_progress_at = excluded.last_progress_at,
  last_progress_quality = excluded.last_progress_quality,
  recent_progress_qualities = excluded.recent_progress_qualities,
  recent_progress_passes = excluded.recent_progress_passes,
  progress_success_count = excluded.progress_success_count,
  progress_failure_count = excluded.progress_failure_count,
  consecutive_success_count = excluded.consecutive_success_count,
  consecutive_failure_count = excluded.consecutive_failure_count,
  schedule_repetition = excluded.schedule_repetition,
  schedule_interval_days = excluded.schedule_interval_days,
  schedule_ease_factor = excluded.schedule_ease_factor,
  next_review_at = excluded.next_review_at,
  suspended_reason = excluded.suspended_reason,
  updated_at = now()
returning *;

-- name: BatchUpsertUserUnitStates :many
with input as (
  select
    (item.value->>'user_id')::uuid as user_id,
    (item.value->>'coarse_unit_id')::bigint as coarse_unit_id,
    (item.value->>'is_target')::boolean as is_target,
    nullif(item.value->>'target_source', '') as target_source,
    nullif(item.value->>'target_source_ref_id', '') as target_source_ref_id,
    (item.value->>'target_priority')::numeric as target_priority,
    item.value->>'status' as status,
    (item.value->>'progress_percent')::numeric as progress_percent,
    (item.value->>'mastery_score')::numeric as mastery_score,
    nullif(item.value->>'first_observed_at', '')::timestamptz as first_observed_at,
    nullif(item.value->>'last_observed_at', '')::timestamptz as last_observed_at,
    (item.value->>'observation_count')::integer as observation_count,
    (item.value->>'progress_event_count')::integer as progress_event_count,
    nullif(item.value->>'last_progress_at', '')::timestamptz as last_progress_at,
    nullif(item.value->>'last_progress_quality', '')::smallint as last_progress_quality,
    coalesce((select array_agg(value::smallint) from jsonb_array_elements_text(coalesce(item.value->'recent_progress_qualities', '[]'::jsonb)) as q(value)), '{}'::smallint[]) as recent_progress_qualities,
    coalesce((select array_agg(value::boolean) from jsonb_array_elements_text(coalesce(item.value->'recent_progress_passes', '[]'::jsonb)) as p(value)), '{}'::boolean[]) as recent_progress_passes,
    (item.value->>'progress_success_count')::integer as progress_success_count,
    (item.value->>'progress_failure_count')::integer as progress_failure_count,
    (item.value->>'consecutive_success_count')::integer as consecutive_success_count,
    (item.value->>'consecutive_failure_count')::integer as consecutive_failure_count,
    (item.value->>'schedule_repetition')::integer as schedule_repetition,
    (item.value->>'schedule_interval_days')::numeric as schedule_interval_days,
    (item.value->>'schedule_ease_factor')::numeric as schedule_ease_factor,
    nullif(item.value->>'next_review_at', '')::timestamptz as next_review_at,
    nullif(item.value->>'suspended_reason', '') as suspended_reason
  from jsonb_array_elements(sqlc.arg(states)::jsonb) as item(value)
)
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
  first_observed_at,
  last_observed_at,
  observation_count,
  progress_event_count,
  last_progress_at,
  last_progress_quality,
  recent_progress_qualities,
  recent_progress_passes,
  progress_success_count,
  progress_failure_count,
  consecutive_success_count,
  consecutive_failure_count,
  schedule_repetition,
  schedule_interval_days,
  schedule_ease_factor,
  next_review_at,
  suspended_reason,
  updated_at
)
select
  user_id,
  coarse_unit_id,
  is_target,
  target_source,
  target_source_ref_id,
  target_priority,
  status,
  progress_percent,
  mastery_score,
  first_observed_at,
  last_observed_at,
  observation_count,
  progress_event_count,
  last_progress_at,
  last_progress_quality,
  coalesce(recent_progress_qualities, '{}'::smallint[]),
  coalesce(recent_progress_passes, '{}'::boolean[]),
  progress_success_count,
  progress_failure_count,
  consecutive_success_count,
  consecutive_failure_count,
  schedule_repetition,
  schedule_interval_days,
  schedule_ease_factor,
  next_review_at,
  suspended_reason,
  now()
from input
on conflict (user_id, coarse_unit_id) do update
set
  is_target = excluded.is_target,
  target_source = excluded.target_source,
  target_source_ref_id = excluded.target_source_ref_id,
  target_priority = excluded.target_priority,
  status = excluded.status,
  progress_percent = excluded.progress_percent,
  mastery_score = excluded.mastery_score,
  first_observed_at = excluded.first_observed_at,
  last_observed_at = excluded.last_observed_at,
  observation_count = excluded.observation_count,
  progress_event_count = excluded.progress_event_count,
  last_progress_at = excluded.last_progress_at,
  last_progress_quality = excluded.last_progress_quality,
  recent_progress_qualities = excluded.recent_progress_qualities,
  recent_progress_passes = excluded.recent_progress_passes,
  progress_success_count = excluded.progress_success_count,
  progress_failure_count = excluded.progress_failure_count,
  consecutive_success_count = excluded.consecutive_success_count,
  consecutive_failure_count = excluded.consecutive_failure_count,
  schedule_repetition = excluded.schedule_repetition,
  schedule_interval_days = excluded.schedule_interval_days,
  schedule_ease_factor = excluded.schedule_ease_factor,
  next_review_at = excluded.next_review_at,
  suspended_reason = excluded.suspended_reason,
  updated_at = now()
returning *;

-- name: EnsureTargetUnits :exec
with raw_input as (
  select
    ordinality::integer as input_index,
    (item->>'coarse_unit_id')::bigint as coarse_unit_id,
    nullif(item->>'target_source', '') as target_source,
    nullif(item->>'target_source_ref_id', '') as target_source_ref_id,
    (item->>'target_priority')::numeric as target_priority
  from jsonb_array_elements(sqlc.arg(targets)::jsonb) with ordinality as targets(item, ordinality)
),
input as (
  select distinct on (coarse_unit_id)
    coarse_unit_id,
    target_source,
    target_source_ref_id,
    target_priority
  from raw_input
  order by coarse_unit_id, input_index desc
)
insert into learning.user_unit_states (
  user_id,
  coarse_unit_id,
  is_target,
  target_source,
  target_source_ref_id,
  target_priority
)
select
  sqlc.arg(user_id),
  coarse_unit_id,
  true,
  target_source,
  target_source_ref_id,
  target_priority
from input
on conflict (user_id, coarse_unit_id) do update
set
  is_target = true,
  target_source = excluded.target_source,
  target_source_ref_id = excluded.target_source_ref_id,
  target_priority = excluded.target_priority,
  updated_at = now();

-- name: SetTargetInactive :exec
update learning.user_unit_states
set
  is_target = false,
  updated_at = now()
where user_id = sqlc.arg(user_id)
  and coarse_unit_id = sqlc.arg(coarse_unit_id);

-- name: DeleteUserUnitStatesByUser :exec
delete from learning.user_unit_states
where user_id = sqlc.arg(user_id);

-- name: ListUserUnitStates :many
select *
from learning.user_unit_states
where user_id = sqlc.arg(user_id)
  and (not sqlc.arg(only_target)::boolean or is_target = true)
  and (not sqlc.arg(exclude_suspended)::boolean or status <> 'suspended')
order by coarse_unit_id asc;
