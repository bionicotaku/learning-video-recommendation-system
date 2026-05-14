from __future__ import annotations

import hashlib
import json
import re
from collections import Counter
from pathlib import Path
from typing import Any

from .models import (
    CatalogIngestError,
    LoadedClipInput,
    ParentClipDescriptor,
    QuestionInput,
    SelectedCoarseUnitRef,
    SelectedCoarseUnitRefs,
    TranscriptSemanticElement,
    TranscriptSentence,
    TranscriptToken,
)


def load_clip_inputs(
    parents_dir: Path,
    transcripts_dir: Path,
    questions_dir: Path,
    source_name: str | None = None,
    clip_key: str | None = None,
    limit: int | None = None,
) -> tuple[LoadedClipInput, ...]:
    """扫描三个输入目录，并组装成单 clip 输入对象列表。

    这是整个脚本的第一步。
    它只负责把输入读出来并完成文件名级别的匹配，不负责完整业务校验。

    参数说明：
    - parents_dir：父视频切片描述文件目录
    - transcripts_dir：clip transcript 文件目录
    - questions_dir：clip question 文件目录
    - source_name：写审计记录时使用的来源名
    - clip_key：若指定，则只保留命中的单个 source_clip_key
    - limit：最多返回多少条 clip 输入，用于小批量调试
    """

    parent_files = _scan_json_files(parents_dir, label="parents_dir")
    transcript_files = _scan_json_files(transcripts_dir, label="transcripts_dir")
    question_files = _scan_json_files(questions_dir, label="questions_dir")

    # transcript 目录预先构造成两份轻量索引：
    # 1. 文件名集合：用于快速判断“文件是否存在”
    # 2. 文件名到 Path 的映射：只有确认存在后才取具体路径
    #
    # 这里显式保留 set，是因为当前 skip 逻辑的第一层判断就是“对应 transcript
    # 文件是否存在”。这样后续遍历每个 clip 时不需要反复碰文件系统。
    transcript_name_set = {path.name for path in transcript_files}
    transcript_index = {path.name: path for path in transcript_files}
    question_name_set = {path.name for path in question_files}
    question_index = {path.name: path for path in question_files}

    loaded_items: list[LoadedClipInput] = []
    for parent_file in parent_files:
        remaining_limit = None if limit is None else max(0, limit - len(loaded_items))
        if remaining_limit == 0:
            break

        parent_items = _load_from_parent_file(
            parent_file=parent_file,
            transcript_name_set=transcript_name_set,
            transcript_index=transcript_index,
            question_name_set=question_name_set,
            question_index=question_index,
            source_name=source_name,
            clip_key=clip_key,
            remaining_limit=remaining_limit,
        )
        loaded_items.extend(parent_items)

    _validate_unique_source_clip_keys(loaded_items)

    return tuple(loaded_items)


def _validate_unique_source_clip_keys(loaded_items: list[LoadedClipInput]) -> None:
    """校验本次输入中的 source_clip_key 唯一。

    用户已经明确要求重复校验直接用 source_clip_key。
    因此这里不再单独维护一套 clip_id 重复规则，而是统一按业务唯一键处理。
    一旦出现重复 source_clip_key，就直接视为坏输入并失败。
    """

    duplicated_source_clip_keys = sorted(
        source_clip_key
        for source_clip_key, count in Counter(item.source_clip_key for item in loaded_items).items()
        if count > 1
    )
    if not duplicated_source_clip_keys:
        return

    raise CatalogIngestError(
        code="manifest_invalid",
        stage="manifest_loader",
        message="输入中存在重复的 source_clip_key",
        context={"duplicated_source_clip_keys": duplicated_source_clip_keys},
    )


