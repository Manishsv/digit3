from __future__ import annotations

import json
from contextlib import asynccontextmanager
from dataclasses import dataclass
from pathlib import Path
from typing import Annotated, Any

import httpx
from fastapi import APIRouter, Depends, FastAPI, HTTPException, Query, Request
from fastapi.responses import RedirectResponse
from pydantic import BaseModel, Field

from app.config import Settings
from app.digit_client import DigitClient, mapping_key
from app.index_store import IndexStore, MappingRow, utc_now_iso
from app.security import (
    DEV_ALL_ROLES,
    ROLE_ADMIN,
    ROLE_AUTHORITY_ADMIN,
    ROLE_EMITTER,
    ROLE_PARTICIPANT_ADMIN,
    ROLE_READER,
    ROLE_WRITER,
    decode_jwt_payload,
    digit_headers,
    extract_roles,
    require_any_role,
    require_emitter,
    require_reader,
    require_writer,
)
from app import vocab


@dataclass
class CoordinationContext:
    headers: dict[str, str]
    roles: set[str]


def _compact(d: dict[str, Any]) -> dict[str, Any]:
    return {k: v for k, v in d.items() if v is not None and v != ""}


def _safe_idgen(digit: DigitClient, headers: dict[str, str], template: str) -> str:
    try:
        return digit.idgen_generate(headers, template)
    except RuntimeError as e:
        raise HTTPException(status_code=502, detail=str(e)) from e


def _safe_registry(digit: DigitClient, headers: dict[str, str], schema: str, payload: dict[str, Any]):
    try:
        return digit.registry_create(headers, schema, _compact(payload))
    except RuntimeError as e:
        raise HTTPException(status_code=502, detail=str(e)) from e


async def get_coordination_context(
    request: Request,
    hdrs: Annotated[dict[str, str], Depends(digit_headers)],
) -> CoordinationContext:
    token = hdrs["Authorization"].split(" ", 1)[1].strip()
    settings = request.app.state.settings
    if settings.dev_auth_enabled and token == settings.dev_auth_token:
        roles = set(DEV_ALL_ROLES)
    else:
        payload = decode_jwt_payload(token)
        roles = extract_roles(payload)
    return CoordinationContext(headers=hdrs, roles=roles)


@asynccontextmanager
async def lifespan(app: FastAPI):
    settings = Settings()
    db_path = settings.coordination_db_path
    Path(db_path).parent.mkdir(parents=True, exist_ok=True)
    store = IndexStore(db_path)
    store.init()
    app.state.settings = settings
    app.state.store = store
    app.state.digit = DigitClient(settings)
    yield


app = FastAPI(
    title="DIGIT Land Coordination Layer API",
    version="1.0.0",
    lifespan=lifespan,
    description="Thin coordination APIs over DIGIT Registry, MDMS, and IdGen. See repository docs for tenant provisioning.",
)

coord_router = APIRouter()


@app.get("/", include_in_schema=False)
def root():
    """Browser-friendly entry: `/` had no handler before (404). APIs live under `/coordination/*`."""
    return RedirectResponse(url="/docs")


def _settings(request: Request) -> Settings:
    return request.app.state.settings


def _store(request: Request) -> IndexStore:
    return request.app.state.store


def _digit(request: Request) -> DigitClient:
    return request.app.state.digit


def _validate(settings: Settings, fn: Any, code: str) -> None:
    if not settings.validate_vocab:
        return
    try:
        fn(code)
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e)) from e


class EntityResolveIn(BaseModel):
    entityType: str
    sourceSystem: str
    localId: str
    canonicalId: str | None = None
    aliasType: str | None = None
    status: str | None = None
    metadata: dict[str, Any] = Field(default_factory=dict)


class EntityResolveOut(BaseModel):
    canonicalId: str
    mappingKey: str
    registryRecordId: str | None = None
    status: str


