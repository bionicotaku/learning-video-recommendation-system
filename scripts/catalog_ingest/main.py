from __future__ import annotations

import argparse
from collections import Counter
from datetime import datetime, timezone
from pathlib import Path

if __package__ in (None, ""):
    import sys

    # 允许直接用 `python scripts/catalog_ingest/main.py` 运行。
    # 这里把项目根目录压入 sys.path，然后改走绝对导入路径。
    sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

    from scripts.catalog_ingest.index_builder import build_normalized_clip_data
    from scripts.catalog_ingest.manifest_loader import load_clip_inputs
    from scripts.catalog_ingest.models import CatalogIngestError, ClipProcessResult, ExistingClipState, IngestionRecordPayload, LoadedClipInput
    from scripts.catalog_ingest.normalizer import normalize_clip_input
    from scripts.catalog_ingest.repository import CatalogRepository, load_database_url
    from scripts.catalog_ingest.validator import ValidationWarning, validate_loaded_clip
else:
    from .index_builder import build_normalized_clip_data
    from .manifest_loader import load_clip_inputs
    from .models import CatalogIngestError, ClipProcessResult, ExistingClipState, IngestionRecordPayload, LoadedClipInput
    from .normalizer import normalize_clip_input
    from .repository import CatalogRepository, load_database_url
    from .validator import ValidationWarning, validate_loaded_clip


BATCH_TERMINAL_AUDIT_SIZE = 200


def build_parser() -> argparse.ArgumentParser:
    """构建 CLI 参数解析器。"""

    parser = argparse.ArgumentParser(
        description="将本地父视频切片描述 JSON 和 transcript JSON 导入 catalog 数据库。"
    )
    parser.add_argument("--parents-dir", required=True, help="父视频切片描述文件目录")
    parser.add_argument("--transcripts-dir", required=True, help="clip transcript 文件目录")
    parser.add_argument("--source-name", default="local-json", help="写入审计记录时使用的来源名称")
    parser.add_argument("--limit", type=int, default=None, help="最多处理多少个 clip")
    parser.add_argument("--dry-run", action="store_true", help="只做读取、校验和归一化，不写数据库")
    parser.add_argument("--clip-key", default=None, help="只处理指定的 source_clip_key")
    parser.add_argument(
        "--time-tolerance-ms",
        type=int,
        default=0,
        help="transcript 时间轴允许偏离 buffered 区间的毫秒数，默认 0",
    )
    return parser


def main(argv: list[str] | None = None) -> int:
    """脚本主入口。

    main 只负责总编排：
    - 读取参数
    - 初始化 repository
    - 调用 loader / validator / normalizer / index_builder / repository
    - 打印逐条结果与最终汇总
    """

    parser = build_parser()
    args = parser.parse_args(argv)

    repository: CatalogRepository | None = None
    pending_terminal_records: list[IngestionRecordPayload] = []
    fatal_error: CatalogIngestError | None = None
    exit_code = 0
    try:
        database_url = load_database_url()
        repository = CatalogRepository(database_url)

        clip_inputs = load_clip_inputs(
            parents_dir=Path(args.parents_dir),
            transcripts_dir=Path(args.transcripts_dir),
            source_name=args.source_name,
            clip_key=args.clip_key,
            limit=args.limit,
        )

        if not clip_inputs:
            print("未找到任何待处理的 clip。")
            return 0

        known_coarse_unit_ids = repository.load_known_coarse_unit_ids()
        existing_states_by_key = repository.load_existing_clip_states(
            [
                clip_input.source_clip_key
                for clip_input in clip_inputs
                if clip_input.skip_reason_code is None
            ]
        )
        results: list[ClipProcessResult] = []

        for clip_input in clip_inputs:
            result = _process_single_clip(
                clip_input=clip_input,
                repository=repository,
                existing_state=existing_states_by_key.get(clip_input.source_clip_key),
                known_coarse_unit_ids=known_coarse_unit_ids,
                dry_run=args.dry_run,
                time_tolerance_ms=args.time_tolerance_ms,
            )
            results.append(result)
            if not args.dry_run and result.terminal_record is not None:
                pending_terminal_records.append(result.terminal_record)
                if len(pending_terminal_records) >= BATCH_TERMINAL_AUDIT_SIZE:
                    repository.write_terminal_records(pending_terminal_records)
                    pending_terminal_records.clear()
            _print_single_result(result)

        if not args.dry_run and pending_terminal_records:
            repository.write_terminal_records(pending_terminal_records)
            pending_terminal_records.clear()

        _print_summary(results, dry_run=args.dry_run)
        exit_code = 1 if any(result.status == "failed" for result in results) else 0
    except CatalogIngestError as error:
        fatal_error = error
        exit_code = 1
    finally:
        # 批量终态审计必须在退出前做最后一次 flush。
        # 这样即使主循环中途因异常提前结束，也尽量不丢已经判定完成的
        # skipped / failed 记录。
        if repository is not None and not args.dry_run and pending_terminal_records:
            try:
                repository.write_terminal_records(pending_terminal_records)
                pending_terminal_records.clear()
            except CatalogIngestError as flush_error:
                if fatal_error is None:
                    fatal_error = flush_error
                else:
                    print(
                        f"[failed] code={flush_error.code} "
                        f"stage={flush_error.stage} message={flush_error.message}"
                    )
        if repository is not None:
            repository.close()

    if fatal_error is not None:
        print(f"[failed] code={fatal_error.code} stage={fatal_error.stage} message={fatal_error.message}")
        return 1
    return exit_code


