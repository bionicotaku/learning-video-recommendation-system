alter table catalog.video_unit_index
drop constraint if exists chk_video_unit_index_best_evidence_bounds;

alter table catalog.video_unit_index
drop column if exists best_evidence_end_ms;

alter table catalog.video_unit_index
drop column if exists best_evidence_start_ms;
