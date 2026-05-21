from __future__ import annotations

import hashlib
import json
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Iterable


OUTPUT_DIR_NAME = "_unit_collection_ingest"


@dataclass(frozen=True, slots=True)
class WordbookEntry:
    word_rank: int
    head_word: str


@dataclass(frozen=True, slots=True)
class ParsedWordbook:
    source_path: Path
    slug: str
    source_sha256: str
    source_payload: list[Any]
    entries: tuple[WordbookEntry, ...]


@dataclass(frozen=True, slots=True)
class CollectionMetadata:
    name: str
    description: str
    internal_description: str


@dataclass(frozen=True, slots=True)
class MatchStageResult:
    source_file: str
    slug: str
    status: str
    match_file: Path
    word_unit_count: int
    matched_head_word_count: int | None = None
    unmatched_head_word_count: int | None = None
    coarse_unit_count: int | None = None


@dataclass(frozen=True, slots=True)
class WriteStageResult:
    source_file: str
    slug: str
    collection_id: str
    word_unit_count: int
    coarse_unit_count: int
    raw_match_count: int
    unique_member_count: int
    duplicate_member_count: int


@dataclass(frozen=True, slots=True)
class CollectionCounts:
    word_unit_count: int
    coarse_unit_count: int


def discover_wordbook_files(input_dir: Path) -> tuple[Path, ...]:
    """Return top-level source JSON files, excluding generated ingest artifacts."""

    return tuple(
        sorted(
            path
            for path in input_dir.glob("*.json")
            if path.is_file() and path.parent.name != OUTPUT_DIR_NAME
        )
    )


def parse_wordbook_file(path: Path) -> ParsedWordbook:
    """Parse one standard JSON wordbook file.

    The current contract is strict: the file must be a top-level JSON array and
    every item must contain integer `wordRank` plus non-empty string `headWord`.
    Other fields are preserved in `source_payload` for internal storage.
    """

    raw_bytes = path.read_bytes()
    try:
        payload = json.loads(raw_bytes.decode("utf-8"))
    except json.JSONDecodeError as exc:
        raise ValueError(f"{path.name}: invalid JSON: {exc}") from exc

    if not isinstance(payload, list):
        raise ValueError(f"{path.name}: expected top-level JSON array")

    entries: list[WordbookEntry] = []
    for index, item in enumerate(payload):
        if not isinstance(item, dict):
            raise ValueError(f"{path.name}: item {index} must be an object")

        word_rank = item.get("wordRank")
        if not isinstance(word_rank, int):
            raise ValueError(f"{path.name}: item {index} wordRank must be an integer")

        raw_head_word = item.get("headWord")
        if not isinstance(raw_head_word, str) or not raw_head_word.strip():
            raise ValueError(f"{path.name}: item {index} headWord must be a non-empty string")

        head_word = raw_head_word.strip()
        entries.append(WordbookEntry(word_rank=word_rank, head_word=head_word))

    return ParsedWordbook(
        source_path=path,
        slug=path.stem,
        source_sha256=hashlib.sha256(raw_bytes).hexdigest(),
        source_payload=payload,
        entries=tuple(entries),
    )


def generated_root(input_dir: Path) -> Path:
    return input_dir / OUTPUT_DIR_NAME


def metadata_path(input_dir: Path) -> Path:
    return generated_root(input_dir) / "collection_metadata.json"


def match_file_path(input_dir: Path, slug: str) -> Path:
    return generated_root(input_dir) / "matches" / f"{slug}.matches.json"


def load_or_create_metadata(path: Path, slugs: Iterable[str]) -> dict[str, CollectionMetadata]:
    """Load metadata map, adding default entries for new slugs without overwriting."""

    existing: dict[str, CollectionMetadata] = {}
    if path.exists():
        data = json.loads(path.read_text(encoding="utf-8"))
        collections = data.get("collections") if isinstance(data, dict) else None
        if not isinstance(collections, dict):
            raise ValueError(f"{path}: expected object with collections map")
        for slug, value in collections.items():
            if not isinstance(value, dict):
                raise ValueError(f"{path}: metadata for {slug!r} must be an object")
            existing[slug] = CollectionMetadata(
                name=str(value.get("name") or slug),
                description=str(value.get("description") or ""),
                internal_description=str(value.get("internal_description") or ""),
            )

    changed = False
    for slug in sorted(set(slugs)):
        if slug not in existing:
            existing[slug] = CollectionMetadata(
                name=slug,
                description="",
                internal_description="",
            )
            changed = True

    if changed or not path.exists():
        path.parent.mkdir(parents=True, exist_ok=True)
        payload = {
            "collections": {
                slug: {
                    "name": metadata.name,
                    "description": metadata.description,
                    "internal_description": metadata.internal_description,
                }
                for slug, metadata in sorted(existing.items())
            }
        }
        path.write_text(
            json.dumps(payload, ensure_ascii=False, indent=2) + "\n",
            encoding="utf-8",
        )

    return existing


