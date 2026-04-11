from __future__ import annotations

import json
import re
from typing import Any

import httpx

from app.config import Settings


def _extract_registry_record_id(obj: Any) -> str | None:
    if obj is None:
        return None
    if isinstance(obj, str):
        return obj
    if not isinstance(obj, dict):
        return None
    for key in ("id", "registryId", "registry_id"):
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

    def mdms_codes_for_category(self, headers: dict[str, str], category: str) -> set[str]:
        """Best-effort: fetch MDMS rows for coordination.vocabulary and filter by category."""
        url = f"{self.s.mdms_base_url.rstrip('/')}/mdms-v2/v2"
        with self._client() as c:
            r = c.get(
                url,
                headers=headers,
                params={"schemaCode": "coordination.vocabulary"},
            )
            if r.status_code >= 400:
                return set()
            try:
                payload = r.json()
            except json.JSONDecodeError:
                return set()
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


def mapping_key(entity_type: str, source_system: str, local_id: str) -> str:
    def norm(x: str) -> str:
        return re.sub(r"\s+", " ", (x or "").strip()).upper()

    return f"{norm(entity_type)}|{norm(source_system)}|{norm(local_id)}"

