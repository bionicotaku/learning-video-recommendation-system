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
    TranscriptSemanticElement,
    TranscriptSentence,
    TranscriptToken,
    VideoRow,
    VideoSemanticSpanRow,
    VideoTranscriptSentenceRow,
)
from scripts.catalog_ingest.validator import validate_loaded_clip


class CatalogIngestAlignmentTest(unittest.TestCase):
    def test_load_clip_inputs_accepts_extra_semantic_display_fields_without_db_rows(self) -> None:
        with tempfile.TemporaryDirectory() as tmp_dir:
            root = Path(tmp_dir)
            parents_dir = root / "parents"
            transcripts_dir = root / "transcripts"
            parents_dir.mkdir()
            transcripts_dir.mkdir()

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

            loaded = load_clip_inputs(parents_dir=parents_dir, transcripts_dir=transcripts_dir)

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

    def test_unit_index_uses_structured_refs_sentence_dedup_limit_five_and_no_surface_forms(self) -> None:
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
        self.assertEqual(
            tuple((ref.sentence_index, ref.span_index) for ref in unit_index.evidence_span_refs),
            ((0, 1), (1, 2), (2, 1), (3, 0), (4, 2)),
        )
        self.assertFalse(hasattr(unit_index, "sample_surface_forms"))

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
    )


if __name__ == "__main__":
    unittest.main()