def _scan_json_files(directory: Path, label: str) -> tuple[Path, ...]:
    """扫描目录根下的 JSON 文件。

    这里严格遵循 README 里的扫描边界：
    - 只扫描根目录
    - 不递归
    - 忽略隐藏文件
    - 忽略非 .json 文件
    """

    if not directory.exists():
        raise CatalogIngestError(
            code="manifest_invalid",
            stage="manifest_loader",
            message=f"{label} 不存在: {directory}",
            context={"directory": str(directory)},
        )

    if not directory.is_dir():
        raise CatalogIngestError(
            code="manifest_invalid",
            stage="manifest_loader",
            message=f"{label} 不是目录: {directory}",
            context={"directory": str(directory)},
        )

    files = sorted(
        path
        for path in directory.iterdir()
        if path.is_file() and not path.name.startswith(".") and path.suffix.lower() == ".json"
    )
    return tuple(files)


def _load_from_parent_file(
    parent_file: Path,
    transcript_name_set: set[str],
    transcript_index: dict[str, Path],
    question_name_set: set[str],
    question_index: dict[str, Path],
    source_name: str | None,
    clip_key: str | None,
    remaining_limit: int | None,
) -> tuple[LoadedClipInput, ...]:
    """读取单个父文件，并展开成多个 clip 输入对象。"""

    parent_payload = _read_json_file(parent_file, code="manifest_invalid")
    clips_payload = parent_payload.get("clips")
    if not isinstance(clips_payload, list):
        raise CatalogIngestError(
            code="manifest_invalid",
            stage="manifest_loader",
            message="父文件顶层必须包含 clips 数组",
            context={"parent_file_path": str(parent_file)},
        )

    parent_video_name = parent_file.stem
    parent_video_slug = _slugify(parent_video_name)
    loaded_items: list[LoadedClipInput] = []

    for raw_clip in clips_payload:
        if remaining_limit is not None and len(loaded_items) >= remaining_limit:
            break

        if not isinstance(raw_clip, dict):
            raise CatalogIngestError(
                code="manifest_invalid",
                stage="manifest_loader",
                message="clips 数组中的每一项都必须是对象",
                context={"parent_file_path": str(parent_file)},
            )

        try:
            descriptor = ParentClipDescriptor(
                clip_id=int(raw_clip["clip_id"]),
                start_index=_optional_int(raw_clip.get("start_index")),
                end_index=_optional_int(raw_clip.get("end_index")),
                start_time=_optional_int(raw_clip.get("start_time")),
                end_time=_optional_int(raw_clip.get("end_time")),
                buffered_start_time=int(raw_clip["buffered_start_time"]),
                buffered_end_time=int(raw_clip["buffered_end_time"]),
                reasoning=_optional_str(raw_clip.get("reasoning")),
            )
        except (KeyError, TypeError, ValueError) as exc:
            raise CatalogIngestError(
                code="manifest_invalid",
                stage="manifest_loader",
                message="父文件中的 clip 结构不合法",
                context={"parent_file_path": str(parent_file), "raw_clip": raw_clip},
            ) from exc

        expected_transcript_filename = f"{parent_video_name}-clip{descriptor.clip_id}.json"
        expected_question_filename = expected_transcript_filename
        title = Path(expected_transcript_filename).stem
        source_clip_key = f"{parent_video_slug}#clip{descriptor.clip_id}"

        # clip_key 过滤尽量前置，避免为了无关 clip 去读 transcript JSON。
        if clip_key is not None and source_clip_key != clip_key:
            continue

        # 先用预扫描得到的文件名集合判断是否存在；只有存在时才拿 Path。
        if expected_transcript_filename not in transcript_name_set:
            loaded_items.append(
                _build_missing_clip_input(
                    source_clip_key=source_clip_key,
                    parent_video_name=parent_video_name,
                    parent_video_slug=parent_video_slug,
                    descriptor=descriptor,
                    title=title,
                    source_name=source_name,
                    parent_file=parent_file,
                    expected_transcript_filename=expected_transcript_filename,
                    expected_question_filename=expected_question_filename,
                    raw_parent_payload=parent_payload,
                    skip_reason_code="transcript_missing",
                    skip_reason_message="未找到对应的 transcript 文件",
                )
            )
            continue

        if expected_question_filename not in question_name_set:
            loaded_items.append(
                _build_missing_clip_input(
                    source_clip_key=source_clip_key,
                    parent_video_name=parent_video_name,
                    parent_video_slug=parent_video_slug,
                    descriptor=descriptor,
                    title=title,
                    source_name=source_name,
                    parent_file=parent_file,
                    expected_transcript_filename=expected_transcript_filename,
                    expected_question_filename=expected_question_filename,
                    raw_parent_payload=parent_payload,
                    skip_reason_code="question_missing",
                    skip_reason_message="未找到对应的 question 文件",
                    transcript_file_path=transcript_index[expected_transcript_filename],
                )
            )
            continue

        transcript_file_path = transcript_index[expected_transcript_filename]
        question_file_path = question_index[expected_question_filename]
        transcript_payload, transcript_bytes = _read_json_file_with_bytes(
            transcript_file_path,
            code="transcript_invalid",
        )
        transcript_sentences = _parse_transcript_sentences(
            transcript_payload=transcript_payload,
            transcript_file_path=transcript_file_path,
        )
        question_payload = _read_json_file(question_file_path, code="question_invalid")
        questions = _parse_questions(
            question_payload=question_payload,
            question_file_path=question_file_path,
        )
        selected_refs = _parse_selected_coarse_unit_refs(
            question_payload=question_payload,
            question_file_path=question_file_path,
        )

        loaded_items.append(
            LoadedClipInput(
                source_clip_key=source_clip_key,
                parent_video_name=parent_video_name,
                parent_video_slug=parent_video_slug,
                clip_seq=descriptor.clip_id,
                source_start_ms=descriptor.buffered_start_time,
                source_end_ms=descriptor.buffered_end_time,
                title=title,
                description=None,
                clip_reason=descriptor.reasoning,
                language="en",
                duration_ms=descriptor.buffered_end_time - descriptor.buffered_start_time,
                hls_master_playlist_path=f"placeholder://video/{title}",
                thumbnail_url=None,
                publish_at=None,
                transcript_object_path=f"placeholder://transcript/{transcript_file_path.stem}",
                transcript_checksum=_sha256_of_bytes(transcript_bytes),
                transcript_format_version=1,
                source_name=source_name,
                parent_file_path=parent_file,
                expected_transcript_filename=expected_transcript_filename,
                expected_question_filename=expected_question_filename,
                transcript_file_path=transcript_file_path,
                question_file_path=question_file_path,
                parent_clip=descriptor,
                transcript_sentences=transcript_sentences,
                questions=questions,
                selected_coarse_unit_refs=selected_refs,
                raw_parent_payload=parent_payload,
                raw_transcript_payload=transcript_payload,
                raw_question_payload=question_payload,
            )
        )

    return tuple(loaded_items)


