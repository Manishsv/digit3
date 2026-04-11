import Keycloak, { type KeycloakConfig, type KeycloakTokenParsed } from 'keycloak-js'
import React, { createContext, useContext, useEffect, useMemo, useState } from 'react'

type AuthState = {
  mode: 'keycloak' | 'dev-local'
  ready: boolean
  isAuthenticated: boolean
  token?: string
  tokenParsed?: KeycloakTokenParsed
  roles: Set<string>
  tenantId?: string
  profile?: { username?: string }
  login: () => void
  logout: () => void
}

const Ctx = createContext<AuthState | null>(null)

const STORAGE_KEY = 'digit.demoUi.authConfig.v1'

/**
 * React 18 Strict Mode runs effects twice on mount. A second Keycloak.init() would run after the
 * first exchange already consumed the OAuth `code` in the URL, leaving no session — user stuck on /login.
 * One client + one init promise per Keycloak config fingerprint; listeners use a generation guard.
 */
let kcBootstrapFingerprint = ''
let kcSingleton: Keycloak | null = null
let kcInitPromise: Promise<boolean> | null = null
let kcListenerGeneration = 0

function resetKeycloakBootstrap() {
  kcBootstrapFingerprint = ''
  kcSingleton = null
  kcInitPromise = null
}

export type AuthConfig = {
  mode?: 'keycloak' | 'dev-local'
  keycloakBaseUrl: string
  realm: string
  clientId: string
  devTenantId?: string
  devClientId?: string
  devRoles?: string[]
}

/**
 * If the user still has Keycloak on :8080 saved, OIDC + silent SSO hit :8080 and the browser blocks
 * framing (CSP frame-ancestors). Vite proxies `/keycloak` on the SPA origin — rewrite saved URL once.
 *
 * OAuth 2.0 redirect_uri must not include a fragment. keycloak-js defaults to `location.href`, which
 * does include the hash — Keycloak may reject the authorize request with HTTP 400.
 */
export function spaOidcRedirectUri(): string {
  if (typeof window === 'undefined') return ''
  const { origin, pathname, search } = window.location
  return `${origin}${pathname}${search}`
}

export function migrateKeycloakBaseUrlForViteProxy(saved: string): string {
  if (typeof window === 'undefined') return saved
  const norm = saved.replace(/\/$/, '')
  if (!/^https?:\/\/(localhost|127\.0\.0\.1):8080\/keycloak$/i.test(norm)) return saved
  let spa: URL
  try {
    spa = new URL(window.location.origin)
  } catch {
    return saved
  }
  if (spa.port === '8080') return saved
  return `${window.location.origin.replace(/\/$/, '')}/keycloak`
}

export function loadAuthConfig(): AuthConfig {
  const raw = localStorage.getItem(STORAGE_KEY)
  if (raw) {
    try {
      const parsed = JSON.parse(raw) as Partial<AuthConfig>
      if (parsed.keycloakBaseUrl && parsed.realm && parsed.clientId) {
        const cfg = parsed as AuthConfig
        const migrated = migrateKeycloakBaseUrlForViteProxy(cfg.keycloakBaseUrl)
        if (migrated !== cfg.keycloakBaseUrl) {
          cfg.keycloakBaseUrl = migrated
          localStorage.setItem(STORAGE_KEY, JSON.stringify(cfg))
        }
        return cfg
      }
    } catch {
      // ignore
    }
  }
  const defaultKeycloakBase =
    typeof window !== 'undefined' && window.location?.origin
      ? `${window.location.origin.replace(/\/$/, '')}/keycloak`
      : 'http://localhost:8080/keycloak'

  return {
    mode: 'keycloak',
    keycloakBaseUrl: defaultKeycloakBase,
    realm: '',
    clientId: 'demo-ui',
    devTenantId: '',
    devClientId: 'demo-ui',
    devRoles: [],
  }
}

export function saveAuthConfig(cfg: AuthConfig) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(cfg))
}