DOMAIN_TEMPLATES: dict[str, str] = {
    "Parcel": "coordination.domain.parcel",
    "Record": "coordination.domain.record",
    "Application": "coordination.domain.application",
    "Case": "coordination.domain.case",
    "Applicant": "coordination.domain.applicant",
    "Document": "coordination.domain.document",
    "MapRef": "coordination.domain.mapref",
    "Participant": "coordination.prt",
    "Payment": "coordination.domain.payment",
    "Deed": "coordination.domain.deed",
}


@coord_router.post("/entity/resolve", response_model=EntityResolveOut)
def resolve_entity(
    body: EntityResolveIn,
    ctx: Annotated[CoordinationContext, Depends(get_coordination_context)],
    request: Request,
):
    require_writer(ctx.roles)
    settings = _settings(request)
    store = _store(request)
    digit = _digit(request)
    _validate(settings, vocab.check_entity_type, body.entityType)
    st = body.status or "ACTIVE"
    _validate(settings, vocab.check_status, st)

    key = mapping_key(body.entityType, body.sourceSystem, body.localId)
    existing = store.get_mapping(key)
    if existing:
        return EntityResolveOut(
            canonicalId=existing.canonical_id,
            mappingKey=key,
            registryRecordId=existing.registry_id,
            status=existing.status,
        )

    if body.canonicalId:
        canonical = body.canonicalId
    else:
        tpl = DOMAIN_TEMPLATES.get(body.entityType)
        if not tpl:
            raise HTTPException(status_code=400, detail=f"No IdGen template for entity type {body.entityType}")
        canonical = _safe_idgen(digit, ctx.headers, tpl)

    now = utc_now_iso()
    row_id = _safe_idgen(digit, ctx.headers, "coordination.eid")
    reg_payload: dict[str, Any] = {
        "id": row_id,
        "entityType": body.entityType,
        "canonicalId": canonical,
        "sourceSystem": body.sourceSystem,
        "localId": body.localId,
        "mappingKey": key,
        "aliasType": body.aliasType or "PRIMARY",
        "status": st,
        "createdTime": now,
        "lastUpdatedTime": now,
        "metadata": body.metadata,
    }
    _, reg_id = _safe_registry(digit, ctx.headers, "coordination.entityId", reg_payload)
    store.put_mapping(
        MappingRow(
            mapping_key=key,
            entity_type=body.entityType,
            source_system=body.sourceSystem,
            local_id=body.localId,
            canonical_id=canonical,
            status=st,
            registry_id=reg_id,
        ),
        created_at=now,
    )
    return EntityResolveOut(canonicalId=canonical, mappingKey=key, registryRecordId=reg_id, status=st)


class LinkIn(BaseModel):
    fromEntityType: str
    fromCanonicalId: str
    relationType: str
    toEntityType: str
    toCanonicalId: str
    sourceSystem: str
    status: str = "ACTIVE"
    confidence: float | None = 1.0
    metadata: dict[str, Any] = Field(default_factory=dict)


class LinkOut(BaseModel):
    id: str
    registryRecordId: str | None = None


@coord_router.post("/link", response_model=LinkOut)
def create_link(
    body: LinkIn,
    ctx: Annotated[CoordinationContext, Depends(get_coordination_context)],
    request: Request,
):
    require_writer(ctx.roles)
    settings = _settings(request)
    store = _store(request)
    digit = _digit(request)
    _validate(settings, vocab.check_entity_type, body.fromEntityType)
    _validate(settings, vocab.check_entity_type, body.toEntityType)
    _validate(settings, vocab.check_relation_type, body.relationType)
    _validate(settings, vocab.check_status, body.status)

    now = utc_now_iso()
    lid = _safe_idgen(digit, ctx.headers, "coordination.lnk")
    reg_payload: dict[str, Any] = {
        "id": lid,
        "fromEntityType": body.fromEntityType,
        "fromCanonicalId": body.fromCanonicalId,
        "relationType": body.relationType,
        "toEntityType": body.toEntityType,
        "toCanonicalId": body.toCanonicalId,
        "sourceSystem": body.sourceSystem,
        "status": body.status,
        "confidence": body.confidence if body.confidence is not None else 1.0,
        "createdTime": now,
        "lastUpdatedTime": now,
        "metadata": body.metadata,
    }
    _, reg_id = _safe_registry(digit, ctx.headers, "coordination.entityLink", reg_payload)
    store.put_link(
        reg_id or lid,
        body.fromEntityType,
        body.fromCanonicalId,
        body.relationType,
        body.toEntityType,
        body.toCanonicalId,
        body.sourceSystem,
        body.status,
        now,
    )
    return LinkOut(id=lid, registryRecordId=reg_id)


