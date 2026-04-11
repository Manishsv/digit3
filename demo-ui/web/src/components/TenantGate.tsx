import { useEffect } from 'react'
import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { isPlatformDirectoryAdmin } from '../console/roles'
import { resolveTenantResolution, selectedAccountIsValid } from '../lib/tenantResolution'
import { useAuth } from '../state/auth'
import { useSelection } from '../state/selection'

/** Platform shell is allowed without tenant context only for directory admins (manual browser directory). */
function allowShellWithoutMembership(pathname: string, roles: Set<string>) {
  return pathname === '/platform' && isPlatformDirectoryAdmin(roles)
}

/**
 * Ensures a tenant (account) context exists before rendering the console shell.
 * - 0 memberships → /welcome (except `/platform` so users can add an account)
 * - 1 membership → auto-select that account
 * - N memberships → require a valid selected account, else /welcome
 */
export function TenantGate() {
  const auth = useAuth()
  const sel = useSelection()
  const loc = useLocation()
  const resolution = resolveTenantResolution(auth, sel.accounts)

  useEffect(() => {
    if (resolution.kind === 'single' && sel.selectedAccountId !== resolution.account.id) {
      sel.setSelectedAccountId(resolution.account.id)
    }
  }, [resolution, sel])

  if (resolution.kind === 'none') {
    if (allowShellWithoutMembership(loc.pathname, auth.roles)) return <Outlet />
    return <Navigate to="/welcome" replace />
  }

  if (resolution.kind === 'many') {
    const ok = selectedAccountIsValid(resolution, sel.selectedAccountId)
    if (!ok) return <Navigate to="/welcome" replace />
  }

  return <Outlet />
}
