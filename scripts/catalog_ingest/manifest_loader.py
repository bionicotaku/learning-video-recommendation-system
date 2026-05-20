from __future__ import annotations

import hashlib
import json
import re
from collections import Counter
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from .models import (
    CatalogIngestError,
    ClipMetadata,
    LoadedClipInput,
    QuestionInput,
    SelectedCoarseUnitRef,
    SelectedCoarseUnitRefs,
    TranscriptSemanticElement,
    TranscriptSentence,
    TranscriptToken,
)

_PUBLIC_ASSET_BASE_URL = "https://storage.googleapis.com/videos2077/test-video"


def load_clip_inputs(
    transcripts_dir: Path,
    questions_dir: Path,
    source_name: str | None = None,
    clip_key: str | None = None,
    limit: int | None = None,
) -> tuple[LoadedClipInput, ...]:
    """扫描 mapped transcript 和 question 输入目录，并组装单 clip 输入对象。

    这是整个脚本的第一步。
    它只负责把输入读出来并完成文件名级别的匹配，不负责完整业务校验。

    参数说明：
    - transcripts_dir：mapped clip transcript 文件目录
    - questions_dir：clip question 文件目录
    - source_name：写审计记录时使用的来源名
    - clip_key：若指定，则只保留命中的单个 source_clip_key
    - limit：最多返回多少条 clip 输入，用于小批量调试
    """

    transcript_files = _scan_json_files(transcripts_dir, label="transcripts_dir")
    question_files = _scan_json_files(questions_dir, label="questions_dir")

    question_name_set = {path.name for path in question_files}
    question_index = {path.name: path for path in question_files}
    execution_publish_at = datetime.now(timezone.utc)

    loaded_items: list[LoadedClipInput] = []
    for transcript_file in transcript_files:
        if limit is not None and len(loaded_items) >= limit:
            break

        clip_input = _load_from_transcript_file(
            transcript_file=transcript_file,
            question_name_set=question_name_set,
            question_index=question_index,
            source_name=source_name,
            clip_key=clip_key,
            publish_at=execution_publish_at,
        )
        if clip_input is not None:
            loaded_items.append(clip_input)

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


def _load_from_transcript_file(
    transcript_file: Path,
    question_name_set: set[str],
    question_index: dict[str, Path],
    source_name: str | None,
    clip_key: str | None,
    publish_at: datetime,
) -> LoadedClipInput | None:
    """读取单个 mapped transcript 文件，并组装成单 clip 输入对象。"""

    transcript_payload, transcript_bytes = _read_json_file_with_bytes(
        transcript_file,
        code="transcript_invalid",
    )
    clip_metadata = _parse_clip_metadata(transcript_payload, transcript_file)
    parent_video_name = _parent_video_name_from_transcript_file(transcript_file, clip_metadata.clip_id)
    parent_video_slug = _slugify(parent_video_name)
    source_clip_key = f"{parent_video_slug}#clip{clip_metadata.clip_id}"

    # clip_key 过滤尽量前置，避免为了无关 clip 去读 question JSON。
    if clip_key is not None and source_clip_key != clip_key:
        return None

    title = _required_str(transcript_payload.get("title"), "title", transcript_file)
    description = _optional_str(transcript_payload.get("description"))
    transcript_sentences = _parse_transcript_sentences(
        transcript_payload=transcript_payload,
        transcript_file_path=transcript_file,
    )
    expected_transcript_filename = transcript_file.name
    expected_question_filename = transcript_file.name

    if expected_question_filename not in question_name_set:
        return _build_missing_question_input(
            source_clip_key=source_clip_key,
            parent_video_name=parent_video_name,
            parent_video_slug=parent_video_slug,
            clip_metadata=clip_metadata,
            title=title,
            description=description,
            source_name=source_name,
            transcript_file=transcript_file,
            expected_transcript_filename=expected_transcript_filename,
            expected_question_filename=expected_question_filename,
            transcript_sentences=transcript_sentences,
            transcript_payload=transcript_payload,
            transcript_checksum=_sha256_of_bytes(transcript_bytes),
            publish_at=publish_at,
        )

    question_file_path = question_index[expected_question_filename]
    question_payload = _read_json_file(question_file_path, code="question_invalid")
    questions = _parse_questions(
        question_payload=question_payload,
        question_file_path=question_file_path,
    )
    selected_refs = _parse_selected_coarse_unit_refs(
        question_payload=question_payload,
        question_file_path=question_file_path,
    )

    return LoadedClipInput(
        source_clip_key=source_clip_key,
        parent_video_name=parent_video_name,
        parent_video_slug=parent_video_slug,
        clip_seq=clip_metadata.clip_id,
        source_start_ms=clip_metadata.buffered_start_time,
        source_end_ms=clip_metadata.buffered_end_time,
        title=title,
        description=description,
        clip_reason=clip_metadata.reasoning,
        language="en",
        duration_ms=clip_metadata.duration_time,
        video_object_path=_video_object_path(transcript_file),
        thumbnail_url=_thumbnail_url(transcript_file),
        publish_at=publish_at,
        transcript_object_path=_transcript_object_path(transcript_file),
        transcript_checksum=_sha256_of_bytes(transcript_bytes),
        transcript_format_version=1,
        source_name=source_name,
        source_file_path=transcript_file,
        expected_transcript_filename=expected_transcript_filename,
        expected_question_filename=expected_question_filename,
        transcript_file_path=transcript_file,
        question_file_path=question_file_path,
        clip_metadata=clip_metadata,
        transcript_sentences=transcript_sentences,
        questions=questions,
        selected_coarse_unit_refs=selected_refs,
        raw_transcript_payload=transcript_payload,
        raw_question_payload=question_payload,
    )


