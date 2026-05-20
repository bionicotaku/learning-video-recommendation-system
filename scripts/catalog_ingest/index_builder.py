from __future__ import annotations

import hashlib
import json
import uuid
from collections import defaultdict
from decimal import Decimal, ROUND_HALF_UP

from .models import (
    BestEvidenceRef,
    CatalogIngestError,
    LoadedClipInput,
    NormalizedClipData,
    NormalizedCoreRows,
    QuestionRow,
    VideoSemanticSpanRow,
    VideoTranscriptRow,
    VideoUnitIndexRow,
)


RATIO_PRECISION = Decimal("0.00001")
QUESTION_ID_NAMESPACE = uuid.UUID("e5a7c9e1-379e-4e0f-9d3b-1c47bf07c701")


def build_normalized_clip_data(
    clip_input: LoadedClipInput,
    core_rows: NormalizedCoreRows,
) -> NormalizedClipData:
    """基于基础行构建完整写库数据。

    这个阶段负责三类派生结果：
    - transcript 顶层摘要
    - video_unit_index 聚合索引
    - catalog.questions 写入行
    """

    transcript_row = _build_transcript_row(clip_input, core_rows)
    unit_index_rows = _build_unit_index_rows(
        clip_input=clip_input,
        core_rows=core_rows,
        video_duration_ms=clip_input.duration_ms,
    )
    question_rows = _build_question_rows(clip_input)

    return NormalizedClipData(
        video=core_rows.video,
        transcript=transcript_row,
        sentences=core_rows.sentences,
        spans=core_rows.spans,
        unit_indexes=unit_index_rows,
        questions=question_rows,
    )


def _build_transcript_row(
    clip_input: LoadedClipInput,
    core_rows: NormalizedCoreRows,
) -> VideoTranscriptRow:
    """构建 transcript 顶层摘要行。"""

    sentence_count = len(core_rows.sentences)
    semantic_span_count = len(core_rows.spans)
    mapped_span_count = sum(1 for span in core_rows.spans if span.coarse_unit_id is not None)
    unmapped_span_count = semantic_span_count - mapped_span_count
    mapped_span_ratio = _safe_ratio(mapped_span_count, semantic_span_count)

    return VideoTranscriptRow(
        transcript_object_path=clip_input.transcript_object_path or "",
        transcript_checksum=clip_input.transcript_checksum or "",
        transcript_format_version=clip_input.transcript_format_version,
        sentence_count=sentence_count,
        semantic_span_count=semantic_span_count,
        mapped_span_count=mapped_span_count,
        unmapped_span_count=unmapped_span_count,
        mapped_span_ratio=mapped_span_ratio,
    )


def _build_unit_index_rows(
    clip_input: LoadedClipInput,
    core_rows: NormalizedCoreRows,
    video_duration_ms: int,
) -> tuple[VideoUnitIndexRow, ...]:
    """按 `(video_id, coarse_unit_id)` 的逻辑构建视频级 unit 索引。

    这里虽然还没有真正的 video_id，但聚合维度和最终落表规则已经完全一致。
    """

    grouped_spans: dict[int, list[VideoSemanticSpanRow]] = defaultdict(list)
    for span in core_rows.spans:
        if span.coarse_unit_id is None:
            continue
        grouped_spans[span.coarse_unit_id].append(span)

    if clip_input.selected_coarse_unit_refs is None:
        raise CatalogIngestError(
            code="question_invalid",
            stage="index_builder",
            message="selected_coarse_unit_refs 不能为空",
            context=clip_input.context,
        )

    selected_ref_by_unit = {
        ref.coarse_unit_id: ref
        for ref in clip_input.selected_coarse_unit_refs.refs
    }
    rows: list[VideoUnitIndexRow] = []
    for coarse_unit_id, spans in sorted(grouped_spans.items(), key=lambda item: item[0]):
        sentence_indexes = tuple(sorted({span.sentence_index for span in spans}))
        coverage_ms = _merge_intervals_and_measure([(span.start_ms, span.end_ms) for span in spans])
        coverage_ratio = _safe_ratio(coverage_ms, video_duration_ms)

        selected_ref = selected_ref_by_unit.get(coarse_unit_id)
        if selected_ref is None:
            raise CatalogIngestError(
                code="question_invalid",
                stage="index_builder",
                message="缺少 coarse unit 对应的 selected best evidence ref",
                context={**clip_input.context, "coarse_unit_id": coarse_unit_id},
            )
        best_evidence_span = _resolve_selected_evidence_span(
            spans=spans,
            coarse_unit_id=coarse_unit_id,
            sentence_index=selected_ref.sentence_index,
            span_index=selected_ref.token_index,
            context=clip_input.context,
        )
        best_evidence_ref = BestEvidenceRef(
            sentence_index=best_evidence_span.sentence_index,
            span_index=best_evidence_span.span_index,
        )

        rows.append(
            VideoUnitIndexRow(
                coarse_unit_id=coarse_unit_id,
                mention_count=len(spans),
                sentence_count=len(sentence_indexes),
                coverage_ms=coverage_ms,
                coverage_ratio=coverage_ratio,
                sentence_indexes=sentence_indexes,
                best_evidence_ref=best_evidence_ref,
                best_evidence_scores=selected_ref.scores,
                best_evidence_question_reject_reason=selected_ref.question_reject_reason,
                best_evidence_selection_reason=selected_ref.selection_reason,
                best_evidence_candidate_score=selected_ref.candidate_score,
                best_evidence_target_text=selected_ref.target_text,
            )
        )

    return tuple(rows)