def _build_missing_clip_input(
    *,
    source_clip_key: str,
    parent_video_name: str,
    parent_video_slug: str,
    descriptor: ParentClipDescriptor,
    title: str,
    source_name: str | None,
    parent_file: Path,
    expected_transcript_filename: str,
    expected_question_filename: str,
    raw_parent_payload: dict[str, Any],
    skip_reason_code: str,
    skip_reason_message: str,
    transcript_file_path: Path | None = None,
) -> LoadedClipInput:
    """构造缺失输入文件时的 skipped clip 对象。"""

    return LoadedClipInput(
        source_clip_key=source_clip_key,
        parent_video_name=parent_video_name,
        parent_video_slug=parent_video_slug,
        clip_seq=descriptor.clip_id,
        source_start_ms=descriptor.buffered_start_time,
        source_end_ms=descriptor.buffered_end_time,
        title=title,
        description=None,
        clip_reason=descriptor.reasoning,
        language="en",
        duration_ms=descriptor.buffered_end_time - descriptor.buffered_start_time,
        hls_master_playlist_path=f"placeholder://video/{title}",
        thumbnail_url=None,
        publish_at=None,
        transcript_object_path=None,
        transcript_checksum=None,
        transcript_format_version=1,
        source_name=source_name,
        parent_file_path=parent_file,
        expected_transcript_filename=expected_transcript_filename,
        expected_question_filename=expected_question_filename,
        transcript_file_path=transcript_file_path,
        question_file_path=None,
        parent_clip=descriptor,
        transcript_sentences=tuple(),
        questions=tuple(),
        selected_coarse_unit_refs=None,
        raw_parent_payload=raw_parent_payload,
        raw_transcript_payload=None,
        raw_question_payload=None,
        skip_reason_code=skip_reason_code,
        skip_reason_message=skip_reason_message,
    )