class EventIn(BaseModel):
    eventType: str
    subjectEntityType: str
    subjectCanonicalId: str
    correlationId: str
    sourceSystem: str
    actor: str
    schemaRef: str | None = None
    sourceEventId: str | None = None
    payloadRef: str | None = None
    payloadSummary: str | None = None
    status: str = "ACTIVE"
    metadata: dict[str, Any] = Field(default_factory=dict)
    appendTrace: bool = True
    traceAction: str | None = None
    # MUTATION_SUBMITTED glue (optional)
    relatedParcelLocalId: str | None = None
    relatedParcelSourceSystem: str | None = None


class EventOut(BaseModel):
    eventId: str
    registryRecordId: str | None = None
    traceId: str | None = None
    traceRegistryRecordId: str | None = None
    relatedParcelCanonicalId: str | None = None


class GovernanceDecideIn(BaseModel):
    correlationId: str
    requestId: str
    channel: str | None = None
    applicantCanonicalId: str | None = None
    rulesetId: str
    rulesetVersion: str
    registryRecordId: str | None = None
    factsSnapshot: dict[str, Any] = Field(default_factory=dict)
    mdmsFactChecks: list[dict[str, Any]] = Field(default_factory=list)
    factsContractCode: str | None = None
    factsContractVersion: str = "1"


class GovernanceDecideOut(BaseModel):
    decisionId: str
    receiptId: str
    traceRegistryRecordId: str | None = None
    receiptRegistryRecordId: str | None = None
    outcome: dict[str, Any]


def _compact_governance_ruleset_ref(body: GovernanceDecideIn) -> dict[str, Any]:
    out: dict[str, Any] = {"rulesetId": body.rulesetId, "version": body.rulesetVersion}
    if body.registryRecordId:
        out["registryRecordId"] = body.registryRecordId
    return out


def _safe_governance_compute(settings: Settings, headers: dict[str, str], case_id: str, body: GovernanceDecideIn) -> dict[str, Any]:
    url = f"{settings.governance_base_url.rstrip('/')}/governance/v1/decisions:compute"
    payload: dict[str, Any] = {
        "decisionType": "SBL_LICENSE",
        "correlationId": body.correlationId,
        "requestId": body.requestId,
        "channel": body.channel,
        "caseRef": {
            "system": "coordination",
            "entityType": "Case",
            "entityId": case_id,
            "tenantId": headers.get("X-Tenant-ID"),
        },
        "applicantRef": (
            {
                "system": "coordination",
                "entityType": "Applicant",
                "entityId": body.applicantCanonicalId,
                "tenantId": headers.get("X-Tenant-ID"),
            }
            if body.applicantCanonicalId
            else None
        ),
        "ruleset": _compact_governance_ruleset_ref(body),
        "factsSnapshot": body.factsSnapshot,
    }
    if body.mdmsFactChecks:
        payload["mdmsFactChecks"] = body.mdmsFactChecks
    if body.factsContractCode:
        payload["factsContractCode"] = body.factsContractCode
        payload["factsContractVersion"] = body.factsContractVersion or "1"
    if payload.get("applicantRef") is None:
        payload.pop("applicantRef", None)
    with httpx.Client(timeout=60.0) as c:
        r = c.post(url, headers={**headers, "Content-Type": "application/json"}, json=payload)
    if r.status_code >= 400:
        raise HTTPException(status_code=502, detail=f"Governance compute {r.status_code}: {r.text}")
    try:
        return r.json()
    except json.JSONDecodeError:
        return {"_raw": r.text}


