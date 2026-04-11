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
class ServiceRow:
    tenant_id: str
    service_code: str
    registry_id: str | None
    status: str
    updated_at: str


@dataclass
class BundleRow:
    tenant_id: str
    bundle_id: str
    service_code: str
    version: str
    registry_id: str | None
    status: str
    updated_at: str


@dataclass
class JobRow:
    tenant_id: str
    job_id: str
    service_code: str
    bundle_id: str | None
    status: str
    started_at: str
    finished_at: str | None


class IndexStore:
    """Local query index: Registry is source of truth; this enables list/status without registry search."""

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
                CREATE TABLE IF NOT EXISTS services (
                  tenant_id TEXT NOT NULL,
                  service_code TEXT NOT NULL,
                  registry_id TEXT,
                  status TEXT NOT NULL,
                  updated_at TEXT NOT NULL,
                  payload_json TEXT,
                  PRIMARY KEY (tenant_id, service_code)
                );
                CREATE TABLE IF NOT EXISTS bundles (
                  tenant_id TEXT NOT NULL,
                  bundle_id TEXT NOT NULL,
                  service_code TEXT NOT NULL,
                  version TEXT NOT NULL,
                  registry_id TEXT,
                  status TEXT NOT NULL,
                  updated_at TEXT NOT NULL,
                  payload_json TEXT,
                  PRIMARY KEY (tenant_id, bundle_id)
                );
                CREATE INDEX IF NOT EXISTS idx_bundles_service ON bundles(tenant_id, service_code);
                CREATE TABLE IF NOT EXISTS jobs (
                  tenant_id TEXT NOT NULL,
                  job_id TEXT NOT NULL,
                  service_code TEXT NOT NULL,
                  bundle_id TEXT,
                  status TEXT NOT NULL,
                  started_at TEXT NOT NULL,
                  finished_at TEXT,
                  payload_json TEXT,
                  PRIMARY KEY (tenant_id, job_id)
                );
                CREATE INDEX IF NOT EXISTS idx_jobs_service ON jobs(tenant_id, service_code);
                """
            )

    def upsert_service(self, tenant_id: str, service_code: str, registry_id: str | None, status: str, payload: dict[str, Any]) -> None:
        now = utc_now_iso()
        with self._conn() as c:
            c.execute(
                """
                INSERT INTO services(tenant_id, service_code, registry_id, status, updated_at, payload_json)
                VALUES (?,?,?,?,?,?)
                ON CONFLICT(tenant_id, service_code) DO UPDATE SET
                  registry_id=excluded.registry_id,
                  status=excluded.status,
                  updated_at=excluded.updated_at,
                  payload_json=excluded.payload_json
                """,
                (tenant_id, service_code, registry_id, status, now, json.dumps(payload)),
            )

    def list_services(self, tenant_id: str, limit: int = 200) -> list[dict[str, Any]]:
        with self._conn() as c:
            rows = c.execute(
                "SELECT * FROM services WHERE tenant_id = ? ORDER BY updated_at DESC LIMIT ?",
                (tenant_id, limit),
            ).fetchall()
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

    def upsert_bundle(
        self,
        tenant_id: str,
        bundle_id: str,
        service_code: str,
        version: str,
        registry_id: str | None,
        status: str,
        payload: dict[str, Any],
    ) -> None:
        now = utc_now_iso()
        with self._conn() as c:
            c.execute(
                """
                INSERT INTO bundles(tenant_id, bundle_id, service_code, version, registry_id, status, updated_at, payload_json)
                VALUES (?,?,?,?,?,?,?,?)
                ON CONFLICT(tenant_id, bundle_id) DO UPDATE SET
                  registry_id=excluded.registry_id,
                  status=excluded.status,
                  updated_at=excluded.updated_at,
                  payload_json=excluded.payload_json
                """,
                (tenant_id, bundle_id, service_code, version, registry_id, status, now, json.dumps(payload)),
            )

    def list_bundles(self, tenant_id: str, service_code: str | None = None, limit: int = 200) -> list[dict[str, Any]]:
        with self._conn() as c:
            if service_code:
                rows = c.execute(
                    "SELECT * FROM bundles WHERE tenant_id = ? AND service_code = ? ORDER BY updated_at DESC LIMIT ?",
                    (tenant_id, service_code, limit),
                ).fetchall()
            else:
                rows = c.execute(
                    "SELECT * FROM bundles WHERE tenant_id = ? ORDER BY updated_at DESC LIMIT ?",
                    (tenant_id, limit),
                ).fetchall()
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

    def get_bundle(self, tenant_id: str, bundle_id: str) -> dict[str, Any] | None:
        with self._conn() as c:
            row = c.execute(
                "SELECT * FROM bundles WHERE tenant_id = ? AND bundle_id = ?",
                (tenant_id, bundle_id),
            ).fetchone()
        if not row:
            return None
        d = dict(row)
        if d.get("payload_json"):
            try:
                d["payload"] = json.loads(d["payload_json"])
            except json.JSONDecodeError:
                d["payload"] = None
        return d

    def create_job(self, tenant_id: str, job_id: str, service_code: str, bundle_id: str | None, status: str, payload: dict[str, Any]) -> None:
        now = utc_now_iso()
        with self._conn() as c:
            c.execute(
                """
                INSERT OR REPLACE INTO jobs(tenant_id, job_id, service_code, bundle_id, status, started_at, finished_at, payload_json)
                VALUES (?,?,?,?,?,?,?,?)
                """,
                (tenant_id, job_id, service_code, bundle_id, status, now, None, json.dumps(payload)),
            )

    def finish_job(self, tenant_id: str, job_id: str, status: str, payload: dict[str, Any]) -> None:
        now = utc_now_iso()
        with self._conn() as c:
            c.execute(
                """
                UPDATE jobs SET status = ?, finished_at = ?, payload_json = ?
                WHERE tenant_id = ? AND job_id = ?
                """,
                (status, now, json.dumps(payload), tenant_id, job_id),
            )

    def get_job(self, tenant_id: str, job_id: str) -> dict[str, Any] | None:
        with self._conn() as c:
            row = c.execute(
                "SELECT * FROM jobs WHERE tenant_id = ? AND job_id = ?",
                (tenant_id, job_id),
            ).fetchone()
        if not row:
            return None
        d = dict(row)
        if d.get("payload_json"):
            try:
                d["payload"] = json.loads(d["payload_json"])
            except json.JSONDecodeError:
                d["payload"] = None
        return d

