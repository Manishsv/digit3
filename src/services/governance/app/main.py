from __future__ import annotations

from contextlib import asynccontextmanager
from dataclasses import dataclass
from pathlib import Path
import hashlib
import json
from typing import Annotated, Any

from fastapi import APIRouter, Depends, FastAPI, HTTPException, Request
from fastapi.responses import RedirectResponse
from pydantic import BaseModel, ConfigDict, Field

from app.config import Settings
from app.contract_enforcement import (
    validate_compiled_rules_against_contract,
    validate_facts_against_contract,
    validate_registry_schema_code_allowed,
)
from app.digit_client import DigitClient
from app.facts_contract import resolve_facts_contract
from app.index_store import IndexStore, ReceiptRow, utc_now_iso
from app.rules_dsl import CompiledRuleset, RulesDslError, compile_ruleset, evaluate_compiled_ruleset
from app.security import (
    DEV_ALL_ROLES,
    decode_jwt_payload,
    digit_headers,
    extract_roles,
)


@dataclass
class GovernanceContext:
    headers: dict[str, str]
    roles: set[str]


async def get_governance_context(
    request: Request,
    hdrs: Annotated[dict[str, str], Depends(digit_headers)],
) -> GovernanceContext:
    token = hdrs["Authorization"].split(" ", 1)[1].strip()
    settings = request.app.state.settings
    if settings.dev_auth_enabled and token == settings.dev_auth_token:
        roles = set(DEV_ALL_ROLES)
    else:
        payload = decode_jwt_payload(token)
        roles = extract_roles(payload)
    return GovernanceContext(headers=hdrs, roles=roles)


@asynccontextmanager
async def lifespan(app: FastAPI):
    settings = Settings()
    Path(settings.governance_db_path).parent.mkdir(parents=True, exist_ok=True)
    store = IndexStore(settings.governance_db_path)
    store.init()
    app.state.settings = settings
    app.state.store = store
    app.state.digit = DigitClient(settings)
    yield


app = FastAPI(
    title="DIGIT Governance Service API (demo)",
    version="1.0.0",
    lifespan=lifespan,
    description="Governance APIs for rulesets, governed decisions, immutable decision traces, and appeals/orders.",
)

router = APIRouter(prefix="/governance/v1", tags=["governance"])


@app.get("/", include_in_schema=False)
def root():
    return RedirectResponse(url="/docs")


@app.get("/health", include_in_schema=False)
def health_root():
    return {"status": "ok"}


@router.get("/health")
def health():
    return {"status": "ok"}


@router.get("/whoami")
def whoami(ctx: Annotated[GovernanceContext, Depends(get_governance_context)]):
    return {
        "tenantId": ctx.headers.get("X-Tenant-ID"),
        "clientId": ctx.headers.get("X-Client-ID"),
        "roles": sorted(ctx.roles),
    }


def _compact(d: dict[str, object]) -> dict[str, object]:
    return {k: v for k, v in d.items() if v is not None and v != ""}


def _digit(request: Request) -> DigitClient:
    return request.app.state.digit


def _store(request: Request) -> IndexStore:
    return request.app.state.store


def _safe_idgen(digit: DigitClient, headers: dict[str, str], template: str) -> str:
    try:
        return digit.idgen_generate(headers, template)
    except RuntimeError as e:
        raise HTTPException(status_code=502, detail=str(e)) from e


def _safe_registry_create(digit: DigitClient, headers: dict[str, str], schema: str, payload: dict[str, object]):
    try:
        parsed, rid = digit.registry_create(headers, schema, payload)
        return parsed, rid
    except RuntimeError as e:
        raise HTTPException(status_code=502, detail=str(e)) from e


def _sha256_hex(b: bytes) -> str:
    return hashlib.sha256(b).hexdigest()


def _canonical_json(obj: object) -> str:
    return json.dumps(obj, sort_keys=True, separators=(",", ":"), ensure_ascii=False)


