/** Left navigation — grouped like a cloud console (Home / Admin / Config / Ops). */

export type NavItem = {
  id: string
  label: string
  description?: string
  to: string
  requiredAnyRoles?: string[]
}

export type NavGroup = { id: string; label: string; items: NavItem[] }

export const NAV_GROUPS: NavGroup[] = [
  {
    id: 'home',
    label: 'Home',
    items: [
      {
        id: 'overview',
        label: 'Overview',
        description: 'Lifecycle map and entry points',
        to: '/',
      },
    ],
  },
  {
    id: 'account-catalogue',
    label: 'Account catalogue',
    items: [
      {
        id: 'acct-rulesets',
        label: 'Rulesets',
        description: 'Versioned governance rules (YAML) for this account',
        to: '/account/setup/rulesets',
        requiredAnyRoles: ['SUPERUSER', 'COORDINATION_ADMIN'],
      },
      {
        id: 'acct-registries',
        label: 'Registries',
        description: 'Versioned registry artefacts and schemas in scope for this account',
        to: '/account/setup/registries',
        requiredAnyRoles: ['SUPERUSER', 'COORDINATION_ADMIN'],
      },
      {
        id: 'acct-audit-groups',
        label: 'Audit groups',
        description: 'Where audit logs are routed and who can read them',
        to: '/account/setup/audit-groups',
        requiredAnyRoles: ['SUPERUSER', 'COORDINATION_ADMIN'],
      },
      {
        id: 'acct-appellate-groups',
        label: 'Appellate groups',
        description: 'Appeal routing and appellate authorities by service or domain',
        to: '/account/setup/appellate-groups',
        requiredAnyRoles: ['SUPERUSER', 'COORDINATION_ADMIN'],
      },
      {
        id: 'acct-service-definitions',
        label: 'Service definitions',
        description: 'Versioned service profiles tying rulesets, registries, workflow, audit, appeals',
        to: '/account/setup/service-definitions',
        requiredAnyRoles: ['SUPERUSER', 'COORDINATION_ADMIN'],
      },
    ],
  },
  {
    id: 'admin',
    label: 'Administration',
    items: [
      {
        id: 'platform',
        label: 'Platform',
        description: 'Shared services health and account directory',
        to: '/platform',
        requiredAnyRoles: ['COORDINATION_ADMIN'],
      },
      {
        id: 'account',
        label: 'Account',
        description: 'Boundaries and users for one account',
        to: '/account',
        requiredAnyRoles: ['COORDINATION_ADMIN'],
      },
      {
        id: 'service',
        label: 'Services',
        description: 'Service catalogue and configuration',
        to: '/service',
        requiredAnyRoles: ['COORDINATION_ADMIN', 'COORDINATION_WRITER'],
      },
    ],
  },
  {
    id: 'config',
    label: 'Configuration',
    items: [
      {
        id: 'regulator',
        label: 'Rules & policies',
        description: 'Regulator / rules engine',
        to: '/regulator',
        requiredAnyRoles: ['AUTHORITY_ADMIN', 'COORDINATION_ADMIN', 'COORDINATION_WRITER'],
      },
      {
        id: 'registries',
        label: 'Registries',
        description: 'Authoritative records and facts',
        to: '/registries',
        requiredAnyRoles: ['COORDINATION_ADMIN', 'COORDINATION_WRITER'],
      },
    ],
  },
  {
    id: 'ops',
    label: 'Channel & operations',
    items: [
      {
        id: 'citizen',
        label: 'Citizen intake',
        description: 'Channel capture',
        to: '/citizen',
      },
      {
        id: 'operator',
        label: 'Operator',
        description: 'Decide and operate workflows',
        to: '/operator',
        requiredAnyRoles: ['COORDINATION_WRITER', 'COORDINATION_ADMIN'],
      },
      {
        id: 'appellate',
        label: 'Appellate',
        description: 'Disputes',
        to: '/appellate',
        requiredAnyRoles: ['AUTHORITY_ADMIN', 'COORDINATION_ADMIN'],
      },
      {
        id: 'audit',
        label: 'Audit',
        description: 'Read-only audit trail',
        to: '/audit',
        requiredAnyRoles: ['COORDINATION_ADMIN', 'COORDINATION_READER'],
      },
    ],
  },
]

export function isNavAllowed(userRoles: Set<string>, requiredAnyRoles?: string[]) {
  if (!requiredAnyRoles || requiredAnyRoles.length === 0) return true
  for (const r of requiredAnyRoles) if (userRoles.has(r)) return true
  return false
}

/** Breadcrumb segments for the main content header. */
export function breadcrumbsForPath(pathname: string): { label: string; to?: string }[] {
  const base = [{ label: 'DIGIT Console', to: '/' }]
  const rest: Record<string, { label: string; to?: string }[]> = {
    '/': [],
    '/platform': [{ label: 'Administration' }, { label: 'Platform', to: '/platform' }],
    '/account': [{ label: 'Administration' }, { label: 'Account', to: '/account' }],
    '/account/setup/rulesets': [
      { label: 'Account catalogue', to: '/account/setup/rulesets' },
      { label: 'Rulesets' },
    ],
    '/account/setup/registries': [
      { label: 'Account catalogue', to: '/account/setup/rulesets' },
      { label: 'Registries', to: '/account/setup/registries' },
    ],
    '/account/setup/audit-groups': [
      { label: 'Account catalogue', to: '/account/setup/rulesets' },
      { label: 'Audit groups', to: '/account/setup/audit-groups' },
    ],
    '/account/setup/appellate-groups': [
      { label: 'Account catalogue', to: '/account/setup/rulesets' },
      { label: 'Appellate groups', to: '/account/setup/appellate-groups' },
    ],
    '/account/setup/service-definitions': [
      { label: 'Account catalogue', to: '/account/setup/rulesets' },
      { label: 'Service definitions', to: '/account/setup/service-definitions' },
    ],
    '/service': [{ label: 'Administration' }, { label: 'Services', to: '/service' }],
    '/regulator': [{ label: 'Configuration' }, { label: 'Rules & policies', to: '/regulator' }],
    '/registries': [{ label: 'Configuration' }, { label: 'Registries', to: '/registries' }],
    '/citizen': [{ label: 'Operations' }, { label: 'Citizen intake', to: '/citizen' }],
    '/operator': [{ label: 'Operations' }, { label: 'Operator', to: '/operator' }],
    '/appellate': [{ label: 'Operations' }, { label: 'Appellate', to: '/appellate' }],
    '/audit': [{ label: 'Operations' }, { label: 'Audit', to: '/audit' }],
  }
  const key = pathname.endsWith('/') && pathname !== '/' ? pathname.slice(0, -1) : pathname
  return [...base, ...(rest[key] || [{ label: pathname }])]
}
