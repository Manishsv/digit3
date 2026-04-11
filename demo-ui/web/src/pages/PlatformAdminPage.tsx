import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { isPlatformDirectoryAdmin, PLATFORM_DIRECTORY_ROLE } from '../console/roles'
import { digitJson, probeDigitService, useDigitHeaders } from '../lib/digitApi'
import { useAuth } from '../state/auth'
import { useSelection } from '../state/selection'

const SERVICE_LABELS: Record<string, string> = {
  studio: 'Studio',
  governance: 'Governance',
  coordination: 'Coordination',
  registry: 'Registry',
  workflow: 'Workflow',
  mdms: 'MDMS',
  idgen: 'IdGen',
  boundary: 'Boundary',
  account: 'Account',
}

function HealthCell({ row }: { row: { key: string; data: unknown } }) {
  const v = row.data
  if (!v || typeof v !== 'object') {
    return (
      <td>
        <span className="console-muted">—</span>
      </td>
    )
  }
  const o = v as Record<string, unknown>
  if (o.status === 'ok' || o.status === 'error') {
    const ok = o.status === 'ok'
    const chip = ok ? 'console-chip console-chip-ok' : 'console-chip console-chip-err'
    return (
      <td>
        <span className={chip}>{ok ? 'Reachable' : 'Issue'}</span>
        {o.probe != null && (
          <div className="console-muted" style={{ marginTop: 6 }}>
            via <code>{String(o.probe)}</code>
            {o.httpStatus != null && <> · HTTP {String(o.httpStatus)}</>}
          </div>
        )}
        {!ok && o.error != null && (
          <div className="console-muted" style={{ marginTop: 6 }}>
            {String(o.error)}
          </div>
        )}
      </td>
    )
  }
  return (
    <td>
      <span className="console-chip console-chip-ok">Up</span>
      <pre className="console-json" style={{ marginTop: 8, maxHeight: 140 }}>
        {JSON.stringify(v, null, 2)}
      </pre>
    </td>
  )
}

