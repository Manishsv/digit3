from __future__ import annotations

import json
from typing import Any
from urllib.parse import quote

import httpx

from app.config import Settings


def _extract_registry_record_id(obj: Any) -> str | None:
    if obj is None:
        return None
    if isinstance(obj, str):
        return obj
    if not isinstance(obj, dict):
        return None
    # Prefer the Registry service's stable business id used by GET .../data/_registry?registryId=
    for key in ("registryId", "registry_id", "id"):
        v = obj.get(key)
        if isinstance(v, str) and v:
            return v
    for wrap in ("Registrydata", "registryData", "data", "Data"):
        inner = obj.get(wrap)
        if isinstance(inner, dict):
            found = _extract_registry_record_id(inner)
            if found:
                return found
    return None


class DigitClient:
    def __init__(self, settings: Settings) -> None:
        self.s = settings

    def _client(self) -> httpx.Client:
        return httpx.Client(timeout=60.0)

    def idgen_generate(self, headers: dict[str, str], template_code: str) -> str:
        url = f"{self.s.idgen_base_url.rstrip('/')}/idgen/v1/generate"
        body = {
            "templateCode": template_code,
            "variables": {"ORG": self.s.idgen_org_variable},
        }
        with self._client() as c:
            r = c.post(url, headers={**headers, "Content-Type": "application/json"}, json=body)
            if r.status_code >= 400:
                raise RuntimeError(f"IdGen {r.status_code}: {r.text}")
            data = r.json()
        gen_id = None
        if isinstance(data, dict):
            gen_id = data.get("id") or data.get("generatedId")
        if not gen_id:
            raise RuntimeError(f"IdGen response missing id: {data!r}")
        return str(gen_id)

    def registry_create(
        self, headers: dict[str, str], schema_code: str, data: dict[str, Any]
    ) -> tuple[dict[str, Any], str | None]:
        url = f"{self.s.registry_base_url.rstrip('/')}/registry/v1/schema/{schema_code}/data"
        with self._client() as c:
            r = c.post(
                url,
                headers={**headers, "Content-Type": "application/json"},
                json={"data": data},
            )
            text = r.text
            if r.status_code >= 400:
                raise RuntimeError(f"Registry error {r.status_code}: {text}")
            try:
                parsed = json.loads(text) if text else {}
            except json.JSONDecodeError:
                parsed = {"_raw": text}
        rid = _extract_registry_record_id(parsed)
        return parsed, rid

    def registry_read(self, headers: dict[str, str], schema_code: str, registry_id: str) -> dict[str, Any]:
        base = self.s.registry_base_url.rstrip("/")
        enc_sc = quote(schema_code, safe="")
        enc_id = quote(registry_id, safe="")
        candidates = [
            f"{base}/registry/v1/schema/{enc_sc}/data/_registry?registryId={enc_id}",
            f"{base}/registry/v1/schema/{enc_sc}/data/{enc_id}",
            f"{base}/registry/v1/_get?schemaCode={quote(schema_code, safe='')}&registryId={enc_id}",
        ]
        last_err: str | None = None
        for url in candidates:
            with self._client() as c:
                r = c.get(url, headers=headers)
            if r.status_code >= 400:
                last_err = f"{r.status_code}: {r.text}"
                continue
            try:
                return r.json()
            except json.JSONDecodeError:
                return {"_raw": r.text}
        raise RuntimeError(f"Registry read failed ({last_err or 'unknown'})")

    def mdms_codes_for_schema_category(
        self, headers: dict[str, str], schema_code: str, category: str
    ) -> set[str]:
        """Fetch MDMS rows and collect `code` values where data.category matches (same pattern as coordination)."""
        url = f"{self.s.mdms_base_url.rstrip('/')}/mdms-v2/v2"
        with self._client() as c:
            r = c.get(url, headers=headers, params={"schemaCode": schema_code})
            if r.status_code >= 400:
                raise RuntimeError(f"MDMS {r.status_code}: {r.text}")
            try:
                payload = r.json()
            except json.JSONDecodeError as e:
                raise RuntimeError(f"MDMS invalid JSON: {e}") from e
        out: set[str] = set()
        rows = None
        if isinstance(payload, dict):
            rows = payload.get("Mdms") or payload.get("mdms") or payload.get("data")
        if not isinstance(rows, list):
            return set()
        for row in rows:
            if not isinstance(row, dict):
                continue
            d = row.get("data") or {}
            if not isinstance(d, dict):
                continue
            if d.get("category") == category and d.get("code"):
                out.add(str(d["code"]))
        return out

    def mdms_list_schema_data(self, headers: dict[str, str], schema_code: str) -> list[dict[str, Any]]:
        """Return raw MDMS rows for a schema (Mdms v2 list API)."""
        url = f"{self.s.mdms_base_url.rstrip('/')}/mdms-v2/v2"
        with self._client() as c:
            r = c.get(url, headers=headers, params={"schemaCode": schema_code})
            if r.status_code >= 400:
                raise RuntimeError(f"MDMS list {r.status_code}: {r.text}")
            try:
                payload = r.json()
            except json.JSONDecodeError as e:
                raise RuntimeError(f"MDMS invalid JSON: {e}") from e
        rows = None
        if isinstance(payload, dict):
            rows = payload.get("Mdms") or payload.get("mdms") or payload.get("data")
        if not isinstance(rows, list):
            return []
        return [x for x in rows if isinstance(x, dict)]

