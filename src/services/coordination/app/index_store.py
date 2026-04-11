from __future__ import annotations

import json
import sqlite3
from contextlib import contextmanager
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Any, Iterator


def utc_now_iso() -> str:
    return datetime.now(timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")


@dataclass
class MappingRow:
    mapping_key: str
    entity_type: str
    source_system: str
    local_id: str
    canonical_id: str
    status: str
    registry_id: str | None


class IndexStore:
    """Local query index: Registry remains source of truth for stored JSON rows."""

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
                CREATE TABLE IF NOT EXISTS entity_mapping (
                  mapping_key TEXT PRIMARY KEY,
                  entity_type TEXT NOT NULL,
                  source_system TEXT NOT NULL,
                  local_id TEXT NOT NULL,
                  canonical_id TEXT NOT NULL,
                  status TEXT NOT NULL,
                  registry_id TEXT,
                  created_at TEXT NOT NULL
                );
                CREATE TABLE IF NOT EXISTS links (
                  id INTEGER PRIMARY KEY AUTOINCREMENT,
                  registry_id TEXT NOT NULL,
                  from_entity_type TEXT NOT NULL,
                  from_canonical_id TEXT NOT NULL,
                  relation_type TEXT NOT NULL,
                  to_entity_type TEXT NOT NULL,
                  to_canonical_id TEXT NOT NULL,
                  source_system TEXT NOT NULL,
                  status TEXT NOT NULL,
                  created_time TEXT NOT NULL
                );
                CREATE INDEX IF NOT EXISTS idx_links_from ON links(from_entity_type, from_canonical_id);
                CREATE INDEX IF NOT EXISTS idx_links_to ON links(to_entity_type, to_canonical_id);
                CREATE TABLE IF NOT EXISTS events (
                  id INTEGER PRIMARY KEY AUTOINCREMENT,
                  registry_id TEXT NOT NULL,
                  event_type TEXT NOT NULL,
                  subject_entity_type TEXT NOT NULL,
                  subject_canonical_id TEXT NOT NULL,
                  correlation_id TEXT NOT NULL,
                  source_system TEXT NOT NULL,
                  actor TEXT NOT NULL,
                  occurred_at TEXT NOT NULL,
                  recorded_at TEXT NOT NULL,
                  payload_json TEXT
                );
                CREATE INDEX IF NOT EXISTS idx_events_subject ON events(subject_entity_type, subject_canonical_id);
                CREATE TABLE IF NOT EXISTS traces (
                  id INTEGER PRIMARY KEY AUTOINCREMENT,
                  registry_id TEXT NOT NULL,
                  trace_type TEXT NOT NULL,
                  entity_type TEXT NOT NULL,
                  canonical_id TEXT NOT NULL,
                  action TEXT NOT NULL,
                  actor TEXT NOT NULL,
                  source_system TEXT NOT NULL,
                  occurred_at TEXT NOT NULL,
                  recorded_at TEXT NOT NULL,
                  related_event_id TEXT,
                  payload_json TEXT
                );
                CREATE INDEX IF NOT EXISTS idx_traces_entity ON traces(entity_type, canonical_id);
                CREATE TABLE IF NOT EXISTS participants (
                  participant_code TEXT PRIMARY KEY,
                  registry_id TEXT NOT NULL,
                  payload_json TEXT NOT NULL
                );
                CREATE TABLE IF NOT EXISTS authority_rules (
                  id INTEGER PRIMARY KEY AUTOINCREMENT,
                  registry_id TEXT NOT NULL,
                  fact_type TEXT NOT NULL,
                  jurisdiction TEXT NOT NULL,
                  effective_from TEXT NOT NULL,
                  effective_to TEXT,
                  payload_json TEXT NOT NULL
                );
                CREATE INDEX IF NOT EXISTS idx_auth_lookup ON authority_rules(fact_type, jurisdiction);
                """
            )

    def get_mapping(self, key: str) -> MappingRow | None:
        with self._conn() as c:
            row = c.execute(
                "SELECT * FROM entity_mapping WHERE mapping_key = ?", (key,)
            ).fetchone()
        if not row:
            return None
        return MappingRow(
            mapping_key=row["mapping_key"],
            entity_type=row["entity_type"],
            source_system=row["source_system"],
            local_id=row["local_id"],
            canonical_id=row["canonical_id"],
            status=row["status"],
            registry_id=row["registry_id"],
        )

    def put_mapping(self, row: MappingRow, created_at: str | None = None) -> None:
        ts = created_at or utc_now_iso()
        with self._conn() as c:
            c.execute(
                """INSERT OR REPLACE INTO entity_mapping
                (mapping_key, entity_type, source_system, local_id, canonical_id, status, registry_id, created_at)
                VALUES (?,?,?,?,?,?,?,?)""",
                (
                    row.mapping_key,
                    row.entity_type,
                    row.source_system,
                    row.local_id,
                    row.canonical_id,
                    row.status,
                    row.registry_id,
                    ts,
                ),
            )

    def put_link(
        self,
        registry_id: str,
        from_entity_type: str,
        from_canonical_id: str,
        relation_type: str,
        to_entity_type: str,
        to_canonical_id: str,
        source_system: str,
        status: str,
        created_time: str,
    ) -> None:
        with self._conn() as c:
            c.execute(
                """INSERT INTO links (registry_id, from_entity_type, from_canonical_id, relation_type,
                to_entity_type, to_canonical_id, source_system, status, created_time)
                VALUES (?,?,?,?,?,?,?,?,?)""",
                (
                    registry_id,
                    from_entity_type,
                    from_canonical_id,
                    relation_type,
                    to_entity_type,
                    to_canonical_id,
                    source_system,
                    status,
                    created_time,
                ),
            )

    def links_for_entity(self, entity_type: str, canonical_id: str) -> list[dict[str, Any]]:
        with self._conn() as c:
            cur = c.execute(
                """
                SELECT * FROM links
                WHERE (from_entity_type = ? AND from_canonical_id = ?)
                   OR (to_entity_type = ? AND to_canonical_id = ?)
                ORDER BY id ASC
                """,
                (entity_type, canonical_id, entity_type, canonical_id),
            )
            rows = cur.fetchall()
        return [dict(r) for r in rows]

    def put_event(
        self,
        registry_id: str,
        event_type: str,
        subject_entity_type: str,
        subject_canonical_id: str,
        correlation_id: str,
        source_system: str,
        actor: str,
        occurred_at: str,
        recorded_at: str,
        payload: dict[str, Any] | None,
    ) -> None:
        with self._conn() as c:
            c.execute(
                """INSERT INTO events (registry_id, event_type, subject_entity_type, subject_canonical_id,
                correlation_id, source_system, actor, occurred_at, recorded_at, payload_json)
                VALUES (?,?,?,?,?,?,?,?,?,?)""",
                (
                    registry_id,
                    event_type,
                    subject_entity_type,
                    subject_canonical_id,
                    correlation_id,
                    source_system,
                    actor,
                    occurred_at,
                    recorded_at,
                    json.dumps(payload) if payload else None,
                ),
            )

    def events_for_subject(self, entity_type: str, canonical_id: str) -> list[dict[str, Any]]:
        with self._conn() as c:
            cur = c.execute(
                "SELECT * FROM events WHERE subject_entity_type = ? AND subject_canonical_id = ? ORDER BY occurred_at ASC",
                (entity_type, canonical_id),
            )
            rows = cur.fetchall()
        return [dict(r) for r in rows]

    def put_trace(
        self,
        registry_id: str,
        trace_type: str,
        entity_type: str,
        canonical_id: str,
        action: str,
        actor: str,
        source_system: str,
        occurred_at: str,
        recorded_at: str,
        related_event_id: str | None,
        payload: dict[str, Any] | None,
    ) -> None:
        with self._conn() as c:
            c.execute(
                """INSERT INTO traces (registry_id, trace_type, entity_type, canonical_id, action, actor,
                source_system, occurred_at, recorded_at, related_event_id, payload_json)
                VALUES (?,?,?,?,?,?,?,?,?,?,?)""",
                (
                    registry_id,
                    trace_type,
                    entity_type,
                    canonical_id,
                    action,
                    actor,
                    source_system,
                    occurred_at,
                    recorded_at,
                    related_event_id,
                    json.dumps(payload) if payload else None,
                ),
            )

    def traces_for_entity(self, entity_type: str, canonical_id: str) -> list[dict[str, Any]]:
        with self._conn() as c:
            cur = c.execute(
                "SELECT * FROM traces WHERE entity_type = ? AND canonical_id = ? ORDER BY occurred_at ASC",
                (entity_type, canonical_id),
            )
            rows = cur.fetchall()
        return [dict(r) for r in rows]

    def put_participant(self, participant_code: str, registry_id: str, payload: dict[str, Any]) -> None:
        with self._conn() as c:
            c.execute(
                """INSERT OR REPLACE INTO participants (participant_code, registry_id, payload_json)
                VALUES (?,?,?)""",
                (participant_code, registry_id, json.dumps(payload)),
            )

    def put_authority(
        self,
        registry_id: str,
        fact_type: str,
        jurisdiction: str,
        effective_from: str,
        effective_to: str | None,
        payload: dict[str, Any],
    ) -> None:
        with self._conn() as c:
            c.execute(
                """INSERT INTO authority_rules (registry_id, fact_type, jurisdiction, effective_from, effective_to, payload_json)
                VALUES (?,?,?,?,?,?)""",
                (
                    registry_id,
                    fact_type,
                    jurisdiction,
                    effective_from,
                    effective_to,
                    json.dumps(payload),
                ),
            )

    def authority_query(self, fact_type: str | None, jurisdiction: str | None) -> list[dict[str, Any]]:
        with self._conn() as c:
            q = "SELECT * FROM authority_rules WHERE 1=1"
            args: list[Any] = []
            if fact_type:
                q += " AND fact_type = ?"
                args.append(fact_type)
            if jurisdiction:
                q += " AND jurisdiction = ?"
                args.append(jurisdiction)
            cur = c.execute(q, args)
            rows = cur.fetchall()
        return [dict(r) for r in rows]

    def list_all_mappings(self, limit: int = 500) -> list[dict[str, Any]]:
        with self._conn() as c:
            cur = c.execute(
                "SELECT * FROM entity_mapping ORDER BY created_at DESC LIMIT ?", (limit,)
            )
            rows = cur.fetchall()
        return [dict(r) for r in rows]

    def list_all_links(self, limit: int = 500) -> list[dict[str, Any]]:
        with self._conn() as c:
            cur = c.execute("SELECT * FROM links ORDER BY id DESC LIMIT ?", (limit,))
            rows = cur.fetchall()
        return [dict(r) for r in rows]

    def list_all_events(self, limit: int = 500) -> list[dict[str, Any]]:
        with self._conn() as c:
            cur = c.execute("SELECT * FROM events ORDER BY occurred_at DESC LIMIT ?", (limit,))
            rows = cur.fetchall()
        return [dict(r) for r in rows]

    def list_all_traces(self, limit: int = 500) -> list[dict[str, Any]]:
        with self._conn() as c:
            cur = c.execute("SELECT * FROM traces ORDER BY occurred_at DESC LIMIT ?", (limit,))
            rows = cur.fetchall()
        return [dict(r) for r in rows]

    def list_all_participants(self) -> list[dict[str, Any]]:
        with self._conn() as c:
            cur = c.execute("SELECT * FROM participants ORDER BY participant_code")
            rows = cur.fetchall()
        out = []
        for r in rows:
            d = dict(r)
            if d.get("payload_json"):
                try:
                    d["payload"] = json.loads(d["payload_json"])
                except json.JSONDecodeError:
                    d["payload"] = None
            out.append(d)
        return out

    def list_all_authority(self, limit: int = 200) -> list[dict[str, Any]]:
        with self._conn() as c:
            cur = c.execute("SELECT * FROM authority_rules ORDER BY id DESC LIMIT ?", (limit,))
            rows = cur.fetchall()
        out = []
        for r in rows:
            d = dict(r)
            if d.get("payload_json"):
                try:
                    d["payload"] = json.loads(d["payload_json"])
                except json.JSONDecodeError:
                    d["payload"] = None
            out.append(d)
        return out