export function AuthProvider(props: { children: React.ReactNode }) {
  const [ready, setReady] = useState(false)
  const [kc, setKc] = useState<Keycloak | null>(null)
  const [token, setToken] = useState<string | undefined>(undefined)
  const [tokenParsed, setTokenParsed] = useState<KeycloakTokenParsed | undefined>(undefined)
  const [roles, setRoles] = useState<Set<string>>(new Set())
  const [mode, setMode] = useState<'keycloak' | 'dev-local'>('keycloak')
  const [devTenantId, setDevTenantId] = useState<string>('')
  const [devClientId, setDevClientId] = useState<string>('demo-ui')

  useEffect(() => {
    const cfg = loadAuthConfig()
    const effMode = (cfg.mode || 'keycloak') as 'keycloak' | 'dev-local'
    setMode(effMode)

    if (effMode === 'dev-local') {
      resetKeycloakBootstrap()
      const tid = (cfg.devTenantId || cfg.realm || '').trim()
      const cid = (cfg.devClientId || 'demo-ui').trim() || 'demo-ui'
      setDevTenantId(tid)
      setDevClientId(cid)
      setKc(null)
      setToken('dev-local')
      setTokenParsed(undefined)
      const r = new Set<string>((cfg.devRoles || []).filter(Boolean))
      // If user hasn't chosen roles yet, default to "all tabs visible" for local demo.
      if (r.size === 0) {
        ;[
          'COORDINATION_ADMIN',
          'COORDINATION_WRITER',
          'COORDINATION_READER',
          'PARTICIPANT_ADMIN',
          'AUTHORITY_ADMIN',
          'SYSTEM_EMITTER',
        ].forEach((x) => r.add(x))
      }
      setRoles(r)
      setReady(true)
      return
    }

    const realm = (cfg.realm || '').trim()
    if (!realm || realm === 'REPLACE_WITH_TENANT_REALM') {
      resetKeycloakBootstrap()
      setKc(null)
      setReady(true)
      return
    }
    const kcc: KeycloakConfig = {
      url: cfg.keycloakBaseUrl.replace(/\/$/, ''),
      realm,
      clientId: cfg.clientId,
    }
    const fp = `${kcc.url ?? ''}|${kcc.realm}|${kcc.clientId}`
    if (kcBootstrapFingerprint !== fp) {
      kcBootstrapFingerprint = fp
      kcSingleton = new Keycloak(kcc)
      kcInitPromise = null
    }
    const client = kcSingleton!
    setKc(client)

    if (!kcInitPromise) {
      kcInitPromise = client.init({
        // Avoid onLoad: 'check-sso': it uses prompt=none in a hidden iframe and Keycloak often returns HTTP 400
        // when there is no session or silent redirect URIs are misconfigured. Interactive login still works;
        // returning from Keycloak is handled via #processInit parsing the URL hash before onLoad runs.
        pkceMethod: 'S256',
        checkLoginIframe: false,
        redirectUri: spaOidcRedirectUri(),
      })
    }

    const myGen = ++kcListenerGeneration

    const applyFromClient = () => {
      if (myGen !== kcListenerGeneration) return
      if (window.location.hash.includes('error=login_required')) {
        window.history.replaceState(null, document.title, window.location.pathname + window.location.search)
      }
      setToken(client.token)
      setTokenParsed(client.tokenParsed as any)
      const r = new Set<string>(((client.tokenParsed as any)?.realm_access?.roles as string[]) || [])
      setRoles(r)
      setReady(true)
    }

    kcInitPromise
      .then(() => {
        if (myGen !== kcListenerGeneration) return
        applyFromClient()
      })
      .catch(() => {
        if (myGen !== kcListenerGeneration) return
        if (window.location.hash.includes('error=login_required')) {
          window.history.replaceState(null, document.title, window.location.pathname + window.location.search)
        }
        setReady(true)
      })

    const onToken = () => {
      if (myGen !== kcListenerGeneration) return
      setToken(client.token)
      setTokenParsed(client.tokenParsed as any)
      const r = new Set<string>(((client.tokenParsed as any)?.realm_access?.roles as string[]) || [])
      setRoles(r)
    }
    client.onAuthSuccess = onToken
    client.onAuthRefreshSuccess = onToken
    client.onTokenExpired = () => {
      client.updateToken(30).catch(() => {
        onToken()
      })
    }

    return () => {
      kcListenerGeneration++
    }
  }, [])

  const tenantId = useMemo(() => {
    if (mode === 'dev-local') return (devTenantId || '').trim() || undefined
    const iss = (tokenParsed as any)?.iss as string | undefined
    if (!iss) return undefined
    const parts = iss.replace(/\/$/, '').split('/')
    return parts[parts.length - 1]
  }, [tokenParsed, mode, devTenantId])

  const value: AuthState = useMemo(
    () => ({
      mode,
      ready,
      isAuthenticated: Boolean(token),
      token,
      tokenParsed,
      roles,
      tenantId,
      profile: {
        username:
          mode === 'dev-local'
            ? devClientId
            : ((tokenParsed as any)?.preferred_username as string | undefined),
      },
      login: () => {
        if (mode === 'dev-local') return
        kc?.login({ redirectUri: spaOidcRedirectUri() })
      },
      logout: () => {
        if (mode === 'dev-local') {
          setToken(undefined)
          setRoles(new Set())
          setReady(true)
          return
        }
        kc?.logout({ redirectUri: spaOidcRedirectUri() })
      },
    }),
    [mode, ready, token, tokenParsed, roles, tenantId, kc, devClientId],
  )

  return <Ctx.Provider value={value}>{props.children}</Ctx.Provider>
}

export function useAuth(): AuthState {
  const v = useContext(Ctx)
  if (!v) throw new Error('useAuth must be used within AuthProvider')
  return v
}

