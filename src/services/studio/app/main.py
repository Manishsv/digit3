from __future__ import annotations

from contextlib import asynccontextmanager
from pathlib import Path
from typing import Annotated, Any

import httpx
from fastapi import APIRouter, Depends, FastAPI, HTTPException, Query, Request
from fastapi.responses import RedirectResponse
from pydantic import BaseModel, Field

from app.config import Settings
from app.digit_client import DigitClient
from app.index_store import IndexStore
from app.security import StudioContext, digit_headers, extract_roles, decode_jwt_payload
from app.service_definition import ServiceDefinitionDoc


def _settings(request: Request) -> Settings:
    return request.app.state.settings


def _digit(request: Request) -> DigitClient:
    return request.app.state.digit


def _store(request: Request) -> IndexStore:
    return request.app.state.store


def _compact(d: dict[str, Any]) -> dict[str, Any]:
    return {k: v for k, v in d.items() if v is not None and v != ""}


async def get_studio_context(
    request: Request,
    hdrs: Annotated[dict[str, str], Depends(digit_headers)],
) -> StudioContext:
    token = hdrs["Authorization"].split(" ", 1)[1].strip()
    settings = _settings(request)
    if settings.dev_auth_enabled and token == settings.dev_auth_token:
        roles = {"STUDIO_ADMIN"}
    else:
        payload = decode_jwt_payload(token)
        roles = extract_roles(payload)
    return StudioContext(headers=hdrs, roles=roles)


@asynccontextmanager
async def lifespan(app: FastAPI):
    settings = Settings()
    Path(settings.studio_db_path).parent.mkdir(parents=True, exist_ok=True)
    store = IndexStore(settings.studio_db_path)
    store.init()
    app.state.settings = settings
    app.state.store = store
    app.state.digit = DigitClient(settings)
    yield


app = FastAPI(
    title="DIGIT Studio Service API (demo)",
    version="1.0.0",
    lifespan=lifespan,
    description="Tenant admin control-plane for service configuration (service directory, bundles, provisioning jobs).",
)

router = APIRouter(prefix="/studio/v1", tags=["studio"])


@app.get("/", include_in_schema=False)
def root():
    return RedirectResponse(url="/docs")


@app.get("/health")
def health():
    return {"status": "ok"}


@router.get("/whoami")
def whoami(ctx: Annotated[StudioContext, Depends(get_studio_context)]):
    return {
        "tenantId": ctx.headers.get("X-Tenant-ID"),
        "clientId": ctx.headers.get("X-Client-ID"),
        "roles": sorted(ctx.roles),
    }


class ServiceUpsertIn(BaseModel):
    serviceCode: str = Field(pattern="^[A-Z0-9_\\.\\-]{2,64}$")
    name: str
    moduleType: str
    status: str = "ENABLED"
    description: str | None = None
    metadata: dict[str, Any] = Field(default_factory=dict)
    serviceDefinition: ServiceDefinitionDoc | None = Field(
        default=None,
        description="Shared control-plane config: intake, governance, workflow, audit, appeals, etc.",
    )


class ServiceUpsertOut(BaseModel):
    serviceCode: str
    id: str
    registryRecordId: str | None = None


@router.post("/services", response_model=ServiceUpsertOut)
def upsert_service(
    body: ServiceUpsertIn,
    ctx: Annotated[StudioContext, Depends(get_studio_context)],
    request: Request,
):
    tenant_id = ctx.headers.get("X-Tenant-ID") or "UNKNOWN"
    digit = _digit(request)
    store = _store(request)

    sid = digit.idgen_generate(ctx.headers, "studio.svc")
    payload = _compact(
        {
            "id": sid,
            "tenantId": tenant_id,
            "serviceCode": body.serviceCode,
            "name": body.name,
            "moduleType": body.moduleType,
            "status": body.status,
            "description": body.description,
            "metadata": body.metadata,
            "serviceDefinition": body.serviceDefinition.model_dump(mode="json", exclude_none=True)
            if body.serviceDefinition
            else None,
        }
    )
    _, reg_id = digit.registry_create(ctx.headers, "studio.service", payload)
    store.upsert_service(tenant_id, body.serviceCode, reg_id, body.status, payload)
    return ServiceUpsertOut(serviceCode=body.serviceCode, id=sid, registryRecordId=reg_id)


@router.get("/services")
def list_services(
    ctx: Annotated[StudioContext, Depends(get_studio_context)],
    request: Request,
    limit: int = Query(200, ge=1, le=1000),
):
    tenant_id = ctx.headers.get("X-Tenant-ID") or "UNKNOWN"
    return {"services": _store(request).list_services(tenant_id, limit=limit)}