def write_match_file_if_missing(
    *,
    wordbook: ParsedWordbook,
    output_path: Path,
    matches_by_label: dict[str, list[int]],
) -> bool:
    """Write match output unless it already exists.

    Return True when a file was created, False when the existing output was
    intentionally kept for repeat-run idempotency.
    """

    if output_path.exists():
        return False

    output_path.parent.mkdir(parents=True, exist_ok=True)
    entries: list[dict[str, Any]] = []
    unmatched: list[str] = []
    coarse_unit_count = 0
    for entry in wordbook.entries:
        coarse_unit_ids = sorted(matches_by_label.get(entry.head_word, []))
        if not coarse_unit_ids:
            unmatched.append(entry.head_word)
        coarse_unit_count += len(coarse_unit_ids)
        entries.append(
            {
                "word_rank": entry.word_rank,
                "head_word": entry.head_word,
                "coarse_unit_ids": coarse_unit_ids,
            }
        )

    payload = {
        "source_file": wordbook.source_path.name,
        "slug": wordbook.slug,
        "source_sha256": wordbook.source_sha256,
        "word_unit_count": len(wordbook.entries),
        "matched_head_word_count": len(wordbook.entries) - len(unmatched),
        "unmatched_head_word_count": len(unmatched),
        "coarse_unit_count": coarse_unit_count,
        "entries": entries,
        "unmatched_head_words": unmatched,
    }
    output_path.write_text(
        json.dumps(payload, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )
    return True


def load_match_payload(path: Path) -> dict[str, Any]:
    payload = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(payload, dict):
        raise ValueError(f"{path}: match payload must be an object")
    entries = payload.get("entries")
    if not isinstance(entries, list):
        raise ValueError(f"{path}: entries must be an array")
    return payload


def build_member_rows(entries: list[dict[str, Any]]) -> list[tuple[int, int, int]]:
    """Build unique member rows as `(coarse_unit_id, sort_order, target_priority)`.

    The table primary key is `(collection_id, coarse_unit_id)`, so duplicate
    coarse units from multiple headWords collapse to the earliest wordRank.
    """

    best_rank_by_unit: dict[int, int] = {}
    for entry in entries:
        word_rank = entry.get("word_rank")
        if not isinstance(word_rank, int):
            raise ValueError("match entry word_rank must be an integer")
        coarse_unit_ids = entry.get("coarse_unit_ids")
        if not isinstance(coarse_unit_ids, list):
            raise ValueError("match entry coarse_unit_ids must be an array")
        for coarse_unit_id in coarse_unit_ids:
            if not isinstance(coarse_unit_id, int):
                raise ValueError("coarse_unit_id must be an integer")
            current = best_rank_by_unit.get(coarse_unit_id)
            if current is None or word_rank < current:
                best_rank_by_unit[coarse_unit_id] = word_rank

    return [
        (coarse_unit_id, sort_order, 0)
        for coarse_unit_id, sort_order in sorted(
            best_rank_by_unit.items(),
            key=lambda item: (item[1], item[0]),
        )
    ]


def compute_collection_counts(
    *,
    entries: list[dict[str, Any]],
    member_rows: list[tuple[int, int, int]],
) -> CollectionCounts:
    matched_head_words: set[str] = set()
    for entry in entries:
        head_word = entry.get("head_word")
        coarse_unit_ids = entry.get("coarse_unit_ids")
        if not isinstance(head_word, str):
            raise ValueError("match entry head_word must be a string")
        if not isinstance(coarse_unit_ids, list):
            raise ValueError("match entry coarse_unit_ids must be an array")
        if coarse_unit_ids:
            matched_head_words.add(head_word)

    return CollectionCounts(
        word_unit_count=len(matched_head_words),
        coarse_unit_count=len(member_rows),
    )


def verify_match_matches_source(match_payload: dict[str, Any], wordbook: ParsedWordbook) -> None:
    if match_payload.get("source_file") != wordbook.source_path.name:
        raise ValueError(f"{wordbook.source_path.name}: match source_file does not match source")
    if match_payload.get("source_sha256") != wordbook.source_sha256:
        raise ValueError(f"{wordbook.source_path.name}: match source_sha256 is stale")