def _emit_event_and_trace(
    digit: DigitClient,
    store: IndexStore,
    headers: dict[str, str],
    body: EventIn,
    settings: Settings,
) -> EventOut:
    _validate(settings, vocab.check_event_type, body.eventType)
    _validate(settings, vocab.check_entity_type, body.subjectEntityType)
    _validate(settings, vocab.check_status, body.status)

    now = utc_now_iso()
    eid = _safe_idgen(digit, headers, "coordination.evt")
    reg_payload: dict[str, Any] = {
        "id": eid,
        "eventType": body.eventType,
        "subjectEntityType": body.subjectEntityType,
        "subjectCanonicalId": body.subjectCanonicalId,
        "correlationId": body.correlationId,
        "sourceSystem": body.sourceSystem,
        "actor": body.actor,
        "occurredAt": now,
        "recordedAt": now,
        "status": body.status,
        "metadata": body.metadata,
    }
    if body.sourceEventId:
        reg_payload["sourceEventId"] = body.sourceEventId
    if body.schemaRef:
        reg_payload["schemaRef"] = body.schemaRef
    if body.payloadRef:
        reg_payload["payloadRef"] = body.payloadRef
    if body.payloadSummary:
        reg_payload["payloadSummary"] = body.payloadSummary
    _, reg_id = _safe_registry(digit, headers, "coordination.eventRecord", reg_payload)
    store.put_event(
        reg_id or eid,
        body.eventType,
        body.subjectEntityType,
        body.subjectCanonicalId,
        body.correlationId,
        body.sourceSystem,
        body.actor,
        now,
        now,
        reg_payload,
    )

    trace_id = None
    trace_reg = None
    if body.appendTrace:
        trace_id = _safe_idgen(digit, headers, "coordination.trc")
        action = body.traceAction or f"EVENT:{body.eventType}"
        trace_payload: dict[str, Any] = {
            "id": trace_id,
            "traceType": "AUDIT",
            "entityType": body.subjectEntityType,
            "canonicalId": body.subjectCanonicalId,
            "action": action,
            "actor": body.actor,
            "sourceSystem": body.sourceSystem,
            "occurredAt": now,
            "recordedAt": now,
            "relatedEventId": eid,
            "metadata": {"eventType": body.eventType},
        }
        _, trace_reg = _safe_registry(digit, headers, "coordination.traceRecord", trace_payload)
        store.put_trace(
            trace_reg or trace_id,
            "AUDIT",
            body.subjectEntityType,
            body.subjectCanonicalId,
            action,
            body.actor,
            body.sourceSystem,
            now,
            now,
            eid,
            trace_payload,
        )

    return EventOut(
        eventId=eid,
        registryRecordId=reg_id,
        traceId=trace_id,
        traceRegistryRecordId=trace_reg,
        relatedParcelCanonicalId=None,
    )


