# Demo UI — API specifications

This document describes how the **demo UI** talks to DIGIT: the **browser-facing URL shape**, the **Node proxy**, **required headers**, **health probes**, and **endpoints used per screen**. It is aligned with the current code under `demo-ui/web/src` and `demo-ui/proxy/`.

## Architecture

```
Browser  →  Vite dev server (e.g. :5177)
              GET/POST  /api/{service}/…  →  proxy (e.g. :3847)
                                                strip /api/{service}
                                                forward to STUDIO_BASE_URL, etc.
```

- **Browser path**: always `/api/{service}{upstreamPath}` where `service` is one of: `studio`, `governance`, `coordination`, `registry`, `workflow`, `mdms`, `idgen`, `boundary`, `account`, `keycloak`.
- **Proxy**: `demo-ui/proxy/index.js` — `createProxyMiddleware` rewrites `/api/{service}` → upstream root (path after the prefix is sent as-is to the target).

## Upstream base URLs (local defaults)

Configure in `demo-ui/proxy/.env` (see `.env.example`).

| Env variable | Default upstream |
|--------------|------------------|
| `STUDIO_BASE_URL` | `http://127.0.0.1:8107` |
| `GOVERNANCE_BASE_URL` | `http://127.0.0.1:8098` |
| `COORDINATION_BASE_URL` | `http://127.0.0.1:8090` |
| `REGISTRY_BASE_URL` | `http://127.0.0.1:8104` |
| `WORKFLOW_BASE_URL` | `http://127.0.0.1:8085` |
| `MDMS_BASE_URL` | `http://127.0.0.1:8099` |
| `IDGEN_BASE_URL` | `http://127.0.0.1:8100` |
| `BOUNDARY_BASE_URL` | `http://127.0.0.1:8093` |
| `ACCOUNT_BASE_URL` | `http://127.0.0.1:8094` |
| `KEYCLOAK_BASE_URL` | `http://127.0.0.1:8080` |

## Request headers (client → proxy → upstream)

The UI builds headers via `useDigitHeaders()` (`web/src/lib/digitApi.ts`):

| Header | Source | Notes |
|--------|--------|--------|
| `Authorization` | Keycloak access token (or dev-local placeholder) | `Bearer …` when present |
| `X-Tenant-ID` | **Selected account** (`SelectionProvider`) if set, else realm from token `iss` | Tenant scoping for DIGIT APIs |
| `X-Client-ID` | Username / dev client id | Also sent as `X-Client-Id` for compatibility |

Proxy `onProxyReq` forwards `X-Tenant-ID` and `X-Client-ID` explicitly.

**CORS** (proxy): if `WEB_ORIGIN` is unset, any `http://localhost:*` or `http://127.0.0.1:*` origin is allowed.

## Client helpers

| Helper | Behavior |
|--------|----------|
| `digitFetch(service, path, init)` | `fetch('/api/' + service + path)` |
| `digitJson(service, path, init)` | Same; status ≥ 400 throws `ApiError`; body parsed as JSON or `{ _raw: text }` |
| `probeDigitService(service, candidates, headers)` | Tries each path until success, 401/403 (reachable), or non-404 &lt; 500; used for services without root `/health` |

## Health and liveness (Platform → Shared services)

Used by `PlatformAdminPage`.

| Service | Probe (upstream path) | Notes |
|---------|------------------------|--------|
| studio | `GET /health` | |
| governance | `GET /health` | |
| coordination | `GET /health` | |
| registry | `GET /health` | |
| workflow | `GET /workflow/v1/process?code=PGR67` then `GET /workflow/v1/process` | No root `/health` in workflow router |
| mdms | `GET /actuator/health` then `POST /mdms-v2/v1/_search` body `{}` | Spring / MDMS paths |
| idgen | `GET /idgen/health` then `GET /health` | Matches container healthcheck |
| boundary | `GET /boundary/v1` | API under context path |
| account | `GET /actuator/health` then `GET /health` | JVM service |

## Endpoints by screen

Paths below are **upstream** paths (what hits the service after the proxy). Method is GET unless noted.

### Platform — Accounts

No backend API; persists account list in `localStorage` (`selection.tsx`).

### Service setup (`ServiceAdminPage`)

| Method | Service | Path |
|--------|---------|------|
| GET | studio | `/studio/v1/services` |

### Regulator — rules (`StudioConfiguratorPage`, route `/regulator`)

| Method | Service | Path | Body (summary) |
|--------|---------|------|----------------|
| POST | studio | `/studio/v1/services` | `serviceCode`, `name`, `moduleType`, `status`, `metadata` |
| POST | studio | `/studio/v1/bundles` | `serviceCode`, `version`, `status`, `factsContractCode`, `factsContractVersion`, `rulesets[]`, `metadata` |
| POST | studio | `/studio/v1/jobs` | `serviceCode`, `bundleId`, `action: APPLY_RULESETS` |

