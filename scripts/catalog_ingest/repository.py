from __future__ import annotations

import os
from datetime import datetime, timezone
from pathlib import Path
from uuid import uuid4

import psycopg
from psycopg.rows import dict_row
from psycopg.types.json import Jsonb

from .models import (
    CatalogIngestError,
    ExistingClipState,
    IngestionRecordPayload,
    NormalizedClipData,
)


def load_database_url(env_file: Path | None = None) -> str:
    """从环境变量或项目根目录 `.env` 中读取 DATABASE_URL。

    读取优先级：
    1. 进程环境变量中的 DATABASE_URL
    2. 项目根目录 `.env`
    """

    database_url = os.environ.get("DATABASE_URL")
    if database_url:
        return database_url

    env_path = env_file or Path(__file__).resolve().parents[2] / ".env"
    if env_path.exists():
        for raw_line in env_path.read_text(encoding="utf-8").splitlines():
            line = raw_line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            key, value = line.split("=", 1)
            if key.strip() == "DATABASE_URL":
                return value.strip().strip('"').strip("'")

    raise CatalogIngestError(
        code="db_connect_failed",
        stage="repository",
        message="未找到 DATABASE_URL，请检查环境变量或项目根目录 .env",
        context={"env_file": str(env_path)},
    )


class CatalogRepository:
    """封装 catalog 入库脚本的数据库访问。

    这里集中处理三件事：
    - 读数据库已有状态，用于 skip 判断
    - 读 semantic.coarse_unit，用于 validator
    - 执行单 clip 单事务写入和审计写入
    """

    def __init__(self, database_url: str) -> None:
        self._database_url = database_url

    def load_known_coarse_unit_ids(self) -> set[int]:
        """一次性加载所有 coarse unit id，避免 validator 逐 token 查库。"""

        sql = "select id from semantic.coarse_unit"
        try:
            with psycopg.connect(self._database_url) as connection:
                with connection.cursor() as cursor:
                    cursor.execute(sql)
                    return {row[0] for row in cursor.fetchall()}
        except Exception as exc:
            raise CatalogIngestError(
                code="db_connect_failed",
                stage="repository",
                message="加载 semantic.coarse_unit 失败",
                context={},
            ) from exc

    def get_existing_clip_state(self, source_clip_key: str) -> ExistingClipState | None:
        """读取数据库中已存在的 clip 快照，用于决定是否可以 skipped。"""

        sql = """
            select
              v.video_id,
              v.parent_video_name,
              v.parent_video_slug,
              v.clip_seq,
              v.source_start_ms,
              v.source_end_ms,
              v.title,
              v.description,
              v.clip_reason,
              v.language,
              v.duration_ms,
              v.hls_master_playlist_path,
              t.transcript_checksum
            from catalog.videos v
            left join catalog.video_transcripts t on t.video_id = v.video_id
            where v.source_clip_key = %s
        """
        try:
            with psycopg.connect(self._database_url, row_factory=dict_row) as connection:
                with connection.cursor() as cursor:
                    cursor.execute(sql, (source_clip_key,))
                    row = cursor.fetchone()
        except Exception as exc:
            raise CatalogIngestError(
                code="db_connect_failed",
                stage="repository",
                message="读取已有 clip 状态失败",
                context={"source_clip_key": source_clip_key},
            ) from exc

        if row is None:
            return None

        return ExistingClipState(
            video_id=str(row["video_id"]),
            parent_video_name=row["parent_video_name"],
            parent_video_slug=row["parent_video_slug"],
            clip_seq=row["clip_seq"],
            source_start_ms=row["source_start_ms"],
            source_end_ms=row["source_end_ms"],
            title=row["title"],
            description=row["description"],
            clip_reason=row["clip_reason"],
            language=row["language"],
            duration_ms=row["duration_ms"],
            hls_master_playlist_path=row["hls_master_playlist_path"],
            transcript_checksum=row["transcript_checksum"],
        )

    def persist_clip(
        self,
        normalized_data: NormalizedClipData,
        source_name: str | None,
        context: dict[str, object],
    ) -> str:
        """将完整 clip 数据以单事务方式写入数据库。"""

        started_at = _utcnow()
        ingestion_record_id = str(uuid4())

        try:
            with psycopg.connect(self._database_url, row_factory=dict_row) as connection:
                with connection.transaction():
                    with connection.cursor() as cursor:
                        self._insert_running_record(
                            cursor=cursor,
                            ingestion_record_id=ingestion_record_id,
                            payload=IngestionRecordPayload(
                                source_clip_key=normalized_data.video.source_clip_key,
                                video_id=None,
                                source_name=source_name,
                                status="running",
                                warning_codes=tuple(),
                                error_code=None,
                                error_message=None,
                                context=context,
                            ),
                            started_at=started_at,
                        )

                        video_id = self._upsert_video(cursor, normalized_data)
                        self._replace_transcript_related_rows(cursor, video_id, normalized_data)

                        cursor.execute(
                            """
                            update catalog.video_ingestion_records
                            set video_id = %s,
                                status = 'succeeded',
                                warning_codes = %s,
                                error_code = null,
                                error_message = null,
                                finished_at = %s
                            where ingestion_record_id = %s
                            """,
                            (video_id, [], _utcnow(), ingestion_record_id),
                        )
                        return video_id
        except CatalogIngestError:
            raise
        except Exception as exc:
            raise CatalogIngestError(
                code="db_write_failed",
                stage="repository",
                message="单 clip 数据库写入失败",
                context={"source_clip_key": normalized_data.video.source_clip_key},
            ) from exc

    def write_skipped_record(self, payload: IngestionRecordPayload) -> None:
        """写入 skipped 审计记录。"""

        self._write_terminal_record(payload)

    def write_failed_record(self, payload: IngestionRecordPayload) -> None:
        """写入 failed 审计记录。"""

        self._write_terminal_record(payload)

    def _write_terminal_record(self, payload: IngestionRecordPayload) -> None:
        """写入最终态审计记录。

        skipped / failed 都不需要业务写入事务，因此单独用一条 insert 即可。
        """

        started_at = _utcnow()
        finished_at = _utcnow()
        sql = """
            insert into catalog.video_ingestion_records (
              ingestion_record_id,
              source_clip_key,
              video_id,
              source_name,
              status,
              warning_codes,
              error_code,
              error_message,
              context,
              started_at,
              finished_at
            )
            values (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
        """
        try:
            with psycopg.connect(self._database_url) as connection:
                with connection.cursor() as cursor:
                    cursor.execute(
                        sql,
                        (
                            str(uuid4()),
                            payload.source_clip_key,
                            payload.video_id,
                            payload.source_name,
                            payload.status,
                            list(payload.warning_codes),
                            payload.error_code,
                            payload.error_message,
                            Jsonb(payload.context),
                            started_at,
                            finished_at,
                        ),
                    )
                connection.commit()
        except Exception as exc:
            raise CatalogIngestError(
                code="db_write_failed",
                stage="repository",
                message="写入审计记录失败",
                context={"source_clip_key": payload.source_clip_key, "status": payload.status},
            ) from exc

    def _insert_running_record(
        self,
        cursor: psycopg.Cursor,
        ingestion_record_id: str,
        payload: IngestionRecordPayload,
        started_at: datetime,
    ) -> None:
        """在事务开始时插入 running 审计记录。"""

        cursor.execute(
            """
            insert into catalog.video_ingestion_records (
              ingestion_record_id,
              source_clip_key,
              video_id,
              source_name,
              status,
              warning_codes,
              error_code,
              error_message,
              context,
              started_at
            )
            values (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
            """,
            (
                ingestion_record_id,
                payload.source_clip_key,
                payload.video_id,
                payload.source_name,
                payload.status,
                list(payload.warning_codes),
                payload.error_code,
                payload.error_message,
                Jsonb(payload.context),
                started_at,
            ),
        )

    def _upsert_video(self, cursor: psycopg.Cursor, normalized_data: NormalizedClipData) -> str:
        """按 source_clip_key 幂等 upsert catalog.videos，并返回稳定 video_id。"""

        video = normalized_data.video
        cursor.execute(
            """
            insert into catalog.videos (
              source_clip_key,
              parent_video_name,
              parent_video_slug,
              clip_seq,
              source_start_ms,
              source_end_ms,
              title,
              description,
              clip_reason,
              language,
              duration_ms,
              hls_master_playlist_path,
              thumbnail_url,
              status,
              visibility_status,
              publish_at
            )
            values (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
            on conflict (source_clip_key) do update
            set parent_video_name = excluded.parent_video_name,
                parent_video_slug = excluded.parent_video_slug,
                clip_seq = excluded.clip_seq,
                source_start_ms = excluded.source_start_ms,
                source_end_ms = excluded.source_end_ms,
                title = excluded.title,
                description = excluded.description,
                clip_reason = excluded.clip_reason,
                language = excluded.language,
                duration_ms = excluded.duration_ms,
                hls_master_playlist_path = excluded.hls_master_playlist_path,
                thumbnail_url = excluded.thumbnail_url,
                status = excluded.status,
                visibility_status = excluded.visibility_status,
                publish_at = excluded.publish_at,
                updated_at = now()
            returning video_id
            """,
            (
                video.source_clip_key,
                video.parent_video_name,
                video.parent_video_slug,
                video.clip_seq,
                video.source_start_ms,
                video.source_end_ms,
                video.title,
                video.description,
                video.clip_reason,
                video.language,
                video.duration_ms,
                video.hls_master_playlist_path,
                video.thumbnail_url,
                video.status,
                video.visibility_status,
                video.publish_at,
            ),
        )
        row = cursor.fetchone()
        if row is None or row["video_id"] is None:
            raise CatalogIngestError(
                code="db_write_failed",
                stage="repository",
                message="upsert catalog.videos 后未返回 video_id",
                context={"source_clip_key": video.source_clip_key},
            )
        return str(row["video_id"])

    def _replace_transcript_related_rows(
        self,
        cursor: psycopg.Cursor,
        video_id: str,
        normalized_data: NormalizedClipData,
    ) -> None:
        """按 video_id 替换 transcript、sentence、span 和 unit_index 四张表。"""

        cursor.execute("delete from catalog.video_unit_index where video_id = %s", (video_id,))
        cursor.execute("delete from catalog.video_semantic_spans where video_id = %s", (video_id,))
        cursor.execute("delete from catalog.video_transcript_sentences where video_id = %s", (video_id,))
        cursor.execute("delete from catalog.video_transcripts where video_id = %s", (video_id,))

        transcript = normalized_data.transcript
        cursor.execute(
            """
            insert into catalog.video_transcripts (
              video_id,
              transcript_object_path,
              transcript_checksum,
              transcript_format_version,
              full_text,
              sentence_count,
              semantic_span_count,
              mapped_span_count,
              unmapped_span_count,
              mapped_span_ratio
            )
            values (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
            """,
            (
                video_id,
                transcript.transcript_object_path,
                transcript.transcript_checksum,
                transcript.transcript_format_version,
                transcript.full_text,
                transcript.sentence_count,
                transcript.semantic_span_count,
                transcript.mapped_span_count,
                transcript.unmapped_span_count,
                transcript.mapped_span_ratio,
            ),
        )

        if normalized_data.sentences:
            cursor.executemany(
                """
                insert into catalog.video_transcript_sentences (
                  video_id,
                  sentence_index,
                  text,
                  start_ms,
                  end_ms,
                  explanation
                )
                values (%s, %s, %s, %s, %s, %s)
                """,
                [
                    (
                        video_id,
                        sentence.sentence_index,
                        sentence.text,
                        sentence.start_ms,
                        sentence.end_ms,
                        sentence.explanation,
                    )
                    for sentence in normalized_data.sentences
                ],
            )

        if normalized_data.spans:
            cursor.executemany(
                """
                insert into catalog.video_semantic_spans (
                  video_id,
                  sentence_index,
                  span_index,
                  text,
                  start_ms,
                  end_ms,
                  explanation,
                  coarse_unit_id,
                  base_form,
                  dictionary_text
                )
                values (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                """,
                [
                    (
                        video_id,
                        span.sentence_index,
                        span.span_index,
                        span.text,
                        span.start_ms,
                        span.end_ms,
                        span.explanation,
                        span.coarse_unit_id,
                        span.base_form,
                        span.dictionary_text,
                    )
                    for span in normalized_data.spans
                ],
            )

        if normalized_data.unit_indexes:
            cursor.executemany(
                """
                insert into catalog.video_unit_index (
                  video_id,
                  coarse_unit_id,
                  mention_count,
                  sentence_count,
                  first_start_ms,
                  last_end_ms,
                  coverage_ms,
                  coverage_ratio,
                  sentence_indexes,
                  evidence_sentence_indexes,
                  evidence_span_indexes,
                  sample_surface_forms
                )
                values (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                """,
                [
                    (
                        video_id,
                        unit.coarse_unit_id,
                        unit.mention_count,
                        unit.sentence_count,
                        unit.first_start_ms,
                        unit.last_end_ms,
                        unit.coverage_ms,
                        unit.coverage_ratio,
                        list(unit.sentence_indexes),
                        list(unit.evidence_sentence_indexes),
                        list(unit.evidence_span_indexes),
                        list(unit.sample_surface_forms),
                    )
                    for unit in normalized_data.unit_indexes
                ],
            )


def _utcnow() -> datetime:
    """统一生成带时区的 UTC 时间。"""

    return datetime.now(timezone.utc)
