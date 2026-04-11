# DIGIT demo UI (precursor)

Role-based tabbed UI for the local DIGIT stack, backed by a small proxy to avoid CORS.

## Run

From `digit3/demo-ui`:

```bash
npm run dev
```

- If port `5177` is already in use (e.g. provision-console), stop the other dev server first.

- Web: `http://127.0.0.1:5177`
- Proxy: `http://127.0.0.1:3847` (Vite proxies `/api/*` to it)

## Keycloak (required)

This UI uses **Keycloak OIDC with PKCE** (`keycloak-js`). You need a **public** client in your tenant realm.

For **`npm run dev`**, point **Keycloak base URL** at **`http://localhost:5177/keycloak`** (same host/port as the SPA). Vite proxies **`/keycloak`** to Keycloak (default **`http://127.0.0.1:8080`**) so silent SSO runs in a **same-origin** iframe. Override the upstream with **`VITE_KEYCLOAK_PROXY_TARGET`** if Keycloak listens elsewhere.

The proxy **drops the `Cookie` header** only for Keycloak **`3p-cookies`** URLs so a bloated `localhost` cookie jar does not trigger **431 Request Header Fields Too Large** on that probe.

- **Realm**: the tenant code created by provision (e.g. `PROVLOCAL...`)
- **Client ID**: default `demo-ui` (editable on the login screen)
- **Client type**: Public
- **Standard flow**: On
- **PKCE**: S256
- **Valid redirect URIs** (include every host/port you use in the browser):
  - `http://127.0.0.1:5177/*`
  - `http://localhost:5177/*`
  - If your Keycloak build is strict, also add explicitly: `http://localhost:5177/silent-check-sso.html` and `http://127.0.0.1:5177/silent-check-sso.html` (used for silent SSO / `check-sso`).
- **Web origins**: match the SPA origin(s), e.g. `http://127.0.0.1:5177` and `http://localhost:5177`

After creating it, set the realm + clientId on the **Login** screen and click **Login**.

### `400` on `.../auth?...&prompt=none&redirect_uri=.../login`

That URL means Keycloak-js fell back to a **non-interactive** SSO redirect to `/login` (often after the **third-party cookie** probe). The adapter then sends `prompt=none` to your current page, which Keycloak may reject with **400**. This repo sets **`silentCheckSsoFallback: false`** so the silent iframe still uses `silent-check-sso.html` instead of clearing it. Ensure **Valid redirect URIs** cover `silent-check-sso.html` as above.

### CSP: `frame-ancestors 'self'` / “Unsafe attempt to load URL … from frame”

Keycloak’s login/authorize responses send **`Content-Security-Policy: … frame-ancestors 'self'`**. **`'self'`** is the Keycloak origin (e.g. `http://localhost:8080`). Your SPA runs on **`http://localhost:5177`**, so **embedding Keycloak in an iframe is cross-origin** and the browser **blocks** it. That breaks **`check-sso`** (hidden iframe to `…/auth?prompt=none&redirect_uri=…/silent-check-sso.html`), which can surface as **400**s or **`chrome-error://chromewebdata`** in DevTools.

**Recommended for local `npm run dev`:** Vite proxies **`/keycloak` → Keycloak on port 8080**. Defaults use **`{currentOrigin}/keycloak`** as the Keycloak base URL so all OIDC traffic (including the silent iframe) is **same-origin** as the SPA and `frame-ancestors 'self'` applies to **`http://localhost:5177`**, which matches the parent page.

1. Keep **Keycloak base URL** on the login screen as **`http://localhost:5177/keycloak`** (or `http://127.0.0.1:5177/keycloak` — **same host you use for the app**).
2. In Keycloak, **Valid redirect URIs** must still allow your SPA origin (e.g. `http://localhost:5177/*`).

**Alternative:** leave Keycloak on `:8080` and change the realm **Security defenses → headers / CSP** so **`frame-ancestors`** includes `http://localhost:5177` (less ideal; same-origin proxy is simpler for dev).

If you previously saved **`http://localhost:8080/keycloak`** in the browser, the app **auto-rewrites** it to **`{origin}/keycloak`** on load when the SPA is not on port 8080 (so Keycloak traffic goes through the Vite proxy). You can still use **Save settings & reload** or clear `localStorage` key `digit.demoUi.authConfig.v1` if anything looks cached wrong.

**Self-service:** open **`/register`** to create a new organization (Account API + Keycloak realm). You are returned to **`/login`** with realm pre-filled; add a `demo-ui` public client in that new realm before signing in with Keycloak.

## Local service defaults

The proxy defaults match `deploy/local/docker-compose.yml`:

- Studio `http://127.0.0.1:8107`
- Governance `http://127.0.0.1:8098`
- Coordination `http://127.0.0.1:8090`
- Registry `http://127.0.0.1:8104`
- Workflow `http://127.0.0.1:8085`
- MDMS `http://127.0.0.1:8099`
- IdGen `http://127.0.0.1:8100`
- Boundary `http://127.0.0.1:8093`
- Account `http://127.0.0.1:8094`

You can override via `demo-ui/proxy/.env` (copy from `.env.example`).

## Documentation

- **[Self-service console journey & UI flows](./docs/SELF_SERVICE_CONSOLE_JOURNEY.md)** — Account / Service administrator experience from registration through concrete screens and routes.

## API specifications

See **[API-SPECS.md](./API-SPECS.md)** for proxy routing, headers, health probes, and every endpoint the demo UI calls per screen.

OpenAPI fragments and optional live probes for the same surface live in the sibling **`digit-api-specs`** repo (`specs/<service>/*-demo-ui.yaml`, `conformance/demo_ui_matrix.yaml`; see `digit-api-specs/specs/demo-ui/README.md`).