export function PlatformAdminPage() {
  const auth = useAuth()
  const headers = useDigitHeaders()
  const sel = useSelection()
  const canManageDirectory = isPlatformDirectoryAdmin(auth.roles)
  const [status, setStatus] = useState<Record<string, unknown> | null>(null)
  const [err, setErr] = useState<string | null>(null)
  const [tab, setTab] = useState<'status' | 'accounts'>('status')
  const [newAccountId, setNewAccountId] = useState('')
  const [newAccountName, setNewAccountName] = useState('')

  useEffect(() => {
    if (!canManageDirectory && tab === 'accounts') setTab('status')
  }, [canManageDirectory, tab])

  useEffect(() => {
    let cancelled = false
    const keys = [
      'studio',
      'governance',
      'coordination',
      'registry',
      'workflow',
      'mdms',
      'idgen',
      'boundary',
      'account',
    ] as const
    ;(async () => {
      try {
        const results = await Promise.all([
          digitJson('studio', '/health', { headers }),
          digitJson('governance', '/health', { headers }),
          digitJson('coordination', '/health', { headers }),
          digitJson('registry', '/health', { headers }),
          probeDigitService(
            'workflow',
            [{ path: '/workflow/v1/process?code=PGR67' }, { path: '/workflow/v1/process' }],
            headers,
          ),
          probeDigitService(
            'mdms',
            [
              { path: '/actuator/health' },
              {
                path: '/mdms-v2/v1/_search',
                init: {
                  method: 'POST',
                  headers: { 'Content-Type': 'application/json' },
                  body: '{}',
                },
              },
            ],
            headers,
          ),
          probeDigitService('idgen', [{ path: '/idgen/health' }, { path: '/health' }], headers),
          probeDigitService('boundary', [{ path: '/boundary/v1' }], headers),
          probeDigitService('account', [{ path: '/actuator/health' }, { path: '/health' }], headers),
        ])
        if (cancelled) return
        const out: Record<string, unknown> = {}
        keys.forEach((k, i) => {
          const r = results[i]
          if (k === 'workflow' || k === 'mdms' || k === 'idgen' || k === 'boundary' || k === 'account') {
            const p = r as Awaited<ReturnType<typeof probeDigitService>>
            out[k] = p.ok
              ? { status: 'ok', probe: p.via, httpStatus: p.status }
              : { status: 'error', error: p.error, probe: p.via, httpStatus: p.status }
          } else {
            out[k] = r
          }
        })
        setStatus(out)
      } catch (e: unknown) {
        if (!cancelled) setErr(String(e))
      }
    })()
    return () => {
      cancelled = true
    }
  }, [headers])

  const rows =
    status &&
    (Object.keys(status) as string[]).map((key) => ({
      key,
      data: status[key],
    }))

  return (
    <div>
      <h1 className="console-page-title">Platform</h1>
      <p className="console-page-lead">
        Shared DIGIT services that power every account.
        {canManageDirectory ? (
          <>
            {' '}
            Use <strong>Accounts directory</strong> to register tenant ids in this browser; the active account in the top bar drives{' '}
            <code>X-Tenant-ID</code> on API calls.
          </>
        ) : (
          <>
            {' '}
            The <strong>Accounts directory</strong> tab is only for users with the <code>{PLATFORM_DIRECTORY_ROLE}</code> role (platform
            administration). Others should use <Link to="/register">Register organization</Link> or ask an admin to add your tenant.
          </>
        )}
      </p>

      <div className="console-tabs">
        <button type="button" className={`console-tab${tab === 'status' ? ' console-tab-active' : ''}`} onClick={() => setTab('status')}>
          Service health
        </button>
        {canManageDirectory && (
          <button type="button" className={`console-tab${tab === 'accounts' ? ' console-tab-active' : ''}`} onClick={() => setTab('accounts')}>
            Accounts directory
          </button>
        )}
      </div>

      {err && <pre className="console-json" style={{ background: '#fce8e6', marginBottom: 16 }}>{err}</pre>}

      {tab === 'accounts' && canManageDirectory ? (
        <div className="console-card">
          <h3>Add or select an account</h3>
          <p className="console-muted">
            Accounts are stored in this browser (demo shortcut). The selected account appears in the top bar for the rest of the console.
          </p>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr auto', gap: 10, alignItems: 'end', marginBottom: 20 }}>
            <div style={{ display: 'grid', gap: 6 }}>
              <label>Account id (realm / tenant)</label>
              <input value={newAccountId} onChange={(e) => setNewAccountId(e.target.value)} placeholder="e.g. PROVLOCAL…" />
            </div>
            <div style={{ display: 'grid', gap: 6 }}>
              <label>Display name (optional)</label>
              <input value={newAccountName} onChange={(e) => setNewAccountName(e.target.value)} placeholder="e.g. Demo City" />
            </div>
            <button
              type="button"
              onClick={() => {
                const id = newAccountId.trim()
                if (!id) return
                sel.upsertAccount({ id, name: newAccountName.trim() || undefined })
                sel.setSelectedAccountId(id)
                setNewAccountId('')
                setNewAccountName('')
              }}
              disabled={!newAccountId.trim()}
            >
              Save &amp; select
            </button>
          </div>

          <h3>Known accounts</h3>
          {sel.accounts.length === 0 ? (
            <div className="console-muted">No accounts yet. Add one above, then pick it in the top bar.</div>
          ) : (
            <div style={{ display: 'grid', gap: 10 }}>
              {sel.accounts.map((a) => (
                <div
                  key={a.id}
                  style={{
                    display: 'grid',
                    gridTemplateColumns: '1fr auto auto',
                    gap: 10,
                    alignItems: 'center',
                    padding: 12,
                    border: '1px solid var(--console-border)',
                    borderRadius: 10,
                    background: sel.selectedAccountId === a.id ? 'var(--console-accent-soft)' : '#fff',
                  }}
                >
                  <div>
                    <div style={{ fontWeight: 600 }}>{a.name || a.id}</div>
                    <div className="console-muted" style={{ fontSize: 12 }}>
                      <code>{a.id}</code>
                    </div>
                  </div>
                  <button type="button" onClick={() => sel.setSelectedAccountId(a.id)}>
                    Select
                  </button>
                  <button type="button" onClick={() => sel.removeAccount(a.id)}>
                    Remove
                  </button>
                </div>
              ))}
            </div>
          )}
          <p className="console-muted" style={{ marginBottom: 0, marginTop: 16 }}>
            User and role administration still lives in Keycloak for this demo: <code>http://localhost:8080/keycloak/admin/</code> ·{' '}
            <Link to="/account">Open Account workspace</Link>
          </p>
        </div>
      ) : (
        <div className="console-card">
          <h3>Connectivity</h3>
          <p className="console-muted" style={{ marginTop: 0 }}>
            Some stacks do not expose <code>/health</code> at the container root; those rows show the first path that responded (see{' '}
            <code>via</code>).
          </p>
          {!status ? (
            <div className="console-muted">Loading…</div>
          ) : (
            <div className="console-table-wrap">
              <table className="console-table">
                <thead>
                  <tr>
                    <th style={{ width: '28%' }}>Service</th>
                    <th>Result</th>
                  </tr>
                </thead>
                <tbody>
                  {rows!.map((row) => (
                    <tr key={row.key}>
                      <td>
                        <strong>{SERVICE_LABELS[row.key] || row.key}</strong>
                        <div className="console-muted" style={{ marginTop: 4 }}>
                          <code>{row.key}</code>
                        </div>
                      </td>
                      <HealthCell row={row} />
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
