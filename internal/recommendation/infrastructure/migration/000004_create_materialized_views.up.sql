create materialized view if not exists recommendation.v_video_unit_recall_index as
with scored as (
  select
    vui.video_id,
    vui.coarse_unit_id,
    vui.mention_count,
    vui.sentence_count,
    vui.coverage_ms,
    vui.coverage_ratio,
    vui.sentence_indexes,
    vui.best_evidence_sentence_index,
    vui.best_evidence_span_index,
    vui.best_evidence_start_ms,
    vui.best_evidence_end_ms,
    vui.best_evidence_scores,
    vui.best_evidence_question_reject_reason,
    vui.best_evidence_selection_reason,
    vui.best_evidence_candidate_score,
    vui.best_evidence_target_text,
    v.duration_ms,
    vt.mapped_span_ratio,
    v.status,
    v.visibility_status,
    v.publish_at,
    round((
      coalesce(vui.best_evidence_candidate_score, 0)::numeric / 10.0 * 0.45
      + vui.coverage_ratio * 0.25
      + least(vui.mention_count::numeric / 4.0, 1.0) * 0.15
      + least(vui.sentence_count::numeric / 3.0, 1.0) * 0.10
      + vt.mapped_span_ratio * 0.05
    ), 6)::numeric(10,6) as content_quality_score
  from catalog.video_unit_index as vui
  join catalog.videos as v on v.video_id = vui.video_id
  join catalog.video_transcripts as vt on vt.video_id = vui.video_id
  where v.status = 'active'
    and v.visibility_status = 'public'
    and (v.publish_at is null or v.publish_at <= now())
)
select
  scored.*,
  row_number() over (
    partition by coarse_unit_id
    order by content_quality_score desc, coverage_ratio desc, mention_count desc, video_id asc
  )::integer as rank_within_unit
from scored;
