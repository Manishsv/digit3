/** Realm role that can manage the browser account directory and bypass tenant gate to Platform (demo). */
export const PLATFORM_DIRECTORY_ROLE = 'COORDINATION_ADMIN' as const

/** Roles that can manage the account catalogue (rulesets, registries, groups, service definitions). */
export const ACCOUNT_CATALOGUE_ROLES = ['SUPERUSER', 'COORDINATION_ADMIN'] as const

export function isPlatformDirectoryAdmin(roles: Set<string>): boolean {
  return roles.has(PLATFORM_DIRECTORY_ROLE)
}

export function isAccountCatalogueAdmin(roles: Set<string>): boolean {
  for (const r of ACCOUNT_CATALOGUE_ROLES) {
    if (roles.has(r)) return true
  }
  return false
}