def _digest(obj: object) -> dict[str, str]:
    return {"alg": "sha256", "value": _sha256_hex(_canonical_json(obj).encode("utf-8"))}


def _facts_without_rules_yaml(snap: dict[str, object]) -> dict[str, object]:
    return {k: v for k, v in snap.items() if k != "rulesYaml"}


def _fact_value_at_path(facts: dict[str, object], path: str) -> object:
    cur: object = facts
    for part in path.split("."):
        p = part.strip()
        if not p:
            continue
        if not isinstance(cur, dict) or p not in cur:
            return None
        cur = cur[p]
    return cur


def _registry_ruleset_business_payload(read_resp: dict[str, Any]) -> dict[str, Any] | None:
    if not isinstance(read_resp, dict):
        return None
    inner: dict[str, Any] | None = None
    for key in ("data", "Data", "registryData", "Registrydata"):
        v = read_resp.get(key)
        if isinstance(v, dict):
            inner = v
            break
    if inner is None:
        inner = read_resp
    nested = inner.get("data")
    if isinstance(nested, dict) and "yamlText" in nested:
        return nested
    if "yamlText" in inner:
        return inner
    return inner if isinstance(inner, dict) else None


def _validate_mdms_fact_checks(
    digit: DigitClient,
    headers: dict[str, str],
    checks: list["MdmsFactCheckIn"],
    facts_snapshot: dict[str, object],
) -> None:
    if not checks:
        return
    probe = _facts_without_rules_yaml(facts_snapshot)
    for chk in checks:
        try:
            codes = digit.mdms_codes_for_schema_category(headers, chk.schemaCode, chk.category)
        except RuntimeError as e:
            raise HTTPException(status_code=502, detail=f"MDMS lookup failed: {e}") from e
        if not codes:
            raise HTTPException(
                status_code=400,
                detail=f"No MDMS codes for schemaCode={chk.schemaCode!r} category={chk.category!r} (empty or unknown)",
            )
        val = _fact_value_at_path(probe, chk.path)
        if val is None:
            raise HTTPException(status_code=400, detail=f"MDMS check: missing value at path {chk.path!r}")
        if str(val) not in codes:
            raise HTTPException(
                status_code=400,
                detail=f"MDMS check: path {chk.path!r} value {val!r} is not allowed for category {chk.category!r}",
            )


def _resolve_ruleset_yaml_for_compute(
    digit: DigitClient,
    headers: dict[str, str],
    ruleset: "RulesetRefIn",
    facts_snapshot: dict[str, object],
) -> tuple[str, dict[str, Any], str]:
    """Return (yaml_text, registry_row_meta, source) where source is inline|registry."""
    inline = facts_snapshot.get("rulesYaml")
    if isinstance(inline, str) and inline.strip():
        return inline.strip(), {}, "inline"

    candidates: list[str] = []
    if ruleset.registryRecordId:
        candidates.append(ruleset.registryRecordId)
    if ruleset.rulesetId and ruleset.rulesetId not in candidates:
        candidates.append(ruleset.rulesetId)

    last: str | None = None
    for rid in candidates:
        if not rid:
            continue
        try:
            raw = digit.registry_read(headers, "governance.ruleset", rid)
        except RuntimeError as e:
            last = str(e)
            continue
        data = _registry_ruleset_business_payload(raw)
        if not data:
            continue
        yt = data.get("yamlText")
        if not isinstance(yt, str) or not yt.strip():
            continue
        if str(data.get("version", "")) != str(ruleset.version):
            raise HTTPException(status_code=400, detail="ruleset.version does not match Registry governance.ruleset record")
        if str(data.get("id", "")) != str(ruleset.rulesetId):
            raise HTTPException(status_code=400, detail="ruleset.rulesetId does not match Registry record id field")
        meta = {
            "issuerAuthorityId": data.get("issuerAuthorityId"),
            "registryReadId": rid,
            "factsContractCode": data.get("factsContractCode"),
            "factsContractVersion": data.get("factsContractVersion"),
        }
        return yt.strip(), meta, "registry"

    detail = (
        "Provide non-empty factsSnapshot.rulesYaml or publish the ruleset and pass matching rulesetId, version, "
        "and optional registryRecordId from RulesetPublishOut."
    )
    if last:
        detail = f"{detail} Last registry error: {last}"
    raise HTTPException(status_code=400, detail=detail)


