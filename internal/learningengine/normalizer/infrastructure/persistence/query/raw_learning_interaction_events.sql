-- name: ListPendingLearningInteractions :many
select
  i.event_id,
  i.user_id,
  i.event_type,
  i.source_surface,
  i.video_id,
  i.watch_session_id,
  i.recommendation_run_id,
  i.related_quiz_event_id,
  i.coarse_unit_id,
  i.token_text,
  i.sentence_index,
  i.span_index,
  i.occurred_at,
  i.exposure_start_ms,
  i.exposure_end_ms,
  i.exposure_count,
  i.lookup_visible_ms,
  i.lookup_sentence_audio_replay_count,
  i.lookup_word_audio_play_count,
  i.lookup_practice_now_clicked,
  i.event_payload
from analytics.learning_interaction_events i
where i.coarse_unit_id is not null
  and i.event_type in ('exposure', 'lookup', 'self_mark_mastered')
  and (sqlc.narg(user_id)::uuid is null or i.user_id = sqlc.narg(user_id)::uuid)
  and (sqlc.narg(occurred_before)::timestamptz is null or i.occurred_at < sqlc.narg(occurred_before)::timestamptz)
  and not exists (
    select 1
    from learning.unit_learning_events e
    where e.user_id = i.user_id
      and e.source_type = 'learning_interaction_event'
      and e.source_ref_id = i.event_id::text
      and e.coarse_unit_id = i.coarse_unit_id
  )
order by i.occurred_at asc, i.event_id asc
limit sqlc.arg(limit_count)::int;
