create index if not exists idx_videos_parent_video_slug_clip_seq
on catalog.videos (parent_video_slug, clip_seq);

create index if not exists idx_videos_status
on catalog.videos (status);

create index if not exists idx_videos_visibility_publish_at
on catalog.videos (visibility_status, publish_at);

create index if not exists idx_videos_created_at_desc
on catalog.videos (created_at desc);

create index if not exists idx_video_transcript_sentences_video_start_ms
on catalog.video_transcript_sentences (video_id, start_ms);

create index if not exists idx_video_transcript_sentences_video_end_ms
on catalog.video_transcript_sentences (video_id, end_ms);

create index if not exists idx_video_semantic_spans_video_sentence
on catalog.video_semantic_spans (video_id, sentence_index);

create index if not exists idx_video_semantic_spans_video_start_ms
on catalog.video_semantic_spans (video_id, start_ms);

create index if not exists idx_video_semantic_spans_coarse_unit_video
on catalog.video_semantic_spans (coarse_unit_id, video_id)
where coarse_unit_id is not null;

create index if not exists idx_video_semantic_spans_video_coarse_unit
on catalog.video_semantic_spans (video_id, coarse_unit_id)
where coarse_unit_id is not null;

create index if not exists idx_video_unit_index_coarse_unit_mention_coverage
on catalog.video_unit_index (coarse_unit_id, mention_count desc, coverage_ratio desc);

create index if not exists idx_video_unit_index_video_id
on catalog.video_unit_index (video_id);

create index if not exists idx_video_ingestion_records_source_clip_key_started_at
on catalog.video_ingestion_records (source_clip_key, started_at desc);

create index if not exists idx_video_ingestion_records_video_id
on catalog.video_ingestion_records (video_id);

create index if not exists idx_video_ingestion_records_status_started_at
on catalog.video_ingestion_records (status, started_at desc);

create index if not exists idx_video_user_states_video_id
on catalog.video_user_states (video_id);

create index if not exists idx_video_user_states_user_last_watched_at
on catalog.video_user_states (user_id, last_watched_at desc);
