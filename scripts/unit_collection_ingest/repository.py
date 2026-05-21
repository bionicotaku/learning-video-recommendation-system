from __future__ import annotations

import os
from pathlib import Path
from typing import Any, Iterable

import psycopg
from psycopg.rows import dict_row
from psycopg.types.json import Jsonb

from .ingest import CollectionMetadata


def load_database_url(env_file: Path | None = None) -> str:
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

    raise RuntimeError(f"未找到 DATABASE_URL，请检查环境变量或 {env_path}")


class UnitCollectionRepository:
    def __init__(self, database_url: str) -> None:
        self._database_url = database_url
        self._connection: psycopg.Connection | None = None

    def close(self) -> None:
        if self._connection is None:
            return
        self._connection.close()
        self._connection = None

    def _get_connection(self) -> psycopg.Connection:
        if self._connection is not None and not self._connection.closed:
            return self._connection
        self._connection = psycopg.connect(
            self._database_url,
            row_factory=dict_row,
            autocommit=True,
        )
        return self._connection

    def load_matches_by_label(self, labels: Iterable[str], chunk_size: int = 5000) -> dict[str, list[int]]:
        unique_labels = sorted(set(labels))
        matches: dict[str, list[int]] = {label: [] for label in unique_labels}
        if not unique_labels:
            return matches

        sql = """
            select label, id
            from semantic.coarse_unit
            where label = any(%s)
            order by label asc, id asc
        """
        with self._get_connection().cursor() as cursor:
            for offset in range(0, len(unique_labels), chunk_size):
                chunk = unique_labels[offset : offset + chunk_size]
                cursor.execute(sql, (chunk,))
                for row in cursor.fetchall():
                    matches[row["label"]].append(int(row["id"]))
        return matches

    def upsert_collection_with_members(
        self,
        *,
        slug: str,
        metadata: CollectionMetadata,
        source_payload: list[Any],
        word_unit_count: int,
        coarse_unit_count: int,
        member_rows: list[tuple[int, int, int]],
    ) -> str:
        connection = self._get_connection()
        with connection.transaction():
            with connection.cursor() as cursor:
                cursor.execute(
                    """
                    insert into semantic.unit_collections (
                      slug,
                      name,
                      description,
                      category,
                      status,
                      coarse_unit_count,
                      word_unit_count,
                      internal_description,
                      source_payload,
                      updated_at
                    )
                    values (%s, %s, %s, 'wordbook', 'active', %s, %s, %s, %s, now())
                    on conflict (slug) do update set
                      name = excluded.name,
                      description = excluded.description,
                      category = excluded.category,
                      status = excluded.status,
                      coarse_unit_count = excluded.coarse_unit_count,
                      word_unit_count = excluded.word_unit_count,
                      internal_description = excluded.internal_description,
                      source_payload = excluded.source_payload,
                      updated_at = now()
                    returning collection_id
                    """,
                    (
                        slug,
                        metadata.name,
                        metadata.description or None,
                        coarse_unit_count,
                        word_unit_count,
                        metadata.internal_description or None,
                        Jsonb(source_payload),
                    ),
                )
                row = cursor.fetchone()
                if row is None:
                    raise RuntimeError(f"{slug}: failed to upsert unit collection")
                collection_id = str(row["collection_id"])

                cursor.execute(
                    "delete from semantic.unit_collection_members where collection_id = %s",
                    (collection_id,),
                )
                if member_rows:
                    cursor.executemany(
                        """
                        insert into semantic.unit_collection_members (
                          collection_id,
                          coarse_unit_id,
                          sort_order,
                          target_priority
                        )
                        values (%s, %s, %s, %s)
                        """,
                        [
                            (collection_id, coarse_unit_id, sort_order, target_priority)
                            for coarse_unit_id, sort_order, target_priority in member_rows
                        ],
                    )
                return collection_id
