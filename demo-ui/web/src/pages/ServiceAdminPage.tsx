import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useConsoleScope } from '../state/consoleScope'
import { useSelection } from '../state/selection'
import type { StudioServiceRow } from '../types/studio'

function payloadName(row: StudioServiceRow): string | undefined {
  const p = row.payload
  if (p && typeof p === 'object' && typeof p.name === 'string') return p.name
  return undefined
}

function studioRecordId(row: StudioServiceRow): string | undefined {
  const p = row.payload
  if (p && typeof p === 'object' && typeof p.id === 'string') return p.id
  return undefined
}

export function ServiceAdminPage() {
  const sel = useSelection()
  const scope = useConsoleScope()
  const [tab, setTab] = useState<'catalogue' | 'details' | 'rules' | 'workflow' | 'notifications'>('catalogue')

  const filtered = sel.selectedAccountId
    ? scope.scopedServices
    : scope.services

  const selected = sel.selectedServiceCode ? filtered.find((s) => s.service_code === sel.selectedServiceCode) : undefined

  return (
    <div>
      <h1 className="console-page-title">Services</h1>
      <p className="console-page-lead">
        Catalogue of services registered in Studio for the active account. Use the top bar to pick an <strong>Account</strong> and{' '}
        <strong>Service</strong> (service code). The fields below mirror what Studio stores: business <em>service code</em>, linked{' '}
        <em>registry</em>, and an internal <em>Studio record id</em> for that row.
      </p>

      {!sel.selectedAccountId && (
        <div className="console-card console-muted" style={{ marginBottom: 16 }}>
          No account selected — showing all services returned for your session. Choose an account in the top bar to narrow the list, or{' '}
          <Link to="/platform">add accounts under Platform</Link>.
        </div>
      )}

      {scope.servicesError && (
        <pre className="console-json" style={{ background: '#fce8e6', marginBottom: 16 }}>
          {scope.servicesError}
        </pre>
      )}

      <div className="console-tabs">
        <button type="button" className={`console-tab${tab === 'catalogue' ? ' console-tab-active' : ''}`} onClick={() => setTab('catalogue')}>
          Catalogue
        </button>
        <button type="button" className={`console-tab${tab === 'details' ? ' console-tab-active' : ''}`} onClick={() => setTab('details')}>
          Details
        </button>
        <button type="button" className={`console-tab${tab === 'rules' ? ' console-tab-active' : ''}`} onClick={() => setTab('rules')}>
          Rules
        </button>
        <button type="button" className={`console-tab${tab === 'workflow' ? ' console-tab-active' : ''}`} onClick={() => setTab('workflow')}>
          Workflow
        </button>
        <button type="button" className={`console-tab${tab === 'notifications' ? ' console-tab-active' : ''}`} onClick={() => setTab('notifications')}>
          Notifications
        </button>
      </div>

      {tab === 'catalogue' ? (
        <div className="console-card">
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12, flexWrap: 'wrap', gap: 8 }}>
            <h3 style={{ margin: 0 }}>Registered services</h3>
            <button type="button" onClick={() => scope.refreshServices()} disabled={scope.servicesLoading}>
              {scope.servicesLoading ? 'Refreshing…' : 'Refresh'}
            </button>
          </div>
          {scope.servicesLoading && filtered.length === 0 ? (
            <div className="console-muted">Loading…</div>
          ) : filtered.length === 0 ? (
            <div className="console-muted">No services for this scope. Register a service in Studio or pick another account.</div>
          ) : (
            <div className="console-table-wrap">
              <table className="console-table">
                <thead>
                  <tr>
                    <th>Service code</th>
                    <th>Display name</th>
                    <th>Registry</th>
                    <th>Studio record</th>
                    <th>Status</th>
                  </tr>
                </thead>
                <tbody>
                  {filtered.map((s) => (
                    <tr key={`${s.tenant_id}:${s.service_code}`}>
                      <td>
                        <code>{s.service_code}</code>
                      </td>
                      <td>{payloadName(s) || '—'}</td>
                      <td>
                        <code>{s.registry_id || '—'}</code>
                      </td>
                      <td>
                        <code>{studioRecordId(s) || '—'}</code>
                      </td>
                      <td>{s.status || s.payload?.status || '—'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          <p className="console-muted" style={{ marginTop: 12, marginBottom: 0 }}>
            <strong>Service code</strong> is what workflows and APIs key on (for example <code>PGR67</code>). <strong>Studio record id</strong> is
            Studio&apos;s internal id for this registration row — not the same as registry or service code.
          </p>
        </div>
      ) : !sel.selectedServiceCode ? (
        <div
          style={{
            color: '#b06000',
            background: '#fef7e0',
            border: '1px solid #fdd663',
            padding: 14,
            borderRadius: 12,
          }}
        >
          Select a <strong>Service</strong> in the top bar. Service code must match a row in the catalogue.
        </div>
      ) : (
        <div className="console-card">
          <h3>Service {sel.selectedServiceCode}</h3>
          {tab === 'details' && selected ? (
            <>
              <dl className="console-dl" style={{ marginBottom: 20 }}>
                <dt>Service code</dt>
                <dd>
                  <code>{selected.service_code}</code> — primary handle for APIs and workflow configuration.
                </dd>
                <dt>Account (tenant)</dt>
                <dd>
                  <code>{selected.tenant_id}</code>
                </dd>
                <dt>Registry id</dt>
                <dd>
                  <code>{selected.registry_id || '—'}</code> — linked registry instance for authoritative data.
                </dd>
                <dt>Studio record id</dt>
                <dd>
                  <code>{studioRecordId(selected) || '—'}</code> — internal Studio row id (module payload <code>id</code>).
                </dd>
                <dt>Row status</dt>
                <dd>{selected.status || selected.payload?.status || '—'}</dd>
                {selected.updated_at && (
                  <>
                    <dt>Updated</dt>
                    <dd>{selected.updated_at}</dd>
                  </>
                )}
              </dl>
              <details>
                <summary style={{ cursor: 'pointer', fontWeight: 600, marginBottom: 8 }}>Raw JSON (advanced)</summary>
                <pre className="console-json">{JSON.stringify(selected, null, 2)}</pre>
              </details>
            </>
          ) : tab === 'details' ? (
            <div className="console-muted">No matching row for this service code in the current catalogue.</div>
          ) : (
            <p className="console-muted" style={{ margin: 0 }}>
              Placeholder: <strong>{tab}</strong> management for <code>{sel.selectedServiceCode}</code> will live here (rules bindings, workflow
              process keys, notification templates).
            </p>
          )}
        </div>
      )}
    </div>
  )
}