class PolicyDocumentRefIn(BaseModel):
    docType: str
    title: str | None = None
    issuedBy: str | None = None
    issuedAt: str | None = None
    jurisdiction: str | None = None
    language: str | None = None
    storage: str = Field(pattern="^(filestore|registry|other)$")
    ref: str
    digest: dict[str, str]


class RulesetPublishIn(BaseModel):
    yamlText: str
    status: str = "ACTIVE"
    humanVersion: str | None = None
    issuerAuthorityId: str
    policyDocuments: list[PolicyDocumentRefIn] = Field(default_factory=list)
    effectiveFrom: str | None = None
    effectiveTo: str | None = None
    factsContractCode: str | None = None
    factsContractVersion: str = "1"


class RulesetPublishOut(BaseModel):
    rulesetId: str
    code: str
    version: str
    registryRecordId: str | None = None
    digest: dict[str, str]


@router.post("/rulesets", response_model=RulesetPublishOut)
def publish_ruleset(
    body: RulesetPublishIn,
    ctx: Annotated[GovernanceContext, Depends(get_governance_context)],
    request: Request,
):
    digit = _digit(request)
    settings = request.app.state.settings

    try:
        compiled: CompiledRuleset = compile_ruleset(body.yamlText)
    except RulesDslError as e:
        raise HTTPException(status_code=400, detail=str(e)) from e

    if body.factsContractCode:
        tenant_id = ctx.headers.get("X-Tenant-ID") or ""
        try:
            fc = resolve_facts_contract(
                digit,
                ctx.headers,
                tenant_id=tenant_id,
                contract_code=body.factsContractCode,
                contract_version=body.factsContractVersion,
                mdms_schema=settings.facts_contract_mdms_schema,
            )
            validate_compiled_rules_against_contract(compiled, fc)
        except ValueError as e:
            raise HTTPException(status_code=400, detail=str(e)) from e
        except RuntimeError as e:
            raise HTTPException(status_code=502, detail=str(e)) from e

    now = utc_now_iso()
    rid = _safe_idgen(digit, ctx.headers, "governance.rul")
    ruleset_digest = _digest({"yamlText": body.yamlText, "compiled": compiled.compiled})

    reg_payload: dict[str, object] = {
        "id": rid,
        "code": compiled.code,
        "version": compiled.version,
        "humanVersion": body.humanVersion,
        "status": body.status,
        "effectiveFrom": body.effectiveFrom,
        "effectiveTo": body.effectiveTo,
        "issuerAuthorityId": body.issuerAuthorityId,
        "policyDocuments": [p.model_dump() for p in body.policyDocuments],
        "yamlText": body.yamlText,
        "compiledRules": compiled.compiled,
        "digest": ruleset_digest,
        "createdAt": now,
        "createdBy": ctx.headers.get("X-Client-ID") or "unknown",
    }
    if body.factsContractCode:
        reg_payload["factsContractCode"] = body.factsContractCode
        reg_payload["factsContractVersion"] = body.factsContractVersion
    _, reg_id = _safe_registry_create(digit, ctx.headers, "governance.ruleset", _compact(reg_payload))
    return RulesetPublishOut(
        rulesetId=rid,
        code=compiled.code,
        version=compiled.version,
        registryRecordId=reg_id,
        digest=ruleset_digest,
    )


class EntityRefIn(BaseModel):
    system: str
    entityType: str
    entityId: str
    tenantId: str | None = None
    version: str | int | float | None = None


