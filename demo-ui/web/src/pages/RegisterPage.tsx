import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { ApiError, registerDigitTenant, registerTenantErrorMessage, type RegisterTenantBody } from '../lib/digitApi'
import { loadAuthConfig, saveAuthConfig } from '../state/auth'
import { useSelection } from '../state/selection'

const POST_REGISTER_KEY = 'digit.postRegister'

export function RegisterPage() {
  const sel = useSelection()
  const nav = useNavigate()
  const [orgName, setOrgName] = useState('')
  const [email, setEmail] = useState('')
  const [realmCode, setRealmCode] = useState('')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string | null>(null)

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    setErr(null)
    const name = orgName.trim()
    const em = email.trim()
    if (!name || !em) {
      setErr('Organization name and email are required.')
      return
    }
    setBusy(true)
    try {
      const body: RegisterTenantBody = {
        tenant: {
          name,
          email: em,
          isActive: true,
          additionalAttributes: {},
          ...(realmCode.trim() ? { code: realmCode.trim() } : {}),
        },
      }
      const resp = await registerDigitTenant(body)
      const t = resp.tenants?.[0]
      const code = (t?.code || '').trim()
      if (!code) {
        setErr('Account service returned no tenant code. Check Account service logs.')
        setBusy(false)
        return
      }

      sel.upsertAccount({ id: code, name: t?.name || name })
      sel.setSelectedAccountId(code)

      sessionStorage.setItem(POST_REGISTER_KEY, JSON.stringify({ realm: code, name: t?.name || name, email: em }))

      const cfg = loadAuthConfig()
      saveAuthConfig({ ...cfg, realm: code, devTenantId: code })

      nav('/login', { replace: true })
    } catch (e: unknown) {
      setErr(e instanceof ApiError ? registerTenantErrorMessage(e.bodyText) : String(e))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="digit-login">
      <div className="digit-login-card" style={{ maxWidth: 460 }}>
        <header className="digit-login-header">
          <h1>Create organization</h1>
          <span className="digit-console-brand-badge">Demo</span>
        </header>
        <p className="digit-login-lead">
          Registers a tenant in the Account service (creates the Keycloak realm, a first admin user from your email, and the database row). The realm
          template sets that user&apos;s initial password to <code>default</code> — sign in on the next screen with <strong>username = your admin email</strong>{' '}
          and that password, then change it in Keycloak if you like. Ensure the Account service is reachable (proxy default <code>8094</code>).
        </p>

        <form className="digit-login-fields" onSubmit={onSubmit}>
          <label className="digit-login-label">
            <span>Organization name</span>
            <input value={orgName} onChange={(e) => setOrgName(e.target.value)} placeholder="e.g. Demo City" autoComplete="organization" required />
          </label>

          <label className="digit-login-label">
            <span>Admin email</span>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="you@example.org"
              autoComplete="email"
              required
            />
          </label>

          <details className="digit-login-details">
            <summary>Advanced — realm code</summary>
            <p className="console-muted" style={{ margin: '8px 0' }}>
              Leave blank to let the server derive a code from the organization name (uppercase, no spaces). Set explicitly if you need a fixed
              Keycloak realm id.
            </p>
            <label className="digit-login-label">
              <span>Realm / tenant code</span>
              <input value={realmCode} onChange={(e) => setRealmCode(e.target.value)} placeholder="e.g. DEMOCITY" autoComplete="off" />
            </label>
          </details>

          {err && (
            <div className="digit-login-banner digit-login-banner--error" role="alert">
              {err}
            </div>
          )}

          <div className="digit-login-actions" style={{ marginTop: 8 }}>
            <button type="submit" className="digit-login-btn-primary" disabled={busy}>
              {busy ? 'Creating…' : 'Create organization'}
            </button>
            <Link to="/login" className="digit-login-link-back">
              Back to sign in
            </Link>
          </div>
        </form>
      </div>
    </div>
  )
}
