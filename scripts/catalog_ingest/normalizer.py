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
        source_start_sentence_index=clip_input.clip_metadata.start_index,
        source_end_sentence_index=clip_input.clip_metadata.end_index,
        title=clip_input.title,
        description=clip_input.description,
        clip_reason=clip_input.clip_reason,
        engagement_score=clip_input.clip_metadata.engagement,
        language=clip_input.language,
        duration_ms=clip_input.duration_ms,
        video_object_path=clip_input.video_object_path,
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
                start_ms=sentence.start_ms,
                end_ms=sentence.end_ms,
                text=sentence.text,
                translation=sentence.translation,
            )
        )

        for token in sentence.tokens:
            semantic_element = token.semantic_element
            span_rows.append(
                VideoSemanticSpanRow(
                    sentence_index=sentence.index,
                    span_index=token.index,
                    start_ms=token.start_ms,
                    end_ms=token.end_ms,
                    coarse_unit_id=semantic_element.coarse_id if semantic_element else None,
                    surface_text=token.text,
                    explanation=token.explanation,
                    base_form=semantic_element.base_form if semantic_element else None,
                    translation=semantic_element.translation if semantic_element else None,
                    dictionary=semantic_element.dictionary if semantic_element else None,
                    mapping_reason=semantic_element.reason if semantic_element else None,
                )
            )

    return NormalizedCoreRows(
        video=video_row,
        sentences=tuple(sentence_rows),
        spans=tuple(span_rows),
    )
