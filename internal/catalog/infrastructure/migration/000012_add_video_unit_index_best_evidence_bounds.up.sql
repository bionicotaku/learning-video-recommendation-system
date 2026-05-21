alter table catalog.video_unit_index
add column if not exists best_evidence_start_ms integer;

alter table catalog.video_unit_index
add column if not exists best_evidence_end_ms integer;

alter table catalog.video_unit_index
add constraint chk_video_unit_index_best_evidence_bounds
check (
  best_evidence_start_ms is null
  or best_evidence_end_ms is null
  or (
    best_evidence_start_ms >= 0
    and best_evidence_end_ms > best_evidence_start_ms
  )
);