def _process_single_clip(
    clip_input: LoadedClipInput,
    repository: CatalogRepository,
    existing_state: ExistingClipState | None,
    known_coarse_unit_ids: set[int],
    dry_run: bool,
    time_tolerance_ms: int,
) -> ClipProcessResult:
    """处理单个 clip 的完整链路。"""

    if clip_input.skip_reason_code is not None:
        terminal_at = _utcnow()
        payload = IngestionRecordPayload(
            source_clip_key=clip_input.source_clip_key,
            video_id=None,
            source_name=clip_input.source_name,
            status="skipped",
            warning_codes=tuple(),
            error_code=clip_input.skip_reason_code,
            error_message=clip_input.skip_reason_message,
            context=clip_input.context,
            started_at=terminal_at,
            finished_at=terminal_at,
        )
        return ClipProcessResult(
            source_clip_key=clip_input.source_clip_key,
            status="skipped",
            video_id=None,
            warning_codes=tuple(),
            error=None,
            terminal_record=None if dry_run else payload,
        )

    try:
        validation_warnings = validate_loaded_clip(
            clip_input=clip_input,
            known_coarse_unit_ids=known_coarse_unit_ids,
            time_tolerance_ms=time_tolerance_ms,
        )
        warning_codes = _collect_warning_codes(validation_warnings)
        warning_context = _build_warning_context(validation_warnings)

        if _should_skip_unchanged_clip(existing_state, clip_input):
            terminal_at = _utcnow()
            payload = IngestionRecordPayload(
                source_clip_key=clip_input.source_clip_key,
                video_id=existing_state.video_id if existing_state else None,
                source_name=clip_input.source_name,
                status="skipped",
                warning_codes=warning_codes,
                error_code=None,
                error_message=None,
                context={**clip_input.context, "skip_reason": "unchanged", **warning_context},
                started_at=terminal_at,
                finished_at=terminal_at,
            )
            return ClipProcessResult(
                source_clip_key=clip_input.source_clip_key,
                status="skipped",
                video_id=existing_state.video_id if existing_state else None,
                warning_codes=warning_codes,
                error=None,
                terminal_record=None if dry_run else payload,
            )

        normalized_core = normalize_clip_input(clip_input)
        normalized_clip = build_normalized_clip_data(clip_input, normalized_core)

        if dry_run:
            return ClipProcessResult(
                source_clip_key=clip_input.source_clip_key,
                status="dry_run_would_write",
                video_id=existing_state.video_id if existing_state else None,
                warning_codes=warning_codes,
                error=None,
            )

        video_id = repository.persist_clip(
            normalized_data=normalized_clip,
            source_name=clip_input.source_name,
            context={**clip_input.context, **warning_context},
            warning_codes=warning_codes,
        )
        return ClipProcessResult(
            source_clip_key=clip_input.source_clip_key,
            status="succeeded",
            video_id=video_id,
            warning_codes=warning_codes,
            error=None,
            terminal_record=None,
        )
    except CatalogIngestError as error:
        terminal_at = _utcnow()
        payload = IngestionRecordPayload(
            source_clip_key=clip_input.source_clip_key,
            video_id=None,
            source_name=clip_input.source_name,
            status="failed",
            warning_codes=tuple(),
            error_code=error.code,
            error_message=error.message,
            context=error.context,
            started_at=terminal_at,
            finished_at=terminal_at,
        )
        return ClipProcessResult(
            source_clip_key=clip_input.source_clip_key,
            status="failed",
            video_id=None,
            warning_codes=tuple(),
            error=error,
            terminal_record=None if dry_run else payload,
        )


