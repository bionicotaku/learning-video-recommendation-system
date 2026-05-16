-- name: HasVisibleVideoForEndQuiz :one
select exists (
  select 1
  from catalog.videos v
  where v.video_id = sqlc.arg(video_id)::uuid
    and v.status = 'active'
    and v.visibility_status = 'public'
    and (v.publish_at is null or v.publish_at <= now())
)::boolean;

-- name: ListVideoUnitQuizQuestionCandidates :many
select
  q.question_id,
  q.scope_type,
  q.question_type,
  q.coarse_unit_id,
  q.target_text,
  q.context_sentence_index,
  q.context_span_index,
  q.context_start_ms,
  q.context_end_ms,
  q.content_payload
from catalog.questions q
where q.video_id = sqlc.arg(video_id)::uuid
  and q.coarse_unit_id = any(sqlc.arg(coarse_unit_ids)::bigint[])
  and q.scope_type = 'video_unit'
  and q.status = 'active'
order by q.coarse_unit_id, q.created_at desc, q.question_id;

-- name: ListUnitQuizQuestionCandidates :many
select
  q.question_id,
  q.scope_type,
  q.question_type,
  q.coarse_unit_id,
  q.target_text,
  q.context_sentence_index,
  q.context_span_index,
  q.context_start_ms,
  q.context_end_ms,
  q.content_payload
from catalog.questions q
where q.video_id is null
  and q.coarse_unit_id = any(sqlc.arg(coarse_unit_ids)::bigint[])
  and q.scope_type = 'unit'
  and q.status = 'active'
order by q.coarse_unit_id, q.created_at desc, q.question_id;
