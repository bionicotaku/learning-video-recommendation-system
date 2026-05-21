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
class ClipMetadata:
    """表示 mapped transcript JSON 顶层的单 clip 元数据。"""

    clip_id: int
    start_index: int | None
    end_index: int | None
    start_time: int | None
    end_time: int | None
    buffered_start_time: int
    buffered_end_time: int
    duration_time: int
    reasoning: str | None
    engagement: dict[str, Any]


@dataclass(slots=True, frozen=True)
class TranscriptSemanticElement:
    """表示 transcript token 下的 semantic_element 结构。"""

    coarse_id: int | None
    base_form: str | None = None
    translation: str | None = None
    dictionary: str | None = None
    reason: str | None = None


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
    translation: str | None
    start_ms: int
    end_ms: int
    tokens: tuple[TranscriptToken, ...]


@dataclass(slots=True, frozen=True)
class QuestionInput:
    """表示 question JSON 中的一道题。"""

    scope_type: str
    question_type: str
    coarse_unit_id: int
    target_text: str
    context_sentence_index: int | None
    context_span_index: int | None
    context_start_ms: int | None
    context_end_ms: int | None
    content_payload: dict[str, Any]
    status: str


@dataclass(slots=True, frozen=True)
class SelectedCoarseUnitRef:
    """表示 question JSON 中为 coarse unit 选出的 best evidence 引用。"""

    coarse_unit_id: int
    target_text: str
    sentence_index: int
    token_index: int
    scores: dict[str, Any]
    candidate_score: float | None
    question_reject_reason: str | None
    selection_reason: str


@dataclass(slots=True, frozen=True)
class SelectedCoarseUnitRefs:
    """表示 selected_coarse_unit_refs 顶层结构。"""

    refs: tuple[SelectedCoarseUnitRef, ...]