@coord_router.post("/event", response_model=EventOut)
def publish_event(
    body: EventIn,
    ctx: Annotated[CoordinationContext, Depends(get_coordination_context)],
    request: Request,
):
    require_emitter(ctx.roles)
    settings = _settings(request)
    store = _store(request)
    digit = _digit(request)

    related_parcel: str | None = None
    if body.eventType == "MUTATION_SUBMITTED" and body.relatedParcelLocalId and body.relatedParcelSourceSystem:
        pk = mapping_key("Parcel", body.relatedParcelSourceSystem, body.relatedParcelLocalId)
        row = store.get_mapping(pk)
        if row:
            related_parcel = row.canonical_id
            link_now = utc_now_iso()
            lid = _safe_idgen(digit, ctx.headers, "coordination.lnk")
            lpayload = {
                "id": lid,
                "fromEntityType": "Application",
                "fromCanonicalId": body.subjectCanonicalId,
                "relationType": "APPLICATION_TO_PARCEL",
                "toEntityType": "Parcel",
                "toCanonicalId": related_parcel,
                "sourceSystem": body.sourceSystem,
                "status": "ACTIVE",
                "confidence": 1.0,
                "createdTime": link_now,
                "lastUpdatedTime": link_now,
                "metadata": {"fromEvent": body.correlationId},
            }
            _, lreg = _safe_registry(digit, ctx.headers, "coordination.entityLink", lpayload)
            store.put_link(
                lreg or lid,
                "Application",
                body.subjectCanonicalId,
                "APPLICATION_TO_PARCEL",
                "Parcel",
                related_parcel,
                body.sourceSystem,
                "ACTIVE",
                link_now,
            )
        else:
            can = _safe_idgen(digit, ctx.headers, "coordination.domain.parcel")
            row_id = _safe_idgen(digit, ctx.headers, "coordination.eid")
            now = utc_now_iso()
            reg_payload = {
                "id": row_id,
                "entityType": "Parcel",
                "canonicalId": can,
                "sourceSystem": body.relatedParcelSourceSystem,
                "localId": body.relatedParcelLocalId,
                "mappingKey": pk,
                "aliasType": "PENDING",
                "status": "PENDING",
                "createdTime": now,
                "lastUpdatedTime": now,
                "metadata": {"reason": "Unresolved parcel at MUTATION_SUBMITTED"},
            }
            _, reg_id = _safe_registry(digit, ctx.headers, "coordination.entityId", reg_payload)
            store.put_mapping(
                MappingRow(
                    mapping_key=pk,
                    entity_type="Parcel",
                    source_system=body.relatedParcelSourceSystem,
                    local_id=body.relatedParcelLocalId,
                    canonical_id=can,
                    status="PENDING",
                    registry_id=reg_id,
                ),
                created_at=now,
            )
            related_parcel = can
            link_now = utc_now_iso()
            lid = _safe_idgen(digit, ctx.headers, "coordination.lnk")
            lpayload = {
                "id": lid,
                "fromEntityType": "Application",
                "fromCanonicalId": body.subjectCanonicalId,
                "relationType": "APPLICATION_TO_PARCEL",
                "toEntityType": "Parcel",
                "toCanonicalId": related_parcel,
                "sourceSystem": body.sourceSystem,
                "status": "PENDING",
                "confidence": 0.5,
                "createdTime": link_now,
                "lastUpdatedTime": link_now,
                "metadata": {"unresolvedParcel": True},
            }
            _, lreg = _safe_registry(digit, ctx.headers, "coordination.entityLink", lpayload)
            store.put_link(
                lreg or lid,
                "Application",
                body.subjectCanonicalId,
                "APPLICATION_TO_PARCEL",
                "Parcel",
                related_parcel,
                body.sourceSystem,
                "PENDING",
                link_now,
            )

    out = _emit_event_and_trace(digit, store, ctx.headers, body, settings)
    out.relatedParcelCanonicalId = related_parcel
    return out


@coord_router.post("/v1/cases/{case_id}/governance:decide", response_model=GovernanceDecideOut)
def governance_decide(
    case_id: str,
    body: GovernanceDecideIn,
    ctx: Annotated[CoordinationContext, Depends(get_coordination_context)],
    request: Request,
):
    require_writer(ctx.roles)
    settings = _settings(request)
    out = _safe_governance_compute(settings, ctx.headers, case_id, body)
    return GovernanceDecideOut(**out)


class TraceIn(BaseModel):
    entityType: str
    canonicalId: str
    action: str
    actor: str
    sourceSystem: str
    relatedEventId: str | None = None
    traceType: str = "AUDIT"
    metadata: dict[str, Any] = Field(default_factory=dict)


class TraceOut(BaseModel):
    traceId: str
    registryRecordId: str | None = None