def _read_json_file(path: Path, code: str) -> dict[str, Any]:
    """读取并解析 JSON 文件。"""

    try:
        payload = json.loads(path.read_text(encoding="utf-8"))
    except FileNotFoundError as exc:
        raise CatalogIngestError(
            code=code,
            stage="manifest_loader",
            message=f"文件不存在: {path}",
            context={"file_path": str(path)},
        ) from exc
    except json.JSONDecodeError as exc:
        raise CatalogIngestError(
            code=code,
            stage="manifest_loader",
            message=f"JSON 解析失败: {path}",
            context={"file_path": str(path), "line": exc.lineno, "column": exc.colno},
        ) from exc

    if not isinstance(payload, dict):
        raise CatalogIngestError(
            code=code,
            stage="manifest_loader",
            message="JSON 顶层必须是对象",
            context={"file_path": str(path)},
        )
    return payload


def _read_json_file_with_bytes(path: Path, code: str) -> tuple[dict[str, Any], bytes]:
    """读取原始字节并解析 JSON。

    transcript 文件后续还要计算 sha256，因此这里一次读取原始字节，
    同时复用给 JSON 解析和 checksum 计算，避免同一文件重复读两遍。
    """

    try:
        raw_bytes = path.read_bytes()
    except FileNotFoundError as exc:
        raise CatalogIngestError(
            code=code,
            stage="manifest_loader",
            message=f"文件不存在: {path}",
            context={"file_path": str(path)},
        ) from exc

    try:
        payload = json.loads(raw_bytes.decode("utf-8"))
    except UnicodeDecodeError as exc:
        raise CatalogIngestError(
            code=code,
            stage="manifest_loader",
            message=f"文件编码不是合法 UTF-8: {path}",
            context={"file_path": str(path)},
        ) from exc
    except json.JSONDecodeError as exc:
        raise CatalogIngestError(
            code=code,
            stage="manifest_loader",
            message=f"JSON 解析失败: {path}",
            context={"file_path": str(path), "line": exc.lineno, "column": exc.colno},
        ) from exc

    if not isinstance(payload, dict):
        raise CatalogIngestError(
            code=code,
            stage="manifest_loader",
            message="JSON 顶层必须是对象",
            context={"file_path": str(path)},
        )
    return payload, raw_bytes