class RulesetRefIn(BaseModel):
    model_config = ConfigDict(extra="ignore")
    rulesetId: str
    version: str
    registryRecordId: str | None = None


class MdmsFactCheckIn(BaseModel):
    """When non-empty, each fact path value must appear as `code` in MDMS rows for schemaCode with matching data.category."""

    model_config = ConfigDict(extra="ignore")
    path: str
    category: str
    schemaCode: str = "coordination.vocabulary"


class DecisionComputeIn(BaseModel):
    decisionType: str = "SBL_LICENSE"
    correlationId: str
    requestId: str
    channel: str | None = None
    caseRef: EntityRefIn
    applicantRef: EntityRefIn | None = None
    ruleset: RulesetRefIn
    factsSnapshot: dict[str, object] = Field(default_factory=dict)
    mdmsFactChecks: list[MdmsFactCheckIn] = Field(default_factory=list)
    factsContractCode: str | None = None
    factsContractVersion: str = "1"


class DecisionComputeOut(BaseModel):
    decisionId: str
    receiptId: str
    traceRegistryRecordId: str | None = None
    receiptRegistryRecordId: str | None = None
    outcome: dict[str, object]


@router.post("/decisions:compute", response_model=DecisionComputeOut)
def compute_decision(
    body: DecisionComputeIn,
    ctx: Annotated[GovernanceContext, Depends(get_governance_context)],
    request: Request,
):
    digit = _digit(request)
    store = _store(request)
    tenant_id = ctx.headers.get("X-Tenant-ID") or body.caseRef.tenantId or "UNKNOWN"

    ruleset_id = body.ruleset.rulesetId
    ruleset_ver = body.ruleset.version

    _validate_mdms_fact_checks(digit, ctx.headers, body.mdmsFactChecks, body.factsSnapshot)

    facts_digest = _digest(_facts_without_rules_yaml(body.factsSnapshot))

    yaml_text, yaml_meta, ruleset_source = _resolve_ruleset_yaml_for_compute(
        digit, ctx.headers, body.ruleset, body.factsSnapshot
    )
    issuer_authority = yaml_meta.get("issuerAuthorityId") if isinstance(yaml_meta.get("issuerAuthorityId"), str) else None
    if not issuer_authority:
        issuer_authority = "unknown"

    try:
        compiled = compile_ruleset(yaml_text)
    except RulesDslError as e:
        raise HTTPException(status_code=400, detail=str(e)) from e

    facts_eval: dict[str, object] = dict(_facts_without_rules_yaml(body.factsSnapshot))
    settings = request.app.state.settings

    eff_contract_code = body.factsContractCode
    eff_contract_version = body.factsContractVersion
    if not eff_contract_code and ruleset_source == "registry":
        cc = yaml_meta.get("factsContractCode")
        if isinstance(cc, str) and cc.strip():
            eff_contract_code = cc.strip()
            cv = yaml_meta.get("factsContractVersion")
            eff_contract_version = str(cv).strip() if cv is not None else "1"

    if eff_contract_code:
        try:
            fc = resolve_facts_contract(
                digit,
                ctx.headers,
                tenant_id=tenant_id,
                contract_code=eff_contract_code,
                contract_version=eff_contract_version,
                mdms_schema=settings.facts_contract_mdms_schema,
            )
            validate_facts_against_contract(dict(facts_eval), fc)
        except ValueError as e:
            raise HTTPException(status_code=400, detail=str(e)) from e
        except RuntimeError as e:
            raise HTTPException(status_code=502, detail=str(e)) from e

    outcome, explanations = evaluate_compiled_ruleset(compiled, facts_eval)

    now = utc_now_iso()
    decision_id = _safe_idgen(digit, ctx.headers, "governance.dec")
    receipt_id = _safe_idgen(digit, ctx.headers, "governance.rcp")

    prev_hash = store.get_last_trace_hash(tenant_id, body.caseRef.entityId)
    trace_base = {
        "decisionId": decision_id,
        "decisionType": body.decisionType,
        "tenantId": tenant_id,
        "correlationId": body.correlationId,
        "requestId": body.requestId,
        "channel": body.channel,
        "caseRef": body.caseRef.model_dump(exclude_none=True),
        "applicantRef": body.applicantRef.model_dump(exclude_none=True) if body.applicantRef else None,
        "actor": {
            "principalId": ctx.headers.get("X-Client-ID") or "unknown",
            "principalType": "service",
            "roles": sorted(ctx.roles),
            "clientId": ctx.headers.get("X-Client-ID"),
        },
        "ruleset": _compact(
            {
                "rulesetId": ruleset_id,
                "version": ruleset_ver,
                "digest": _digest({"yamlText": yaml_text}),
                "issuerAuthorityId": issuer_authority,
                "source": ruleset_source,
                "registryReadId": yaml_meta.get("registryReadId"),
            }
        ),
        "facts": _compact(
            {
                "factsRefs": [],
                "factsDigest": facts_digest,
                "factsContractCode": eff_contract_code,
                "factsContractVersion": eff_contract_version if eff_contract_code else None,
            }
        ),
        "explanations": explanations,
        "outcome": outcome,
        "timestamp": now,
        "integrity": {"traceDigest": _digest({"decisionId": decision_id, "outcome": outcome})},
        "lineage": {},
    }
    if trace_base.get("applicantRef") is None:
        trace_base.pop("applicantRef", None)
    if trace_base.get("channel") is None:
        trace_base.pop("channel", None)

    canonical_hash = _sha256_hex(_canonical_json(trace_base).encode("utf-8"))
    trace_hash = _sha256_hex(((prev_hash or "") + canonical_hash).encode("utf-8"))
    trace_payload = {**trace_base, "traceCanonicalJsonHash": canonical_hash, "traceHash": trace_hash}
    if prev_hash:
        trace_payload["prevTraceHash"] = prev_hash

    _, trace_reg_id = _safe_registry_create(digit, ctx.headers, "governance.decisionTrace", trace_payload)
    store.set_last_trace_hash(tenant_id, body.caseRef.entityId, trace_hash)

    receipt_payload = {
        "receiptId": receipt_id,
        "decisionId": decision_id,
        "tenantId": tenant_id,
        "caseRef": body.caseRef.model_dump(exclude_none=True),
        "applicantRef": body.applicantRef.model_dump(exclude_none=True) if body.applicantRef else None,
        "ruleset": _compact(
            {
                "rulesetId": ruleset_id,
                "version": ruleset_ver,
                "digest": _digest({"yamlText": yaml_text}),
                "issuerAuthorityId": issuer_authority,
                "source": ruleset_source,
                "registryReadId": yaml_meta.get("registryReadId"),
            }
        ),
        "factsDigest": facts_digest,
        "explanations": [{"ruleId": e["ruleId"], "matched": e["matched"], "reason": e.get("reason")} for e in explanations],
        "outcome": outcome,
        "issuedAt": now,
        "signature": {"kid": "demo", "alg": "none", "signature": "demo"},
    }
    if receipt_payload.get("applicantRef") is None:
        receipt_payload.pop("applicantRef", None)
    _, receipt_reg_id = _safe_registry_create(digit, ctx.headers, "governance.decisionReceipt", receipt_payload)
    if receipt_reg_id:
        store.put_receipt(
            ReceiptRow(
                receipt_id=receipt_id,
                registry_id=receipt_reg_id,
                decision_id=decision_id,
                case_entity_id=body.caseRef.entityId,
            )
        )

    return DecisionComputeOut(
        decisionId=decision_id,
        receiptId=receipt_id,
        traceRegistryRecordId=trace_reg_id,
        receiptRegistryRecordId=receipt_reg_id,
        outcome=outcome,
    )


