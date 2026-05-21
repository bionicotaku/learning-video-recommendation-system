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
  and status in ('new', 'learning', 'reviewing')
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

-- name: ListMasteredTargetFillVideoCandidates :many
with matched as (
  select
    rvu.video_id,
    max(rvu.duration_ms)::integer as duration_ms,
    count(distinct rvu.coarse_unit_id)::integer as matched_unit_count,
    coalesce(sum(rvu.mention_count), 0)::bigint as total_mention_count,
    coalesce(max(rvu.coverage_ratio), 0)::numeric(10,5) as max_coverage_ratio,
    coalesce(avg(rvu.mapped_span_ratio), 0)::numeric(10,5) as mapped_span_ratio
  from learning.user_unit_states as states
  join recommendation.v_recommendable_video_units as rvu
    on rvu.coarse_unit_id = states.coarse_unit_id
  where states.user_id = sqlc.arg(user_id)
    and states.is_target = true
    and states.status = 'mastered'
    and not (rvu.video_id = any(sqlc.arg(excluded_video_ids)::uuid[]))
  group by rvu.video_id
)
select
  matched.video_id,
  matched.duration_ms,
  matched.matched_unit_count,
  matched.total_mention_count,
  matched.max_coverage_ratio,
  matched.mapped_span_ratio,
  coalesce(stats.view_count, 0)::bigint as view_count,
  coalesce(stats.like_count, 0)::bigint as like_count,
  coalesce(stats.favorite_count, 0)::bigint as favorite_count,
  serving.last_served_at,
  coalesce(serving.served_count, 0)::integer as served_count,
  user_state.last_watched_at,
  coalesce(user_state.watch_count, 0)::integer as watch_count,
  coalesce(user_state.completed_count, 0)::integer as completed_count,
  coalesce(user_state.max_position_ms, 0)::integer as max_position_ms
from matched
left join catalog.video_engagement_stats as stats
  on stats.video_id = matched.video_id
left join recommendation.user_video_serving_states as serving
  on serving.user_id = sqlc.arg(user_id)
 and serving.video_id = matched.video_id
left join catalog.video_user_states as user_state
  on user_state.user_id = sqlc.arg(user_id)
 and user_state.video_id = matched.video_id
order by
  matched.matched_unit_count desc,
  matched.total_mention_count desc,
  matched.max_coverage_ratio desc,
  matched.mapped_span_ratio desc,
  matched.video_id asc
limit sqlc.arg(fill_limit)::integer;

-- name: ListPopularFillVideoCandidates :many
select
  videos.video_id,
  videos.duration_ms,
  0::integer as matched_unit_count,
  0::bigint as total_mention_count,
  0::numeric(10,5) as max_coverage_ratio,
  0::numeric(10,5) as mapped_span_ratio,
  coalesce(stats.view_count, 0)::bigint as view_count,
  coalesce(stats.like_count, 0)::bigint as like_count,
  coalesce(stats.favorite_count, 0)::bigint as favorite_count,
  serving.last_served_at,
  coalesce(serving.served_count, 0)::integer as served_count,
  user_state.last_watched_at,
  coalesce(user_state.watch_count, 0)::integer as watch_count,
  coalesce(user_state.completed_count, 0)::integer as completed_count,
  coalesce(user_state.max_position_ms, 0)::integer as max_position_ms
from catalog.videos as videos
left join catalog.video_engagement_stats as stats
  on stats.video_id = videos.video_id
left join recommendation.user_video_serving_states as serving
  on serving.user_id = sqlc.arg(user_id)
 and serving.video_id = videos.video_id
left join catalog.video_user_states as user_state
  on user_state.user_id = sqlc.arg(user_id)
 and user_state.video_id = videos.video_id
where videos.status = 'active'
  and videos.visibility_status = 'public'
  and (videos.publish_at is null or videos.publish_at <= now())
  and not (videos.video_id = any(sqlc.arg(excluded_video_ids)::uuid[]))
order by
  coalesce(stats.view_count, 0) desc,
  coalesce(stats.like_count, 0) desc,
  coalesce(stats.favorite_count, 0) desc,
  videos.video_id asc
limit sqlc.arg(fill_limit)::integer;

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