### Registries (`RegistriesPage`, route `/registries`)

| Method | Service | Path | Notes |
|--------|---------|------|--------|
| GET | registry | `/registry/v1/schema/{schemaCode}` | Schema metadata |
| GET | registry | `/registry/v1/schema/{schemaCode}/data/_registry?registryId=` | Single record |
| POST | registry | `/registry/v1/schema/{schemaCode}/data/_search` | Body `{ query: { serviceRequestId, serviceCode? } }` — shape depends on Registry API support |

### Citizen — intake (`CitizenPage`)

| Method | Service | Path | Body (summary) |
|--------|---------|------|----------------|
| GET | workflow | `/workflow/v1/process?code={serviceCode}` | |
| POST | workflow | `/workflow/v1/transition` | `processId`, `entityId` (caseId), `action: APPLY`, `init: true`, `comment`, `attributes.roles` |
| POST | registry | `/registry/v1/schema/complaints.case/data` | `data`: `serviceRequestId`, `tenantId`, `serviceCode`, `processId`, `workflowInstanceId`, `description`, `applicationStatus` |
| GET | registry | `/registry/v1/schema/complaints.case/data/_registry?registryId=` | Load by registry id |

Successful submit appends to `localStorage` key `digit.demoUi.cases.v1` for Operator case list.

### Operator (`OperatorPage`)

| Method | Service | Path | Body (summary) |
|--------|---------|------|----------------|
| POST | coordination | `/coordination/v1/cases/{caseId}/governance:decide` | `correlationId`, `requestId`, `channel`, `rulesetId`, `rulesetVersion`, `registryRecordId?`, `factsContractCode`, `factsContractVersion`, `factsSnapshot`, `mdmsFactChecks` |
| GET | workflow | `/workflow/v1/process?code={serviceCode}` | |
| POST | workflow | `/workflow/v1/transition` | `processId`, `entityId`, `action`, `init: false`, `comment`, `attributes.roles: OPERATOR` |
| GET | workflow | `/workflow/v1/transition?entityId=&processId=&history=true` | History |

### Appellate (`AppellatePage`)

| Method | Service | Path | Body (summary) |
|--------|---------|------|----------------|
| POST | governance | `/governance/v1/appeals` | `receiptId`, `decisionId`, `filedBy`, `grounds`, `status`, `metadata` |
| POST | governance | `/governance/v1/orders` | `appealId`, `decisionId`, `receiptId`, `issuedBy`, `outcome`, `instructions`, `metadata` |
| POST | governance | `/governance/v1/decisions:recompute` | Full recompute payload including `parentDecisionId`, `caseRef`, `ruleset`, `factsSnapshot`, contracts |
| GET | governance | `/governance/v1/decisions/{id}` | Used by “Fetch receipt” control with `receiptId` field (demo wiring; prefer `GET /governance/v1/receipts/{receiptId}` for receipts — see Auditor) |

### Auditor (`AuditorPage`)

| Method | Service | Path |
|--------|---------|------|
| GET | governance | `/governance/v1/receipts/{receiptId}` |
| GET | governance | `/governance/v1/decisions/{decisionId}/trace` |

### Legacy / unused in router

- `RegulatorPage.tsx` — not mounted in `App.tsx` (Operator owns decide); file may still reference coordination decide.
- `CsrPage.tsx` — not mounted; same workflow calls as Operator workflow section.
- `RegistrarPage.tsx` — superseded by `RegistriesPage`.

## OpenAPI / authoritative contracts

For full service contracts, use the **service-owned** OpenAPI / docs in the `digit3` repo (e.g. `docs/services/workflow/workflow-3.0.0.yaml`) and each service’s own spec. This file only documents **what the demo UI actually calls**.

### Machine-readable specs + conformance (`digit-api-specs`)

Use the existing **`digit-api-specs`** repo (sibling of `digit3`, e.g. `../digit-api-specs`):

- OpenAPI fragments: `specs/<service>/*-demo-ui.yaml` (e.g. `specs/governance/governance-demo-ui.yaml`, `specs/coordination/coordination-demo-ui.yaml`; index in `specs/demo-ui/README.md`)
- Live probe matrix: `conformance/demo_ui_matrix.yaml` + `conformance/test_demo_ui_probes.py` (`DEMO_UI_PROBES=1 pytest …`)

Keep those files in sync when you change demo UI call patterns or Platform health probes.

## Changelog (recent)

- Platform health: workflow / mdms / idgen / boundary / account use **context-specific probes**, not bare `/health`.
- Tenant header: `X-Tenant-ID` prefers **selected account** from UI state over token-derived tenant.
- Registries screen: schema + data + optional `_search`.
- Operator: coordination `governance:decide` + workflow transition/history.