def _parse_transcript_sentences(
    transcript_payload: dict[str, Any],
    transcript_file_path: Path,
) -> tuple[TranscriptSentence, ...]:
    """将 transcript JSON 解析成脚本内部 sentence / token 结构。

    这里会做最基础的字段读取和类型转换。
    更细的业务校验，例如索引唯一性、时间落区间、coarse_id 是否存在，交给 validator。
    """

    sentences_payload = transcript_payload.get("sentences")
    if not isinstance(sentences_payload, list):
        raise CatalogIngestError(
            code="transcript_invalid",
            stage="manifest_loader",
            message="transcript 顶层必须包含 sentences 数组",
            context={"transcript_file_path": str(transcript_file_path)},
        )

    parsed_sentences: list[TranscriptSentence] = []
    for raw_sentence in sentences_payload:
        if not isinstance(raw_sentence, dict):
            raise CatalogIngestError(
                code="transcript_invalid",
                stage="manifest_loader",
                message="sentences 数组中的每一项都必须是对象",
                context={"transcript_file_path": str(transcript_file_path)},
            )

        tokens_payload = raw_sentence.get("tokens")
        if not isinstance(tokens_payload, list):
            raise CatalogIngestError(
                code="transcript_invalid",
                stage="manifest_loader",
                message="sentence.tokens 必须是数组",
                context={
                    "transcript_file_path": str(transcript_file_path),
                    "sentence_index": raw_sentence.get("index"),
                },
            )

        parsed_tokens: list[TranscriptToken] = []
        for raw_token in tokens_payload:
            if not isinstance(raw_token, dict):
                raise CatalogIngestError(
                    code="transcript_invalid",
                    stage="manifest_loader",
                    message="token 必须是对象",
                    context={"transcript_file_path": str(transcript_file_path)},
                )

            semantic_payload = raw_token.get("semantic_element")
            semantic_element = None
            if semantic_payload is not None:
                if not isinstance(semantic_payload, dict):
                    raise CatalogIngestError(
                        code="transcript_invalid",
                        stage="manifest_loader",
                        message="semantic_element 必须是对象",
                        context={
                            "transcript_file_path": str(transcript_file_path),
                            "sentence_index": raw_sentence.get("index"),
                            "token_index": raw_token.get("index"),
                        },
                    )
                semantic_element = TranscriptSemanticElement(
                    coarse_id=_optional_int(semantic_payload.get("coarse_id")),
                )

            try:
                parsed_tokens.append(
                    TranscriptToken(
                        index=int(raw_token["index"]),
                        text=str(raw_token["text"]),
                        explanation=_optional_str(raw_token.get("explanation")),
                        start_ms=int(raw_token["start"]),
                        end_ms=int(raw_token["end"]),
                        semantic_element=semantic_element,
                    )
                )
            except (KeyError, TypeError, ValueError) as exc:
                raise CatalogIngestError(
                    code="transcript_invalid",
                    stage="manifest_loader",
                    message="token 结构不合法",
                    context={
                        "transcript_file_path": str(transcript_file_path),
                        "sentence_index": raw_sentence.get("index"),
                        "raw_token": raw_token,
                    },
                ) from exc

        try:
            parsed_sentences.append(
                TranscriptSentence(
                    index=int(raw_sentence["index"]),
                    text=str(raw_sentence["text"]),
                    translation=_optional_str(raw_sentence.get("translation")),
                    start_ms=int(raw_sentence["start"]),
                    end_ms=int(raw_sentence["end"]),
                    tokens=tuple(parsed_tokens),
                )
            )
        except (KeyError, TypeError, ValueError) as exc:
            raise CatalogIngestError(
                code="transcript_invalid",
                stage="manifest_loader",
                message="sentence 结构不合法",
                context={"transcript_file_path": str(transcript_file_path), "raw_sentence": raw_sentence},
            ) from exc

    return tuple(parsed_sentences)


def _parse_questions(
    question_payload: dict[str, Any],
    question_file_path: Path,
) -> tuple[QuestionInput, ...]:
    """将 question JSON 中的 questions 数组解析为内部结构。"""

    questions_payload = question_payload.get("questions")
    if not isinstance(questions_payload, list):
        raise CatalogIngestError(
            code="question_invalid",
            stage="manifest_loader",
            message="question 顶层必须包含 questions 数组",
            context={"question_file_path": str(question_file_path)},
        )

    parsed_questions: list[QuestionInput] = []
    for raw_question in questions_payload:
        if not isinstance(raw_question, dict):
            raise CatalogIngestError(
                code="question_invalid",
                stage="manifest_loader",
                message="questions 数组中的每一项都必须是对象",
                context={"question_file_path": str(question_file_path)},
            )
        content_payload = raw_question.get("content_payload")
        if not isinstance(content_payload, dict):
            raise CatalogIngestError(
                code="question_invalid",
                stage="manifest_loader",
                message="question.content_payload 必须是对象",
                context={"question_file_path": str(question_file_path), "raw_question": raw_question},
            )

        try:
            parsed_questions.append(
                QuestionInput(
                    scope_type=str(raw_question.get("scope_type", "video_unit")),
                    question_type=str(raw_question["question_type"]),
                    coarse_unit_id=int(raw_question["coarse_unit_id"]),
                    target_text=str(raw_question["target_text"]),
                    context_sentence_index=_optional_int(raw_question.get("context_sentence_index")),
                    context_span_index=_optional_int(raw_question.get("context_span_index")),
                    context_start_ms=_optional_int(raw_question.get("context_start_ms")),
                    context_end_ms=_optional_int(raw_question.get("context_end_ms")),
                    content_payload=content_payload,
                    status=str(raw_question.get("status", "active")),
                )
            )
        except (KeyError, TypeError, ValueError) as exc:
            raise CatalogIngestError(
                code="question_invalid",
                stage="manifest_loader",
                message="question 结构不合法",
                context={"question_file_path": str(question_file_path), "raw_question": raw_question},
            ) from exc

    return tuple(parsed_questions)


