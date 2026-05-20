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
    _validate_clip_metadata(clip_input)
    _validate_transcript_level_fields(clip_input)
    _validate_time_range(clip_input, time_tolerance_ms=time_tolerance_ms)
    warnings = _validate_sentence_and_token_structure(clip_input)
    _validate_coarse_ids(clip_input, known_coarse_set)
    _validate_questions(clip_input, known_coarse_set)
    _validate_selected_coarse_unit_refs(clip_input)
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
    if not clip_input.video_object_path:
        raise _error(clip_input, "manifest_invalid", "video_object_path 不能为空")


def _validate_clip_metadata(clip_input: LoadedClipInput) -> None:
    """校验 mapped transcript 顶层 clip metadata 的基础事实。"""

    clip = clip_input.clip_metadata
    if clip.clip_id <= 0:
        raise _error(clip_input, "manifest_invalid", "clip_id 必须为正整数")
    if clip.start_index is not None and clip.start_index < 0:
        raise _error(clip_input, "manifest_invalid", "start_index 不能为负数")
    if clip.end_index is not None and clip.end_index < 0:
        raise _error(clip_input, "manifest_invalid", "end_index 不能为负数")
    if clip.start_index is not None and clip.end_index is not None and clip.end_index < clip.start_index:
        raise _error(clip_input, "manifest_invalid", "end_index 必须大于等于 start_index")
    if clip.buffered_end_time <= clip.buffered_start_time:
        raise _error(clip_input, "manifest_invalid", "buffered_end_time 必须大于 buffered_start_time")
    if clip_input.source_start_ms != clip.buffered_start_time:
        raise _error(clip_input, "manifest_invalid", "source_start_ms 必须等于 buffered_start_time")
    if clip_input.source_end_ms != clip.buffered_end_time:
        raise _error(clip_input, "manifest_invalid", "source_end_ms 必须等于 buffered_end_time")
    expected_duration_ms = clip.buffered_end_time - clip.buffered_start_time
    if clip.duration_time != expected_duration_ms:
        raise _error(
            clip_input,
            "manifest_invalid",
            "duration_time 必须等于 buffered 区间长度",
            {"duration_time": clip.duration_time, "expected_duration_ms": expected_duration_ms},
        )
    if clip_input.duration_ms != clip.duration_time:
        raise _error(clip_input, "manifest_invalid", "duration_ms 必须等于 duration_time")
    _validate_engagement_score(clip_input)


def _validate_engagement_score(clip_input: LoadedClipInput) -> None:
    """校验可选内容打分字段的轻量结构。"""

    engagement = clip_input.clip_metadata.engagement
    for key in ("drama", "humor", "payoff", "standalone"):
        value = engagement.get(key)
        if value is None:
            continue
        if not isinstance(value, int):
            raise _error(
                clip_input,
                "manifest_invalid",
                "engagement 数值字段必须为整数",
                {"field": key, "value": value},
            )
        if value < 0 or value > 10:
            raise _error(
                clip_input,
                "manifest_invalid",
                "engagement 数值字段必须在 0 到 10 之间",
                {"field": key, "value": value},
            )

    reasoning = engagement.get("reasoning")
    if reasoning is not None and not isinstance(reasoning, str):
        raise _error(
            clip_input,
            "manifest_invalid",
            "engagement.reasoning 必须为字符串",
            {"value": reasoning},
        )


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
                    "token 必须包含 semantic_element 对象",
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


