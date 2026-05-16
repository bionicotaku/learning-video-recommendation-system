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
on conflict (user_id, client_event_id) do nothing
returning event_id;

-- name: GetLearningInteractionEventByClientID :one
select event_id
from analytics.learning_interaction_events
where user_id = sqlc.arg(user_id)
  and client_event_id = sqlc.arg(client_event_id);

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
on conflict (user_id, client_event_id) do nothing
returning event_id;

-- name: GetQuizEventByClientID :one
select event_id
from analytics.quiz_events
where user_id = sqlc.arg(user_id)
  and client_event_id = sqlc.arg(client_event_id);