@router.get("/decisions/{receipt_id}")
def get_decision(
    receipt_id: str,
    ctx: Annotated[GovernanceContext, Depends(get_governance_context)],
    request: Request,
):
    digit = _digit(request)
    store = _store(request)
    reg_id = store.get_receipt_registry_id(receipt_id)
    if not reg_id:
        raise HTTPException(status_code=404, detail="Unknown receiptId (not indexed locally)")
    payload = digit.registry_read(ctx.headers, "governance.decisionReceipt", reg_id)
    return {"receiptId": receipt_id, "registryRecordId": reg_id, "payload": payload}


class AppealCreateIn(BaseModel):
    receiptId: str
    decisionId: str
    filedBy: str
    grounds: str
    status: str = "FILED"
    metadata: dict[str, object] = Field(default_factory=dict)


class AppealCreateOut(BaseModel):
    appealId: str
    registryRecordId: str | None = None


@router.post("/appeals", response_model=AppealCreateOut)
def create_appeal(
    body: AppealCreateIn,
    ctx: Annotated[GovernanceContext, Depends(get_governance_context)],
    request: Request,
):
    digit = _digit(request)
    tenant_id = ctx.headers.get("X-Tenant-ID") or "UNKNOWN"
    now = utc_now_iso()
    appeal_id = _safe_idgen(digit, ctx.headers, "governance.apl")
    payload: dict[str, object] = {
        "appealId": appeal_id,
        "tenantId": tenant_id,
        "decisionId": body.decisionId,
        "receiptId": body.receiptId,
        "filedBy": body.filedBy,
        "grounds": body.grounds,
        "status": body.status,
        "filedAt": now,
        "metadata": body.metadata,
    }
    _, reg_id = _safe_registry_create(digit, ctx.headers, "governance.appeal", payload)
    return AppealCreateOut(appealId=appeal_id, registryRecordId=reg_id)


