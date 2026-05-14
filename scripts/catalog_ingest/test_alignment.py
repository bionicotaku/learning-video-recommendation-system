from __future__ import annotations

import json
import tempfile
import unittest
from pathlib import Path

from scripts.catalog_ingest.index_builder import build_normalized_clip_data
from scripts.catalog_ingest.manifest_loader import load_clip_inputs
from scripts.catalog_ingest.models import (
    LoadedClipInput,
    NormalizedCoreRows,
    ParentClipDescriptor,
    QuestionInput,
    SelectedCoarseUnitRef,
    SelectedCoarseUnitRefs,
    TranscriptSemanticElement,
    TranscriptSentence,
    TranscriptToken,
    VideoRow,
    VideoSemanticSpanRow,
    VideoTranscriptSentenceRow,
)
from scripts.catalog_ingest.repository import CatalogRepository
from scripts.catalog_ingest.validator import validate_loaded_clip


class CatalogIngestAlignmentTest(unittest.TestCase):
    def test_load_clip_inputs_accepts_extra_semantic_display_fields_without_db_rows(self) -> None:
        with tempfile.TemporaryDirectory() as tmp_dir:
            root = Path(tmp_dir)
            parents_dir = root / "parents"
            transcripts_dir = root / "transcripts"
            questions_dir = root / "questions"
            parents_dir.mkdir()
            transcripts_dir.mkdir()
            questions_dir.mkdir()

            (parents_dir / "demo.json").write_text(
                json.dumps(
                    {
                        "clips": [
                            {
                                "clip_id": 1,
                                "start_index": 0,
                                "end_index": 0,
                                "start_time": 100,
                                "end_time": 200,
                                "buffered_start_time": 100,
                                "buffered_end_time": 200,
                                "reasoning": "demo",
                            }
                        ]
                    }
                ),
                encoding="utf-8",
            )
            (transcripts_dir / "demo-clip1.json").write_text(
                json.dumps(
                    {
                        "sentences": [
                            {
                                "index": 0,
                                "text": "Pam!",
                                "translation": "帕姆！",
                                "start": 110,
                                "end": 150,
                                "tokens": [
                                    {
                                        "index": 0,
                                        "text": "Pam!",
                                        "explanation": "Pam",
                                        "start": 110,
                                        "end": 150,
                                        "semantic_element": {
                                            "base_form": "Pam",
                                            "translation": "帕姆",
                                            "dictionary": "Pam",
                                            "coarse_id": 7,
                                            "reason": "name",
                                        },
                                    }
                                ],
                            }
                        ]
                    }
                ),
                encoding="utf-8",
            )
            (questions_dir / "demo-clip1.json").write_text(
                json.dumps(_question_payload(coarse_unit_id=7, sentence_index=0, token_index=0)),
                encoding="utf-8",
            )

            loaded = load_clip_inputs(
                parents_dir=parents_dir,
                transcripts_dir=transcripts_dir,
                questions_dir=questions_dir,
            )

        token = loaded[0].transcript_sentences[0].tokens[0]
        sentence = loaded[0].transcript_sentences[0]
        self.assertEqual(sentence.translation, "帕姆！")
        self.assertFalse(hasattr(sentence, "explanation"))
        self.assertIsNotNone(token.semantic_element)
        self.assertEqual(token.semantic_element.coarse_id, 7)
        core_rows = NormalizedCoreRows(
            video=VideoRow(
                source_clip_key=loaded[0].source_clip_key,
                parent_video_name=loaded[0].parent_video_name,
                parent_video_slug=loaded[0].parent_video_slug,
                clip_seq=loaded[0].clip_seq,
                source_start_ms=loaded[0].source_start_ms,
                source_end_ms=loaded[0].source_end_ms,
                title=loaded[0].title,
                description=loaded[0].description,
                clip_reason=loaded[0].clip_reason,
                language=loaded[0].language,
                duration_ms=loaded[0].duration_ms,
                hls_master_playlist_path=loaded[0].hls_master_playlist_path,
                thumbnail_url=loaded[0].thumbnail_url,
                status="active",
                visibility_status="public",
                publish_at=loaded[0].publish_at,
            ),
            sentences=(
                VideoTranscriptSentenceRow(
                    sentence_index=0,
                    start_ms=110,
                    end_ms=150,
                ),
            ),
            spans=(
                VideoSemanticSpanRow(
                    sentence_index=0,
                    span_index=0,
                    start_ms=110,
                    end_ms=150,
                    coarse_unit_id=7,
                ),
            ),
        )
        normalized = build_normalized_clip_data(loaded[0], core_rows)
        self.assertFalse(hasattr(normalized.sentences[0], "translation"))
        self.assertFalse(hasattr(normalized.sentences[0], "explanation"))
        self.assertFalse(hasattr(normalized.spans[0], "base_form"))
        self.assertFalse(hasattr(normalized.spans[0], "dictionary_text"))
        self.assertFalse(hasattr(normalized.spans[0], "translation"))

    def test_unit_index_uses_selected_best_evidence_and_no_surface_forms(self) -> None:
        clip_input = _build_clip_input(
            selected_refs=(
                SelectedCoarseUnitRef(coarse_unit_id=42, sentence_index=2, token_index=3),
            )
        )
        core_rows = NormalizedCoreRows(
            video=VideoRow(
                source_clip_key=clip_input.source_clip_key,
                parent_video_name=clip_input.parent_video_name,
                parent_video_slug=clip_input.parent_video_slug,
                clip_seq=clip_input.clip_seq,
                source_start_ms=clip_input.source_start_ms,
                source_end_ms=clip_input.source_end_ms,
                title=clip_input.title,
                description=clip_input.description,
                clip_reason=clip_input.clip_reason,
                language=clip_input.language,
                duration_ms=clip_input.duration_ms,
                hls_master_playlist_path=clip_input.hls_master_playlist_path,
                thumbnail_url=clip_input.thumbnail_url,
                status="active",
                visibility_status="public",
                publish_at=clip_input.publish_at,
            ),
            sentences=tuple(
                VideoTranscriptSentenceRow(
                    sentence_index=index,
                    start_ms=index * 100,
                    end_ms=index * 100 + 90,
                )
                for index in range(6)
            ),
            spans=(
                VideoSemanticSpanRow(2, 3, 230, 240, 42),
                VideoSemanticSpanRow(0, 4, 40, 50, 42),
                VideoSemanticSpanRow(5, 1, 510, 520, 42),
                VideoSemanticSpanRow(1, 7, 170, 180, 42),
                VideoSemanticSpanRow(4, 2, 420, 430, 42),
                VideoSemanticSpanRow(3, 0, 300, 310, 42),
                VideoSemanticSpanRow(1, 2, 120, 130, 42),
                VideoSemanticSpanRow(0, 1, 10, 20, 42),
                VideoSemanticSpanRow(2, 1, 210, 220, 42),
            ),
        )

        normalized = build_normalized_clip_data(clip_input, core_rows)
        unit_index = normalized.unit_indexes[0]

        self.assertEqual(unit_index.sentence_indexes, (0, 1, 2, 3, 4, 5))
        self.assertEqual((unit_index.best_evidence_ref.sentence_index, unit_index.best_evidence_ref.span_index), (2, 3))
        self.assertEqual(unit_index.best_evidence_source, "selected_coarse_unit_refs")
        self.assertFalse(hasattr(unit_index, "sample_surface_forms"))
        self.assertFalse(hasattr(unit_index, "evidence_span_refs"))

    def test_load_clip_inputs_skips_when_question_file_missing(self) -> None:
        with tempfile.TemporaryDirectory() as tmp_dir:
            root = Path(tmp_dir)
            parents_dir = root / "parents"
            transcripts_dir = root / "transcripts"
            questions_dir = root / "questions"
            parents_dir.mkdir()
            transcripts_dir.mkdir()
            questions_dir.mkdir()
            _write_parent_file(parents_dir / "demo.json")
            _write_transcript_file(transcripts_dir / "demo-clip1.json", coarse_unit_id=7)

            loaded = load_clip_inputs(
                parents_dir=parents_dir,
                transcripts_dir=transcripts_dir,
                questions_dir=questions_dir,
            )

        self.assertEqual(loaded[0].skip_reason_code, "question_missing")

    def test_selected_refs_must_match_transcript_mapped_units(self) -> None:
        clip_input = _build_clip_input(
            transcript_sentences=(
                _sentence(index=0, coarse_unit_id=7),
                _sentence(index=1, coarse_unit_id=8),
            ),
            selected_refs=(
                SelectedCoarseUnitRef(coarse_unit_id=7, sentence_index=0, token_index=0),
            ),
        )

        with self.assertRaisesRegex(Exception, "一对一"):
            validate_loaded_clip(
                clip_input=clip_input,
                known_coarse_unit_ids={7, 8},
                time_tolerance_ms=0,
            )

    def test_selected_ref_must_point_to_matching_token(self) -> None:
        clip_input = _build_clip_input(
            transcript_sentences=(
                _sentence(index=0, coarse_unit_id=7),
            ),
            selected_refs=(
                SelectedCoarseUnitRef(coarse_unit_id=7, sentence_index=0, token_index=9),
            ),
        )

        with self.assertRaisesRegex(Exception, "不存在的 token"):
            validate_loaded_clip(
                clip_input=clip_input,
                known_coarse_unit_ids={7},
                time_tolerance_ms=0,
            )

    def test_question_id_is_deterministic_for_same_question_payload(self) -> None:
        clip_input = _build_clip_input()
        core_rows = NormalizedCoreRows(
            video=VideoRow(
                source_clip_key=clip_input.source_clip_key,
                parent_video_name=clip_input.parent_video_name,
                parent_video_slug=clip_input.parent_video_slug,
                clip_seq=clip_input.clip_seq,
                source_start_ms=clip_input.source_start_ms,
                source_end_ms=clip_input.source_end_ms,
                title=clip_input.title,
                description=clip_input.description,
                clip_reason=clip_input.clip_reason,
                language=clip_input.language,
                duration_ms=clip_input.duration_ms,
                hls_master_playlist_path=clip_input.hls_master_playlist_path,
                thumbnail_url=clip_input.thumbnail_url,
                status="active",
                visibility_status="public",
                publish_at=clip_input.publish_at,
            ),
            sentences=(VideoTranscriptSentenceRow(sentence_index=0, start_ms=0, end_ms=100),),
            spans=(VideoSemanticSpanRow(0, 0, 0, 50, 1),),
        )

        first = build_normalized_clip_data(clip_input, core_rows)
        second = build_normalized_clip_data(clip_input, core_rows)

        self.assertEqual(first.questions[0].question_id, second.questions[0].question_id)
        self.assertEqual(first.questions[0].scope_type, "video_unit")

    def test_ingest_questions_are_video_unit_when_scope_type_is_omitted(self) -> None:
        with tempfile.TemporaryDirectory() as tmp_dir:
            root = Path(tmp_dir)
            parents_dir = root / "parents"
            transcripts_dir = root / "transcripts"
            questions_dir = root / "questions"
            parents_dir.mkdir()
            transcripts_dir.mkdir()
            questions_dir.mkdir()
            _write_parent_file(parents_dir / "demo.json")
            _write_transcript_file(transcripts_dir / "demo-clip1.json", coarse_unit_id=7)
            question_payload = _question_payload(7, 0, 0)
            del question_payload["questions"][0]["scope_type"]
            (questions_dir / "demo-clip1.json").write_text(
                json.dumps(question_payload),
                encoding="utf-8",
            )

            loaded = load_clip_inputs(
                parents_dir=parents_dir,
                transcripts_dir=transcripts_dir,
                questions_dir=questions_dir,
            )

        self.assertEqual(loaded[0].questions[0].scope_type, "video_unit")

    def test_ingest_rejects_non_video_unit_questions(self) -> None:
        clip_input = _build_clip_input(
            questions=(
                QuestionInput(
                    scope_type="unit",
                    question_type="context_meaning_choice",
                    coarse_unit_id=1,
                    target_text="demo",
                    context_sentence_index=0,
                    context_span_index=0,
                    context_start_ms=0,
                    context_end_ms=50,
                    content_payload=_content_payload(),
                    status="active",
                ),
            )
        )

        with self.assertRaisesRegex(Exception, "Catalog ingest 只允许写入 video_unit"):
            validate_loaded_clip(
                clip_input=clip_input,
                known_coarse_unit_ids={1},
                time_tolerance_ms=0,
            )

    def test_repository_upserts_questions_and_retires_stale_video_questions(self) -> None:
        clip_input = _build_clip_input()
        core_rows = NormalizedCoreRows(
            video=VideoRow(
                source_clip_key=clip_input.source_clip_key,
                parent_video_name=clip_input.parent_video_name,
                parent_video_slug=clip_input.parent_video_slug,
                clip_seq=clip_input.clip_seq,
                source_start_ms=clip_input.source_start_ms,
                source_end_ms=clip_input.source_end_ms,
                title=clip_input.title,
                description=clip_input.description,
                clip_reason=clip_input.clip_reason,
                language=clip_input.language,
                duration_ms=clip_input.duration_ms,
                hls_master_playlist_path=clip_input.hls_master_playlist_path,
                thumbnail_url=clip_input.thumbnail_url,
                status="active",
                visibility_status="public",
                publish_at=clip_input.publish_at,
            ),
            sentences=(VideoTranscriptSentenceRow(sentence_index=0, start_ms=0, end_ms=100),),
            spans=(VideoSemanticSpanRow(0, 0, 0, 50, 1),),
        )
        normalized = build_normalized_clip_data(clip_input, core_rows)
        cursor = _FakeCursor()

        CatalogRepository("postgresql://unused")._replace_question_rows(cursor, "video-1", normalized)

        self.assertEqual(len(cursor.executemany_calls), 1)
        insert_sql, params = cursor.executemany_calls[0]
        self.assertIn("insert into catalog.questions", insert_sql)
        self.assertEqual(params[0][0], normalized.questions[0].question_id)
        self.assertEqual(params[0][5], "video-1")
        self.assertEqual(len(cursor.execute_calls), 1)
        retire_sql, retire_params = cursor.execute_calls[0]
        self.assertIn("status = 'retired'", retire_sql)
        self.assertEqual(retire_params[0], "video-1")
        self.assertEqual(retire_params[1], [normalized.questions[0].question_id])

    def test_token_outside_sentence_stays_warning_for_assemblyai_edge_case(self) -> None:
        clip_input = _build_clip_input(
            transcript_sentences=(
                TranscriptSentence(
                    index=0,
                    text="edge case",
                    translation="边界情况",
                    start_ms=100,
                    end_ms=200,
                    tokens=(
                        TranscriptToken(
                            index=0,
                            text="edge",
                            explanation=None,
                            start_ms=110,
                            end_ms=205,
                            semantic_element=TranscriptSemanticElement(
                                coarse_id=7,
                            ),
                        ),
                    ),
                ),
            )
        )

        warnings = validate_loaded_clip(
            clip_input=clip_input,
            known_coarse_unit_ids={7},
            time_tolerance_ms=0,
        )

        self.assertEqual(len(warnings), 1)
        self.assertEqual(warnings[0].code, "token_time_outside_sentence")


