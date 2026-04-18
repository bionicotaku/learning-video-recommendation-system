from __future__ import annotations

from collections import Counter
from dataclasses import dataclass
from typing import Iterable

from .models import CatalogIngestError, LoadedClipInput


@dataclass(slots=True, frozen=True)
class ValidationWarning:
    """表示校验阶段发现的非阻断性告警。

    这类问题不会阻止当前 clip 入库，但需要：
    - 在命令行结果里暴露
    - 在审计表 warning_codes 中留下痕迹
    - 必要时把细节带到 repository 的 context 中，便于后续排查
    """

    code: str
    message: str
    context: dict[str, object]


def validate_loaded_clip(
    clip_input: LoadedClipInput,
    known_coarse_unit_ids: Iterable[int],
    time_tolerance_ms: int = 0,
) -> tuple[ValidationWarning, ...]:
    """校验单 clip 输入是否满足 catalog 入库规则。

    这里专门做“规则判断”，不做数据库写入，也不做标准化映射。
    main 在拿到 loader 输出后，应先走这里，再进入 normalizer。

    如果发现不合法数据，直接抛出 CatalogIngestError。
    """

    if clip_input.skip_reason_code is not None:
        # 缺 transcript 的情况属于已知跳过分支，不是失败分支。
        # 这种输入对象不进入完整校验流程，直接由 main 写 skipped 审计。
        return tuple()

    known_coarse_set = set(known_coarse_unit_ids)

    _validate_video_level_fields(clip_input)
    _validate_parent_clip(clip_input)
    _validate_transcript_level_fields(clip_input)
    _validate_time_range(clip_input, time_tolerance_ms=time_tolerance_ms)
    warnings = _validate_sentence_and_token_structure(clip_input)
    _validate_coarse_ids(clip_input, known_coarse_set)
    return warnings


def _validate_video_level_fields(clip_input: LoadedClipInput) -> None:
    """校验 video 主表层面的字段。"""

    if not clip_input.source_clip_key:
        raise _error(clip_input, "manifest_invalid", "source_clip_key 不能为空")
    if not clip_input.parent_video_name:
        raise _error(clip_input, "manifest_invalid", "parent_video_name 不能为空")
    if not clip_input.parent_video_slug:
        raise _error(clip_input, "manifest_invalid", "parent_video_slug 不能为空")
    if clip_input.clip_seq <= 0:
        raise _error(clip_input, "manifest_invalid", "clip_seq 必须为正整数")
    if not clip_input.title:
        raise _error(clip_input, "manifest_invalid", "title 不能为空")
    if clip_input.duration_ms <= 0:
        raise _error(clip_input, "manifest_invalid", "duration_ms 必须大于 0")
    if not clip_input.hls_master_playlist_path:
        raise _error(clip_input, "manifest_invalid", "hls_master_playlist_path 不能为空")


def _validate_parent_clip(clip_input: LoadedClipInput) -> None:
    """校验父文件中单个 clip 的基础事实。"""

    clip = clip_input.parent_clip
    if clip.clip_id <= 0:
        raise _error(clip_input, "manifest_invalid", "clip_id 必须为正整数")
    if clip.buffered_end_time <= clip.buffered_start_time:
        raise _error(clip_input, "manifest_invalid", "buffered_end_time 必须大于 buffered_start_time")
    if clip_input.source_start_ms != clip.buffered_start_time:
        raise _error(clip_input, "manifest_invalid", "source_start_ms 必须等于 buffered_start_time")
    if clip_input.source_end_ms != clip.buffered_end_time:
        raise _error(clip_input, "manifest_invalid", "source_end_ms 必须等于 buffered_end_time")
    if clip_input.duration_ms != (clip.buffered_end_time - clip.buffered_start_time):
        raise _error(clip_input, "manifest_invalid", "duration_ms 必须等于 buffered 区间长度")


def _validate_transcript_level_fields(clip_input: LoadedClipInput) -> None:
    """校验 transcript 顶层必须存在的输入。"""

    if clip_input.transcript_file_path is None:
        raise _error(clip_input, "transcript_invalid", "transcript_file_path 不能为空")
    if clip_input.transcript_object_path is None:
        raise _error(clip_input, "transcript_invalid", "transcript_object_path 不能为空")
    if clip_input.transcript_checksum is None:
        raise _error(clip_input, "transcript_invalid", "transcript_checksum 不能为空")
    if clip_input.raw_transcript_payload is None:
        raise _error(clip_input, "transcript_invalid", "raw_transcript_payload 不能为空")
    if not clip_input.transcript_sentences:
        raise _error(clip_input, "transcript_invalid", "空 transcript 不能入库")


