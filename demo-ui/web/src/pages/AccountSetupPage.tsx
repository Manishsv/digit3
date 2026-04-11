import { Navigate, NavLink, useParams } from 'react-router-dom'
import { isAccountCatalogueAdmin } from '../console/roles'
import { useAuth } from '../state/auth'
import { useSelection } from '../state/selection'

const TABS = [
  { id: 'rulesets', label: 'Rulesets', path: 'rulesets' as const },
  { id: 'registries', label: 'Registries', path: 'registries' as const },
  { id: 'audit-groups', label: 'Audit groups', path: 'audit-groups' as const },
  { id: 'appellate-groups', label: 'Appellate groups', path: 'appellate-groups' as const },
  { id: 'service-definitions', label: 'Service definitions', path: 'service-definitions' as const },
] as const

type TabId = (typeof TABS)[number]['path']

function basePath(tab: TabId) {
  return `/account/setup/${tab}`
}

export function AccountSetupPage() {
  const auth = useAuth()
  const sel = useSelection()
  const { tab } = useParams<{ tab: TabId }>()

  if (!isAccountCatalogueAdmin(auth.roles)) {
    return (
      <div className="console-card">
        <h1 className="console-page-title">Account catalogue</h1>
        <p className="console-muted">You need the <code>SUPERUSER</code> or <code>COORDINATION_ADMIN</code> realm role to manage the account catalogue.</p>
      </div>
    )
  }

  const active = (TABS.find((t) => t.path === tab)?.path ?? null) as TabId | null
  if (!tab || !active) {
    return <Navigate to="/account/setup/rulesets" replace />
  }

  return (
    <div>
      <h1 className="console-page-title">Account catalogue</h1>
      <p className="console-page-lead">
        Account administrators maintain <strong>versioned</strong> artefacts for the selected account, then wire them together in{' '}
        <strong>service definitions</strong>. Order is typical: rulesets and registries first, then audit and appellate routing, then service
        definitions that reference those ids.
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
          Select an <strong>Account</strong> in the top bar — catalogue operations are scoped with <code>X-Tenant-ID</code>.
        </div>
      ) : (
        <div className="console-card" style={{ marginBottom: 16, padding: '12px 16px' }}>
          <span className="console-muted">Catalogue for </span>
          <strong>{sel.accounts.find((a) => a.id === sel.selectedAccountId)?.name || sel.selectedAccountId}</strong>
          <span className="console-muted"> · </span>
          <code>{sel.selectedAccountId}</code>
        </div>
      )}

      <div className="console-tabs" style={{ flexWrap: 'wrap', gap: 4 }}>
        {TABS.map((t) => (
          <NavLink
            key={t.id}
            to={basePath(t.path)}
            className={({ isActive }) => `console-tab${isActive ? ' console-tab-active' : ''}`}
            style={{ textDecoration: 'none' }}
          >
            {t.label}
          </NavLink>
        ))}
      </div>

      <div style={{ marginTop: 20 }}>
        {active === 'rulesets' && <RulesetsPanel />}
        {active === 'registries' && <RegistriesPanel />}
        {active === 'audit-groups' && <AuditGroupsPanel />}
        {active === 'appellate-groups' && <AppellateGroupsPanel />}
        {active === 'service-definitions' && <ServiceDefinitionsPanel />}
      </div>
    </div>
  )
}

function VersionNote() {
  return (
    <p className="console-muted" style={{ marginTop: 0 }}>
      Each artefact is <strong>versioned</strong> (semver or monotonic id). Publish creates an immutable row; service definitions pin compatible
      versions or ranges. Registry remains the durable store; this console surfaces the intended flow.
    </p>
  )
}

function RulesetsPanel() {
  return (
    <div className="console-card">
      <h2 style={{ marginTop: 0 }}>Rulesets</h2>
      <VersionNote />
      <ul className="console-muted" style={{ lineHeight: 1.7 }}>
        <li>
          Publish via Governance API → Registry schema <code>governance.ruleset</code> (id + <code>version</code> + digest).
        </li>
        <li>
          Studio bundle jobs can push YAML bundles; facts contracts validate in Governance before publish.
        </li>
        <li>
          Console shortcut: <NavLink to="/regulator">Rules & policies</NavLink> (regulator / rules engine demo).
        </li>
      </ul>
    </div>
  )
}

function RegistriesPanel() {
  return (
    <div className="console-card">
      <h2 style={{ marginTop: 0 }}>Registries</h2>
      <VersionNote />
      <ul className="console-muted" style={{ lineHeight: 1.7 }}>
        <li>
          Registry schemas and rows are tenant-scoped via <code>X-Tenant-ID</code>; schema evolution is versioned at the schema registry layer.
        </li>
        <li>
          Service definitions reference registry schemas and record ids (e.g. pulls, evidence bindings).
        </li>
        <li>
          Console shortcut: <NavLink to="/registries">Registries</NavLink> for record exploration.
        </li>
      </ul>
    </div>
  )
}

function AuditGroupsPanel() {
  return (
    <div className="console-card">
      <h2 style={{ marginTop: 0 }}>Audit groups</h2>
      <VersionNote />
      <ul className="console-muted" style={{ lineHeight: 1.7 }}>
        <li>
          Model as versioned <strong>audit profiles</strong>: coordination traces, governance receipts, retention, PII masking — referenced from{' '}
          <code>serviceDefinition.audit</code> on <code>studio.service</code> / <code>studio.bundle</code>.
        </li>
        <li>Multiple services may share one audit group; one service may emit to several sinks (declared as bindings).</li>
        <li>
          Console shortcut: <NavLink to="/audit">Audit</NavLink> (read trail for operators).
        </li>
      </ul>
    </div>
  )
}

function AppellateGroupsPanel() {
  return (
    <div className="console-card">
      <h2 style={{ marginTop: 0 }}>Appellate groups</h2>
      <VersionNote />
      <ul className="console-muted" style={{ lineHeight: 1.7 }}>
        <li>
          Model as versioned <strong>routing rules</strong> (authority participant refs, windows, grounds taxonomy) — referenced from{' '}
          <code>serviceDefinition.appeals</code>.
        </li>
        <li>Map service types or service codes to appellate authorities; many-to-many is allowed.</li>
        <li>
          Governance persists <code>governance.appeal</code> / <code>governance.order</code> at runtime.
        </li>
        <li>
          Console shortcut: <NavLink to="/appellate">Appellate</NavLink>.
        </li>
      </ul>
    </div>
  )
}

function ServiceDefinitionsPanel() {
  return (
    <div className="console-card">
      <h2 style={{ marginTop: 0 }}>Service definitions</h2>
      <VersionNote />
      <ul className="console-muted" style={{ lineHeight: 1.7 }}>
        <li>
          Canonical payload: <code>serviceDefinition</code> on Registry <code>studio.service</code> (and optional snapshot on <code>studio.bundle</code>
          ).
        </li>
        <li>
          Ties <strong>case model</strong> (intake, evidence, registry pulls, facts contract), <strong>governance.rulesetBindings</strong>,{' '}
          <strong>workflow</strong> process codes, <strong>notifications</strong>, <strong>payments</strong>, <strong>audit</strong>,{' '}
          <strong>appeals</strong>, and <strong>bindings</strong> index.
        </li>
        <li>
          Publish via Studio <code>POST /studio/v1/services</code> with <code>serviceDefinition</code> JSON (schema version <code>1.0</code>).
        </li>
        <li>
          Console shortcut: <NavLink to="/service">Services</NavLink> for catalogue and bundle jobs.
        </li>
      </ul>
    </div>
  )
}