def _parse_selected_coarse_unit_refs(
    question_payload: dict[str, Any],
    question_file_path: Path,
) -> SelectedCoarseUnitRefs:
    """解析 selected_coarse_unit_refs。"""

    refs_payload = question_payload.get("selected_coarse_unit_refs")
    if not isinstance(refs_payload, dict):
        raise CatalogIngestError(
            code="question_invalid",
            stage="manifest_loader",
            message="question 顶层必须包含 selected_coarse_unit_refs 对象",
            context={"question_file_path": str(question_file_path)},
        )

    raw_refs = refs_payload.get("refs")
    if not isinstance(raw_refs, list):
        raise CatalogIngestError(
            code="question_invalid",
            stage="manifest_loader",
            message="selected_coarse_unit_refs.refs 必须是数组",
            context={"question_file_path": str(question_file_path)},
        )

    allowed_question_types_payload = refs_payload.get("allowed_question_types", [])
    if not isinstance(allowed_question_types_payload, list):
        raise CatalogIngestError(
            code="question_invalid",
            stage="manifest_loader",
            message="selected_coarse_unit_refs.allowed_question_types 必须是数组",
            context={"question_file_path": str(question_file_path)},
        )

    parsed_refs: list[SelectedCoarseUnitRef] = []
    for raw_ref in raw_refs:
        if not isinstance(raw_ref, dict):
            raise CatalogIngestError(
                code="question_invalid",
                stage="manifest_loader",
                message="selected_coarse_unit_refs.refs 每一项都必须是对象",
                context={"question_file_path": str(question_file_path)},
            )
        try:
            parsed_refs.append(
                SelectedCoarseUnitRef(
                    coarse_unit_id=int(raw_ref["coarse_unit_id"]),
                    sentence_index=int(raw_ref["sentence_index"]),
                    token_index=int(raw_ref["token_index"]),
                )
            )
        except (KeyError, TypeError, ValueError) as exc:
            raise CatalogIngestError(
                code="question_invalid",
                stage="manifest_loader",
                message="selected_coarse_unit_refs ref 结构不合法",
                context={"question_file_path": str(question_file_path), "raw_ref": raw_ref},
            ) from exc

    try:
        version = int(refs_payload.get("version", 1))
        selection_top_k = _optional_int(refs_payload.get("selection_top_k"))
    except (TypeError, ValueError) as exc:
        raise CatalogIngestError(
            code="question_invalid",
            stage="manifest_loader",
            message="selected_coarse_unit_refs version/top_k 结构不合法",
            context={"question_file_path": str(question_file_path)},
        ) from exc

    return SelectedCoarseUnitRefs(
        version=version,
        selection_model=_optional_str(refs_payload.get("selection_model")),
        selection_top_k=selection_top_k,
        allowed_question_types=tuple(str(value) for value in allowed_question_types_payload),
        refs=tuple(parsed_refs),
    )


def _sha256_of_bytes(raw_bytes: bytes) -> str:
    """对 transcript 原始字节计算 sha256。

    按 README 的规则，这里必须基于原始字节，而不是基于重排后的 JSON 文本。
    """

    return hashlib.sha256(raw_bytes).hexdigest()


def _slugify(value: str) -> str:
    """将父视频名转成稳定的 slug。"""

    slug = re.sub(r"[^a-z0-9]+", "-", value.lower()).strip("-")
    slug = re.sub(r"-+", "-", slug)
    if not slug:
        raise CatalogIngestError(
            code="manifest_invalid",
            stage="manifest_loader",
            message="parent_video_slug 规范化后不能为空",
            context={"value": value},
        )
    return slug


def _optional_int(value: Any) -> int | None:
    if value is None:
        return None
    return int(value)


def _optional_str(value: Any) -> str | None:
    if value is None:
        return None
    return str(value)