def _validate_time_range(clip_input: LoadedClipInput, time_tolerance_ms: int) -> None:
    """校验 transcript 时间轴整体是否落在 buffered 区间内。"""

    earliest_start = min(sentence.start_ms for sentence in clip_input.transcript_sentences)
    latest_end = max(sentence.end_ms for sentence in clip_input.transcript_sentences)

    if earliest_start < (clip_input.source_start_ms - time_tolerance_ms):
        raise _error(
            clip_input,
            "transcript_invalid",
            "transcript 最早时间早于 buffered_start_time",
            {"earliest_start": earliest_start, "source_start_ms": clip_input.source_start_ms},
        )
    if latest_end > (clip_input.source_end_ms + time_tolerance_ms):
        raise _error(
            clip_input,
            "transcript_invalid",
            "transcript 最晚时间晚于 buffered_end_time",
            {"latest_end": latest_end, "source_end_ms": clip_input.source_end_ms},
        )


def _validate_sentence_and_token_structure(clip_input: LoadedClipInput) -> tuple[ValidationWarning, ...]:
    """校验 sentence / token 的索引、文本和时间结构。"""

    warnings: list[ValidationWarning] = []
    sentence_indexes = [sentence.index for sentence in clip_input.transcript_sentences]
    duplicated_sentence_indexes = [index for index, count in Counter(sentence_indexes).items() if count > 1]
    if duplicated_sentence_indexes:
        raise _error(
            clip_input,
            "transcript_invalid",
            "同一 transcript 内 sentence.index 不能重复",
            {"duplicated_sentence_indexes": duplicated_sentence_indexes},
        )

    for sentence in clip_input.transcript_sentences:
        if sentence.index < 0:
            raise _error(clip_input, "transcript_invalid", "sentence.index 不能为负数", {"sentence_index": sentence.index})
        if not sentence.text.strip():
            raise _error(clip_input, "transcript_invalid", "sentence.text 不能为空字符串", {"sentence_index": sentence.index})
        if sentence.start_ms < 0:
            raise _error(clip_input, "transcript_invalid", "sentence.start_ms 不能为负数", {"sentence_index": sentence.index})
        if sentence.end_ms <= sentence.start_ms:
            raise _error(clip_input, "transcript_invalid", "sentence.end_ms 必须大于 sentence.start_ms", {"sentence_index": sentence.index})

        token_indexes = [token.index for token in sentence.tokens]
        duplicated_token_indexes = [index for index, count in Counter(token_indexes).items() if count > 1]
        if duplicated_token_indexes:
            raise _error(
                clip_input,
                "transcript_invalid",
                "同一句内 token.index 不能重复",
                {"sentence_index": sentence.index, "duplicated_token_indexes": duplicated_token_indexes},
            )

        for token in sentence.tokens:
            if token.semantic_element is None:
                raise _error(
                    clip_input,
                    "transcript_invalid",
                    "token 必须包含 semanticElement 对象",
                    {"sentence_index": sentence.index, "token_index": token.index},
                )
            if token.index < 0:
                raise _error(
                    clip_input,
                    "transcript_invalid",
                    "token.index 不能为负数",
                    {"sentence_index": sentence.index, "token_index": token.index},
                )
            if not token.text.strip():
                raise _error(
                    clip_input,
                    "transcript_invalid",
                    "token.text 不能为空字符串",
                    {"sentence_index": sentence.index, "token_index": token.index},
                )
            if token.start_ms < 0:
                raise _error(
                    clip_input,
                    "transcript_invalid",
                    "token.start_ms 不能为负数",
                    {"sentence_index": sentence.index, "token_index": token.index},
                )
            if token.end_ms <= token.start_ms:
                raise _error(
                    clip_input,
                    "transcript_invalid",
                    "token.end_ms 必须大于 token.start_ms",
                    {"sentence_index": sentence.index, "token_index": token.index},
                )
            if token.start_ms < sentence.start_ms or token.end_ms > sentence.end_ms:
                warnings.append(
                    ValidationWarning(
                        code="token_time_outside_sentence",
                        message="token 时间超出所属 sentence 区间，按 warning 继续入库",
                        context={
                            "sentence_index": sentence.index,
                            "token_index": token.index,
                            "token_start_ms": token.start_ms,
                            "token_end_ms": token.end_ms,
                            "sentence_start_ms": sentence.start_ms,
                            "sentence_end_ms": sentence.end_ms,
                        },
                    )
                )

    return tuple(warnings)


def _validate_coarse_ids(clip_input: LoadedClipInput, known_coarse_set: set[int]) -> None:
    """校验所有非空 coarse_id 都真实存在于 semantic.coarse_unit。"""

    for sentence in clip_input.transcript_sentences:
        for token in sentence.tokens:
            if token.semantic_element.coarse_id is None:
                continue
            if token.semantic_element.coarse_id not in known_coarse_set:
                raise _error(
                    clip_input,
                    "coarse_unit_missing",
                    "coarse_id 在 semantic.coarse_unit 中不存在",
                    {
                        "sentence_index": sentence.index,
                        "token_index": token.index,
                        "coarse_id": token.semantic_element.coarse_id,
                    },
                )


def _error(
    clip_input: LoadedClipInput,
    code: str,
    message: str,
    extra_context: dict[str, object] | None = None,
) -> CatalogIngestError:
    """统一构造带 clip 上下文的结构化错误。"""

    context = dict(clip_input.context)
    if extra_context:
        context.update(extra_context)
    return CatalogIngestError(
        code=code,
        stage="validator",
        message=message,
        context=context,
    )
