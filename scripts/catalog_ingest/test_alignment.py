from __future__ import annotations

import json
import tempfile
import unittest
from dataclasses import replace
from datetime import datetime, timezone
from pathlib import Path
from unittest import mock

from scripts.catalog_ingest import main as ingest_main
from scripts.catalog_ingest.index_builder import build_normalized_clip_data
from scripts.catalog_ingest.manifest_loader import load_clip_inputs
from scripts.catalog_ingest.models import (
    CatalogIngestError,
    ClipMetadata,
    LoadedClipInput,
    NormalizedCoreRows,
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
from scripts.catalog_ingest.normalizer import normalize_clip_input
from scripts.catalog_ingest.repository import CatalogRepository
from scripts.catalog_ingest.validator import validate_loaded_clip


class CatalogIngestAlignmentTest(unittest.TestCase):
    def test_load_clip_inputs_reads_clip_metadata_from_mapped_transcript_file(self) -> None:
        with tempfile.TemporaryDirectory() as tmp_dir:
            root = Path(tmp_dir)
            transcripts_dir = root / "transcripts"
            questions_dir = root / "questions"
            transcripts_dir.mkdir()
            questions_dir.mkdir()

            (transcripts_dir / "demo-clip1.json").write_text(
                json.dumps(
                    {
                        "clip_id": 1,
                        "title": "真实标题",
                        "description": "真实描述",
                        "engagement": {
                            "drama": 4,
                            "humor": 7,
                            "payoff": 6,
                            "standalone": 7,
                            "reasoning": "engaging",
                        },
                        "start_index": 0,
                        "end_index": 0,
                        "start_time": 100,
                        "end_time": 200,
                        "buffered_start_time": 100,
                        "buffered_end_time": 200,
                        "duration_time": 100,
                        "reasoning": "demo",
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

            before_load = datetime.now(timezone.utc)
            loaded = load_clip_inputs(
                transcripts_dir=transcripts_dir,
                questions_dir=questions_dir,
            )
            after_load = datetime.now(timezone.utc)

        clip_input = loaded[0]
        self.assertEqual(clip_input.source_clip_key, "demo#clip1")
        self.assertEqual(clip_input.parent_video_name, "demo")
        self.assertEqual(clip_input.title, "真实标题")
        self.assertEqual(clip_input.description, "真实描述")
        self.assertEqual(clip_input.clip_reason, "demo")
        self.assertEqual(clip_input.duration_ms, 100)
        self.assertEqual(clip_input.source_start_ms, 100)
        self.assertEqual(clip_input.source_end_ms, 200)
        self.assertEqual(
            clip_input.video_object_path,
            "https://storage.googleapis.com/videos2077/test-video/portrait_videos/demo-clip1.mp4",
        )
        self.assertEqual(
            clip_input.transcript_object_path,
            "https://storage.googleapis.com/videos2077/test-video/transcript/demo-clip1.json",
        )
        self.assertEqual(
            clip_input.thumbnail_url,
            "https://storage.googleapis.com/videos2077/test-video/cover/demo-clip1.webp",
        )
        self.assertIsNotNone(clip_input.publish_at)
        self.assertLessEqual(before_load, clip_input.publish_at)
        self.assertLessEqual(clip_input.publish_at, after_load)
        self.assertEqual(clip_input.clip_metadata.engagement["humor"], 7)
        self.assertEqual(clip_input.questions[0].status, "active")
        self.assertEqual(clip_input.context["question_source"], {"model": "test-source"})
        self.assertEqual(clip_input.context["question_audit"], {})
        self.assertEqual(
            clip_input.context["selected_coarse_unit_refs_metadata"]["selection_model"],
            "test-model",
        )
        token = clip_input.transcript_sentences[0].tokens[0]
        sentence = clip_input.transcript_sentences[0]
        self.assertEqual(sentence.translation, "帕姆！")
        self.assertFalse(hasattr(sentence, "explanation"))
        self.assertIsNotNone(token.semantic_element)
        self.assertEqual(token.semantic_element.coarse_id, 7)
        self.assertEqual(token.semantic_element.base_form, "Pam")
        self.assertEqual(token.semantic_element.dictionary, "Pam")
        self.assertEqual(token.semantic_element.translation, "帕姆")
        self.assertEqual(token.semantic_element.reason, "name")
        core_rows = NormalizedCoreRows(
            video=VideoRow(
                source_clip_key=clip_input.source_clip_key,
                parent_video_name=clip_input.parent_video_name,
                parent_video_slug=clip_input.parent_video_slug,
                clip_seq=clip_input.clip_seq,
                source_start_ms=clip_input.source_start_ms,
                source_end_ms=clip_input.source_end_ms,
                source_start_sentence_index=clip_input.clip_metadata.start_index,
                source_end_sentence_index=clip_input.clip_metadata.end_index,
                title=clip_input.title,
                description=clip_input.description,
                clip_reason=clip_input.clip_reason,
                engagement_score=clip_input.clip_metadata.engagement,
                language=clip_input.language,
                duration_ms=clip_input.duration_ms,
                video_object_path=clip_input.video_object_path,
                thumbnail_url=clip_input.thumbnail_url,
                status="active",
                visibility_status="public",
                publish_at=clip_input.publish_at,
            ),
            sentences=(
                VideoTranscriptSentenceRow(
                    sentence_index=0,
                    start_ms=110,
                    end_ms=150,
                    text="Pam!",
                    translation="帕姆！",
                ),
            ),
            spans=(
                VideoSemanticSpanRow(
                    sentence_index=0,
                    span_index=0,
                    start_ms=110,
                    end_ms=150,
                    coarse_unit_id=7,
                    surface_text="Pam!",
                    explanation="Pam",
                    base_form="Pam",
                    translation="帕姆",
                    dictionary="Pam",
                    mapping_reason="name",
                ),
            ),
        )
        normalized = build_normalized_clip_data(clip_input, core_rows)
        self.assertEqual(normalized.video.source_start_sentence_index, 0)
        self.assertEqual(normalized.video.source_end_sentence_index, 0)
        self.assertEqual(normalized.video.engagement_score["humor"], 7)
        self.assertEqual(normalized.sentences[0].text, "Pam!")
        self.assertEqual(normalized.sentences[0].translation, "帕姆！")
        self.assertEqual(normalized.spans[0].surface_text, "Pam!")
        self.assertEqual(normalized.spans[0].explanation, "Pam")
        self.assertEqual(normalized.spans[0].base_form, "Pam")
        self.assertEqual(normalized.spans[0].dictionary, "Pam")
        self.assertEqual(normalized.spans[0].translation, "帕姆")
        self.assertEqual(normalized.spans[0].mapping_reason, "name")
        self.assertFalse(hasattr(normalized.sentences[0], "explanation"))
        self.assertFalse(hasattr(normalized.spans[0], "dictionary_text"))

    def test_unit_index_uses_selected_best_evidence_metadata(self) -> None:
        clip_input = _build_clip_input(
            selected_refs=(
                SelectedCoarseUnitRef(
                    coarse_unit_id=42,
                    target_text="chosen target",
                    sentence_index=2,
                    token_index=3,
                    scores={
                        "visual_context": 3,
                        "context_clarity": 8,
                        "learning_value": 9,
                        "representative_salience": 8,
                    },
                    candidate_score=8.35,
                    question_reject_reason=None,
                    selection_reason="clear context",
                ),
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
                source_start_sentence_index=clip_input.clip_metadata.start_index,
                source_end_sentence_index=clip_input.clip_metadata.end_index,
                title=clip_input.title,
                description=clip_input.description,
                clip_reason=clip_input.clip_reason,
                engagement_score=clip_input.clip_metadata.engagement,
                language=clip_input.language,
                duration_ms=clip_input.duration_ms,
                video_object_path=clip_input.video_object_path,
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
        self.assertEqual(unit_index.best_evidence_start_ms, 230)
        self.assertEqual(unit_index.best_evidence_end_ms, 240)
        self.assertEqual(unit_index.best_evidence_scores["visual_context"], 3)
        self.assertEqual(unit_index.best_evidence_selection_reason, "clear context")
        self.assertIsNone(unit_index.best_evidence_question_reject_reason)
        self.assertEqual(unit_index.best_evidence_candidate_score, 8.35)
        self.assertEqual(unit_index.best_evidence_target_text, "chosen target")
        self.assertFalse(hasattr(unit_index, "first_start_ms"))
        self.assertFalse(hasattr(unit_index, "last_end_ms"))
        self.assertFalse(hasattr(unit_index, "best_evidence_source"))
        self.assertFalse(hasattr(unit_index, "best_evidence_model"))
        self.assertFalse(hasattr(unit_index, "best_evidence_version"))
        self.assertFalse(hasattr(unit_index, "best_evidence_metadata"))
        self.assertFalse(hasattr(unit_index, "sample_surface_forms"))
        self.assertFalse(hasattr(unit_index, "evidence_span_refs"))

    def test_load_clip_inputs_skips_when_question_file_missing(self) -> None:
        with tempfile.TemporaryDirectory() as tmp_dir:
            root = Path(tmp_dir)
            transcripts_dir = root / "transcripts"
            questions_dir = root / "questions"
            transcripts_dir.mkdir()
            questions_dir.mkdir()
            _write_transcript_file(transcripts_dir / "demo-clip1.json", coarse_unit_id=7)

            loaded = load_clip_inputs(
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
                _selected_ref(coarse_unit_id=7, sentence_index=0, token_index=0),
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
                _selected_ref(coarse_unit_id=7, sentence_index=0, token_index=9),
            ),
        )

        with self.assertRaisesRegex(Exception, "不存在的 token"):
            validate_loaded_clip(
                clip_input=clip_input,
                known_coarse_unit_ids={7},
                time_tolerance_ms=0,
            )

    def test_validate_accepts_clip_local_transcript_times_with_absolute_source_range(self) -> None:
        clip_input = _build_clip_input()
        clip_input = replace(
            clip_input,
            source_start_ms=50_000,
            source_end_ms=51_000,
            clip_metadata=replace(
                clip_input.clip_metadata,
                start_time=50_100,
                end_time=50_900,
                buffered_start_time=50_000,
                buffered_end_time=51_000,
            ),
        )

        warnings = validate_loaded_clip(
            clip_input=clip_input,
            known_coarse_unit_ids={1},
            time_tolerance_ms=0,
        )

        self.assertEqual(warnings, tuple())

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
                source_start_sentence_index=clip_input.clip_metadata.start_index,
                source_end_sentence_index=clip_input.clip_metadata.end_index,
                title=clip_input.title,
                description=clip_input.description,
                clip_reason=clip_input.clip_reason,
                engagement_score=clip_input.clip_metadata.engagement,
                language=clip_input.language,
                duration_ms=clip_input.duration_ms,
                video_object_path=clip_input.video_object_path,
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
            transcripts_dir = root / "transcripts"
            questions_dir = root / "questions"
            transcripts_dir.mkdir()
            questions_dir.mkdir()
            _write_transcript_file(transcripts_dir / "demo-clip1.json", coarse_unit_id=7)
            question_payload = _question_payload(7, 0, 0)
            del question_payload["questions"][0]["scope_type"]
            (questions_dir / "demo-clip1.json").write_text(
                json.dumps(question_payload),
                encoding="utf-8",
            )

            loaded = load_clip_inputs(
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
                source_start_sentence_index=clip_input.clip_metadata.start_index,
                source_end_sentence_index=clip_input.clip_metadata.end_index,
                title=clip_input.title,
                description=clip_input.description,
                clip_reason=clip_input.clip_reason,
                engagement_score=clip_input.clip_metadata.engagement,
                language=clip_input.language,
                duration_ms=clip_input.duration_ms,
                video_object_path=clip_input.video_object_path,
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
        self.assertEqual(params[0][11], "active")
        self.assertEqual(len(cursor.execute_calls), 1)
        retire_sql, retire_params = cursor.execute_calls[0]
        self.assertIn("status = 'retired'", retire_sql)
        self.assertEqual(retire_params[0], "video-1")
        self.assertEqual(retire_params[1], [normalized.questions[0].question_id])

    def test_repository_writes_sentence_span_and_best_evidence_metadata(self) -> None:
        clip_input = _build_clip_input(
            transcript_sentences=(
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
                            explanation="default explanation",
                            start_ms=0,
                            end_ms=50,
                            semantic_element=TranscriptSemanticElement(
                                coarse_id=1,
                                base_form="default",
                                translation="默认",
                                dictionary="default dictionary",
                                reason="default reason",
                            ),
                        ),
                    ),
                ),
            ),
        )
        normalized = build_normalized_clip_data(clip_input, normalize_clip_input(clip_input))
        cursor = _FakeCursor()

        CatalogRepository("postgresql://unused")._replace_transcript_related_rows(cursor, "video-1", normalized)

        sentence_sql, sentence_params = cursor.executemany_calls[0]
        self.assertIn("text", sentence_sql)
        self.assertIn("translation", sentence_sql)
        self.assertEqual(sentence_params[0][4], "default sentence")
        self.assertEqual(sentence_params[0][5], "默认句子")

        span_sql, span_params = cursor.executemany_calls[1]
        self.assertIn("surface_text", span_sql)
        self.assertIn("mapping_reason", span_sql)
        self.assertEqual(span_params[0][6], "default")
        self.assertEqual(span_params[0][7], "default explanation")
        self.assertEqual(span_params[0][8], "default")
        self.assertEqual(span_params[0][9], "默认")
        self.assertEqual(span_params[0][10], "default dictionary")
        self.assertEqual(span_params[0][11], "default reason")

        unit_sql, unit_params = cursor.executemany_calls[2]
        self.assertIn("best_evidence_start_ms", unit_sql)
        self.assertIn("best_evidence_end_ms", unit_sql)
        self.assertIn("best_evidence_candidate_score", unit_sql)
        self.assertIn("best_evidence_target_text", unit_sql)
        self.assertEqual(unit_params[0][9], 0)
        self.assertEqual(unit_params[0][10], 50)
        self.assertEqual(unit_params[0][14], 8.35)
        self.assertEqual(unit_params[0][15], "token")

    def test_repository_upserts_video_sentence_indexes_and_engagement_score(self) -> None:
        clip_input = _build_clip_input()
        video = VideoRow(
            source_clip_key=clip_input.source_clip_key,
            parent_video_name=clip_input.parent_video_name,
            parent_video_slug=clip_input.parent_video_slug,
            clip_seq=clip_input.clip_seq,
            source_start_ms=clip_input.source_start_ms,
            source_end_ms=clip_input.source_end_ms,
            source_start_sentence_index=clip_input.clip_metadata.start_index,
            source_end_sentence_index=clip_input.clip_metadata.end_index,
            title=clip_input.title,
            description=clip_input.description,
            clip_reason=clip_input.clip_reason,
            engagement_score=clip_input.clip_metadata.engagement,
            language=clip_input.language,
            duration_ms=clip_input.duration_ms,
            video_object_path=clip_input.video_object_path,
            thumbnail_url=clip_input.thumbnail_url,
            status="active",
            visibility_status="public",
            publish_at=clip_input.publish_at,
        )
        normalized = build_normalized_clip_data(
            clip_input,
            NormalizedCoreRows(
                video=video,
                sentences=(VideoTranscriptSentenceRow(sentence_index=0, start_ms=0, end_ms=100),),
                spans=(VideoSemanticSpanRow(0, 0, 0, 50, 1),),
            ),
        )
        cursor = _FakeCursor()

        video_id = CatalogRepository("postgresql://unused")._upsert_video(cursor, normalized)

        self.assertEqual(video_id, "video-1")
        sql, params = cursor.execute_calls[0]
        self.assertIn("source_start_sentence_index", sql)
        self.assertIn("source_end_sentence_index", sql)
        self.assertIn("engagement_score", sql)
        self.assertEqual(params[6], 0)
        self.assertEqual(params[7], 1)
        self.assertEqual(params[11].obj["humor"], 8)

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

    def test_main_refreshes_recommendation_projection_after_successful_write(self) -> None:
        with tempfile.TemporaryDirectory() as tmp_dir:
            root = Path(tmp_dir)
            transcripts_dir = root / "transcripts"
            questions_dir = root / "questions"
            transcripts_dir.mkdir()
            questions_dir.mkdir()
            _write_transcript_file(transcripts_dir / "demo-clip1.json", coarse_unit_id=7)
            (questions_dir / "demo-clip1.json").write_text(
                json.dumps(_question_payload(7, 0, 0)),
                encoding="utf-8",
            )

            fake_repo = _FakeMainRepository()
            refresh = mock.Mock()
            with (
                mock.patch.object(ingest_main, "load_database_url", return_value="postgresql://unused"),
                mock.patch.object(ingest_main, "CatalogRepository", return_value=fake_repo),
                mock.patch.object(ingest_main, "run_recommendation_refresh", refresh),
            ):
                exit_code = ingest_main.main(
                    [
                        "--transcripts-dir",
                        str(transcripts_dir),
                        "--questions-dir",
                        str(questions_dir),
                        "--source-name",
                        "local-json",
                    ]
                )

        self.assertEqual(exit_code, 0)
        self.assertEqual(fake_repo.persist_count, 1)
        refresh.assert_called_once_with()

    def test_main_skips_recommendation_refresh_for_dry_run_and_skip_flag(self) -> None:
        for extra_args in (("--dry-run",), ("--skip-recommendation-refresh",)):
            with self.subTest(extra_args=extra_args):
                with tempfile.TemporaryDirectory() as tmp_dir:
                    root = Path(tmp_dir)
                    transcripts_dir = root / "transcripts"
                    questions_dir = root / "questions"
                    transcripts_dir.mkdir()
                    questions_dir.mkdir()
                    _write_transcript_file(transcripts_dir / "demo-clip1.json", coarse_unit_id=7)
                    (questions_dir / "demo-clip1.json").write_text(
                        json.dumps(_question_payload(7, 0, 0)),
                        encoding="utf-8",
                    )

                    fake_repo = _FakeMainRepository()
                    refresh = mock.Mock()
                    with (
                        mock.patch.object(ingest_main, "load_database_url", return_value="postgresql://unused"),
                        mock.patch.object(ingest_main, "CatalogRepository", return_value=fake_repo),
                        mock.patch.object(ingest_main, "run_recommendation_refresh", refresh),
                    ):
                        exit_code = ingest_main.main(
                            [
                                "--transcripts-dir",
                                str(transcripts_dir),
                                "--questions-dir",
                                str(questions_dir),
                                "--source-name",
                                "local-json",
                                *extra_args,
                            ]
                        )

                self.assertEqual(exit_code, 0)
                refresh.assert_not_called()

    def test_main_returns_nonzero_when_recommendation_refresh_fails_after_write(self) -> None:
        with tempfile.TemporaryDirectory() as tmp_dir:
            root = Path(tmp_dir)
            transcripts_dir = root / "transcripts"
            questions_dir = root / "questions"
            transcripts_dir.mkdir()
            questions_dir.mkdir()
            _write_transcript_file(transcripts_dir / "demo-clip1.json", coarse_unit_id=7)
            (questions_dir / "demo-clip1.json").write_text(
                json.dumps(_question_payload(7, 0, 0)),
                encoding="utf-8",
            )

            fake_repo = _FakeMainRepository()
            with (
                mock.patch.object(ingest_main, "load_database_url", return_value="postgresql://unused"),
                mock.patch.object(ingest_main, "CatalogRepository", return_value=fake_repo),
                mock.patch.object(
                    ingest_main,
                    "run_recommendation_refresh",
                    side_effect=CatalogIngestError(
                        code="recommendation_refresh_failed",
                        stage="recommendation_refresh",
                        message="refresh failed",
                    ),
                ),
            ):
                exit_code = ingest_main.main(
                    [
                        "--transcripts-dir",
                        str(transcripts_dir),
                        "--questions-dir",
                        str(questions_dir),
                        "--source-name",
                        "local-json",
                    ]
                )

        self.assertEqual(exit_code, 1)
        self.assertEqual(fake_repo.persist_count, 1)


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
        video_object_path="https://storage.googleapis.com/videos2077/test-video/portrait_videos/parent-clip1.mp4",
        thumbnail_url=None,
        publish_at=None,
        transcript_object_path="transcript.json",
        transcript_checksum="checksum",
        transcript_format_version=1,
        source_name="test",
        source_file_path=Path("parent-clip1.json"),
        expected_transcript_filename="parent-clip1.json",
        transcript_file_path=Path("parent-clip1.json"),
        clip_metadata=ClipMetadata(
            clip_id=1,
            start_index=0,
            end_index=1,
            start_time=0,
            end_time=900,
            buffered_start_time=0,
            buffered_end_time=1000,
            duration_time=1000,
            reasoning=None,
            engagement={
                "drama": 5,
                "humor": 8,
                "payoff": 8,
                "standalone": 7,
                "reasoning": "demo score",
            },
        ),
        transcript_sentences=sentences,
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
            refs=selected_refs
            or (
                _selected_ref(
                        coarse_unit_id=sentences[0].tokens[0].semantic_element.coarse_id,
                        sentence_index=sentences[0].index,
                        token_index=sentences[0].tokens[0].index,
                    ),
            ),
        ),
        raw_question_payload={"source": {"model": "test-source"}, "questions": []},
    )


def _write_transcript_file(path: Path, coarse_unit_id: int) -> None:
    path.write_text(
        json.dumps(
            {
                "clip_id": 1,
                "title": "demo title",
                "description": "demo description",
                "engagement": {},
                "start_index": 0,
                "end_index": 0,
                "start_time": 100,
                "end_time": 250,
                "buffered_start_time": 100,
                "buffered_end_time": 300,
                "duration_time": 200,
                "reasoning": "demo",
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
                    "target_text": "demo",
                    "sentence_index": sentence_index,
                    "token_index": token_index,
                    "scores": {
                        "visual_context": 3,
                        "context_clarity": 8,
                        "learning_value": 9,
                        "representative_salience": 8,
                    },
                    "candidate_score": 8.35,
                    "question_reject_reason": None,
                    "selection_reason": "clear context",
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


def _selected_ref(coarse_unit_id: int, sentence_index: int, token_index: int) -> SelectedCoarseUnitRef:
    return SelectedCoarseUnitRef(
        coarse_unit_id=coarse_unit_id,
        target_text="token",
        sentence_index=sentence_index,
        token_index=token_index,
        scores={
            "visual_context": 3,
            "context_clarity": 8,
            "learning_value": 9,
            "representative_salience": 8,
        },
        candidate_score=8.35,
        question_reject_reason=None,
        selection_reason="clear context",
    )


class _FakeCursor:
    def __init__(self) -> None:
        self.execute_calls: list[tuple[str, tuple[object, ...]]] = []
        self.executemany_calls: list[tuple[str, list[tuple[object, ...]]]] = []
        self.fetchone_result: dict[str, object] | None = {"video_id": "video-1"}

    def execute(self, sql: str, params: tuple[object, ...]) -> None:
        self.execute_calls.append((sql, params))

    def executemany(self, sql: str, params: list[tuple[object, ...]]) -> None:
        self.executemany_calls.append((sql, params))

    def fetchone(self) -> dict[str, object] | None:
        return self.fetchone_result


class _FakeMainRepository:
    def __init__(self) -> None:
        self.persist_count = 0
        self.closed = False

    def load_known_coarse_unit_ids(self) -> set[int]:
        return {7}

    def load_existing_clip_states(self, source_clip_keys: list[str]) -> dict[str, object]:
        return {}

    def persist_clip(self, **kwargs: object) -> str:
        self.persist_count += 1
        return "11111111-1111-4111-8111-111111111111"

    def write_terminal_records(self, records: list[object]) -> None:
        return None

    def close(self) -> None:
        self.closed = True


if __name__ == "__main__":
    unittest.main()
