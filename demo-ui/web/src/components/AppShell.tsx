import { Link, NavLink, Outlet, useLocation } from 'react-router-dom'
import { isPlatformDirectoryAdmin } from '../console/roles'
import { breadcrumbsForPath, isNavAllowed, NAV_GROUPS } from '../console/navConfig'
import { useAuth } from '../state/auth'
import { useConsoleScope } from '../state/consoleScope'
import { useSelection } from '../state/selection'

export function AppShell() {
  const auth = useAuth()
  const loc = useLocation()
  const sel = useSelection()
  const scope = useConsoleScope()
  const crumbs = breadcrumbsForPath(loc.pathname)

  return (
    <div className="digit-console">
      <header className="digit-console-topbar">
        <div className="digit-console-topbar-left">
          <Link to="/" className="digit-console-brand">
            <span className="digit-console-brand-title">DIGIT Console</span>
            <span className="digit-console-brand-badge">Demo</span>
          </Link>
        </div>

        <nav className="digit-console-breadcrumbs" aria-label="Breadcrumb">
          {crumbs.map((c, i) => (
            <span key={`${c.label}-${i}`} className="digit-console-breadcrumb-seg">
              {i > 0 && <span className="digit-console-breadcrumb-sep">/</span>}
              {c.to ? (
                <Link to={c.to} className="digit-console-breadcrumb-link">
                  {c.label}
                </Link>
              ) : (
                <span className="digit-console-breadcrumb-current">{c.label}</span>
              )}
            </span>
          ))}
        </nav>

        <div className="digit-console-topbar-right">
          <div className="digit-console-context" title="API calls use this account as X-Tenant-ID when set">
            <label className="digit-console-context-label">Account</label>
            <select
              className="digit-console-context-select"
              value={sel.selectedAccountId || ''}
              onChange={(e) => sel.setSelectedAccountId(e.target.value || undefined)}
            >
              <option value="">— Select account —</option>
              {sel.accounts.map((a) => (
                <option key={a.id} value={a.id}>
                  {a.name ? `${a.name} (${a.id})` : a.id}
                </option>
              ))}
            </select>
            {isPlatformDirectoryAdmin(auth.roles) && (
              <Link to="/platform" className="digit-console-context-link" title="Platform administration — account directory">
                Directory
              </Link>
            )}
          </div>

          <div className="digit-console-context" title="Service code for Studio-backed configuration">
            <label className="digit-console-context-label">Service</label>
            <select
              className="digit-console-context-select"
              value={sel.selectedServiceCode || ''}
              disabled={!sel.selectedAccountId}
              onChange={(e) => sel.setSelectedServiceCode(e.target.value || undefined)}
            >
              <option value="">{sel.selectedAccountId ? '— Select service —' : 'Select account first'}</option>
              {scope.scopedServices.map((s) => (
                <option key={`${s.tenant_id}:${s.service_code}`} value={s.service_code}>
                  {s.service_code}
                  {s.payload && typeof s.payload === 'object' && s.payload.name ? ` — ${s.payload.name}` : ''}
                </option>
              ))}
            </select>
            <Link to="/service" className="digit-console-context-link">
              Open
            </Link>
          </div>

          <div className="digit-console-user">
            <span className="digit-console-user-line" title="JWT / session tenant">
              Tenant: <code>{auth.tenantId || '—'}</code>
            </span>
            <span className="digit-console-user-line">
              {auth.profile?.username || '—'}
            </span>
            <button type="button" className="digit-console-logout" onClick={auth.logout}>
              Sign out
            </button>
          </div>
        </div>
      </header>

      <div className="digit-console-body">
        <aside className="digit-console-sidebar" aria-label="Product navigation">
          {NAV_GROUPS.map((group) => {
            const items = group.items.filter((t) => isNavAllowed(auth.roles, t.requiredAnyRoles))
            if (items.length === 0) return null
            return (
              <div key={group.id} className="digit-console-nav-group">
                <div className="digit-console-nav-group-label">{group.label}</div>
                {items.map((t) => (
                  <NavLink
                    key={t.id}
                    to={t.to}
                    end={t.to === '/'}
                    className={({ isActive }) =>
                      `digit-console-nav-item${isActive ? ' digit-console-nav-item-active' : ''}`
                    }
                    title={t.description}
                  >
                    <span className="digit-console-nav-item-label">{t.label}</span>
                  </NavLink>
                ))}
              </div>
            )
          })}
        </aside>

        <main className="digit-console-main">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