class OrderIssueIn(BaseModel):
    appealId: str
    decisionId: str
    receiptId: str
    issuedBy: str
    outcome: str = Field(description="UPHOLD | MODIFY | REMAND")
    instructions: str | None = None
    metadata: dict[str, object] = Field(default_factory=dict)


class OrderIssueOut(BaseModel):
    orderId: str
    registryRecordId: str | None = None


@router.post("/orders", response_model=OrderIssueOut)
def issue_order(
    body: OrderIssueIn,
    ctx: Annotated[GovernanceContext, Depends(get_governance_context)],
    request: Request,
):
    digit = _digit(request)
    tenant_id = ctx.headers.get("X-Tenant-ID") or "UNKNOWN"
    now = utc_now_iso()
    order_id = _safe_idgen(digit, ctx.headers, "governance.ord")
    payload: dict[str, object] = {
        "orderId": order_id,
        "tenantId": tenant_id,
        "appealId": body.appealId,
        "decisionId": body.decisionId,
        "receiptId": body.receiptId,
        "issuedBy": body.issuedBy,
        "outcome": body.outcome,
        "instructions": body.instructions,
        "issuedAt": now,
        "metadata": body.metadata,
    }
    _, reg_id = _safe_registry_create(digit, ctx.headers, "governance.order", payload)
    return OrderIssueOut(orderId=order_id, registryRecordId=reg_id)


class DecisionRecomputeIn(DecisionComputeIn):
    parentDecisionId: str
    appealId: str | None = None
    orderId: str | None = None


