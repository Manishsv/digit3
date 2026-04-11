# DIGIT Cloud Console — self-service journey (Account & Service admin)

This document anchors **who** the console is for, **registration → login → tenant → service** self-service principles, and **concrete UI flows** aligned with the current demo console routes (`web/src/App.tsx`, `web/src/console/navConfig.ts`).

---

## Part A — Experience principles

### Cloud control plane mental model

Users should be able to **arrive without a ticket**, **prove identity**, **get or join a tenant (account)**, and **attach services** with guardrails—not depend on a platform engineer for routine steps.

### Personas

| Persona | Scope | Primary outcomes |
|--------|--------|-------------------|
| **Account administrator** | One tenant (account) | Org settings, **members & roles**, boundaries / master data, tenant-scoped health, optional quotas / integrations. |
| **Service administrator** | One or more **services** inside a tenant | Studio-backed **service registration**, registry linkage, workflow / rules / notifications configuration. |

### Registration models (policy, not just UI)

| Model | Typical use | Console behavior |
|--------|-------------|------------------|
| **Open signup** | Trial / sandbox | Email verify → create personal org or waitlist. |
| **Invite-only** | Production gov / enterprise | Email invite → accept → membership created. |
| **Workforce SSO** | Employees | No self-registration; **Login with organization** (OIDC/SAML). |
| **Join existing tenant** | Partners / contractors | Request access with **tenant / org code** + approval workflow. |

**DIGIT Cloud** will usually combine: **SSO + invite** for production; **open or low-friction signup** for sandbox (same UI patterns, different backend flags).

### Login

- **Primary**: OIDC (e.g. Keycloak) with **MFA** when policy requires.
- **Lab / demo**: optional **dev-local** login (clearly labeled; disabled in production builds).

### After authentication: tenant resolution

Land the user according to membership count:

1. **One tenant** → Home with that tenant pre-selected (top bar **Account**).
2. **Zero tenants** → **Create or join an account** (self-service wizard); block destructive actions until resolved.
3. **Many tenants** → **Account picker** first (same idea as GCP project picker).

Session **JWT tenant** and console **selected account** must converge on a documented rule (today: selection drives `X-Tenant-ID` when set; see `useDigitHeaders`).

---

## Part B — Journeys by persona (narrative)

### Account administrator

1. Landing with clear scope: “Managing **{Account}**.”
2. **Organization** (optional): display name, region, contacts.
3. **Members**: invite users, assign roles (Account admin vs Service admin vs read-only).
4. **Security**: MFA policy, API keys / service accounts (if product supports).
5. **Boundaries / master data**: guided steps with validation (not raw JSON as the only path).
6. **Health**: tenant-scoped slice of dependencies (“what blocks my program”).

Guardrails: RBAC, approval for sensitive actions, audit of invitations.

### Service administrator

1. **Services** home: catalogue + **Register service**.
2. **Guided registration**: service code, display name, module type, **registry** association, status.
3. **In-product ID glossary**: **service code** (APIs / workflow) vs **Studio record id** (internal row) vs **registry id**.
4. **Configuration**: rules, workflow, notifications—progressive disclosure; safe **Validate** / **Test** actions.
5. Errors actionable without platform access.

### Overlap

Small tenant: one user holds both roles → single nav, sections gated by role.  
Large tenant: split responsibilities strictly in RBAC and UI (service admin cannot invite org users unless delegated).

---

## Part C — Concrete UI flows and screens

Below: **screen** = route + primary UI state. **Current** = implemented today in `digit3/demo-ui/web`. **Target** = self-service evolution (same route where possible).

### C1. Unauthenticated — registration & login

| Screen | Route | Purpose | Current | Target |
|--------|-------|---------|---------|--------|
| **Login** | `/login` | OIDC PKCE or dev-local; realm/client hints | Yes (`LoginPage`) | Add **“Create account”** / **“Join with code”** links to new flows when backend exists; prod hides dev-local. |
| **Register (sandbox)** | `/register` | Org name + email → `POST /account/v1` (creates realm + tenant) | **Yes** (`RegisterPage`) | Add email verify / policy flags when product hardens. |
| **Accept invite** | `/invite/:token` *(new)* | Consume one-time token → join tenant | No | Deep link from email. |
| **SSO broker** | external | IdP redirect | Via Keycloak | Document realm-per-tenant vs central realm. |

**Flow — returning user**

1. `/login` → Keycloak → redirect back with tokens.
2. App resolves **tenant list** (target: directory API; current: manual “known accounts” + JWT tenant).
3. Branch: **0 / 1 / N** tenants → C2.

**Flow — new sandbox user (target)**

1. `/register` → verify email → **Create organization** (tenant id generated or chosen per policy).
2. Auto-select new tenant → `/` Overview with **first-run checklist** (C4).

---

### C2. Post-login — tenant resolution (shell)

