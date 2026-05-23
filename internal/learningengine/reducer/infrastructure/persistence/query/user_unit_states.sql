-- name: GetUserUnitStateForUpdate :one
select *
from learning.user_unit_states
where user_id = sqlc.arg(user_id)
  and coarse_unit_id = sqlc.arg(coarse_unit_id)
for update;

-- name: GetUserUnitState :one
select *
from learning.user_unit_states
where user_id = sqlc.arg(user_id)
  and coarse_unit_id = sqlc.arg(coarse_unit_id);

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
  latest_learning_event_occurred_at,
  latest_reset_boundary_at,
  latest_learning_event_ledger_seq,
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
  sqlc.narg(latest_learning_event_occurred_at),
  sqlc.narg(latest_reset_boundary_at),
  sqlc.arg(latest_learning_event_ledger_seq),
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
  latest_learning_event_occurred_at = excluded.latest_learning_event_occurred_at,
  latest_reset_boundary_at = excluded.latest_reset_boundary_at,
  latest_learning_event_ledger_seq = excluded.latest_learning_event_ledger_seq,
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
    nullif(item.value->>'latest_learning_event_occurred_at', '')::timestamptz as latest_learning_event_occurred_at,
    nullif(item.value->>'latest_reset_boundary_at', '')::timestamptz as latest_reset_boundary_at,
    coalesce((item.value->>'latest_learning_event_ledger_seq')::bigint, 0) as latest_learning_event_ledger_seq
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
  latest_learning_event_occurred_at,
  latest_reset_boundary_at,
  latest_learning_event_ledger_seq,
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
  latest_learning_event_occurred_at,
  latest_reset_boundary_at,
  latest_learning_event_ledger_seq,
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
  latest_learning_event_occurred_at = excluded.latest_learning_event_occurred_at,
  latest_reset_boundary_at = excluded.latest_reset_boundary_at,
  latest_learning_event_ledger_seq = excluded.latest_learning_event_ledger_seq,
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

-- name: ActivateUnitCollectionTarget :one
with selected_collection as (
  select
    collection_id,
    slug,
    coarse_unit_count
  from semantic.unit_collections
  where slug = sqlc.arg(collection_slug)
    and status = 'active'
),
new_members as (
  select
    m.collection_id,
    m.coarse_unit_id,
    0::numeric as target_priority
  from semantic.unit_collection_members m
  join selected_collection c
    on c.collection_id = m.collection_id
),
profile_upsert as (
  insert into learning.user_learning_profiles (
    user_id,
    active_collection_id,
    active_collection_slug,
    active_collection_activated_at,
    updated_at
  )
  select
    sqlc.arg(user_id),
    collection_id,
    slug,
    now(),
    now()
  from selected_collection
  on conflict (user_id) do update
  set
    active_collection_id = excluded.active_collection_id,
    active_collection_slug = excluded.active_collection_slug,
    active_collection_activated_at = now(),
    updated_at = now()
  returning user_id
),
deactivated as (
  update learning.user_unit_states s
  set
    is_target = false,
    updated_at = now()
  where s.user_id = sqlc.arg(user_id)
    and s.target_source = 'unit_collection'
    and s.is_target = true
    and not exists (
      select 1
      from new_members nm
      where nm.coarse_unit_id = s.coarse_unit_id
    )
  returning s.coarse_unit_id
),
upserted as (
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
    nm.coarse_unit_id,
    true,
    'unit_collection',
    nm.collection_id::text,
    nm.target_priority
  from new_members nm
  on conflict (user_id, coarse_unit_id) do update
  set
    is_target = true,
    target_source = 'unit_collection',
    target_source_ref_id = excluded.target_source_ref_id,
    target_priority = excluded.target_priority,
    updated_at = now()
  returning coarse_unit_id
)
select
  c.collection_id,
  c.slug,
  coalesce(count(u.coarse_unit_id), 0)::integer as target_count
from selected_collection c
left join upserted u on true
group by c.collection_id, c.slug;

-- name: GetActiveUnitCollection :one
select
  active_collection_id,
  active_collection_slug
from learning.user_learning_profiles
where user_id = sqlc.arg(user_id);

-- name: GetActiveLearningTargetCoarseUnitIDs :one
with profile as (
  select p.active_collection_slug
  from learning.user_learning_profiles p
  where p.user_id = sqlc.arg(user_id)
),
targets as (
  select coalesce(array_agg(s.coarse_unit_id order by s.coarse_unit_id), '{}'::bigint[]) as coarse_unit_ids
  from learning.user_unit_states s
  where s.user_id = sqlc.arg(user_id)
    and s.is_target = true
    and s.status in ('new', 'learning', 'reviewing')
)
select
  coalesce((select active_collection_slug from profile), '')::text as active_collection_slug,
  coalesce((select coarse_unit_ids from targets), '{}'::bigint[])::bigint[] as coarse_unit_ids,
  exists(select 1 from profile) as has_active_profile;

-- name: DeleteUserUnitStatesByUser :exec
delete from learning.user_unit_states
where user_id = sqlc.arg(user_id);

-- name: ListUserUnitStates :many
select *
from learning.user_unit_states
where user_id = sqlc.arg(user_id)
  and (not sqlc.arg(only_target)::boolean or is_target = true)
order by coarse_unit_id asc;