def _resolve_selected_evidence_span(
    spans: list[VideoSemanticSpanRow],
    coarse_unit_id: int,
    sentence_index: int,
    span_index: int,
    context: dict[str, object],
) -> VideoSemanticSpanRow:
    """按 selected ref 精确解析 best evidence span。"""

    for span in spans:
        if span.sentence_index == sentence_index and span.span_index == span_index:
            return span
    raise CatalogIngestError(
        code="question_invalid",
        stage="index_builder",
        message="selected best evidence ref 无法回查到 semantic span",
        context={
            **context,
            "coarse_unit_id": coarse_unit_id,
            "sentence_index": sentence_index,
            "span_index": span_index,
        },
    )


def _build_question_rows(clip_input: LoadedClipInput) -> tuple[QuestionRow, ...]:
    """构建 catalog.questions 写入行。"""

    rows: list[QuestionRow] = []
    for question in clip_input.questions:
        rows.append(
            QuestionRow(
                question_id=_deterministic_question_id(clip_input.source_clip_key, question),
                scope_type="video_unit",
                question_type=question.question_type,
                coarse_unit_id=question.coarse_unit_id,
                target_text=question.target_text,
                context_sentence_index=question.context_sentence_index,
                context_span_index=question.context_span_index,
                context_start_ms=question.context_start_ms,
                context_end_ms=question.context_end_ms,
                content_payload=question.content_payload,
                status="active",
            )
        )
    return tuple(rows)


def _deterministic_question_id(source_clip_key: str, question) -> str:
    """基于稳定输入生成可重复的 question_id。"""

    canonical_payload = json.dumps(
        question.content_payload,
        ensure_ascii=False,
        sort_keys=True,
        separators=(",", ":"),
    )
    payload_hash = hashlib.sha256(canonical_payload.encode("utf-8")).hexdigest()
    name = "|".join(
        [
            source_clip_key,
            str(question.coarse_unit_id),
            question.question_type,
            str(question.context_sentence_index),
            str(question.context_span_index),
            payload_hash,
        ]
    )
    return str(uuid.uuid5(QUESTION_ID_NAMESPACE, name))


def _merge_intervals_and_measure(intervals: list[tuple[int, int]]) -> int:
    """合并重叠区间并计算总覆盖时长。"""

    if not intervals:
        return 0

    sorted_intervals = sorted(intervals)
    merged: list[list[int]] = [[sorted_intervals[0][0], sorted_intervals[0][1]]]
    for start_ms, end_ms in sorted_intervals[1:]:
        current = merged[-1]
        if start_ms <= current[1]:
            current[1] = max(current[1], end_ms)
        else:
            merged.append([start_ms, end_ms])
    return sum(end_ms - start_ms for start_ms, end_ms in merged)


def _safe_ratio(numerator: int, denominator: int) -> Decimal:
    """计算保留 5 位小数的比例值。"""

    if denominator <= 0:
        return Decimal("0").quantize(RATIO_PRECISION)
    return (Decimal(numerator) / Decimal(denominator)).quantize(RATIO_PRECISION, rounding=ROUND_HALF_UP)