def _build_clip_input(
    transcript_sentences: tuple[TranscriptSentence, ...] | None = None,
    selected_refs: tuple[SelectedCoarseUnitRef, ...] | None = None,
    questions: tuple[QuestionInput, ...] | None = None,
) -> LoadedClipInput:
    sentences = transcript_sentences or (
        TranscriptSentence(
            index=0,
            text="default sentence",
            translation="默认句子",
            start_ms=0,
            end_ms=100,
            tokens=(
                TranscriptToken(
                    index=0,
                    text="default",
                    explanation=None,
                    start_ms=0,
                    end_ms=50,
                    semantic_element=TranscriptSemanticElement(
                        coarse_id=1,
                    ),
                ),
            ),
        ),
    )
    return LoadedClipInput(
        source_clip_key="parent#clip1",
        parent_video_name="parent",
        parent_video_slug="parent",
        clip_seq=1,
        source_start_ms=0,
        source_end_ms=1000,
        title="clip title",
        description=None,
        clip_reason=None,
        language="en",
        duration_ms=1000,
        hls_master_playlist_path="hls/master.m3u8",
        thumbnail_url=None,
        publish_at=None,
        transcript_object_path="transcript.json",
        transcript_checksum="checksum",
        transcript_format_version=1,
        source_name="test",
        parent_file_path=Path("parent.json"),
        expected_transcript_filename="parent-clip1.json",
        transcript_file_path=Path("parent-clip1.json"),
        parent_clip=ParentClipDescriptor(
            clip_id=1,
            start_index=0,
            end_index=1,
            start_time=0,
            end_time=900,
            buffered_start_time=0,
            buffered_end_time=1000,
            reasoning=None,
        ),
        transcript_sentences=sentences,
        raw_parent_payload={"clips": []},
        raw_transcript_payload={"sentences": []},
        expected_question_filename="parent-clip1.json",
        question_file_path=Path("parent-clip1.json"),
        questions=questions
        or (
            QuestionInput(
                scope_type="video_unit",
                question_type="context_meaning_choice",
                coarse_unit_id=(selected_refs[0].coarse_unit_id if selected_refs else sentences[0].tokens[0].semantic_element.coarse_id),
                target_text="default",
                context_sentence_index=(selected_refs[0].sentence_index if selected_refs else sentences[0].index),
                context_span_index=(selected_refs[0].token_index if selected_refs else sentences[0].tokens[0].index),
                context_start_ms=sentences[0].start_ms,
                context_end_ms=sentences[0].end_ms,
                content_payload=_content_payload(),
                status="draft",
            ),
        ),
        selected_coarse_unit_refs=SelectedCoarseUnitRefs(
            version=1,
            selection_model="test-model",
            selection_top_k=5,
            allowed_question_types=("context_meaning_choice",),
            refs=selected_refs
            or (
                SelectedCoarseUnitRef(
                    coarse_unit_id=sentences[0].tokens[0].semantic_element.coarse_id,
                    sentence_index=sentences[0].index,
                    token_index=sentences[0].tokens[0].index,
                ),
            ),
        ),
        raw_question_payload={"source": {"model": "test-source"}, "questions": []},
    )


