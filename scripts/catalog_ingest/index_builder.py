from __future__ import annotations

from collections import defaultdict
from decimal import Decimal, ROUND_HALF_UP

from .models import (
    LoadedClipInput,
    NormalizedClipData,
    NormalizedCoreRows,
    VideoSemanticSpanRow,
    VideoTranscriptRow,
    VideoUnitIndexRow,
)


RATIO_PRECISION = Decimal("0.00001")


def build_normalized_clip_data(
    clip_input: LoadedClipInput,
    core_rows: NormalizedCoreRows,
) -> NormalizedClipData:
    """基于基础行构建完整写库数据。

    这个阶段负责两类派生结果：
    - transcript 顶层摘要
    - video_unit_index 聚合索引
    """

    transcript_row = _build_transcript_row(clip_input, core_rows)
    unit_index_rows = _build_unit_index_rows(core_rows, video_duration_ms=clip_input.duration_ms)

    return NormalizedClipData(
        video=core_rows.video,
        transcript=transcript_row,
        sentences=core_rows.sentences,
        spans=core_rows.spans,
        unit_indexes=unit_index_rows,
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
    full_text = " ".join(sentence.text.strip() for sentence in core_rows.sentences if sentence.text.strip())

    return VideoTranscriptRow(
        transcript_object_path=clip_input.transcript_object_path or "",
        transcript_checksum=clip_input.transcript_checksum or "",
        transcript_format_version=clip_input.transcript_format_version,
        full_text=full_text,
        sentence_count=sentence_count,
        semantic_span_count=semantic_span_count,
        mapped_span_count=mapped_span_count,
        unmapped_span_count=unmapped_span_count,
        mapped_span_ratio=mapped_span_ratio,
    )


def _build_unit_index_rows(
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

    rows: list[VideoUnitIndexRow] = []
    for coarse_unit_id, spans in sorted(grouped_spans.items(), key=lambda item: item[0]):
        sentence_indexes = tuple(sorted({span.sentence_index for span in spans}))
        first_start_ms = min(span.start_ms for span in spans)
        last_end_ms = max(span.end_ms for span in spans)
        coverage_ms = _merge_intervals_and_measure([(span.start_ms, span.end_ms) for span in spans])
        coverage_ratio = _safe_ratio(coverage_ms, video_duration_ms)

        # 当前 evidence 策略保持简单且稳定：
        # 先按时间顺序排序，再取前 3 个代表 span。
        # 这足够支撑 recall 的轻量证据定位，不额外引入复杂评分。
        evidence_spans = sorted(spans, key=lambda span: (span.start_ms, span.end_ms, span.sentence_index, span.span_index))[:3]

        # 这里必须明确遵守 README 里的约定：
        # evidence_sentence_indexes[i] 与 evidence_span_indexes[i] 按位置配对解释。
        evidence_sentence_indexes = tuple(span.sentence_index for span in evidence_spans)
        evidence_span_indexes = tuple(span.span_index for span in evidence_spans)

        # sample_surface_forms 只保留去重后的前几个表面形式，避免数组无限膨胀。
        sample_surface_forms = _dedupe_surface_forms([span.text for span in evidence_spans] + [span.text for span in spans])

        rows.append(
            VideoUnitIndexRow(
                coarse_unit_id=coarse_unit_id,
                mention_count=len(spans),
                sentence_count=len(sentence_indexes),
                first_start_ms=first_start_ms,
                last_end_ms=last_end_ms,
                coverage_ms=coverage_ms,
                coverage_ratio=coverage_ratio,
                sentence_indexes=sentence_indexes,
                evidence_sentence_indexes=evidence_sentence_indexes,
                evidence_span_indexes=evidence_span_indexes,
                sample_surface_forms=sample_surface_forms,
            )
        )

    return tuple(rows)


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


def _dedupe_surface_forms(surface_forms: list[str], limit: int = 5) -> tuple[str, ...]:
    """对表面形式去重并限制个数。"""

    deduped: list[str] = []
    seen: set[str] = set()
    for text in surface_forms:
        normalized = text.strip()
        if not normalized or normalized in seen:
            continue
        seen.add(normalized)
        deduped.append(normalized)
        if len(deduped) >= limit:
            break
    return tuple(deduped)
