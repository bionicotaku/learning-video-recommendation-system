from __future__ import annotations

import json
import tempfile
import unittest
from pathlib import Path

from scripts.unit_collection_ingest.ingest import (
    CollectionMetadata,
    WordbookEntry,
    build_member_rows,
    compute_collection_counts,
    load_or_create_metadata,
    parse_wordbook_file,
    write_match_file_if_missing,
)


class UnitCollectionIngestTest(unittest.TestCase):
    def test_parse_wordbook_file_accepts_standard_json_array(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "demo.json"
            path.write_text(
                json.dumps(
                    [
                        {"wordRank": 2, "headWord": "beta", "content": {"ignored": True}},
                        {"wordRank": 1, "headWord": "alpha"},
                    ]
                ),
                encoding="utf-8",
            )

            parsed = parse_wordbook_file(path)

        self.assertEqual(parsed.slug, "demo")
        self.assertEqual(
            parsed.entries,
            (
                WordbookEntry(word_rank=2, head_word="beta"),
                WordbookEntry(word_rank=1, head_word="alpha"),
            ),
        )
        self.assertEqual(parsed.source_payload[0]["content"], {"ignored": True})

    def test_parse_wordbook_file_rejects_non_array_json(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "demo.json"
            path.write_text(json.dumps({"entries": []}), encoding="utf-8")

            with self.assertRaisesRegex(ValueError, "top-level JSON array"):
                parse_wordbook_file(path)

    def test_parse_wordbook_file_allows_duplicate_head_words_as_separate_entries(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "demo.json"
            path.write_text(
                json.dumps(
                    [
                        {"wordRank": 272, "headWord": "cliche"},
                        {"wordRank": 418, "headWord": "cliche"},
                    ]
                ),
                encoding="utf-8",
            )

            parsed = parse_wordbook_file(path)

        self.assertEqual(
            parsed.entries,
            (
                WordbookEntry(word_rank=272, head_word="cliche"),
                WordbookEntry(word_rank=418, head_word="cliche"),
            ),
        )

    def test_write_match_file_skips_existing_output(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source = root / "demo.json"
            source.write_text(json.dumps([{"wordRank": 1, "headWord": "alpha"}]), encoding="utf-8")
            wordbook = parse_wordbook_file(source)
            output = root / "_unit_collection_ingest" / "matches" / "demo.matches.json"
            output.parent.mkdir(parents=True)
            output.write_text('{"sentinel": true}', encoding="utf-8")

            wrote = write_match_file_if_missing(
                wordbook=wordbook,
                output_path=output,
                matches_by_label={"alpha": [101]},
            )

            self.assertFalse(wrote)
            self.assertEqual(json.loads(output.read_text(encoding="utf-8")), {"sentinel": True})

    def test_load_or_create_metadata_preserves_existing_values_and_adds_missing(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "collection_metadata.json"
            path.write_text(
                json.dumps(
                    {
                        "collections": {
                            "demo": {
                                "name": "Custom",
                                "description": "Keep me",
                                "internal_description": "Curated",
                            }
                        }
                    }
                ),
                encoding="utf-8",
            )

            metadata = load_or_create_metadata(
                path,
                slugs=("demo", "new-book"),
            )

        self.assertEqual(metadata["demo"].name, "Custom")
        self.assertEqual(metadata["demo"].description, "Keep me")
        self.assertEqual(metadata["demo"].internal_description, "Curated")
        self.assertEqual(
            metadata["new-book"],
            CollectionMetadata(
                name="new-book",
                description="",
                internal_description="",
            ),
        )

    def test_build_member_rows_dedupes_coarse_units_with_min_word_rank(self) -> None:
        rows = build_member_rows(
            [
                {"word_rank": 5, "head_word": "later", "coarse_unit_ids": [10, 11]},
                {"word_rank": 2, "head_word": "earlier", "coarse_unit_ids": [10]},
                {"word_rank": 9, "head_word": "none", "coarse_unit_ids": []},
            ]
        )

        self.assertEqual(
            rows,
            [
                (10, 2, 0),
                (11, 5, 0),
            ],
        )

    def test_compute_collection_counts_uses_matched_distinct_head_words_and_member_rows(self) -> None:
        entries = [
            {"word_rank": 1, "head_word": "alpha", "coarse_unit_ids": [10, 11]},
            {"word_rank": 2, "head_word": "alpha", "coarse_unit_ids": [10, 11]},
            {"word_rank": 3, "head_word": "beta", "coarse_unit_ids": []},
            {"word_rank": 4, "head_word": "gamma", "coarse_unit_ids": [12]},
        ]
        member_rows = build_member_rows(entries)

        counts = compute_collection_counts(entries=entries, member_rows=member_rows)

        self.assertEqual(counts.word_unit_count, 2)
        self.assertEqual(counts.coarse_unit_count, 3)


if __name__ == "__main__":
    unittest.main()