def _build_missing_question_input(
    *,
    source_clip_key: str,
    parent_video_name: str,
    parent_video_slug: str,
    clip_metadata: ClipMetadata,
    title: str,
    description: str | None,
    source_name: str | None,
    transcript_file: Path,
    expected_transcript_filename: str,
    expected_question_filename: str,
    transcript_sentences: tuple[TranscriptSentence, ...],
    transcript_payload: dict[str, Any],
    transcript_checksum: str,
    publish_at: datetime,
) -> LoadedClipInput:
    """构造缺失 question 文件时的 skipped clip 对象。"""

    return LoadedClipInput(
        source_clip_key=source_clip_key,
        parent_video_name=parent_video_name,
        parent_video_slug=parent_video_slug,
        clip_seq=clip_metadata.clip_id,
        source_start_ms=clip_metadata.buffered_start_time,
        source_end_ms=clip_metadata.buffered_end_time,
        title=title,
        description=description,
        clip_reason=clip_metadata.reasoning,
        language="en",
        duration_ms=clip_metadata.duration_time,
        video_object_path=_video_object_path(transcript_file),
        thumbnail_url=_thumbnail_url(transcript_file),
        publish_at=publish_at,
        transcript_object_path=_transcript_object_path(transcript_file),
        transcript_checksum=transcript_checksum,
        transcript_format_version=1,
        source_name=source_name,
        source_file_path=transcript_file,
        expected_transcript_filename=expected_transcript_filename,
        expected_question_filename=expected_question_filename,
        transcript_file_path=transcript_file,
        question_file_path=None,
        clip_metadata=clip_metadata,
        transcript_sentences=transcript_sentences,
        questions=tuple(),
        selected_coarse_unit_refs=None,
        raw_transcript_payload=transcript_payload,
        raw_question_payload=None,
        skip_reason_code="question_missing",
        skip_reason_message="未找到对应的 question 文件",
    )


def _video_object_path(transcript_file: Path) -> str:
    return f"{_PUBLIC_ASSET_BASE_URL}/portrait_videos/{transcript_file.stem}.mp4"


def _transcript_object_path(transcript_file: Path) -> str:
    return f"{_PUBLIC_ASSET_BASE_URL}/transcript/{transcript_file.name}"


def _thumbnail_url(transcript_file: Path) -> str:
    return f"{_PUBLIC_ASSET_BASE_URL}/cover/{transcript_file.stem}.webp"


def _parse_clip_metadata(transcript_payload: dict[str, Any], transcript_file_path: Path) -> ClipMetadata:
    """从 mapped transcript JSON 顶层解析单 clip 元数据。"""

    engagement_payload = transcript_payload.get("engagement", {})
    if not isinstance(engagement_payload, dict):
        raise CatalogIngestError(
            code="transcript_invalid",
            stage="manifest_loader",
            message="transcript.engagement 必须是对象",
            context={"transcript_file_path": str(transcript_file_path)},
        )

    try:
        return ClipMetadata(
            clip_id=int(transcript_payload["clip_id"]),
            start_index=_optional_int(transcript_payload.get("start_index")),
            end_index=_optional_int(transcript_payload.get("end_index")),
            start_time=_optional_int(transcript_payload.get("start_time")),
            end_time=_optional_int(transcript_payload.get("end_time")),
            buffered_start_time=int(transcript_payload["buffered_start_time"]),
            buffered_end_time=int(transcript_payload["buffered_end_time"]),
            duration_time=int(transcript_payload["duration_time"]),
            reasoning=_optional_str(transcript_payload.get("reasoning")),
            engagement=dict(engagement_payload),
        )
    except (KeyError, TypeError, ValueError) as exc:
        raise CatalogIngestError(
            code="transcript_invalid",
            stage="manifest_loader",
            message="transcript 顶层 clip metadata 结构不合法",
            context={"transcript_file_path": str(transcript_file_path)},
        ) from exc