def _validate_questions(clip_input: LoadedClipInput, known_coarse_set: set[int]) -> None:
    """校验 question JSON 的题目结构是否符合 catalog.questions 契约。"""

    if clip_input.question_file_path is None:
        raise _error(clip_input, "question_invalid", "question_file_path 不能为空")
    if clip_input.raw_question_payload is None:
        raise _error(clip_input, "question_invalid", "raw_question_payload 不能为空")
    if clip_input.selected_coarse_unit_refs is None:
        raise _error(clip_input, "question_invalid", "selected_coarse_unit_refs 不能为空")

    valid_question_types = {
        "context_meaning_choice",
        "unit_meaning_choice",
        "context_cloze_choice",
        "reverse_identification_choice",
    }
    valid_statuses = {"draft", "active", "retired", "rejected"}

    for question in clip_input.questions:
        if question.scope_type != "video_unit":
            raise _error(
                clip_input,
                "question_invalid",
                "Catalog ingest 只允许写入 video_unit question",
                {"scope_type": question.scope_type},
            )
        if question.question_type not in valid_question_types:
            raise _error(
                clip_input,
                "question_invalid",
                "question.question_type 不合法",
                {"question_type": question.question_type},
            )
        if question.status not in valid_statuses:
            raise _error(clip_input, "question_invalid", "question.status 不合法", {"status": question.status})
        if question.coarse_unit_id not in known_coarse_set:
            raise _error(
                clip_input,
                "coarse_unit_missing",
                "question.coarse_unit_id 在 semantic.coarse_unit 中不存在",
                {"coarse_id": question.coarse_unit_id},
            )
        if not question.target_text.strip():
            raise _error(clip_input, "question_invalid", "question.target_text 不能为空")
        _validate_content_payload(clip_input, question.content_payload)
        missing_context_fields = [
            name
            for name, value in (
                ("context_sentence_index", question.context_sentence_index),
                ("context_span_index", question.context_span_index),
                ("context_start_ms", question.context_start_ms),
                ("context_end_ms", question.context_end_ms),
            )
            if value is None
        ]
        if missing_context_fields:
            raise _error(
                clip_input,
                "question_invalid",
                "video_unit question 必须包含完整 context 字段",
                {"missing_context_fields": missing_context_fields},
            )
        if (
            question.context_start_ms is not None
            and question.context_end_ms is not None
            and question.context_end_ms <= question.context_start_ms
        ):
            raise _error(
                clip_input,
                "question_invalid",
                "question.context_end_ms 必须大于 context_start_ms",
                {
                    "context_start_ms": question.context_start_ms,
                    "context_end_ms": question.context_end_ms,
                },
            )


def _validate_content_payload(clip_input: LoadedClipInput, content_payload: dict[str, object]) -> None:
    """校验前端题目 payload 的最低稳定契约。"""

    question_text = content_payload.get("question")
    if not isinstance(question_text, str) or not question_text.strip():
        raise _error(clip_input, "question_invalid", "content_payload.question 不能为空")

    options = content_payload.get("options")
    if not isinstance(options, list) or not options:
        raise _error(clip_input, "question_invalid", "content_payload.options 必须是非空数组")

    option_ids: list[str] = []
    for option in options:
        if not isinstance(option, dict):
            raise _error(clip_input, "question_invalid", "content_payload.options 每一项都必须是对象")
        option_id = option.get("id")
        option_text = option.get("text")
        if not isinstance(option_id, str) or not option_id.strip():
            raise _error(clip_input, "question_invalid", "content_payload.options[].id 不能为空")
        if not isinstance(option_text, str) or not option_text.strip():
            raise _error(clip_input, "question_invalid", "content_payload.options[].text 不能为空")
        option_ids.append(option_id)

    if "correct" not in option_ids:
        raise _error(clip_input, "question_invalid", "content_payload.options 必须包含 id=correct")
    duplicated_option_ids = [option_id for option_id, count in Counter(option_ids).items() if count > 1]
    if duplicated_option_ids:
        raise _error(
            clip_input,
            "question_invalid",
            "content_payload.options[].id 不能重复",
            {"duplicated_option_ids": duplicated_option_ids},
        )


