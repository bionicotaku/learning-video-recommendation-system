from __future__ import annotations

import argparse
from pathlib import Path

if __package__ in (None, ""):
    import sys

    sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

    from scripts.unit_collection_ingest.ingest import (
        MatchStageResult,
        WriteStageResult,
        build_member_rows,
        compute_collection_counts,
        discover_wordbook_files,
        load_match_payload,
        load_or_create_metadata,
        match_file_path,
        metadata_path,
        parse_wordbook_file,
        verify_match_matches_source,
        write_match_file_if_missing,
    )
    from scripts.unit_collection_ingest.repository import UnitCollectionRepository, load_database_url
else:
    from .ingest import (
        MatchStageResult,
        WriteStageResult,
        build_member_rows,
        compute_collection_counts,
        discover_wordbook_files,
        load_match_payload,
        load_or_create_metadata,
        match_file_path,
        metadata_path,
        parse_wordbook_file,
        verify_match_matches_source,
        write_match_file_if_missing,
    )
    from .repository import UnitCollectionRepository, load_database_url


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="将本地标准 JSON 词书导入 semantic.unit_collections。"
    )
    subparsers = parser.add_subparsers(dest="command", required=True)

    for command, help_text in (
        ("match", "读取所有 JSON，严格匹配 semantic.coarse_unit.label 并生成 match 文件。"),
        ("write", "根据原始 JSON、metadata 和 match 文件写入 unit collection 表。"),
        ("all", "先对所有文件执行 match，再对所有文件执行 write。"),
    ):
        subparser = subparsers.add_parser(command, help=help_text)
        subparser.add_argument("--input-dir", required=True, help="词书 JSON 文件目录")
        subparser.add_argument("--limit", type=int, default=None, help="最多处理多少个 JSON 文件")

    return parser


def main(argv: list[str] | None = None) -> int:
    args = build_parser().parse_args(argv)
    input_dir = Path(args.input_dir)
    if not input_dir.exists() or not input_dir.is_dir():
        print(f"[failed] input dir not found: {input_dir}")
        return 1

    files = discover_wordbook_files(input_dir)
    if args.limit is not None:
        files = files[: args.limit]
    if not files:
        print("未找到词书 JSON 文件。")
        return 0

    repository = UnitCollectionRepository(load_database_url())
    try:
        if args.command in ("match", "all"):
            match_results = run_match_stage(input_dir=input_dir, files=files, repository=repository)
            _print_match_summary(match_results)
        if args.command in ("write", "all"):
            write_results = run_write_stage(input_dir=input_dir, files=files, repository=repository)
            _print_write_summary(write_results)
    finally:
        repository.close()

    return 0


def run_match_stage(
    *,
    input_dir: Path,
    files: tuple[Path, ...],
    repository: UnitCollectionRepository,
) -> list[MatchStageResult]:
    wordbooks = [parse_wordbook_file(path) for path in files]
    load_or_create_metadata(metadata_path(input_dir), (wordbook.slug for wordbook in wordbooks))

    pending = [
        wordbook
        for wordbook in wordbooks
        if not match_file_path(input_dir, wordbook.slug).exists()
    ]
    labels = [entry.head_word for wordbook in pending for entry in wordbook.entries]
    matches_by_label = repository.load_matches_by_label(labels)

    results: list[MatchStageResult] = []
    for wordbook in wordbooks:
        output_path = match_file_path(input_dir, wordbook.slug)
        wrote = write_match_file_if_missing(
            wordbook=wordbook,
            output_path=output_path,
            matches_by_label=matches_by_label,
        )
        if not wrote:
            results.append(
                MatchStageResult(
                    source_file=wordbook.source_path.name,
                    slug=wordbook.slug,
                    status="skipped_existing",
                    match_file=output_path,
                    word_unit_count=len(wordbook.entries),
                )
            )
            continue
        payload = load_match_payload(output_path)
        results.append(
            MatchStageResult(
                source_file=wordbook.source_path.name,
                slug=wordbook.slug,
                status="created",
                match_file=output_path,
                word_unit_count=int(payload["word_unit_count"]),
                matched_head_word_count=int(payload["matched_head_word_count"]),
                unmatched_head_word_count=int(payload["unmatched_head_word_count"]),
                coarse_unit_count=int(payload["coarse_unit_count"]),
            )
        )
    return results


def run_write_stage(
    *,
    input_dir: Path,
    files: tuple[Path, ...],
    repository: UnitCollectionRepository,
) -> list[WriteStageResult]:
    wordbooks = [parse_wordbook_file(path) for path in files]
    metadata = load_or_create_metadata(metadata_path(input_dir), (wordbook.slug for wordbook in wordbooks))

    results: list[WriteStageResult] = []
    for wordbook in wordbooks:
        output_path = match_file_path(input_dir, wordbook.slug)
        if not output_path.exists():
            raise FileNotFoundError(f"{wordbook.source_path.name}: missing match file {output_path}")
        match_payload = load_match_payload(output_path)
        verify_match_matches_source(match_payload, wordbook)

        entries = match_payload["entries"]
        member_rows = build_member_rows(entries)
        counts = compute_collection_counts(entries=entries, member_rows=member_rows)
        raw_match_count = int(match_payload["coarse_unit_count"])
        collection_id = repository.upsert_collection_with_members(
            slug=wordbook.slug,
            metadata=metadata[wordbook.slug],
            source_payload=wordbook.source_payload,
            word_unit_count=counts.word_unit_count,
            coarse_unit_count=counts.coarse_unit_count,
            member_rows=member_rows,
        )
        results.append(
            WriteStageResult(
                source_file=wordbook.source_path.name,
                slug=wordbook.slug,
                collection_id=collection_id,
                word_unit_count=counts.word_unit_count,
                coarse_unit_count=counts.coarse_unit_count,
                raw_match_count=raw_match_count,
                unique_member_count=len(member_rows),
                duplicate_member_count=raw_match_count - len(member_rows),
            )
        )
    return results


def _print_match_summary(results: list[MatchStageResult]) -> None:
    for result in results:
        if result.status == "skipped_existing":
            print(
                f"[match skipped] slug={result.slug} words={result.word_unit_count} "
                f"file={result.match_file}",
                flush=True,
            )
        else:
            print(
                f"[match created] slug={result.slug} words={result.word_unit_count} "
                f"matched={result.matched_head_word_count} unmatched={result.unmatched_head_word_count} "
                f"coarse_units={result.coarse_unit_count}",
                flush=True,
            )
    print(
        "[match summary] "
        f"created={sum(1 for result in results if result.status == 'created')} "
        f"skipped={sum(1 for result in results if result.status == 'skipped_existing')}",
        flush=True,
    )


def _print_write_summary(results: list[WriteStageResult]) -> None:
    for result in results:
        print(
            f"[write succeeded] slug={result.slug} collection_id={result.collection_id} "
            f"matched_distinct_words={result.word_unit_count} collection_units={result.coarse_unit_count} "
            f"raw_matches={result.raw_match_count} "
            f"members={result.unique_member_count} duplicates={result.duplicate_member_count}",
            flush=True,
        )
    print(f"[write summary] succeeded={len(results)}", flush=True)


if __name__ == "__main__":
    raise SystemExit(main())
