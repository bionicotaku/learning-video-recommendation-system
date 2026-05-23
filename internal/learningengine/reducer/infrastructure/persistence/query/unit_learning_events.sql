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
    coalesce((item.value->>'counts_toward_success_streak')::boolean, false) as counts_toward_success_streak,
    array(
      select jsonb_array_elements_text(
        case
          when jsonb_typeof(item.value->'consumed_watch_session_ids') = 'array'
            then item.value->'consumed_watch_session_ids'
          else '[]'::jsonb
        end
      )::uuid
    ) as consumed_watch_session_ids,
	    coalesce(item.value->'metadata', '{}'::jsonb) as metadata,
	    (item.value->>'occurred_at')::timestamptz as occurred_at,
	    nullif(item.value->>'reset_boundary_at', '')::timestamptz as reset_boundary_at
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
  counts_toward_success_streak,
	  consumed_watch_session_ids,
	  metadata,
	  occurred_at,
	  reset_boundary_at
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
  counts_toward_success_streak,
	  consumed_watch_session_ids,
	  coalesce(metadata, '{}'::jsonb),
	  occurred_at,
	  reset_boundary_at
	from input
	order by input_index asc
	on conflict (user_id, source_type, source_ref_id, coarse_unit_id) do nothing
returning *
)
select *
from inserted
order by ledger_seq asc;

-- name: ListLearningEventsByUserOrdered :many
select *
from learning.unit_learning_events
where user_id = sqlc.arg(user_id)
order by ledger_seq asc;

-- name: ListLearningEventsByUserUnitOrdered :many
select *
from learning.unit_learning_events
where user_id = sqlc.arg(user_id)
  and coarse_unit_id = sqlc.arg(coarse_unit_id)
order by ledger_seq asc;

-- name: GetLearningEventByUserSourceRef :one
select *
from learning.unit_learning_events
where user_id = sqlc.arg(user_id)
  and source_type = sqlc.arg(source_type)
  and source_ref_id = sqlc.arg(source_ref_id)
order by ledger_seq asc
limit 1;
