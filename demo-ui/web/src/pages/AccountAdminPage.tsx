import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useSelection } from '../state/selection'

export function AccountAdminPage() {
  const sel = useSelection()
  const [tab, setTab] = useState<'boundaries' | 'users'>('boundaries')

  return (
    <div>
      <h1 className="console-page-title">Account</h1>
      <p className="console-page-lead">
        Workspace for a single tenant: boundaries, directory data, and (eventually) delegated user administration. Choose the active account in
        the <strong>top bar</strong>; add accounts under <Link to="/platform">Platform → Accounts directory</Link>. Account administrators maintain
        versioned <Link to="/account/setup/rulesets">rulesets, registries, audit & appellate groups, and service definitions</Link> under{' '}
        <strong>Account catalogue</strong>.
      </p>

      {!sel.selectedAccountId ? (
        <div
          style={{
            color: '#b06000',
            background: '#fef7e0',
            border: '1px solid #fdd663',
            padding: 14,
            borderRadius: 12,
            marginBottom: 16,
          }}
        >
          Select an <strong>Account</strong> in the top bar to scope this page.
        </div>
      ) : (
        <div className="console-card" style={{ marginBottom: 16, padding: '12px 16px' }}>
          <span className="console-muted">Managing </span>
          <strong>{sel.accounts.find((a) => a.id === sel.selectedAccountId)?.name || sel.selectedAccountId}</strong>
          <span className="console-muted"> · </span>
          <code>{sel.selectedAccountId}</code>
        </div>
      )}

      <div className="console-tabs">
        <button type="button" className={`console-tab${tab === 'boundaries' ? ' console-tab-active' : ''}`} onClick={() => setTab('boundaries')}>
          Boundaries
        </button>
        <button type="button" className={`console-tab${tab === 'users' ? ' console-tab-active' : ''}`} onClick={() => setTab('users')}>
          Users
        </button>
      </div>

      {sel.selectedAccountId ? (
        tab === 'boundaries' ? (
          <div className="console-card">
            <h3>Boundaries</h3>
            <p className="console-muted">
              Placeholder: create and list administrative boundaries for account <code>{sel.selectedAccountId}</code>.
            </p>
          </div>
        ) : (
          <div className="console-card">
            <h3>Users</h3>
            <p className="console-muted">
              Placeholder: user and role assignments for this account. For the demo, use Keycloak Admin:{' '}
              <code>http://localhost:8080/keycloak/admin/</code>
            </p>
          </div>
        )
      ) : null}
    </div>
  )
}