class BundleCreateIn(BaseModel):
    serviceCode: str = Field(pattern="^[A-Z0-9_\\.\\-]{2,64}$")
    version: str = Field(pattern="^[0-9A-Za-z\\.\\-]{1,32}$")
    status: str = "DRAFT"
    # Minimal bundle contents for now: rulesets (YAML DSL) to publish via governance
    rulesets: list[dict[str, Any]] = Field(default_factory=list, description="Each: {yamlText, issuerAuthorityId, humanVersion?}")
    metadata: dict[str, Any] = Field(default_factory=dict)
    factsContractCode: str | None = None
    factsContractVersion: str = "1"
    serviceDefinition: ServiceDefinitionDoc | None = Field(
        default=None,
        description="Optional definition snapshot/version slice for this bundle (same shape as studio.service).",
    )


class BundleCreateOut(BaseModel):
    bundleId: str
    registryRecordId: str | None = None


@router.post("/bundles", response_model=BundleCreateOut)
def create_bundle(
    body: BundleCreateIn,
    ctx: Annotated[StudioContext, Depends(get_studio_context)],
    request: Request,
):
    tenant_id = ctx.headers.get("X-Tenant-ID") or "UNKNOWN"
    digit = _digit(request)
    store = _store(request)

    bid = digit.idgen_generate(ctx.headers, "studio.bndl")
    payload = _compact(
        {
            "id": bid,
            "tenantId": tenant_id,
            "bundleId": bid,
            "serviceCode": body.serviceCode,
            "version": body.version,
            "status": body.status,
            "rulesets": body.rulesets,
            "metadata": body.metadata,
            "factsContractCode": body.factsContractCode,
            "factsContractVersion": body.factsContractVersion,
            "serviceDefinition": body.serviceDefinition.model_dump(mode="json", exclude_none=True)
            if body.serviceDefinition
            else None,
        }
    )
    _, reg_id = digit.registry_create(ctx.headers, "studio.bundle", payload)
    store.upsert_bundle(tenant_id, bid, body.serviceCode, body.version, reg_id, body.status, payload)
    return BundleCreateOut(bundleId=bid, registryRecordId=reg_id)


@router.get("/bundles")
def list_bundles(
    ctx: Annotated[StudioContext, Depends(get_studio_context)],
    request: Request,
    serviceCode: str | None = Query(None),
    limit: int = Query(200, ge=1, le=1000),
):
    tenant_id = ctx.headers.get("X-Tenant-ID") or "UNKNOWN"
    return {"bundles": _store(request).list_bundles(tenant_id, service_code=serviceCode, limit=limit)}


class JobCreateIn(BaseModel):
    serviceCode: str
    bundleId: str | None = None
    action: str = Field(description="Currently supports: APPLY_RULESETS")
    # Optional: apply these rulesets directly (same shape as bundle.rulesets items) without loading bundle from Registry
    rulesets: list[dict[str, Any]] | None = None
    factsContractCode: str | None = None
    factsContractVersion: str = "1"
    requireFactsContract: bool = True


class JobCreateOut(BaseModel):
    jobId: str
    status: str
    results: dict[str, Any] = Field(default_factory=dict)


def _publish_ruleset(governance_base_url: str, headers: dict[str, str], body: dict[str, Any]) -> dict[str, Any]:
    url = f"{governance_base_url.rstrip('/')}/governance/v1/rulesets"
    with httpx.Client(timeout=120.0) as c:
        r = c.post(url, headers={**headers, "Content-Type": "application/json"}, json=body)
    if r.status_code >= 400:
        raise RuntimeError(f"Governance publish {r.status_code}: {r.text}")
    try:
        return r.json()
    except Exception:  # noqa: BLE001
        return {"_raw": r.text}


def _registry_data_object(read_resp: dict[str, Any]) -> dict[str, Any] | None:
    """Best-effort unwrap Registry _get response to the stored business payload."""
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
    if isinstance(nested, dict) and ("rulesets" in nested or "bundleId" in nested):
        return nested
    return inner