def _validate_selected_coarse_unit_refs(clip_input: LoadedClipInput) -> None:
    """校验 selected refs 与 transcript 中 mapped coarse unit 严格一对一。"""

    selected_refs = clip_input.selected_coarse_unit_refs
    if selected_refs is None:
        raise _error(clip_input, "question_invalid", "selected_coarse_unit_refs 不能为空")

    token_by_ref: dict[tuple[int, int], int | None] = {}
    mapped_coarse_unit_ids: set[int] = set()
    for sentence in clip_input.transcript_sentences:
        for token in sentence.tokens:
            coarse_id = token.semantic_element.coarse_id if token.semantic_element else None
            token_by_ref[(sentence.index, token.index)] = coarse_id
            if coarse_id is not None:
                mapped_coarse_unit_ids.add(coarse_id)

    ref_coarse_unit_ids = [ref.coarse_unit_id for ref in selected_refs.refs]
    duplicated_ref_coarse_unit_ids = [
        coarse_unit_id for coarse_unit_id, count in Counter(ref_coarse_unit_ids).items() if count > 1
    ]
    if duplicated_ref_coarse_unit_ids:
        raise _error(
            clip_input,
            "question_invalid",
            "selected_coarse_unit_refs.refs 中 coarse_unit_id 不能重复",
            {"duplicated_coarse_unit_ids": duplicated_ref_coarse_unit_ids},
        )

    ref_coarse_set = set(ref_coarse_unit_ids)
    if ref_coarse_set != mapped_coarse_unit_ids:
        raise _error(
            clip_input,
            "question_invalid",
            "selected_coarse_unit_refs.refs 必须与 transcript mapped coarse units 一对一",
            {
                "missing_ref_coarse_unit_ids": sorted(mapped_coarse_unit_ids - ref_coarse_set),
                "extra_ref_coarse_unit_ids": sorted(ref_coarse_set - mapped_coarse_unit_ids),
            },
        )

    for ref in selected_refs.refs:
        token_coarse_id = token_by_ref.get((ref.sentence_index, ref.token_index))
        if (ref.sentence_index, ref.token_index) not in token_by_ref:
            raise _error(
                clip_input,
                "question_invalid",
                "selected_coarse_unit_refs ref 指向不存在的 token",
                {
                    "coarse_unit_id": ref.coarse_unit_id,
                    "sentence_index": ref.sentence_index,
                    "token_index": ref.token_index,
                },
            )
        if token_coarse_id != ref.coarse_unit_id:
            raise _error(
                clip_input,
                "question_invalid",
                "selected_coarse_unit_refs ref 与 token coarse_id 不一致",
                {
                    "ref_coarse_unit_id": ref.coarse_unit_id,
                    "token_coarse_unit_id": token_coarse_id,
                    "sentence_index": ref.sentence_index,
                    "token_index": ref.token_index,
                },
            )
        _validate_selected_ref_metadata(clip_input, ref)


def _validate_selected_ref_metadata(clip_input: LoadedClipInput, ref) -> None:
    """校验 selected best evidence 的可审计打分字段。"""

    required_score_keys = (
        "visual_context",
        "context_clarity",
        "learning_value",
        "representative_salience",
    )
    missing_score_keys = [key for key in required_score_keys if key not in ref.scores]
    if missing_score_keys:
        raise _error(
            clip_input,
            "question_invalid",
            "selected_coarse_unit_refs ref.scores 缺少必填字段",
            {
                "coarse_unit_id": ref.coarse_unit_id,
                "missing_score_keys": missing_score_keys,
            },
        )

    for key in required_score_keys:
        value = ref.scores.get(key)
        if type(value) is not int or value < 0 or value > 10:
            raise _error(
                clip_input,
                "question_invalid",
                "selected_coarse_unit_refs ref.scores 数值字段必须为 0 到 10 的整数",
                {
                    "coarse_unit_id": ref.coarse_unit_id,
                    "field": key,
                    "value": value,
                },
            )

    if ref.question_reject_reason is not None and not ref.question_reject_reason.strip():
        raise _error(
            clip_input,
            "question_invalid",
            "selected_coarse_unit_refs ref.question_reject_reason 不能是空字符串",
            {"coarse_unit_id": ref.coarse_unit_id},
        )
    if not isinstance(ref.selection_reason, str) or not ref.selection_reason.strip():
        raise _error(
            clip_input,
            "question_invalid",
            "selected_coarse_unit_refs ref.selection_reason 不能为空字符串",
            {"coarse_unit_id": ref.coarse_unit_id},
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
