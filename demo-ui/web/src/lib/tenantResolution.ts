import type { AccountRef } from '../state/selection'

/** Minimal auth fields needed for tenant membership (avoid importing full AuthState). */
export type TenantAuthSnapshot = {
  isAuthenticated: boolean
  tenantId?: string
}

/**
 * Console "memberships" until a directory API exists: saved accounts plus the JWT realm tenant
 * (Keycloak iss) or dev-local tenant when present — de-duplicated by id.
 */
export function resolveMemberships(auth: TenantAuthSnapshot, accounts: AccountRef[]): AccountRef[] {
  const map = new Map<string, AccountRef>()
  for (const a of accounts) {
    if (a.id?.trim()) map.set(a.id.trim(), { id: a.id.trim(), name: a.name })
  }
  const jwt = auth.tenantId?.trim()
  if (jwt && !map.has(jwt)) map.set(jwt, { id: jwt, name: 'Signed-in realm' })
  return Array.from(map.values()).sort((a, b) => a.id.localeCompare(b.id))
}

export type TenantResolution =
  | { kind: 'none' }
  | { kind: 'single'; account: AccountRef }
  | { kind: 'many'; accounts: AccountRef[] }

export function resolveTenantResolution(auth: TenantAuthSnapshot, accounts: AccountRef[]): TenantResolution {
  if (!auth.isAuthenticated) return { kind: 'none' }
  const memberships = resolveMemberships(auth, accounts)
  if (memberships.length === 0) return { kind: 'none' }
  if (memberships.length === 1) return { kind: 'single', account: memberships[0]! }
  return { kind: 'many', accounts: memberships }
}

export function selectedAccountIsValid(resolution: TenantResolution, selectedAccountId?: string): boolean {
  if (!selectedAccountId?.trim()) return false
  if (resolution.kind === 'none') return false
  if (resolution.kind === 'single') return resolution.account.id === selectedAccountId
  return resolution.accounts.some((a) => a.id === selectedAccountId)
}

/** First screen after successful login (before shell guards). */
export function defaultPostLoginPath(
  auth: TenantAuthSnapshot,
  accounts: AccountRef[],
  selectedAccountId?: string,
): '/welcome' | '/' {
  const r = resolveTenantResolution(auth, accounts)
  if (r.kind === 'none') return '/welcome'
  if (r.kind === 'single') return '/'
  return selectedAccountIsValid(r, selectedAccountId) ? '/' : '/welcome'
}