@dataclass(slots=True, frozen=True)
class LoadedClipInput:
    """表示 loader 阶段输出的“单 clip 原始输入对象”。

    这是整个脚本后续流程的主输入。
    它同时持有：
    - mapped transcript 文件路径、顶层 clip 元数据与 transcript 原文
    - question 文件路径、题目与 selected refs
    - 为写审计记录准备的上下文

    注意：如果 question 文件缺失，这个对象仍然会被创建，但会带上
    skip_reason_code，供 main 直接走 skipped 分支。
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
    video_object_path: str
    thumbnail_url: str | None
    publish_at: datetime | None
    transcript_object_path: str | None
    transcript_checksum: str | None
    transcript_format_version: int
    source_name: str | None
    source_file_path: Path
    expected_transcript_filename: str
    expected_question_filename: str
    transcript_file_path: Path | None
    question_file_path: Path | None
    clip_metadata: ClipMetadata
    transcript_sentences: tuple[TranscriptSentence, ...]
    questions: tuple[QuestionInput, ...]
    selected_coarse_unit_refs: SelectedCoarseUnitRefs | None
    raw_transcript_payload: dict[str, Any] | None
    raw_question_payload: dict[str, Any] | None
    skip_reason_code: str | None = None
    skip_reason_message: str | None = None

    @property
    def context(self) -> dict[str, Any]:
        """生成统一审计上下文。

        审计上下文尽量只放排障需要的信息，不放整坨 transcript。
        """

        return {
            "source_file_path": str(self.source_file_path),
            "expected_transcript_filename": self.expected_transcript_filename,
            "expected_question_filename": self.expected_question_filename,
            "transcript_file_path": str(self.transcript_file_path) if self.transcript_file_path else None,
            "question_file_path": str(self.question_file_path) if self.question_file_path else None,
            "clip_id": self.clip_metadata.clip_id,
            "parent_video_name": self.parent_video_name,
            "source_clip_key": self.source_clip_key,
            "engagement": self.clip_metadata.engagement,
            **self._question_generation_context(),
        }

    def _question_generation_context(self) -> dict[str, Any]:
        if self.raw_question_payload is None:
            return {}

        selected_refs_payload = self.raw_question_payload.get("selected_coarse_unit_refs")
        selected_refs_metadata: dict[str, Any] = {}
        if isinstance(selected_refs_payload, dict):
            selected_refs_metadata = {
                key: selected_refs_payload[key]
                for key in (
                    "version",
                    "selection_model",
                    "selection_top_k",
                    "allowed_question_types",
                    "candidate_score_threshold",
                    "score_weights",
                )
                if key in selected_refs_payload
            }

        return {
            "question_source": self.raw_question_payload.get("source"),
            "question_audit": self.raw_question_payload.get("audit"),
            "selected_coarse_unit_refs_metadata": selected_refs_metadata,
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
    source_start_sentence_index: int | None
    source_end_sentence_index: int | None
    title: str
    description: str | None
    clip_reason: str | None
    engagement_score: dict[str, Any]
    language: str
    duration_ms: int
    video_object_path: str
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
    sentence_count: int
    semantic_span_count: int
    mapped_span_count: int
    unmapped_span_count: int
    mapped_span_ratio: Decimal


@dataclass(slots=True, frozen=True)
class VideoTranscriptSentenceRow:
    """表示将写入 catalog.video_transcript_sentences 的一行数据。"""

    sentence_index: int
    start_ms: int
    end_ms: int
    text: str = ""
    translation: str | None = None


@dataclass(slots=True, frozen=True)
class VideoSemanticSpanRow:
    """表示将写入 catalog.video_semantic_spans 的一行数据。"""

    sentence_index: int
    span_index: int
    start_ms: int
    end_ms: int
    coarse_unit_id: int | None
    surface_text: str = ""
    explanation: str | None = None
    base_form: str | None = None
    translation: str | None = None
    dictionary: str | None = None
    mapping_reason: str | None = None


@dataclass(slots=True, frozen=True)
class BestEvidenceRef:
    """表示 video_unit_index 中已选定的 best evidence span 引用。"""

    sentence_index: int
    span_index: int


@dataclass(slots=True, frozen=True)
class VideoUnitIndexRow:
    """表示将写入 catalog.video_unit_index 的一行聚合结果。"""

    coarse_unit_id: int
    mention_count: int
    sentence_count: int
    coverage_ms: int
    coverage_ratio: Decimal
    sentence_indexes: tuple[int, ...]
    best_evidence_ref: BestEvidenceRef
    best_evidence_start_ms: int
    best_evidence_end_ms: int
    best_evidence_scores: dict[str, Any]
    best_evidence_question_reject_reason: str | None
    best_evidence_selection_reason: str
    best_evidence_candidate_score: float | None = None
    best_evidence_target_text: str | None = None


@dataclass(slots=True, frozen=True)
class QuestionRow:
    """表示将写入 catalog.questions 的一行题目内容。"""

    question_id: str
    scope_type: str
    question_type: str
    coarse_unit_id: int
    target_text: str
    context_sentence_index: int | None
    context_span_index: int | None
    context_start_ms: int | None
    context_end_ms: int | None
    content_payload: dict[str, Any]
    status: str


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
    questions: tuple[QuestionRow, ...]


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
    started_at: datetime | None = None
    finished_at: datetime | None = None


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
    source_start_sentence_index: int | None
    source_end_sentence_index: int | None
    title: str
    description: str | None
    clip_reason: str | None
    engagement_score: dict[str, Any]
    language: str
    duration_ms: int
    video_object_path: str
    thumbnail_url: str | None
    publish_at: datetime | None
    transcript_checksum: str | None


@dataclass(slots=True, frozen=True)
class ClipProcessResult:
    """表示 main 汇总时使用的单 clip 最终结果。"""

    source_clip_key: str
    status: str
    video_id: str | None
    warning_codes: tuple[str, ...]
    error: CatalogIngestError | None
    terminal_record: IngestionRecordPayload | None = None
