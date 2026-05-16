-- name: InsertLearningInteractionEvent :one
insert into analytics.learning_interaction_events (
  client_event_id,
  user_id,
  client_context,
  event_type,
  source_surface,
  video_id,
  watch_session_id,
  recommendation_run_id,
  related_quiz_event_id,
  coarse_unit_id,
  token_text,
  sentence_index,
  span_index,
  occurred_at,
  exposure_start_ms,
  exposure_end_ms,
  exposure_count,
  lookup_visible_ms,
  lookup_sentence_audio_replay_count,
  lookup_word_audio_play_count,
  lookup_practice_now_clicked,
  event_payload
) values (
  sqlc.arg(client_event_id),
  sqlc.arg(user_id),
  sqlc.arg(client_context),
  sqlc.arg(event_type),
  sqlc.arg(source_surface),
  sqlc.narg(video_id),
  sqlc.narg(watch_session_id),
  sqlc.narg(recommendation_run_id),
  sqlc.narg(related_quiz_event_id),
  sqlc.narg(coarse_unit_id),
  sqlc.narg(token_text),
  sqlc.narg(sentence_index),
  sqlc.narg(span_index),
  sqlc.arg(occurred_at),
  sqlc.narg(exposure_start_ms),
  sqlc.narg(exposure_end_ms),
  sqlc.narg(exposure_count),
  sqlc.narg(lookup_visible_ms),
  sqlc.arg(lookup_sentence_audio_replay_count),
  sqlc.arg(lookup_word_audio_play_count),
  sqlc.arg(lookup_practice_now_clicked),
  sqlc.arg(event_payload)
)
on conflict (user_id, client_event_id) do update
set client_event_id = excluded.client_event_id
returning event_id, (xmax = 0) as inserted;

-- name: InsertLearningInteractionEvents :many
with input as (
  select
    ordinality::integer as input_index,
    item->>'client_event_id' as client_event_id,
    (item->>'user_id')::uuid as user_id,
    coalesce(item->'client_context', '{}'::jsonb) as client_context,
    item->>'event_type' as event_type,
    item->>'source_surface' as source_surface,
    case when nullif(item->>'video_id', '') is null then null else (item->>'video_id')::uuid end as video_id,
    case when nullif(item->>'watch_session_id', '') is null then null else (item->>'watch_session_id')::uuid end as watch_session_id,
    case when nullif(item->>'recommendation_run_id', '') is null then null else (item->>'recommendation_run_id')::uuid end as recommendation_run_id,
    case when nullif(item->>'related_quiz_event_id', '') is null then null else (item->>'related_quiz_event_id')::uuid end as related_quiz_event_id,
    case when nullif(item->>'coarse_unit_id', '') is null then null else (item->>'coarse_unit_id')::bigint end as coarse_unit_id,
    nullif(item->>'token_text', '') as token_text,
    case when nullif(item->>'sentence_index', '') is null then null else (item->>'sentence_index')::integer end as sentence_index,
    case when nullif(item->>'span_index', '') is null then null else (item->>'span_index')::integer end as span_index,
    (item->>'occurred_at')::timestamptz as occurred_at,
    case when nullif(item->>'exposure_start_ms', '') is null then null else (item->>'exposure_start_ms')::integer end as exposure_start_ms,
    case when nullif(item->>'exposure_end_ms', '') is null then null else (item->>'exposure_end_ms')::integer end as exposure_end_ms,
    case when nullif(item->>'exposure_count', '') is null then null else (item->>'exposure_count')::integer end as exposure_count,
    case when nullif(item->>'lookup_visible_ms', '') is null then null else (item->>'lookup_visible_ms')::integer end as lookup_visible_ms,
    coalesce((item->>'lookup_sentence_audio_replay_count')::integer, 0) as lookup_sentence_audio_replay_count,
    coalesce((item->>'lookup_word_audio_play_count')::integer, 0) as lookup_word_audio_play_count,
    coalesce((item->>'lookup_practice_now_clicked')::boolean, false) as lookup_practice_now_clicked,
    coalesce(item->'event_payload', '{}'::jsonb) as event_payload
  from jsonb_array_elements(sqlc.arg(events)::jsonb) with ordinality as events(item, ordinality)
),
upserted as (
  insert into analytics.learning_interaction_events (
    client_event_id,
    user_id,
    client_context,
    event_type,
    source_surface,
    video_id,
    watch_session_id,
    recommendation_run_id,
    related_quiz_event_id,
    coarse_unit_id,
    token_text,
    sentence_index,
    span_index,
    occurred_at,
    exposure_start_ms,
    exposure_end_ms,
    exposure_count,
    lookup_visible_ms,
    lookup_sentence_audio_replay_count,
    lookup_word_audio_play_count,
    lookup_practice_now_clicked,
    event_payload
  )
  select
    client_event_id,
    user_id,
    client_context,
    event_type,
    source_surface,
    video_id,
    watch_session_id,
    recommendation_run_id,
    related_quiz_event_id,
    coarse_unit_id,
    token_text,
    sentence_index,
    span_index,
    occurred_at,
    exposure_start_ms,
    exposure_end_ms,
    exposure_count,
    lookup_visible_ms,
    lookup_sentence_audio_replay_count,
    lookup_word_audio_play_count,
    lookup_practice_now_clicked,
    event_payload
  from input
  on conflict (user_id, client_event_id) do update
  set client_event_id = excluded.client_event_id
  returning user_id, client_event_id, event_id, (xmax = 0) as inserted
)
select input.client_event_id::text as client_event_id, upserted.event_id, upserted.inserted
from input
join upserted
  on upserted.user_id = input.user_id
 and upserted.client_event_id = input.client_event_id
order by input.input_index;

-- name: InsertQuizEvent :one
insert into analytics.quiz_events (
  client_event_id,
  user_id,
  client_context,
  question_id,
  coarse_unit_id,
  video_id,
  recommendation_run_id,
  trigger_type,
  selected_option_ids,
  selection_interval_ms,
  is_first_try_correct,
  total_elapsed_ms,
  shown_at,
  completed_at
) values (
  sqlc.arg(client_event_id),
  sqlc.arg(user_id),
  sqlc.arg(client_context),
  sqlc.arg(question_id),
  sqlc.arg(coarse_unit_id),
  sqlc.narg(video_id),
  sqlc.narg(recommendation_run_id),
  sqlc.arg(trigger_type),
  sqlc.arg(selected_option_ids),
  sqlc.arg(selection_interval_ms),
  sqlc.arg(is_first_try_correct),
  sqlc.arg(total_elapsed_ms),
  sqlc.arg(shown_at),
  sqlc.arg(completed_at)
)
on conflict (user_id, client_event_id) do update
set client_event_id = excluded.client_event_id
returning event_id, (xmax = 0) as inserted;