def _utcnow() -> datetime:
    """返回带 UTC 时区的当前时间。"""

    return datetime.now(timezone.utc)


def _should_skip_unchanged_clip(existing_state, clip_input: LoadedClipInput) -> bool:
    """判断当前 clip 是否可直接 skipped。

    这里严格按 README 中的“无变化跳过”规则比较。
    只要 transcript checksum、HLS 路径和关键元数据都没变，就不需要重写四张内容表。
    """

    if existing_state is None:
        return False
    if existing_state.transcript_checksum != clip_input.transcript_checksum:
        return False
    if existing_state.hls_master_playlist_path != clip_input.hls_master_playlist_path:
        return False

    return (
        existing_state.parent_video_name == clip_input.parent_video_name
        and existing_state.parent_video_slug == clip_input.parent_video_slug
        and existing_state.clip_seq == clip_input.clip_seq
        and existing_state.source_start_ms == clip_input.source_start_ms
        and existing_state.source_end_ms == clip_input.source_end_ms
        and existing_state.title == clip_input.title
        and existing_state.description == clip_input.description
        and existing_state.clip_reason == clip_input.clip_reason
        and existing_state.language == clip_input.language
        and existing_state.duration_ms == clip_input.duration_ms
        and existing_state.thumbnail_url == clip_input.thumbnail_url
        and existing_state.publish_at == clip_input.publish_at
    )


def _collect_warning_codes(warnings: tuple[ValidationWarning, ...]) -> tuple[str, ...]:
    """提取去重后的 warning code，供命令行和审计表复用。"""

    seen: set[str] = set()
    ordered_codes: list[str] = []
    for warning in warnings:
        if warning.code in seen:
            continue
        seen.add(warning.code)
        ordered_codes.append(warning.code)
    return tuple(ordered_codes)


def _build_warning_context(warnings: tuple[ValidationWarning, ...]) -> dict[str, object]:
    """把 warning 细节压入审计上下文。

    审计表当前只有 warning_codes，没有单独的 warning_message 字段。
    因此这里把详细位置和时间信息塞进 context，避免 warning 只剩一个裸 code。
    """

    if not warnings:
        return {}

    return {
        "warnings": [
            {
                "code": warning.code,
                "message": warning.message,
                **warning.context,
            }
            for warning in warnings
        ]
    }


def _print_single_result(result: ClipProcessResult) -> None:
    """打印单条执行结果，方便在命令行里追踪进度。"""

    if result.error is None:
        if result.warning_codes:
            print(
                f"[{result.status}] {result.source_clip_key} "
                f"warnings={','.join(result.warning_codes)}"
            )
        else:
            print(f"[{result.status}] {result.source_clip_key}")
        return

    print(
        f"[failed] {result.source_clip_key} "
        f"code={result.error.code} stage={result.error.stage} message={result.error.message}"
    )


def _print_summary(results: list[ClipProcessResult], dry_run: bool) -> None:
    """打印最终汇总。"""

    status_counter = Counter(result.status for result in results)
    print("")
    print("执行汇总：")
    print(f"- 总数: {len(results)}")
    for status, count in sorted(status_counter.items()):
        print(f"- {status}: {count}")
    if dry_run:
        print("- 当前为 dry-run，本次未写数据库。")


if __name__ == "__main__":
    raise SystemExit(main())
