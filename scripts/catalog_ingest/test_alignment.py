from __future__ import annotations

import unittest
from pathlib import Path

from scripts.catalog_ingest.index_builder import build_normalized_clip_data
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
    def test_unit_index_uses_structured_refs_sentence_dedup_and_limit_five(self) -> None:
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
                    text=f"sentence-{index}",
                    start_ms=index * 100,
                    end_ms=index * 100 + 90,
                    explanation=None,
                )
                for index in range(6)
            ),
            spans=(
                VideoSemanticSpanRow(2, 3, "two-late", 230, 240, None, 42, "two", "two"),
                VideoSemanticSpanRow(0, 4, "zero-late", 40, 50, None, 42, "zero", "zero"),
                VideoSemanticSpanRow(5, 1, "five-keep", 510, 520, None, 42, "five", "five"),
                VideoSemanticSpanRow(1, 7, "one-drop", 170, 180, None, 42, "one", "one"),
                VideoSemanticSpanRow(4, 2, "four-keep", 420, 430, None, 42, "four", "four"),
                VideoSemanticSpanRow(3, 0, "three-keep", 300, 310, None, 42, "three", "three"),
                VideoSemanticSpanRow(1, 2, "one-keep", 120, 130, None, 42, "one", "one"),
                VideoSemanticSpanRow(0, 1, "zero-keep", 10, 20, None, 42, "zero", "zero"),
                VideoSemanticSpanRow(2, 1, "two-keep", 210, 220, None, 42, "two", "two"),
            ),
        )

        normalized = build_normalized_clip_data(clip_input, core_rows)
        unit_index = normalized.unit_indexes[0]

        self.assertEqual(unit_index.sentence_indexes, (0, 1, 2, 3, 4, 5))
        self.assertEqual(
            tuple((ref.sentence_index, ref.span_index) for ref in unit_index.evidence_span_refs),
            ((0, 1), (1, 2), (2, 1), (3, 0), (4, 2)),
        )
        self.assertEqual(unit_index.sample_surface_forms, ("zero-keep", "one-keep", "two-keep", "three-keep", "four-keep"))

    def test_token_outside_sentence_stays_warning_for_assemblyai_edge_case(self) -> None:
        clip_input = _build_clip_input(
            transcript_sentences=(
                TranscriptSentence(
                    index=0,
                    text="edge case",
                    explanation=None,
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
                                base_form="edge",
                                dictionary_text="edge",
                                coarse_id=7,
                                reason=None,
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
            explanation=None,
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
                        base_form="default",
                        dictionary_text="default",
                        coarse_id=1,
                        reason=None,
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
