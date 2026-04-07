from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime
from decimal import Decimal
from pathlib import Path
from typing import Any


@dataclass(slots=True)
class CatalogIngestError(Exception):
    """表示入库脚本中的结构化错误。

    这里不用裸字符串异常，而是统一带上错误码、阶段和上下文。
    这样 main 在汇总失败结果、写审计记录和打印日志时都可以复用同一份信息。
    """

    code: str
    stage: str
    message: str
    context: dict[str, Any] = field(default_factory=dict)

    def __str__(self) -> str:
        return f"[{self.stage}:{self.code}] {self.message}"


@dataclass(slots=True, frozen=True)
class ParentClipDescriptor:
    """表示父视频描述文件中的单个 clip 条目。

    这个对象只承接父文件里的切片事实，不混入 transcript 结构。
    它是 manifest_loader 从父文件 JSON 里提炼出的最小稳定输入。
    """

    clip_id: int
    start_index: int | None
    end_index: int | None
    start_time: int | None
    end_time: int | None
    buffered_start_time: int
    buffered_end_time: int
    reasoning: str | None


@dataclass(slots=True, frozen=True)
class TranscriptSemanticElement:
    """表示 transcript token 下的 semanticElement 结构。"""

    base_form: str | None
    dictionary_text: str | None
    coarse_id: int | None
    reason: str | None


@dataclass(slots=True, frozen=True)
class TranscriptToken:
    """表示 transcript 中的单个语义 span。

    虽然上游 JSON 字段名叫 token，但按当前 catalog 设计，
    它在数据库里最终会落成 semantic span，所以这里保留完整信息。
    """

    index: int
    text: str
    explanation: str | None
    start_ms: int
    end_ms: int
    semantic_element: TranscriptSemanticElement | None


@dataclass(slots=True, frozen=True)
class TranscriptSentence:
    """表示 transcript 中的单个 sentence。"""

    index: int
    text: str
    explanation: str | None
    start_ms: int
    end_ms: int
    tokens: tuple[TranscriptToken, ...]


@dataclass(slots=True, frozen=True)
class LoadedClipInput:
    """表示 loader 阶段输出的“单 clip 原始输入对象”。

    这是整个脚本后续流程的主输入。
    它同时持有：
    - 由父文件推导出的 clip 元数据
    - transcript 文件路径与原文
    - 为写审计记录准备的上下文

    注意：如果 transcript 文件缺失，这个对象仍然会被创建，
    但会带上 skip_reason_code，供 main 直接走 skipped 分支。
    """

    source_clip_key: str
    parent_video_name: str
    parent_video_slug: str
    clip_seq: int
    source_start_ms: int
    source_end_ms: int
    title: str
    description: str | None
    clip_reason: str | None
    language: str
    duration_ms: int
    hls_master_playlist_path: str
    thumbnail_url: str | None
    publish_at: datetime | None
    transcript_object_path: str | None
    transcript_checksum: str | None
    transcript_format_version: int
    source_name: str | None
    parent_file_path: Path
    expected_transcript_filename: str
    transcript_file_path: Path | None
    parent_clip: ParentClipDescriptor
    transcript_sentences: tuple[TranscriptSentence, ...]
    raw_parent_payload: dict[str, Any]
    raw_transcript_payload: dict[str, Any] | None
    skip_reason_code: str | None = None
    skip_reason_message: str | None = None

    @property
    def context(self) -> dict[str, Any]:
        """生成统一审计上下文。

        审计上下文尽量只放排障需要的信息，不放整坨 transcript。
        """

        return {
            "parent_file_path": str(self.parent_file_path),
            "expected_transcript_filename": self.expected_transcript_filename,
            "transcript_file_path": str(self.transcript_file_path) if self.transcript_file_path else None,
            "clip_id": self.parent_clip.clip_id,
            "parent_video_name": self.parent_video_name,
            "source_clip_key": self.source_clip_key,
        }