@router.post("/decisions:recompute", response_model=DecisionComputeOut)
def recompute_decision(
    body: DecisionRecomputeIn,
    ctx: Annotated[GovernanceContext, Depends(get_governance_context)],
    request: Request,
):
    digit = _digit(request)
    store = _store(request)
    tenant_id = ctx.headers.get("X-Tenant-ID") or body.caseRef.tenantId or "UNKNOWN"

    ruleset_id = body.ruleset.rulesetId
    ruleset_ver = body.ruleset.version

    _validate_mdms_fact_checks(digit, ctx.headers, body.mdmsFactChecks, body.factsSnapshot)

    facts_digest = _digest(_facts_without_rules_yaml(body.factsSnapshot))

    yaml_text, yaml_meta, ruleset_source = _resolve_ruleset_yaml_for_compute(
        digit, ctx.headers, body.ruleset, body.factsSnapshot
    )
    issuer_authority = yaml_meta.get("issuerAuthorityId") if isinstance(yaml_meta.get("issuerAuthorityId"), str) else None
    if not issuer_authority:
        issuer_authority = "unknown"

    try:
        compiled = compile_ruleset(yaml_text)
    except RulesDslError as e:
        raise HTTPException(status_code=400, detail=str(e)) from e

    facts_eval = dict(_facts_without_rules_yaml(body.factsSnapshot))
    settings = request.app.state.settings

    eff_contract_code = body.factsContractCode
    eff_contract_version = body.factsContractVersion
    if not eff_contract_code and ruleset_source == "registry":
        cc = yaml_meta.get("factsContractCode")
        if isinstance(cc, str) and cc.strip():
            eff_contract_code = cc.strip()
            cv = yaml_meta.get("factsContractVersion")
            eff_contract_version = str(cv).strip() if cv is not None else "1"

    if eff_contract_code:
        try:
            fc = resolve_facts_contract(
                digit,
                ctx.headers,
                tenant_id=tenant_id,
                contract_code=eff_contract_code,
                contract_version=eff_contract_version,
                mdms_schema=settings.facts_contract_mdms_schema,
            )
            validate_facts_against_contract(dict(facts_eval), fc)
        except ValueError as e:
            raise HTTPException(status_code=400, detail=str(e)) from e
        except RuntimeError as e:
            raise HTTPException(status_code=502, detail=str(e)) from e

    outcome, explanations = evaluate_compiled_ruleset(compiled, facts_eval)

    now = utc_now_iso()
    decision_id = _safe_idgen(digit, ctx.headers, "governance.dec")
    receipt_id = _safe_idgen(digit, ctx.headers, "governance.rcp")

    prev_hash = store.get_last_trace_hash(tenant_id, body.caseRef.entityId)
    lineage: dict[str, object] = {"parentDecisionId": body.parentDecisionId}
    if body.appealId:
        lineage["appealRef"] = {
            "system": "registry",
            "entityType": "Appeal",
            "entityId": body.appealId,
            "tenantId": tenant_id,
        }
    if body.orderId:
        lineage["orderRef"] = {
            "system": "registry",
            "entityType": "Order",
            "entityId": body.orderId,
            "tenantId": tenant_id,
        }

    trace_base = {
        "decisionId": decision_id,
        "decisionType": body.decisionType,
        "tenantId": tenant_id,
        "correlationId": body.correlationId,
        "requestId": body.requestId,
        "channel": body.channel,
        "caseRef": body.caseRef.model_dump(exclude_none=True),
        "applicantRef": body.applicantRef.model_dump(exclude_none=True) if body.applicantRef else None,
        "actor": {
            "principalId": ctx.headers.get("X-Client-ID") or "unknown",
            "principalType": "service",
            "roles": sorted(ctx.roles),
            "clientId": ctx.headers.get("X-Client-ID"),
        },
        "ruleset": _compact(
            {
                "rulesetId": ruleset_id,
                "version": ruleset_ver,
                "digest": _digest({"yamlText": yaml_text}),
                "issuerAuthorityId": issuer_authority,
                "source": ruleset_source,
                "registryReadId": yaml_meta.get("registryReadId"),
            }
        ),
        "facts": _compact(
            {
                "factsRefs": [],
                "factsDigest": facts_digest,
                "factsContractCode": eff_contract_code,
                "factsContractVersion": eff_contract_version if eff_contract_code else None,
            }
        ),
        "explanations": explanations,
        "outcome": outcome,
        "timestamp": now,
        "integrity": {"traceDigest": _digest({"decisionId": decision_id, "outcome": outcome})},
        "lineage": lineage,
    }
    if trace_base.get("applicantRef") is None:
        trace_base.pop("applicantRef", None)
    if trace_base.get("channel") is None:
        trace_base.pop("channel", None)

    canonical_hash = _sha256_hex(_canonical_json(trace_base).encode("utf-8"))
    trace_hash = _sha256_hex(((prev_hash or "") + canonical_hash).encode("utf-8"))
    trace_payload = {**trace_base, "traceCanonicalJsonHash": canonical_hash, "traceHash": trace_hash}
    if prev_hash:
        trace_payload["prevTraceHash"] = prev_hash

    _, trace_reg_id = _safe_registry_create(digit, ctx.headers, "governance.decisionTrace", trace_payload)
    store.set_last_trace_hash(tenant_id, body.caseRef.entityId, trace_hash)

    receipt_payload = {
        "receiptId": receipt_id,
        "decisionId": decision_id,
        "tenantId": tenant_id,
        "caseRef": body.caseRef.model_dump(exclude_none=True),
        "applicantRef": body.applicantRef.model_dump(exclude_none=True) if body.applicantRef else None,
        "ruleset": _compact(
            {
                "rulesetId": ruleset_id,
                "version": ruleset_ver,
                "digest": _digest({"yamlText": yaml_text}),
                "issuerAuthorityId": issuer_authority,
                "source": ruleset_source,
                "registryReadId": yaml_meta.get("registryReadId"),
            }
        ),
        "factsDigest": facts_digest,
        "explanations": [{"ruleId": e["ruleId"], "matched": e["matched"], "reason": e.get("reason")} for e in explanations],
        "outcome": outcome,
        "issuedAt": now,
        "signature": {"kid": "demo", "alg": "none", "signature": "demo"},
    }
    if receipt_payload.get("applicantRef") is None:
        receipt_payload.pop("applicantRef", None)
    _, receipt_reg_id = _safe_registry_create(digit, ctx.headers, "governance.decisionReceipt", receipt_payload)
    if receipt_reg_id:
        store.put_receipt(
            ReceiptRow(
                receipt_id=receipt_id,
                registry_id=receipt_reg_id,
                decision_id=decision_id,
                case_entity_id=body.caseRef.entityId,
            )
        )

    return DecisionComputeOut(
        decisionId=decision_id,
        receiptId=receipt_id,
        traceRegistryRecordId=trace_reg_id,
        receiptRegistryRecordId=receipt_reg_id,
        outcome=outcome,
    )


