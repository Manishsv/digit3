import { Navigate, Link, useNavigate } from 'react-router-dom'
import { isPlatformDirectoryAdmin } from '../console/roles'
import { resolveTenantResolution } from '../lib/tenantResolution'
import { useAuth } from '../state/auth'
import { useSelection } from '../state/selection'

export function WelcomePage() {
  const auth = useAuth()
  const sel = useSelection()
  const nav = useNavigate()
  const resolution = resolveTenantResolution(auth, sel.accounts)
  const canUseAccountDirectory = isPlatformDirectoryAdmin(auth.roles)

  if (resolution.kind === 'single') {
    return <Navigate to="/" replace />
  }

  if (resolution.kind === 'many') {
    return (
      <div className="digit-console-welcome">
        <div className="digit-console-welcome-card">
          <h1 className="console-page-title" style={{ marginTop: 0 }}>
            Choose an account
          </h1>
          <p className="console-page-lead">
            You have access to more than one account. Pick one to continue — you can switch later from the top bar.
          </p>
          <ul className="digit-console-welcome-list">
            {resolution.accounts.map((a) => (
              <li key={a.id}>
                <button
                  type="button"
                  className="digit-console-welcome-pick"
                  onClick={() => {
                    sel.setSelectedAccountId(a.id)
                    nav('/', { replace: true })
                  }}
                >
                  <span className="digit-console-welcome-pick-name">{a.name || a.id}</span>
                  <span className="digit-console-welcome-pick-id">{a.id}</span>
                </button>
              </li>
            ))}
          </ul>
          <p style={{ marginTop: 20, marginBottom: 0 }}>
            <button type="button" className="digit-console-welcome-secondary" onClick={() => auth.logout()}>
              Sign out
            </button>
          </p>
        </div>
      </div>
    )
  }

  return (
    <div className="digit-console-welcome">
      <div className="digit-console-welcome-card">
        <h1 className="console-page-title" style={{ marginTop: 0 }}>
          Set up your account context
        </h1>
        <p className="console-page-lead">
          The console did not find a <strong>tenant id</strong> from your login (for example an empty dev-local tenant) and you have no saved
          accounts yet. Add at least one account to scope API calls (<code>X-Tenant-ID</code>) and unlock the rest of the console.
        </p>

        <div className="console-card" style={{ marginBottom: 16 }}>
          <h3 className="console-page-title" style={{ fontSize: 16 }}>
            Recommended (local demo)
          </h3>
          <ol style={{ margin: '0 0 12px', paddingLeft: 20, color: 'var(--console-text-muted)', lineHeight: 1.6 }}>
            <li>
              <Link to="/register">Register a new organization</Link> — calls Account service to create the tenant and Keycloak realm, then returns
              you to sign-in with the realm pre-filled.
            </li>
            {canUseAccountDirectory && (
              <li>
                Or, as a platform admin, open <Link to="/platform">Platform → Accounts directory</Link> and add a tenant id manually (browser
                shortcut; same value as your Keycloak realm if you use realm-per-tenant).
              </li>
            )}
            <li>Use the <strong>Account</strong> picker in the top bar once you are inside the console.</li>
          </ol>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 10 }}>
            <Link to="/register" className="digit-console-welcome-cta">
              Register organization
            </Link>
            {canUseAccountDirectory && (
              <button type="button" className="digit-console-welcome-secondary" onClick={() => nav('/platform')}>
                Platform (directory)
              </button>
            )}
            <button type="button" className="digit-console-welcome-secondary" onClick={() => auth.logout()}>
              Sign out
            </button>
          </div>
        </div>

        <div className="console-card">
          <h3 className="console-page-title" style={{ fontSize: 16 }}>
            Join with organization code
          </h3>
          <p className="console-muted" style={{ marginBottom: 12 }}>
            Self-service invite acceptance will call a directory API. For now, ask your coordination admin or use <Link to="/register">register</Link>{' '}
            if you are provisioning a new org.
          </p>
          <span className="console-chip console-chip-warn">Coming soon</span>
        </div>
      </div>
    </div>
  )
}
