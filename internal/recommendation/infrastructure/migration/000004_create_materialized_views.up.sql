create materialized view if not exists recommendation.v_recommendable_video_units as
select
  vui.video_id,
  vui.coarse_unit_id,
  vui.mention_count,
  vui.sentence_count,
  vui.first_start_ms,
  vui.last_end_ms,
  vui.coverage_ms,
  vui.coverage_ratio,
  vui.sentence_indexes,
  vui.evidence_span_refs,
  vui.sample_surface_forms,
  v.duration_ms,
  vt.mapped_span_ratio,
  v.status,
  v.visibility_status,
  v.publish_at
from catalog.video_unit_index as vui
join catalog.videos as v on v.video_id = vui.video_id
join catalog.video_transcripts as vt on vt.video_id = vui.video_id
where v.status = 'active'
  and v.visibility_status = 'public'
  and (v.publish_at is null or v.publish_at <= now());