@coord_router.post("/trace", response_model=TraceOut)
def append_trace(
    body: TraceIn,
    ctx: Annotated[CoordinationContext, Depends(get_coordination_context)],
    request: Request,
):
    require_writer(ctx.roles)
    settings = _settings(request)
    store = _store(request)
    digit = _digit(request)
    _validate(settings, vocab.check_entity_type, body.entityType)

    now = utc_now_iso()
    tid = _safe_idgen(digit, ctx.headers, "coordination.trc")
    reg_payload: dict[str, Any] = {
        "id": tid,
        "traceType": body.traceType,
        "entityType": body.entityType,
        "canonicalId": body.canonicalId,
        "action": body.action,
        "actor": body.actor,
        "sourceSystem": body.sourceSystem,
        "occurredAt": now,
        "recordedAt": now,
        "metadata": body.metadata,
    }
    if body.relatedEventId:
        reg_payload["relatedEventId"] = body.relatedEventId
    _, reg_id = _safe_registry(digit, ctx.headers, "coordination.traceRecord", reg_payload)
    store.put_trace(
        reg_id or tid,
        body.traceType,
        body.entityType,
        body.canonicalId,
        body.action,
        body.actor,
        body.sourceSystem,
        now,
        now,
        body.relatedEventId,
        reg_payload,
    )
    return TraceOut(traceId=tid, registryRecordId=reg_id)


@coord_router.get("/entity/{entity_type}/{canonical_id}/links")
def get_links(
    entity_type: str,
    canonical_id: str,
    ctx: Annotated[CoordinationContext, Depends(get_coordination_context)],
    request: Request,
):
    require_reader(ctx.roles)
    settings = _settings(request)
    _validate(settings, vocab.check_entity_type, entity_type)
    return {
        "entityType": entity_type,
        "canonicalId": canonical_id,
        "links": _store(request).links_for_entity(entity_type, canonical_id),
    }


@coord_router.get("/entity/{entity_type}/{canonical_id}/timeline")
def get_timeline(
    entity_type: str,
    canonical_id: str,
    ctx: Annotated[CoordinationContext, Depends(get_coordination_context)],
    request: Request,
):
    require_reader(ctx.roles)
    settings = _settings(request)
    store = _store(request)
    _validate(settings, vocab.check_entity_type, entity_type)
    events = store.events_for_subject(entity_type, canonical_id)
    traces = store.traces_for_entity(entity_type, canonical_id)
    links = store.links_for_entity(entity_type, canonical_id)
    return {
        "entityType": entity_type,
        "canonicalId": canonical_id,
        "events": events,
        "trace": traces,
        "linkedReferences": links,
    }


class ParticipantIn(BaseModel):
    participantType: str
    participantCode: str
    name: str
    organisation: str | None = None
    jurisdiction: str
    roleType: str | None = None
    endpointRef: str | None = None
    status: str = "ACTIVE"
    effectiveFrom: str
    effectiveTo: str | None = None
    metadata: dict[str, Any] = Field(default_factory=dict)


class ParticipantOut(BaseModel):
    id: str
    registryRecordId: str | None = None


@coord_router.post("/participant", response_model=ParticipantOut)
def register_participant(
    body: ParticipantIn,
    ctx: Annotated[CoordinationContext, Depends(get_coordination_context)],
    request: Request,
):
    require_any_role(ctx.roles, ROLE_PARTICIPANT_ADMIN, ROLE_ADMIN)
    settings = _settings(request)
    store = _store(request)
    digit = _digit(request)
    _validate(settings, vocab.check_participant_type, body.participantType)
    _validate(settings, vocab.check_status, body.status)

    pid = _safe_idgen(digit, ctx.headers, "coordination.prt")
    reg_payload: dict[str, Any] = {
        "id": pid,
        "participantType": body.participantType,
        "participantCode": body.participantCode,
        "name": body.name,
        "jurisdiction": body.jurisdiction,
        "status": body.status,
        "effectiveFrom": body.effectiveFrom,
        "metadata": body.metadata,
    }
    if body.organisation:
        reg_payload["organisation"] = body.organisation
    if body.roleType:
        reg_payload["roleType"] = body.roleType
    if body.endpointRef:
        reg_payload["endpointRef"] = body.endpointRef
    if body.effectiveTo:
        reg_payload["effectiveTo"] = body.effectiveTo
    _, reg_id = _safe_registry(digit, ctx.headers, "coordination.participant", reg_payload)
    store.put_participant(body.participantCode, reg_id or pid, reg_payload)
    return ParticipantOut(id=pid, registryRecordId=reg_id)


