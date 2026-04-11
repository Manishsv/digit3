from __future__ import annotations

import json
from dataclasses import dataclass
from typing import Any

from app.digit_client import DigitClient

MDMS_SCHEMA = "governance.factsContract"


@dataclass(frozen=True)
class ResolvedFactsContract:
    contract_code: str
    contract_version: str
    facts_json_schema: dict[str, Any]
    allowed_outcome_statuses: frozenset[str] | None
    allowed_registry_schema_codes: frozenset[str] | None


def _as_str_set(val: Any) -> set[str] | None:
    if val is None:
        return None
    if isinstance(val, str) and val.strip().startswith("["):
        try:
            parsed = json.loads(val)
            if isinstance(parsed, list):
                return {str(x) for x in parsed if x is not None}
        except json.JSONDecodeError:
            return None
    if isinstance(val, list):
        return {str(x) for x in val if x is not None}
    return None


def _facts_schema_from_data(data: dict[str, Any]) -> dict[str, Any]:
    raw = data.get("factsJsonSchema")
    if isinstance(raw, dict):
        return raw
    if isinstance(raw, str) and raw.strip():
        try:
            obj = json.loads(raw)
            if isinstance(obj, dict):
                return obj
        except json.JSONDecodeError:
            pass
    return {}


def _merge_facts_schemas(layers: list[dict[str, Any]]) -> dict[str, Any]:
    schemas = [_facts_schema_from_data(d) for d in layers]
    schemas = [s for s in schemas if s]
    if not schemas:
        return {"type": "object", "additionalProperties": True}
    if len(schemas) == 1:
        return schemas[0]
    return {"allOf": schemas}


def _merge_allowed_sets(layers: list[dict[str, Any]], key: str) -> frozenset[str] | None:
    acc: set[str] | None = None
    for d in layers:
        s = _as_str_set(d.get(key))
        if not s:
            continue
        acc = s if acc is None else (acc & s)
    return frozenset(acc) if acc is not None else None


def _pick_contract_row(
    rows: list[dict[str, Any]], contract_code: str, contract_version: str, tenant_id: str
) -> dict[str, Any] | None:
    ver = str(contract_version or "1")
    candidates: list[tuple[str, dict[str, Any]]] = []
    for row in rows:
        if row.get("isActive") is False:
            continue
        d = row.get("data")
        if not isinstance(d, dict):
            continue
        if d.get("contractCode") != contract_code:
            continue
        if str(d.get("contractVersion") or "1") != ver:
            continue
        tid = str(d.get("tenantId") or "").strip()
        candidates.append((tid, d))
    for tid, d in candidates:
        if tid == tenant_id.strip():
            return d
    for tid, d in candidates:
        if not tid:
            return d
    return None


def _resolve_layer_chain(
    tip: dict[str, Any],
    by_key: dict[tuple[str, str], dict[str, Any]],
    contract_version: str,
    stack: list[tuple[str, str]],
) -> list[dict[str, Any]]:
    code = str(tip.get("contractCode") or "")
    ver = str(contract_version or "1")
    key = (code, ver)
    if key in stack:
        raise ValueError(f"Cycle in facts contract extends: {key}")
    ext = str(tip.get("extendsContractCode") or "").strip()
    if not ext:
        return [tip]
    parent = by_key.get((ext, ver))
    if not parent:
        raise ValueError(f"Unknown parent contract {ext!r} for version {ver!r}")
    inner = _resolve_layer_chain(parent, by_key, ver, stack + [key])
    return inner + [tip]


def resolve_facts_contract(
    digit: DigitClient,
    headers: dict[str, str],
    *,
    tenant_id: str,
    contract_code: str,
    contract_version: str = "1",
    mdms_schema: str = MDMS_SCHEMA,
) -> ResolvedFactsContract:
    rows = digit.mdms_list_schema_data(headers, mdms_schema)
    tip = _pick_contract_row(rows, contract_code, contract_version, tenant_id)
    if not tip:
        raise ValueError(
            f"No MDMS row for contract {contract_code!r} version {contract_version!r} "
            f"(tenant {tenant_id!r} or platform default)"
        )
    by_key: dict[tuple[str, str], dict[str, Any]] = {}
    ver = str(contract_version or "1")
    for row in rows:
        if row.get("isActive") is False:
            continue
        d = row.get("data")
        if not isinstance(d, dict):
            continue
        cc = d.get("contractCode")
        if not cc:
            continue
        cv = str(d.get("contractVersion") or "1")
        by_key[(str(cc), cv)] = d

    layers = _resolve_layer_chain(tip, by_key, ver, [])
    merged_schema = _merge_facts_schemas(layers)
    outcomes = _merge_allowed_sets(layers, "allowedOutcomeStatuses")
    reg_codes = _merge_allowed_sets(layers, "allowedRegistrySchemaCodes")
    return ResolvedFactsContract(
        contract_code=str(tip.get("contractCode") or contract_code),
        contract_version=ver,
        facts_json_schema=merged_schema,
        allowed_outcome_statuses=outcomes,
        allowed_registry_schema_codes=reg_codes,
    )