def _governance_ruleset_request(
    rs: dict[str, Any],
    *,
    bundle_facts_contract_code: str | None = None,
    bundle_facts_contract_version: str = "1",
) -> dict[str, Any]:
    """Build JSON body for POST /governance/v1/rulesets from a bundle ruleset item."""
    yaml_text = rs.get("yamlText")
    if not isinstance(yaml_text, str) or not yaml_text.strip():
        raise ValueError("ruleset item missing yamlText (string)")
    issuer = rs.get("issuerAuthorityId")
    if not isinstance(issuer, str) or not issuer.strip():
        raise ValueError("ruleset item missing issuerAuthorityId (string)")
    out: dict[str, Any] = {
        "yamlText": yaml_text,
        "issuerAuthorityId": issuer,
        "status": rs.get("status") or "ACTIVE",
        "policyDocuments": rs.get("policyDocuments") if isinstance(rs.get("policyDocuments"), list) else [],
    }
    if isinstance(rs.get("humanVersion"), str):
        out["humanVersion"] = rs["humanVersion"]
    if isinstance(rs.get("effectiveFrom"), str):
        out["effectiveFrom"] = rs["effectiveFrom"]
    if isinstance(rs.get("effectiveTo"), str):
        out["effectiveTo"] = rs["effectiveTo"]
    fcc = rs.get("factsContractCode") or bundle_facts_contract_code
    fcv = rs.get("factsContractVersion") or bundle_facts_contract_version
    if isinstance(fcc, str) and fcc.strip():
        out["factsContractCode"] = fcc.strip()
        out["factsContractVersion"] = str(fcv).strip() or "1"
    return out


