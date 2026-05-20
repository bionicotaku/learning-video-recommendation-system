-- name: ListLearningStatesForRecommendation :many
select
  user_id,
  coarse_unit_id,
  is_target,
  target_priority,
  status,
  mastery_score,
  last_progress_quality,
  next_review_at,
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
  coverage_ms,
  coverage_ratio,
  sentence_indexes,
  best_evidence_sentence_index,
  best_evidence_span_index,
  best_evidence_candidate_score,
  best_evidence_target_text,
  duration_ms,
  mapped_span_ratio
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

-- name: ListSemanticSpansByRefs :many
with input as (
  select distinct
    (item->>'video_id')::uuid as video_id,
    (item->>'coarse_unit_id')::bigint as coarse_unit_id,
    (item->>'sentence_index')::integer as sentence_index,
    (item->>'span_index')::integer as span_index
  from jsonb_array_elements(sqlc.arg(refs)::jsonb) as refs(item)
)
select
  spans.video_id,
  spans.sentence_index,
  spans.span_index,
  spans.coarse_unit_id,
  spans.start_ms,
  spans.end_ms,
  spans.surface_text,
  spans.explanation,
  spans.base_form,
  spans.translation,
  spans.dictionary,
  spans.mapping_reason
from catalog.video_semantic_spans spans
join input
  on input.video_id = spans.video_id
 and input.coarse_unit_id = spans.coarse_unit_id
 and input.sentence_index = spans.sentence_index
 and input.span_index = spans.span_index
order by spans.video_id, spans.coarse_unit_id, spans.sentence_index, spans.span_index;

-- name: ListTranscriptSentencesByRefs :many
with input as (
  select distinct
    (item->>'video_id')::uuid as video_id,
    (item->>'sentence_index')::integer as sentence_index
  from jsonb_array_elements(sqlc.arg(refs)::jsonb) as refs(item)
)
select
  sentences.video_id,
  sentences.sentence_index,
  sentences.start_ms,
  sentences.end_ms,
  sentences.text,
  sentences.translation
from catalog.video_transcript_sentences sentences
join input
  on input.video_id = sentences.video_id
 and input.sentence_index = sentences.sentence_index
order by sentences.video_id, sentences.sentence_index;

-- name: ListVideoUserStatesByUserAndVideoIDs :many
select user_id, video_id, last_watched_at, watch_count, completed_count, last_position_ms, max_position_ms, total_watch_ms
from catalog.video_user_states
where user_id = sqlc.arg(user_id)
  and video_id = any(sqlc.arg(video_ids)::uuid[])
order by video_id asc;
