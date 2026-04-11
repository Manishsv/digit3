from __future__ import annotations

import sqlite3
from contextlib import contextmanager
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Iterator


def utc_now_iso() -> str:
    return datetime.now(timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")


@dataclass
class ReceiptRow:
    receipt_id: str
    registry_id: str
    decision_id: str
    case_entity_id: str | None


class IndexStore:
    """Small local index for hash-chain + lookups (Registry is source of truth)."""

    def __init__(self, path: str) -> None:
        self.path = path

    @contextmanager
    def _conn(self) -> Iterator[sqlite3.Connection]:
        conn = sqlite3.connect(self.path)
        conn.row_factory = sqlite3.Row
        try:
            yield conn
            conn.commit()
        finally:
            conn.close()

    def init(self) -> None:
        with self._conn() as c:
            c.executescript(
                """
                CREATE TABLE IF NOT EXISTS case_trace_chain (
                  tenant_id TEXT NOT NULL,
                  case_entity_id TEXT NOT NULL,
                  last_trace_hash TEXT,
                  updated_at TEXT NOT NULL,
                  PRIMARY KEY (tenant_id, case_entity_id)
                );
                CREATE TABLE IF NOT EXISTS receipts (
                  receipt_id TEXT PRIMARY KEY,
                  registry_id TEXT NOT NULL,
                  decision_id TEXT NOT NULL,
                  case_entity_id TEXT,
                  created_at TEXT NOT NULL
                );
                CREATE INDEX IF NOT EXISTS idx_receipts_decision ON receipts(decision_id);
                """
            )

    def get_last_trace_hash(self, tenant_id: str, case_entity_id: str) -> str | None:
        with self._conn() as c:
            row = c.execute(
                "SELECT last_trace_hash FROM case_trace_chain WHERE tenant_id = ? AND case_entity_id = ?",
                (tenant_id, case_entity_id),
            ).fetchone()
        return str(row["last_trace_hash"]) if row and row["last_trace_hash"] else None

    def set_last_trace_hash(self, tenant_id: str, case_entity_id: str, last_trace_hash: str) -> None:
        now = utc_now_iso()
        with self._conn() as c:
            c.execute(
                """
                INSERT INTO case_trace_chain (tenant_id, case_entity_id, last_trace_hash, updated_at)
                VALUES (?,?,?,?)
                ON CONFLICT(tenant_id, case_entity_id) DO UPDATE SET
                  last_trace_hash=excluded.last_trace_hash,
                  updated_at=excluded.updated_at
                """,
                (tenant_id, case_entity_id, last_trace_hash, now),
            )

    def put_receipt(self, row: ReceiptRow) -> None:
        now = utc_now_iso()
        with self._conn() as c:
            c.execute(
                """
                INSERT OR REPLACE INTO receipts (receipt_id, registry_id, decision_id, case_entity_id, created_at)
                VALUES (?,?,?,?,?)
                """,
                (row.receipt_id, row.registry_id, row.decision_id, row.case_entity_id, now),
            )

    def get_receipt_registry_id(self, receipt_id: str) -> str | None:
        with self._conn() as c:
            row = c.execute(
                "SELECT registry_id FROM receipts WHERE receipt_id = ?",
                (receipt_id,),
            ).fetchone()
        return str(row["registry_id"]) if row and row["registry_id"] else None