@dataclass(slots=True, frozen=True)
class VideoRow:
    """表示将写入 catalog.videos 的一行数据。"""

    source_clip_key: str
    parent_video_name: str
    parent_video_slug: str
    clip_seq: int
    source_start_ms: int
    source_end_ms: int
    title: str
    description: str | None
    clip_reason: str | None
    language: str
    duration_ms: int
    hls_master_playlist_path: str
    thumbnail_url: str | None
    status: str
    visibility_status: str
    publish_at: datetime | None


@dataclass(slots=True, frozen=True)
class VideoTranscriptRow:
    """表示将写入 catalog.video_transcripts 的一行数据。"""

    transcript_object_path: str
    transcript_checksum: str
    transcript_format_version: int
    full_text: str
    sentence_count: int
    semantic_span_count: int
    mapped_span_count: int
    unmapped_span_count: int
    mapped_span_ratio: Decimal


@dataclass(slots=True, frozen=True)
class VideoTranscriptSentenceRow:
    """表示将写入 catalog.video_transcript_sentences 的一行数据。"""

    sentence_index: int
    text: str
    start_ms: int
    end_ms: int
    explanation: str | None


@dataclass(slots=True, frozen=True)
class VideoSemanticSpanRow:
    """表示将写入 catalog.video_semantic_spans 的一行数据。"""

    sentence_index: int
    span_index: int
    text: str
    start_ms: int
    end_ms: int
    explanation: str | None
    coarse_unit_id: int | None
    base_form: str | None
    dictionary_text: str | None


@dataclass(slots=True, frozen=True)
class VideoUnitIndexRow:
    """表示将写入 catalog.video_unit_index 的一行聚合结果。"""

    coarse_unit_id: int
    mention_count: int
    sentence_count: int
    first_start_ms: int
    last_end_ms: int
    coverage_ms: int
    coverage_ratio: Decimal
    sentence_indexes: tuple[int, ...]
    evidence_sentence_indexes: tuple[int, ...]
    evidence_span_indexes: tuple[int, ...]
    sample_surface_forms: tuple[str, ...]


@dataclass(slots=True, frozen=True)
class NormalizedCoreRows:
    """表示 normalizer 阶段产出的基础行集合。

    这里故意不包含 transcript 摘要和 unit index。
    原因是这两类数据属于派生聚合，应该由 index_builder 负责。
    """

    video: VideoRow
    sentences: tuple[VideoTranscriptSentenceRow, ...]
    spans: tuple[VideoSemanticSpanRow, ...]


@dataclass(slots=True, frozen=True)
class NormalizedClipData:
    """表示 normalizer 和 index_builder 产出的完整写库数据。"""

    video: VideoRow
    transcript: VideoTranscriptRow
    sentences: tuple[VideoTranscriptSentenceRow, ...]
    spans: tuple[VideoSemanticSpanRow, ...]
    unit_indexes: tuple[VideoUnitIndexRow, ...]


@dataclass(slots=True, frozen=True)
class IngestionRecordPayload:
    """表示写入审计表时需要的字段集合。

    这里不包含 ingestion_record_id 和时间戳，因为这两个值应由 repository 在真正写库时生成，
    从而保证数据库写入阶段对时间和主键拥有最终控制权。
    """

    source_clip_key: str
    video_id: str | None
    source_name: str | None
    status: str
    warning_codes: tuple[str, ...]
    error_code: str | None
    error_message: str | None
    context: dict[str, Any]


@dataclass(slots=True, frozen=True)
class ExistingClipState:
    """表示数据库里已存在的 clip 快照。

    这个对象专门给 main 做“是否可以 skipped”判断用。
    它只保留幂等判断所需的字段，不承担完整读模型职责。
    """

    video_id: str
    parent_video_name: str
    parent_video_slug: str
    clip_seq: int
    source_start_ms: int
    source_end_ms: int
    title: str
    description: str | None
    clip_reason: str | None
    language: str
    duration_ms: int
    hls_master_playlist_path: str
    transcript_checksum: str | None


@dataclass(slots=True, frozen=True)
class ClipProcessResult:
    """表示 main 汇总时使用的单 clip 最终结果。"""

    source_clip_key: str
    status: str
    video_id: str | None
    warning_codes: tuple[str, ...]
    error: CatalogIngestError | None
