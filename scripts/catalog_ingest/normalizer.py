from __future__ import annotations

from .models import (
    LoadedClipInput,
    NormalizedCoreRows,
    VideoRow,
    VideoSemanticSpanRow,
    VideoTranscriptSentenceRow,
)


def normalize_clip_input(clip_input: LoadedClipInput) -> NormalizedCoreRows:
    """将单 clip 原始输入对象映射为基础数据库行。

    这个模块只做“字段如何落表”的确定性映射，不做：
    - 业务校验
    - transcript 摘要统计
    - unit index 聚合

    后两项都交给 index_builder，避免职责混杂。
    """

    video_row = VideoRow(
        source_clip_key=clip_input.source_clip_key,
        parent_video_name=clip_input.parent_video_name,
        parent_video_slug=clip_input.parent_video_slug,
        clip_seq=clip_input.clip_seq,
        source_start_ms=clip_input.source_start_ms,
        source_end_ms=clip_input.source_end_ms,
        title=clip_input.title,
        description=clip_input.description,
        clip_reason=clip_input.clip_reason,
        language=clip_input.language,
        duration_ms=clip_input.duration_ms,
        hls_master_playlist_path=clip_input.hls_master_playlist_path,
        thumbnail_url=clip_input.thumbnail_url,
        status="active",
        visibility_status="public",
        publish_at=clip_input.publish_at,
    )

    sentence_rows: list[VideoTranscriptSentenceRow] = []
    span_rows: list[VideoSemanticSpanRow] = []

    for sentence in clip_input.transcript_sentences:
        # sentence 行直接沿用 transcript 的绝对时间轴，不做 clip 内归零。
        sentence_rows.append(
            VideoTranscriptSentenceRow(
                sentence_index=sentence.index,
                text=sentence.text,
                start_ms=sentence.start_ms,
                end_ms=sentence.end_ms,
                explanation=sentence.explanation,
            )
        )

        for token in sentence.tokens:
            semantic_element = token.semantic_element
            span_rows.append(
                VideoSemanticSpanRow(
                    sentence_index=sentence.index,
                    span_index=token.index,
                    text=token.text,
                    start_ms=token.start_ms,
                    end_ms=token.end_ms,
                    explanation=token.explanation,
                    coarse_unit_id=semantic_element.coarse_id if semantic_element else None,
                    base_form=semantic_element.base_form if semantic_element else None,
                    dictionary_text=semantic_element.dictionary_text if semantic_element else None,
                )
            )

    return NormalizedCoreRows(
        video=video_row,
        sentences=tuple(sentence_rows),
        spans=tuple(span_rows),
    )