| Screen | Route | Purpose | Current | Target |
|--------|-------|---------|---------|--------|
| **Account picker** | modal or `/welcome` *(new)* | Choose tenant when N > 1 | **Yes** — `/welcome` lists accounts; top bar persists choice | Full-screen or modal on first load; persist last selection. |
| **No access** | `/welcome` *(new)* | 0 tenants: create / join / request | **Yes** — `/welcome` + `TenantGate` redirect; `/platform` allowed to add first account | Directory API when available. |
| **Console shell** | `/*` under `RequireAuth` | Top bar + nav + outlet | Yes (`AppShell`) | Inject **first-run** banner until tenant resolved. |

**Flow — 0 tenants (target)**

1. After login → `/welcome` with CTAs: **Create organization**, **Join with organization code**, **Contact support** (link only).
2. On success → populate membership → redirect `/` with tenant selected.

**Flow — 1 tenant**

1. Auto-set selected account to sole membership → `/`.

**Flow — N tenants**

1. Show picker once → persist → `/`.

---

### C3. Account administrator — screens

| Screen | Route | Tabs / sections | Current | Target |
|--------|-------|-----------------|---------|--------|
| **Account home** | `/account` | Summary + shortcuts | Partial (`AccountAdminPage`) | **Dashboard**: members count, boundaries status, linked services count, alerts. |
| **Members** | `/account/members` *(new)* | List, invite, roles | No | Table + Invite modal; map to Keycloak / Account API. |
| **Boundaries** | `/account/boundaries` *(new)* | CRUD wizard | Placeholder tab today | Replace placeholder with guided UI. |
| **Security** | `/account/security` *(new)* | MFA readout, keys | No | Read-only first; then API keys. |
| **Platform directory** | `/platform` tab Accounts | Local directory | Yes | **Replace localStorage** with server-backed tenant list user belongs to + optional **platform** health for admins only. |

**Flow — invite user (target)**

1. `/account` → **Members** → **Invite** → email + role template → POST invite → show pending state until accept (C1 invite link).

**Flow — boundaries (target)**

1. `/account/boundaries` → **Create boundary** → form validated against Boundary API → success toast → list row.

---

### C4. Service administrator — screens

| Screen | Route | Tabs / sections | Current | Target |
|--------|-------|-----------------|---------|--------|
| **Services catalogue** | `/service` | Catalogue table | Yes | Add row actions: **Open**, **Configure**. |
| **Register service** | `/service/register` *(new)* | Multi-step wizard | No | Steps: identity → registry → review → POST Studio. |
| **Service details** | `/service` tab Details | DL + raw JSON | Yes | Keep; add **Edit** (limited fields) when API allows. |
| **Service rules / workflow / notifications** | `/service` tabs | Placeholders | Yes | Replace with forms + “Test connection” using safe probes. |

**Flow — register service (target)**

1. Top bar: select **Account**.
2. `/service` → **Register service** → wizard → success → auto-select new **Service** in top bar → deep-link to **Workflow** tab with empty-state guidance.

**Flow — configure existing service**

1. Top bar: **Account** + **Service** → `/service` → tab **Workflow** / **Rules** / **Notifications** → save with validation.

---

### C5. Cross-cutting console chrome (already aligned)

| Element | Behavior |
|---------|----------|
| **Top bar — Account** | Selects tenant for `X-Tenant-ID` (with JWT fallback); **Manage** → Platform directory. |
| **Top bar — Service** | Selects `service_code` for contextual pages; **Open** → Services. |
| **Breadcrumbs** | `DIGIT Console / …` from `breadcrumbsForPath`. |
| **Left nav** | `NAV_GROUPS`: Home, Administration, Configuration, Channel & operations. |

**Target enhancements**

- **First-run checklist** on `/` when tenant new: “Add members”, “Register a service”, “Link registry”.
- **Notifications** icon area (future): pending invites, failed health checks.

---

### C6. RBAC vs nav (reference)

Map Keycloak realm roles (existing) to **visible nav** and **route guards**:

| Role (examples) | Platform | Account | Services | Regulator / Registries | Ops tabs |
|------------------|----------|---------|----------|------------------------|----------|
| `COORDINATION_ADMIN` | Yes | Yes | Yes | Yes | per existing rules |
| `COORDINATION_WRITER` | No | No | Yes | Yes | Operator, etc. |
| `COORDINATION_READER` | No | No | Read-only target | TBD | Audit read |

**Target:** route-level `RequiredRoles` wrapper matching `navConfig` so deep links cannot bypass policy.

---

### C7. Implementation backlog (ordered)

1. **Tenant resolution** route `/welcome` + post-login redirect logic (stub OK with feature flag).
2. **Account → Members** (`/account/members`) minimal invite UI (stub API).
3. **Service → Register** (`/service/register`) wizard shell + Studio POST when contract is stable.
4. **Persist tenants** from API instead of only `localStorage` (`selection.tsx` evolution).
5. **Route guards** aligned with `navConfig`.
6. **First-run checklist** component on Overview.

---

## Revision history

| Date | Change |
|------|--------|
| 2026-04-11 | Initial journey + concrete UI flow map. |
| 2026-04-11 | Implemented `/welcome`, `TenantGate`, and post-login redirect (`tenantResolution.ts`, `LoginPage`). |