@router.post("/jobs", response_model=JobCreateOut)
def create_job(
    body: JobCreateIn,
    ctx: Annotated[StudioContext, Depends(get_studio_context)],
    request: Request,
):
    tenant_id = ctx.headers.get("X-Tenant-ID") or "UNKNOWN"
    digit = _digit(request)
    store = _store(request)
    settings = _settings(request)

    jid = digit.idgen_generate(ctx.headers, "studio.job")
    job_payload: dict[str, Any] = {
        "jobId": jid,
        "tenantId": tenant_id,
        "serviceCode": body.serviceCode,
        "bundleId": body.bundleId,
        "action": body.action,
        "status": "STARTED",
    }
    store.create_job(tenant_id, jid, body.serviceCode, body.bundleId, "STARTED", job_payload)

    results: dict[str, Any] = {"publishedRulesets": []}
    final_status = "SUCCEEDED"

    if body.action != "APPLY_RULESETS":
        final_status = "FAILED"
        job_payload["status"] = "FAILED"
        job_payload["error"] = "Unsupported action"
        store.finish_job(tenant_id, jid, "FAILED", job_payload)
        return JobCreateOut(jobId=jid, status="FAILED", results={"error": "Unsupported action"})

    bundle_contract_code: str | None = None
    bundle_contract_version = "1"

    rulesets_to_apply: list[dict[str, Any]] = []
    if body.rulesets:
        rulesets_to_apply = list(body.rulesets)
    elif body.bundleId:
        bundle_row = store.get_bundle(tenant_id, body.bundleId)
        if not bundle_row:
            final_status = "FAILED"
            job_payload["status"] = "FAILED"
            job_payload["error"] = f"Unknown bundleId: {body.bundleId}"
            store.finish_job(tenant_id, jid, "FAILED", job_payload)
            return JobCreateOut(jobId=jid, status="FAILED", results={"error": job_payload["error"]})
        if bundle_row.get("service_code") != body.serviceCode:
            final_status = "FAILED"
            job_payload["status"] = "FAILED"
            job_payload["error"] = "bundleId belongs to a different serviceCode"
            store.finish_job(tenant_id, jid, "FAILED", job_payload)
            return JobCreateOut(jobId=jid, status="FAILED", results={"error": job_payload["error"]})
        reg_id = bundle_row.get("registry_id")
        bundle_data: dict[str, Any] | None = None
        if reg_id:
            try:
                raw = digit.registry_read(ctx.headers, "studio.bundle", str(reg_id))
                bundle_data = _registry_data_object(raw)
            except RuntimeError as e:
                # Some Registry deployments may not expose the `_get` helper endpoint.
                # We can still apply from the local index payload for newly-created bundles.
                msg = str(e)
                if " 404" in msg or "404 page not found" in msg:
                    bundle_data = None
                else:
                    final_status = "FAILED"
                    job_payload["status"] = "FAILED"
                    job_payload["error"] = msg
                    store.finish_job(tenant_id, jid, "FAILED", job_payload)
                    return JobCreateOut(jobId=jid, status="FAILED", results={"error": msg})
        if not bundle_data and isinstance(bundle_row.get("payload"), dict):
            bundle_data = bundle_row["payload"]
        if not bundle_data:
            final_status = "FAILED"
            job_payload["status"] = "FAILED"
            job_payload["error"] = "Could not load bundle payload"
            store.finish_job(tenant_id, jid, "FAILED", job_payload)
            return JobCreateOut(jobId=jid, status="FAILED", results={"error": job_payload["error"]})
        rs = bundle_data.get("rulesets")
        if not isinstance(rs, list):
            final_status = "FAILED"
            job_payload["status"] = "FAILED"
            job_payload["error"] = "Bundle has no rulesets array"
            store.finish_job(tenant_id, jid, "FAILED", job_payload)
            return JobCreateOut(jobId=jid, status="FAILED", results={"error": job_payload["error"]})
        rulesets_to_apply = [x for x in rs if isinstance(x, dict)]
        if isinstance(bundle_data.get("factsContractCode"), str) and bundle_data["factsContractCode"].strip():
            bundle_contract_code = bundle_data["factsContractCode"].strip()
        if isinstance(bundle_data.get("factsContractVersion"), str) and bundle_data["factsContractVersion"].strip():
            bundle_contract_version = bundle_data["factsContractVersion"].strip()
    else:
        final_status = "FAILED"
        job_payload["status"] = "FAILED"
        job_payload["error"] = "Provide bundleId or rulesets[]"
        store.finish_job(tenant_id, jid, "FAILED", job_payload)
        return JobCreateOut(jobId=jid, status="FAILED", results={"error": job_payload["error"]})

    if not rulesets_to_apply:
        final_status = "FAILED"
        job_payload["status"] = "FAILED"
        job_payload["error"] = "No rulesets to apply"
        store.finish_job(tenant_id, jid, "FAILED", job_payload)
        return JobCreateOut(jobId=jid, status="FAILED", results={"error": job_payload["error"]})

    eff_contract_code = body.factsContractCode or bundle_contract_code
    if body.factsContractCode:
        eff_contract_version = body.factsContractVersion or "1"
    else:
        eff_contract_version = bundle_contract_version

    if body.requireFactsContract and not (isinstance(eff_contract_code, str) and eff_contract_code.strip()):
        final_status = "FAILED"
        job_payload["status"] = "FAILED"
        job_payload["error"] = "Missing factsContractCode (set on job, bundle, or each ruleset; or set requireFactsContract=false)"
        store.finish_job(tenant_id, jid, final_status, job_payload)
        return JobCreateOut(jobId=jid, status=final_status, results={"error": job_payload["error"]})

    gov_url = settings.governance_base_url
    published: list[dict[str, Any]] = []
    try:
        for i, rs in enumerate(rulesets_to_apply):
            try:
                req = _governance_ruleset_request(
                    rs,
                    bundle_facts_contract_code=eff_contract_code,
                    bundle_facts_contract_version=eff_contract_version,
                )
            except ValueError as ve:
                raise RuntimeError(f"rulesets[{i}]: {ve}") from ve
            out = _publish_ruleset(gov_url, ctx.headers, req)
            published.append(out)
    except RuntimeError as e:
        final_status = "FAILED"
        results["publishedRulesets"] = published
        results["error"] = str(e)
        job_payload["status"] = final_status
        job_payload["results"] = results
        job_payload["error"] = str(e)
        store.finish_job(tenant_id, jid, final_status, job_payload)
        raise HTTPException(status_code=502, detail=str(e)) from e

    results["publishedRulesets"] = published
    if body.bundleId:
        b = store.get_bundle(tenant_id, body.bundleId)
        if b and isinstance(b.get("payload"), dict):
            p = dict(b["payload"])
            p["status"] = "APPLIED"
            store.upsert_bundle(
                tenant_id,
                body.bundleId,
                str(b["service_code"]),
                str(b["version"]),
                b.get("registry_id"),
                "APPLIED",
                p,
            )

    job_payload["status"] = final_status
    job_payload["results"] = results
    store.finish_job(tenant_id, jid, final_status, job_payload)
    return JobCreateOut(jobId=jid, status=final_status, results=results)


@router.get("/jobs/{job_id}")
def get_job(
    job_id: str,
    ctx: Annotated[StudioContext, Depends(get_studio_context)],
    request: Request,
):
    tenant_id = ctx.headers.get("X-Tenant-ID") or "UNKNOWN"
    row = _store(request).get_job(tenant_id, job_id)
    if not row:
        raise HTTPException(status_code=404, detail="Unknown jobId")
    return row


app.include_router(router)

