from __future__ import annotations

import hashlib
import json
import re
from pathlib import Path
from typing import Any

from .models import (
    CatalogIngestError,
    LoadedClipInput,
    ParentClipDescriptor,
    TranscriptSemanticElement,
    TranscriptSentence,
    TranscriptToken,
)


def load_clip_inputs(
    parents_dir: Path,
    transcripts_dir: Path,
    source_name: str | None = None,
    clip_key: str | None = None,
    limit: int | None = None,
) -> tuple[LoadedClipInput, ...]:
    """扫描两个输入目录，并组装成单 clip 输入对象列表。

    这是整个脚本的第一步。
    它只负责把输入读出来并完成文件名级别的匹配，不负责完整业务校验。

    参数说明：
    - parents_dir：父视频切片描述文件目录
    - transcripts_dir：clip transcript 文件目录
    - source_name：写审计记录时使用的来源名
    - clip_key：若指定，则只保留命中的单个 source_clip_key
    - limit：最多返回多少条 clip 输入，用于小批量调试
    """

    parent_files = _scan_json_files(parents_dir, label="parents_dir")
    transcript_files = _scan_json_files(transcripts_dir, label="transcripts_dir")

    # transcript 目录以“文件名 -> Path”建索引，后续按规则 O(1) 匹配。
    # 当前规则是由父文件 basename 和 clip_id 直接推导 transcript 文件名。
    transcript_index = {path.name: path for path in transcript_files}

    loaded_items: list[LoadedClipInput] = []
    for parent_file in parent_files:
        loaded_items.extend(
            _load_from_parent_file(
                parent_file=parent_file,
                transcript_index=transcript_index,
                source_name=source_name,
            )
        )

    if clip_key:
        loaded_items = [item for item in loaded_items if item.source_clip_key == clip_key]

    if limit is not None:
        loaded_items = loaded_items[:limit]

    return tuple(loaded_items)


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
    transcript_index: dict[str, Path],
    source_name: str | None,
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
        transcript_file_path = transcript_index.get(expected_transcript_filename)
        title = Path(expected_transcript_filename).stem
        source_clip_key = f"{parent_video_slug}#clip{descriptor.clip_id}"

        if transcript_file_path is None:
            # 缺 transcript 时，仍然返回一个 LoadedClipInput。
            # main 后续会根据 skip_reason_code 直接写 skipped 审计，而不是报错中断整批执行。
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
                    transcript_object_path=None,
                    transcript_checksum=None,
                    transcript_format_version=1,
                    source_name=source_name,
                    parent_file_path=parent_file,
                    expected_transcript_filename=expected_transcript_filename,
                    transcript_file_path=None,
                    parent_clip=descriptor,
                    transcript_sentences=tuple(),
                    raw_parent_payload=parent_payload,
                    raw_transcript_payload=None,
                    skip_reason_code="transcript_missing",
                    skip_reason_message="未找到对应的 transcript 文件",
                )
            )
            continue

        transcript_payload = _read_json_file(transcript_file_path, code="transcript_invalid")
        transcript_sentences = _parse_transcript_sentences(
            transcript_payload=transcript_payload,
            transcript_file_path=transcript_file_path,
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
                transcript_checksum=_sha256_of_file(transcript_file_path),
                transcript_format_version=1,
                source_name=source_name,
                parent_file_path=parent_file,
                expected_transcript_filename=expected_transcript_filename,
                transcript_file_path=transcript_file_path,
                parent_clip=descriptor,
                transcript_sentences=transcript_sentences,
                raw_parent_payload=parent_payload,
                raw_transcript_payload=transcript_payload,
            )
        )

    return tuple(loaded_items)


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

            semantic_payload = raw_token.get("semanticElement")
            semantic_element = None
            if semantic_payload is not None:
                if not isinstance(semantic_payload, dict):
                    raise CatalogIngestError(
                        code="transcript_invalid",
                        stage="manifest_loader",
                        message="semanticElement 必须是对象",
                        context={
                            "transcript_file_path": str(transcript_file_path),
                            "sentence_index": raw_sentence.get("index"),
                            "token_index": raw_token.get("index"),
                        },
                    )
                semantic_element = TranscriptSemanticElement(
                    base_form=_optional_str(semantic_payload.get("baseForm")),
                    dictionary_text=_optional_str(semantic_payload.get("dictionary")),
                    coarse_id=_optional_int(semantic_payload.get("coarse_id")),
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
                    explanation=_optional_str(raw_sentence.get("explanation")),
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


def _sha256_of_file(path: Path) -> str:
    """对 transcript 原始字节计算 sha256。

    按 README 的规则，这里必须基于原始字节，而不是基于重排后的 JSON 文本。
    """

    digest = hashlib.sha256()
    with path.open("rb") as file_obj:
        for chunk in iter(lambda: file_obj.read(8192), b""):
            digest.update(chunk)
    return digest.hexdigest()


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