def _write_parent_file(path: Path) -> None:
    path.write_text(
        json.dumps(
            {
                "clips": [
                    {
                        "clip_id": 1,
                        "buffered_start_time": 100,
                        "buffered_end_time": 200,
                    }
                ]
            }
        ),
        encoding="utf-8",
    )


def _write_transcript_file(path: Path, coarse_unit_id: int) -> None:
    path.write_text(
        json.dumps(
            {
                "sentences": [
                    {
                        "index": 0,
                        "text": "demo",
                        "translation": "demo",
                        "start": 110,
                        "end": 150,
                        "tokens": [
                            {
                                "index": 0,
                                "text": "demo",
                                "start": 110,
                                "end": 150,
                                "semantic_element": {"coarse_id": coarse_unit_id},
                            }
                        ],
                    }
                ]
            }
        ),
        encoding="utf-8",
    )


def _question_payload(coarse_unit_id: int, sentence_index: int, token_index: int) -> dict[str, object]:
    return {
        "source": {"model": "test-source"},
        "questions": [
            {
                "scope_type": "video_unit",
                "question_type": "context_meaning_choice",
                "coarse_unit_id": coarse_unit_id,
                "target_text": "demo",
                "context_sentence_index": sentence_index,
                "context_span_index": token_index,
                "context_start_ms": 110,
                "context_end_ms": 150,
                "content_payload": _content_payload(),
                "status": "draft",
            }
        ],
        "audit": {},
        "selected_coarse_unit_refs": {
            "version": 1,
            "selection_model": "test-model",
            "selection_top_k": 5,
            "allowed_question_types": ["context_meaning_choice"],
            "refs": [
                {
                    "coarse_unit_id": coarse_unit_id,
                    "sentence_index": sentence_index,
                    "token_index": token_index,
                }
            ],
        },
    }


def _content_payload() -> dict[str, object]:
    return {
        "question": "demo?",
        "options": [
            {"id": "correct", "text": "right"},
            {"id": "wrong_1", "text": "wrong"},
        ],
        "explanation": "demo",
    }


def _sentence(index: int, coarse_unit_id: int) -> TranscriptSentence:
    return TranscriptSentence(
        index=index,
        text=f"sentence {index}",
        translation=None,
        start_ms=index * 100,
        end_ms=index * 100 + 50,
        tokens=(
            TranscriptToken(
                index=0,
                text="token",
                explanation=None,
                start_ms=index * 100,
                end_ms=index * 100 + 40,
                semantic_element=TranscriptSemanticElement(coarse_id=coarse_unit_id),
            ),
        ),
    )


class _FakeCursor:
    def __init__(self) -> None:
        self.execute_calls: list[tuple[str, tuple[object, ...]]] = []
        self.executemany_calls: list[tuple[str, list[tuple[object, ...]]]] = []

    def execute(self, sql: str, params: tuple[object, ...]) -> None:
        self.execute_calls.append((sql, params))

    def executemany(self, sql: str, params: list[tuple[object, ...]]) -> None:
        self.executemany_calls.append((sql, params))


if __name__ == "__main__":
    unittest.main()