class ValidateRegistrySchemaContractIn(BaseModel):
    factsContractCode: str
    factsContractVersion: str = "1"
    registrySchemaCode: str


class ValidateRegistrySchemaContractOut(BaseModel):
    valid: bool
    contractCode: str
    contractVersion: str


@router.post("/contracts:validateRegistrySchema", response_model=ValidateRegistrySchemaContractOut)
def validate_registry_schema_against_contract(
    body: ValidateRegistrySchemaContractIn,
    ctx: Annotated[GovernanceContext, Depends(get_governance_context)],
    request: Request,
):
    digit = _digit(request)
    settings = request.app.state.settings
    tenant_id = ctx.headers.get("X-Tenant-ID") or ""
    try:
        fc = resolve_facts_contract(
            digit,
            ctx.headers,
            tenant_id=tenant_id,
            contract_code=body.factsContractCode,
            contract_version=body.factsContractVersion,
            mdms_schema=settings.facts_contract_mdms_schema,
        )
        validate_registry_schema_code_allowed(body.registrySchemaCode, fc)
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e)) from e
    except RuntimeError as e:
        raise HTTPException(status_code=502, detail=str(e)) from e
    return ValidateRegistrySchemaContractOut(
        valid=True,
        contractCode=body.factsContractCode,
        contractVersion=body.factsContractVersion,
    )


app.include_router(router)