def _parent_video_name_from_transcript_file(transcript_file: Path, clip_id: int) -> str:
    """从 `<parent_video_name>-clipN.json` 文件名解析父视频名并校验 clip_id。"""

    match = re.match(r"^(?P<parent>.+)-clip(?P<clip_id>\d+)$", transcript_file.stem)
    if match is None:
        raise CatalogIngestError(
            code="manifest_invalid",
            stage="manifest_loader",
            message="mapped transcript 文件名必须形如 <parent>-clip<clip_id>.json",
            context={"transcript_file_path": str(transcript_file)},
        )

    filename_clip_id = int(match.group("clip_id"))
    if filename_clip_id != clip_id:
        raise CatalogIngestError(
            code="manifest_invalid",
            stage="manifest_loader",
            message="文件名中的 clip_id 与 transcript 顶层 clip_id 不一致",
            context={
                "transcript_file_path": str(transcript_file),
                "filename_clip_id": filename_clip_id,
                "payload_clip_id": clip_id,
            },
        )

    return match.group("parent")


def _required_str(value: Any, field_name: str, transcript_file_path: Path) -> str:
    if not isinstance(value, str) or not value.strip():
        raise CatalogIngestError(
            code="transcript_invalid",
            stage="manifest_loader",
            message=f"transcript.{field_name} 不能为空字符串",
            context={"transcript_file_path": str(transcript_file_path)},
        )
    return value


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
                    base_form=_optional_str(semantic_payload.get("base_form")),
                    translation=_optional_str(semantic_payload.get("translation")),
                    dictionary=_optional_str(semantic_payload.get("dictionary")),
                    reason=_optional_str(semantic_payload.get("reason")),
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
                    status="active",
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

    parsed_refs: list[SelectedCoarseUnitRef] = []
    for raw_ref in raw_refs:
        if not isinstance(raw_ref, dict):
            raise CatalogIngestError(
                code="question_invalid",
                stage="manifest_loader",
                message="selected_coarse_unit_refs.refs 每一项都必须是对象",
                context={"question_file_path": str(question_file_path)},
            )
        scores = raw_ref.get("scores")
        if not isinstance(scores, dict):
            raise CatalogIngestError(
                code="question_invalid",
                stage="manifest_loader",
                message="selected_coarse_unit_refs ref.scores 必须是对象",
                context={"question_file_path": str(question_file_path), "raw_ref": raw_ref},
            )
        question_reject_reason = raw_ref.get("question_reject_reason")
        if question_reject_reason is not None and not isinstance(question_reject_reason, str):
            raise CatalogIngestError(
                code="question_invalid",
                stage="manifest_loader",
                message="selected_coarse_unit_refs ref.question_reject_reason 必须是字符串或 null",
                context={"question_file_path": str(question_file_path), "raw_ref": raw_ref},
            )
        selection_reason = raw_ref.get("selection_reason")
        if not isinstance(selection_reason, str):
            raise CatalogIngestError(
                code="question_invalid",
                stage="manifest_loader",
                message="selected_coarse_unit_refs ref.selection_reason 必须是字符串",
                context={"question_file_path": str(question_file_path), "raw_ref": raw_ref},
            )

        try:
            parsed_refs.append(
                SelectedCoarseUnitRef(
                    coarse_unit_id=int(raw_ref["coarse_unit_id"]),
                    target_text=_required_raw_str(raw_ref["target_text"]),
                    sentence_index=int(raw_ref["sentence_index"]),
                    token_index=int(raw_ref["token_index"]),
                    scores=dict(scores),
                    candidate_score=_optional_float(raw_ref.get("candidate_score")),
                    question_reject_reason=question_reject_reason,
                    selection_reason=selection_reason,
                )
            )
        except (KeyError, TypeError, ValueError) as exc:
            raise CatalogIngestError(
                code="question_invalid",
                stage="manifest_loader",
                message="selected_coarse_unit_refs ref 结构不合法",
                context={"question_file_path": str(question_file_path), "raw_ref": raw_ref},
            ) from exc

    return SelectedCoarseUnitRefs(
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


def _optional_float(value: Any) -> float | None:
    if value is None:
        return None
    if type(value) not in (int, float):
        raise ValueError("expected number")
    return float(value)


def _required_raw_str(value: Any) -> str:
    if not isinstance(value, str) or not value.strip():
        raise ValueError("required non-empty string")
    return value


def _optional_str(value: Any) -> str | None:
    if value is None:
        return None
    return str(value)
