-- name: ListPendingQuizEvents :many
select
  q.event_id,
  q.user_id,
  q.question_id,
  q.coarse_unit_id,
  q.video_id,
  q.recommendation_run_id,
  q.trigger_type,
  q.selected_option_ids,
  q.selection_interval_ms,
  q.is_first_try_correct,
  q.total_elapsed_ms,
  q.shown_at,
  q.completed_at
from analytics.quiz_events q
where (sqlc.narg(user_id)::uuid is null or q.user_id = sqlc.narg(user_id)::uuid)
  and (sqlc.narg(occurred_before)::timestamptz is null or q.completed_at < sqlc.narg(occurred_before)::timestamptz)
  and q.completed_at > coalesce((
    select max(e.reset_boundary_at)
    from learning.unit_learning_events e
    where e.user_id = q.user_id
      and e.coarse_unit_id = q.coarse_unit_id
      and e.source_type = 'learning_unit_reset'
      and e.event_type = 'reset_unlearned'
      and e.reducer_effect = 'reset_unlearned'
  ), '-infinity'::timestamptz)
  and not exists (
    select 1
    from learning.unit_learning_events e
    where e.user_id = q.user_id
      and e.source_type = 'quiz_event'
      and e.source_ref_id = q.event_id::text
      and e.coarse_unit_id = q.coarse_unit_id
  )
order by q.completed_at asc, q.event_id asc
limit sqlc.arg(limit_count)::int;

-- name: ListQuizEventsByIDs :many
select
  q.event_id,
  q.user_id,
  q.question_id,
  q.coarse_unit_id,
  q.video_id,
  q.recommendation_run_id,
  q.trigger_type,
  q.selected_option_ids,
  q.selection_interval_ms,
  q.is_first_try_correct,
  q.total_elapsed_ms,
  q.shown_at,
  q.completed_at
from analytics.quiz_events q
where q.user_id = sqlc.arg(user_id)
  and q.event_id = any(sqlc.arg(event_ids)::uuid[])
  and q.completed_at > coalesce((
    select max(e.reset_boundary_at)
    from learning.unit_learning_events e
    where e.user_id = q.user_id
      and e.coarse_unit_id = q.coarse_unit_id
      and e.source_type = 'learning_unit_reset'
      and e.event_type = 'reset_unlearned'
      and e.reducer_effect = 'reset_unlearned'
  ), '-infinity'::timestamptz)
order by q.completed_at asc, q.event_id asc;
