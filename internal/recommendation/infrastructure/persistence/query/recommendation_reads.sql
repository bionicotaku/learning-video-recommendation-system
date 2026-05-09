-- name: ListLearningStatesForRecommendation :many
select
  user_id,
  coarse_unit_id,
  is_target,
  target_priority,
  status,
  progress_percent,
  mastery_score,
  last_quality,
  next_review_at,
  recent_quality_window,
  recent_correctness_window,
  strong_event_count,
  review_count,
  updated_at
from learning.user_unit_states
where user_id = sqlc.arg(user_id)
  and is_target = true
  and status <> 'suspended'
order by target_priority desc, coarse_unit_id asc;

-- name: ListRecommendableVideoUnitsByUnitIDs :many
select
  video_id,
  coarse_unit_id,
  mention_count,
  sentence_count,
  first_start_ms,
  last_end_ms,
  coverage_ms,
  coverage_ratio,
  sentence_indexes,
  evidence_span_refs,
  duration_ms,
  mapped_span_ratio,
  status,
  visibility_status,
  publish_at
from recommendation.v_recommendable_video_units
where coarse_unit_id = any(sqlc.arg(coarse_unit_ids)::bigint[])
order by coarse_unit_id asc, coverage_ratio desc, mention_count desc;

-- name: ListUnitVideoInventoryByUnitIDs :many
select
  coarse_unit_id,
  distinct_video_count,
  avg_mention_count,
  avg_sentence_count,
  avg_coverage_ms,
  avg_coverage_ratio,
  strong_video_count,
  supply_grade,
  updated_at
from recommendation.v_unit_video_inventory
where coarse_unit_id = any(sqlc.arg(coarse_unit_ids)::bigint[])
order by coarse_unit_id asc;

-- name: ListUserUnitServingStatesByUnitIDs :many
select user_id, coarse_unit_id, last_served_at, last_run_id, served_count, created_at, updated_at
from recommendation.user_unit_serving_states
where user_id = sqlc.arg(user_id)
  and coarse_unit_id = any(sqlc.arg(coarse_unit_ids)::bigint[])
order by coarse_unit_id asc;

-- name: ListUserVideoServingStatesByVideoIDs :many
select user_id, video_id, last_served_at, last_run_id, served_count, created_at, updated_at
from recommendation.user_video_serving_states
where user_id = sqlc.arg(user_id)
  and video_id = any(sqlc.arg(video_ids)::uuid[])
order by video_id asc;

-- name: ListSemanticSpansByVideoAndUnit :many
select video_id, sentence_index, span_index, coarse_unit_id, start_ms, end_ms
from catalog.video_semantic_spans
where video_id = sqlc.arg(video_id)
  and coarse_unit_id = sqlc.arg(coarse_unit_id)
order by sentence_index asc, span_index asc;

-- name: ListTranscriptSentencesByVideoAndIndexes :many
select video_id, sentence_index, start_ms, end_ms
from catalog.video_transcript_sentences
where video_id = sqlc.arg(video_id)
  and sentence_index = any(sqlc.arg(sentence_indexes)::integer[])
order by sentence_index asc;

-- name: ListVideoUserStatesByUserAndVideoIDs :many
select user_id, video_id, last_watched_at, watch_count, completed_count, last_watch_ratio, max_watch_ratio
from catalog.video_user_states
where user_id = sqlc.arg(user_id)
  and video_id = any(sqlc.arg(video_ids)::uuid[])
order by video_id asc;
