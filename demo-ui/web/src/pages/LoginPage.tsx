import { useEffect, useMemo, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { defaultPostLoginPath } from '../lib/tenantResolution'
import { loadAuthConfig, saveAuthConfig, type AuthConfig, useAuth } from '../state/auth'
import { useSelection } from '../state/selection'

const ROLE_CHOICES = [
  'COORDINATION_ADMIN',
  'COORDINATION_WRITER',
  'COORDINATION_READER',
  'PARTICIPANT_ADMIN',
  'AUTHORITY_ADMIN',
  'SYSTEM_EMITTER',
] as const

function saveAndReload(next: AuthConfig) {
  saveAuthConfig(next)
  window.location.reload()
}

const POST_REGISTER_SESSION = 'digit.postRegister'

export function LoginPage() {
  const auth = useAuth()
  const sel = useSelection()
  const nav = useNavigate()
  const [cfg, setCfg] = useState(loadAuthConfig())
  const [postRegisterNotice, setPostRegisterNotice] = useState<string | null>(null)

  useEffect(() => {
    if (!auth.ready) return

    const raw = sessionStorage.getItem(POST_REGISTER_SESSION)
    if (raw) {
      sessionStorage.removeItem(POST_REGISTER_SESSION)
      try {
        const o = JSON.parse(raw) as { realm: string; name?: string; email?: string }
        if (o.realm) {
          const base = loadAuthConfig()
          const next = { ...base, realm: o.realm, devTenantId: o.realm }
          saveAuthConfig(next)
          setCfg(next)
          const emailHint = o.email?.trim()
          const cred = emailHint
            ? ` Sign in with Keycloak: username = ${emailHint} (same as admin email), password = default — change it in Keycloak Admin if you like.`
            : ` Sign in with Keycloak: username = the admin email you used at registration, password = default (set by the Account realm template).`
          setPostRegisterNotice(
            o.name
              ? `Organization “${o.name}” was created. Realm: ${o.realm}.${cred} Then use Continue with Keycloak.`
              : `Realm ${o.realm} was created.${cred} Then use Continue with Keycloak.`,
          )
        }
      } catch {
        /* ignore */
      }
      if (auth.isAuthenticated) return
    }

    if (!auth.isAuthenticated) return
    nav(
      defaultPostLoginPath(
        { isAuthenticated: auth.isAuthenticated, tenantId: auth.tenantId },
        sel.accounts,
        sel.selectedAccountId,
      ),
      { replace: true },
    )
  }, [auth.ready, auth.isAuthenticated, auth.tenantId, sel.accounts, sel.selectedAccountId, nav])

  const mode = (cfg.mode || 'keycloak') as 'keycloak' | 'dev-local'
  const devRoles = useMemo(() => new Set(cfg.devRoles || []), [cfg.devRoles])

  function update(next: Partial<AuthConfig>) {
    setCfg((c) => ({ ...c, ...next }))
  }

  function setMode(m: 'keycloak' | 'dev-local') {
    update({ mode: m })
  }

  const tenantField =
    mode === 'keycloak' ? (
      <label className="digit-login-label">
        <span>Realm (tenant)</span>
        <input
          value={cfg.realm}
          onChange={(e) => update({ realm: e.target.value })}
          placeholder="e.g. PROVLOCAL…"
          autoComplete="off"
        />
      </label>
    ) : (
      <label className="digit-login-label">
        <span>Tenant id</span>
        <input
          value={cfg.devTenantId || ''}
          onChange={(e) => update({ devTenantId: e.target.value })}
          placeholder="Sent as X-Tenant-ID when set"
          autoComplete="off"
        />
      </label>
    )

  return (
    <div className="digit-login">
      <div className="digit-login-card">
        <header className="digit-login-header">
          <h1>DIGIT Console</h1>
          <span className="digit-console-brand-badge">Demo</span>
        </header>
        <p className="digit-login-lead">Sign in to continue. Local sandbox only — not for production secrets.</p>

        <p className="digit-login-register-prompt">
          New organization? <Link to="/register">Create an account</Link>
        </p>

        {postRegisterNotice && (
          <div className="digit-login-banner digit-login-banner--success" style={{ marginBottom: 18 }}>
            {postRegisterNotice}
            <button
              type="button"
              onClick={() => {
                setPostRegisterNotice(null)
                if (auth.isAuthenticated) {
                  nav(
                    defaultPostLoginPath(
                      { isAuthenticated: auth.isAuthenticated, tenantId: auth.tenantId },
                      sel.accounts,
                      sel.selectedAccountId,
                    ),
                    { replace: true },
                  )
                }
              }}
              style={{
                marginTop: 10,
                display: 'block',
                fontSize: 12,
                border: 'none',
                background: 'transparent',
                color: 'inherit',
                textDecoration: 'underline',
                cursor: 'pointer',
                padding: 0,
              }}
            >
              {auth.isAuthenticated ? 'Continue to console' : 'Dismiss'}
            </button>
          </div>
        )}

        <div className="digit-login-mode" role="group" aria-label="Authentication mode">
          <button
            type="button"
            className={`digit-login-mode-btn${mode === 'keycloak' ? ' digit-login-mode-btn-active' : ''}`}
            onClick={() => setMode('keycloak')}
          >
            Keycloak
          </button>
          <button
            type="button"
            className={`digit-login-mode-btn${mode === 'dev-local' ? ' digit-login-mode-btn-active' : ''}`}
            onClick={() => setMode('dev-local')}
          >
            Dev local
          </button>
        </div>

        <div className="digit-login-fields">
          {tenantField}

          <label className="digit-login-label">
            <span>OAuth client id</span>
            <input value={cfg.clientId} onChange={(e) => setCfg({ ...cfg, clientId: e.target.value })} placeholder="demo-ui" autoComplete="off" />
          </label>

          {mode === 'keycloak' && (
            <details className="digit-login-details">
              <summary>Advanced — Keycloak URL</summary>
              <label className="digit-login-label" style={{ marginTop: 4 }}>
                <span>Server base URL</span>
                <input
                  value={cfg.keycloakBaseUrl}
                  onChange={(e) => update({ keycloakBaseUrl: e.target.value })}
                  placeholder="http://localhost:5177/keycloak (dev, same origin)"
                />
              </label>
            </details>
          )}

          {mode === 'dev-local' && (
            <>
              <label className="digit-login-label">
                <span>X-Client-ID</span>
                <input
                  value={cfg.devClientId || 'demo-ui'}
                  onChange={(e) => update({ devClientId: e.target.value })}
                  placeholder="demo-ui"
                  autoComplete="off"
                />
              </label>

              <details className="digit-login-details">
                <summary>Simulated roles</summary>
                <p className="console-muted" style={{ margin: '0 0 8px' }}>
                  Controls which console areas appear. Leave all unchecked to use the default full demo set after reload.
                </p>
                <div className="digit-login-roles">
                  {ROLE_CHOICES.map((r) => (
                    <label key={r} className="digit-login-role">
                      <input
                        type="checkbox"
                        checked={devRoles.has(r)}
                        onChange={(e) => {
                          const next = new Set(devRoles)
                          if (e.target.checked) next.add(r)
                          else next.delete(r)
                          update({ devRoles: Array.from(next) })
                        }}
                      />
                      <span>{r}</span>
                    </label>
                  ))}
                </div>
              </details>
            </>
          )}
        </div>

        <div className="digit-login-actions">
          {mode === 'keycloak' ? (
            <button type="button" className="digit-login-btn-primary" onClick={() => auth.login()} disabled={!auth.ready}>
              Continue with Keycloak
            </button>
          ) : (
            <button type="button" className="digit-login-btn-primary" onClick={() => saveAndReload(cfg)}>
              Enter console
            </button>
          )}

          <button type="button" className="digit-login-btn-ghost" onClick={() => saveAndReload(cfg)}>
            Save settings &amp; reload
          </button>
        </div>

        <footer className="digit-login-footer">
          {mode === 'keycloak' ? (
            <>
              If you changed <strong>realm</strong> or the Keycloak <strong>server URL</strong>, use <strong>Save settings &amp; reload</strong>{' '}
              before <em>Continue with Keycloak</em> (the client reads config on page load).
              <br />
              <br />
              Use a <strong>public</strong> SPA client (e.g. <code>demo-ui</code>) with redirect URIs matching how you open the app (e.g. <code>http://localhost:5177/*</code> and <code>http://127.0.0.1:5177/*</code>) and PKCE enabled in Keycloak.
            </>
          ) : (
            <>
              Sends <code>Authorization: Bearer dev-local</code>. Your local compose stack must allow dev auth for APIs.
            </>
          )}
        </footer>
      </div>
    </div>
  )
}
