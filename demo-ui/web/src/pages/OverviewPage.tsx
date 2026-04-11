import { Link } from 'react-router-dom'

const CARDS = [
  {
    title: 'Platform',
    body: 'Check shared services and maintain the account directory.',
    to: '/platform',
    roles: 'Coordination admin',
  },
  {
    title: 'Account catalogue',
    body: 'Versioned rulesets, registries, audit groups, appellate groups, then service definitions.',
    to: '/account/setup/rulesets',
    roles: 'Superuser / coordination admin',
  },
  {
    title: 'Account',
    body: 'Per-tenant boundaries and users (placeholders link to Keycloak for now).',
    to: '/account',
    roles: 'Coordination admin',
  },
  {
    title: 'Services',
    body: 'Studio service catalogue — codes, registries, and internal Studio ids explained.',
    to: '/service',
    roles: 'Coordination admin / writer',
  },
  {
    title: 'Rules & policies',
    body: 'Regulator and rules engine configuration.',
    to: '/regulator',
    roles: 'Authority / coordination',
  },
  {
    title: 'Registries',
    body: 'Authoritative registry records used in decisions.',
    to: '/registries',
    roles: 'Coordination admin / writer',
  },
  {
    title: 'Operations',
    body: 'Citizen intake, operator decisions, appellate, and audit.',
    to: '/operator',
    roles: 'Role varies by screen',
  },
] as const

export function OverviewPage() {
  return (
    <div>
      <h1 className="console-page-title">Overview</h1>
      <p className="console-page-lead">
        DIGIT Console (demo) groups tasks the way cloud consoles group products: administration, configuration, then day‑to‑day operations. Use the
        top bar to set <strong>Account</strong> and <strong>Service</strong> context before running flows that depend on tenant or service code.
      </p>

      <h2 className="console-page-title" style={{ fontSize: 16, marginBottom: 12 }}>
        Quick links
      </h2>
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fill, minmax(240px, 1fr))',
          gap: 14,
        }}
      >
        {CARDS.map((c) => (
          <Link key={c.to} to={c.to} className="console-card" style={{ textDecoration: 'none', color: 'inherit', margin: 0 }}>
            <h3 style={{ marginTop: 0, color: 'var(--console-accent)' }}>{c.title}</h3>
            <p className="console-muted" style={{ margin: '0 0 8px' }}>
              {c.body}
            </p>
            <span className="console-muted" style={{ fontSize: 12 }}>
              Typical roles: {c.roles}
            </span>
          </Link>
        ))}
      </div>

      <h2 className="console-page-title" style={{ fontSize: 16, margin: '28px 0 12px' }}>
        End-to-end lifecycle
      </h2>
      <ol style={{ margin: 0, paddingLeft: 20, color: 'var(--console-text-muted)', lineHeight: 1.7, fontSize: '14px' }}>
        <li>
          <b>Account</b> → <Link to="/account">Account</Link>
        </li>
        <li>
          <b>Account catalogue</b> (versioned artefacts) → <Link to="/account/setup/rulesets">Rulesets → Registries → Audit → Appellate → Service definitions</Link>
        </li>
        <li>
          <b>Platform health + directory</b> → <Link to="/platform">Platform</Link>
        </li>
        <li>
          <b>Register services in Studio</b> → <Link to="/service">Services</Link>
        </li>
        <li>
          <b>Rules</b> → <Link to="/regulator">Rules & policies</Link>
        </li>
        <li>
          <b>Registries</b> → <Link to="/registries">Registries</Link>
        </li>
        <li>
          <b>Citizen intake</b> → <Link to="/citizen">Citizen intake</Link>
        </li>
        <li>
          <b>Operator</b> → <Link to="/operator">Operator</Link>
        </li>
        <li>
          <b>Appellate</b> → <Link to="/appellate">Appellate</Link>
        </li>
        <li>
          <b>Audit</b> → <Link to="/audit">Audit</Link>
        </li>
      </ol>
    </div>
  )
}
