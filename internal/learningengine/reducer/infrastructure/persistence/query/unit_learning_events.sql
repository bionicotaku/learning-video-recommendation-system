-- name: AppendLearningEvents :many
with input as (
  select
    (item.value->>'input_index')::integer as input_index,
    (item.value->>'user_id')::uuid as user_id,
    (item.value->>'coarse_unit_id')::bigint as coarse_unit_id,
    nullif(item.value->>'video_id', '')::uuid as video_id,
    item.value->>'event_type' as event_type,
    item.value->>'reducer_effect' as reducer_effect,
    nullif(item.value->>'progress_quality', '')::smallint as progress_quality,
    item.value->>'source_type' as source_type,
    item.value->>'source_ref_id' as source_ref_id,
    nullif(item.value->>'is_correct', '')::boolean as is_correct,
    coalesce(item.value->'metadata', '{}'::jsonb) as metadata,
    (item.value->>'occurred_at')::timestamptz as occurred_at
  from jsonb_array_elements(sqlc.arg(events)::jsonb) as item(value)
),
inserted as (
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
)
select
  user_id,
  coarse_unit_id,
  video_id,
  event_type,
  reducer_effect,
  progress_quality,
  source_type,
  source_ref_id,
  is_correct,
  coalesce(metadata, '{}'::jsonb),
  occurred_at
from input
order by input_index asc
on conflict (user_id, source_type, source_ref_id, coarse_unit_id) do nothing
returning *
)
select *
from inserted
order by coarse_unit_id asc, occurred_at asc, event_id asc;

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