class AuthorityIn(BaseModel):
    factType: str
    authoritativeParticipantId: str
    jurisdiction: str
    effectiveFrom: str
    effectiveTo: str | None = None
    allowedEmitters: list[str] = Field(default_factory=list)
    allowedApprovers: list[str] = Field(default_factory=list)
    verificationMode: str = "SOURCE_ATTESTATION"
    status: str = "ACTIVE"
    metadata: dict[str, Any] = Field(default_factory=dict)


class AuthorityOut(BaseModel):
    id: str
    registryRecordId: str | None = None


@coord_router.post("/authority", response_model=AuthorityOut)
def register_authority(
    body: AuthorityIn,
    ctx: Annotated[CoordinationContext, Depends(get_coordination_context)],
    request: Request,
):
    require_any_role(ctx.roles, ROLE_AUTHORITY_ADMIN, ROLE_ADMIN)
    settings = _settings(request)
    store = _store(request)
    digit = _digit(request)
    _validate(settings, vocab.check_fact_type, body.factType)
    _validate(settings, vocab.check_status, body.status)

    aid = _safe_idgen(digit, ctx.headers, "coordination.aut")
    reg_payload: dict[str, Any] = {
        "id": aid,
        "factType": body.factType,
        "authoritativeParticipantId": body.authoritativeParticipantId,
        "jurisdiction": body.jurisdiction,
        "effectiveFrom": body.effectiveFrom,
        "allowedEmitters": body.allowedEmitters,
        "allowedApprovers": body.allowedApprovers,
        "verificationMode": body.verificationMode,
        "status": body.status,
        "metadata": body.metadata,
    }
    if body.effectiveTo:
        reg_payload["effectiveTo"] = body.effectiveTo
    _, reg_id = _safe_registry(digit, ctx.headers, "coordination.authorityRule", reg_payload)
    store.put_authority(
        reg_id or aid,
        body.factType,
        body.jurisdiction,
        body.effectiveFrom,
        body.effectiveTo,
        reg_payload,
    )
    return AuthorityOut(id=aid, registryRecordId=reg_id)


@coord_router.get("/authority")
def resolve_authority(
    ctx: Annotated[CoordinationContext, Depends(get_coordination_context)],
    request: Request,
    factType: str | None = Query(None),
    jurisdiction: str | None = Query(None),
):
    require_reader(ctx.roles)
    rows = _store(request).authority_query(factType, jurisdiction)
    parsed = []
    for r in rows:
        try:
            parsed.append(json.loads(r["payload_json"]))
        except Exception:  # noqa: BLE001
            parsed.append(r)
    return {"rules": parsed}


@coord_router.get("/observability/summary")
def observability_summary(
    ctx: Annotated[CoordinationContext, Depends(get_coordination_context)],
    request: Request,
):
    """Demo: expose index-store views of coordination glue (not for untrusted production use)."""
    require_reader(ctx.roles)
    store = _store(request)
    return {
        "mappings": store.list_all_mappings(),
        "links": store.list_all_links(),
        "events": store.list_all_events(),
        "traces": store.list_all_traces(),
        "participants": store.list_all_participants(),
        "authorityRules": store.list_all_authority(),
    }


@coord_router.get("/vocab/check")
def vocab_check(request: Request, ctx: Annotated[CoordinationContext, Depends(get_coordination_context)]):
    """Optional: compare embedded vocabulary with MDMS coordination.vocabulary rows (category ENTITY_TYPE)."""
    require_reader(ctx.roles)
    settings = _settings(request)
    digit = _digit(request)
    if not settings.mdms_base_url:
        return {"mdms": "skipped"}
    remote = digit.mdms_codes_for_category(ctx.headers, "ENTITY_TYPE")
    local = set(vocab.ENTITY_TYPES)
    return {
        "localEntityTypes": sorted(local),
        "mdmsEntityTypes": sorted(remote),
        "onlyLocal": sorted(local - remote),
        "onlyMdms": sorted(remote - local),
    }


app.include_router(coord_router, prefix="/coordination", tags=["coordination"])
app.include_router(coord_router, prefix="/api/coordination", tags=["api-coordination"])


@app.get("/health")
def health():
    return {"status": "ok"}

