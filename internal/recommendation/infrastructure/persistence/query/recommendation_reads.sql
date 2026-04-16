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
select *
from recommendation.v_recommendable_video_units
where coarse_unit_id = any(sqlc.arg(coarse_unit_ids)::bigint[])
order by coarse_unit_id asc, coverage_ratio desc, mention_count desc;

-- name: ListUnitVideoInventoryByUnitIDs :many
select *
from recommendation.v_unit_video_inventory
where coarse_unit_id = any(sqlc.arg(coarse_unit_ids)::bigint[])
order by coarse_unit_id asc;

-- name: ListUserUnitServingStatesByUnitIDs :many
select *
from recommendation.user_unit_serving_states
where user_id = sqlc.arg(user_id)
  and coarse_unit_id = any(sqlc.arg(coarse_unit_ids)::bigint[])
order by coarse_unit_id asc;

-- name: ListUserVideoServingStatesByVideoIDs :many
select *
from recommendation.user_video_serving_states
where user_id = sqlc.arg(user_id)
  and video_id = any(sqlc.arg(video_ids)::uuid[])
order by video_id asc;

-- name: ListSemanticSpansByVideoAndUnit :many
select *
from catalog.video_semantic_spans
where video_id = sqlc.arg(video_id)
  and coarse_unit_id = sqlc.arg(coarse_unit_id)
order by sentence_index asc, span_index asc;

-- name: ListTranscriptSentencesByVideoAndIndexes :many
select *
from catalog.video_transcript_sentences
where video_id = sqlc.arg(video_id)
  and sentence_index = any(sqlc.arg(sentence_indexes)::integer[])
order by sentence_index asc;

-- name: ListVideoUserStatesByUserAndVideoIDs :many
select *
from catalog.video_user_states
where user_id = sqlc.arg(user_id)
  and video_id = any(sqlc.arg(video_ids)::uuid[])
order by video_id asc;
