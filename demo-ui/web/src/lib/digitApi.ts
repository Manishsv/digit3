import { useMemo } from 'react'
import { useAuth } from '../state/auth'
import { useSelection } from '../state/selection'

export type DigitService =
  | 'studio'
  | 'governance'
  | 'coordination'
  | 'registry'
  | 'workflow'
  | 'mdms'
  | 'idgen'
  | 'boundary'
  | 'account'

export class ApiError extends Error {
  status: number
  bodyText: string
  constructor(message: string, status: number, bodyText: string) {
    super(message)
    this.status = status
    this.bodyText = bodyText
  }
}

export async function digitFetch(
  service: DigitService,
  path: string,
  init: RequestInit & { headers?: Record<string, string> } = {},
): Promise<Response> {
  const url = `/api/${service}${path.startsWith('/') ? path : `/${path}`}`
  const headers = new Headers(init.headers || {})
  return fetch(url, { ...init, headers })
}

/** Try several paths; many DIGIT services have no root `/health` (context path lives under e.g. `/workflow`, `/idgen`). */
export async function probeDigitService(
  service: DigitService,
  candidates: { path: string; init?: RequestInit }[],
  headers: Record<string, string>,
): Promise<{ ok: boolean; via?: string; status?: number; body?: unknown; error?: string }> {
  const tried: string[] = []
  let lastFail: { via?: string; status?: number; body?: unknown; error?: string } = {}
  for (const { path, init } of candidates) {
    tried.push(path)
    try {
      const r = await digitFetch(service, path, {
        ...init,
        headers: { ...headers, ...(init?.headers as Record<string, string> | undefined) },
      })
      const text = await r.text()
      let body: unknown = text
      try {
        body = JSON.parse(text) as unknown
      } catch {
        // keep raw string
      }
      if (r.ok) return { ok: true, via: path, status: r.status, body }
      // Service is up but probe path may be wrong or auth required.
      if (r.status === 401 || r.status === 403)
        return { ok: true, via: path, status: r.status, body: { note: 'reachable', auth: true } }
      if (r.status === 404) {
        lastFail = { via: path, status: r.status, body }
        continue
      }
      if (r.status < 500) return { ok: true, via: path, status: r.status, body }
      lastFail = { via: path, status: r.status, body, error: `${service} ${r.status}` }
    } catch (e: unknown) {
      lastFail = { error: String(e) }
      continue
    }
  }
  return { ok: false, ...lastFail, error: lastFail.error || `no matching route (tried ${tried.join(', ')})` }
}

/** Self-service tenant creation (Account API). No bearer token — uses X-Client-ID only. */
const REGISTER_CLIENT_ID = 'digit-demo-ui-register'

export type RegisterTenantBody = {
  tenant: {
    name: string
    email: string
    isActive: true
    code?: string
    additionalAttributes?: Record<string, unknown>
  }
}

export type RegisterTenantResponse = {
  tenants: Array<{
    id?: string
    code?: string
    name?: string
    email?: string
  }>
}

export async function registerDigitTenant(body: RegisterTenantBody): Promise<RegisterTenantResponse> {
  const r = await digitFetch('account', '/account/v1', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Client-ID': REGISTER_CLIENT_ID,
    },
    body: JSON.stringify(body),
  })
  const text = await r.text()
  if (r.status >= 400) {
    throw new ApiError(`account ${r.status}`, r.status, text)
  }
  try {
    return JSON.parse(text) as RegisterTenantResponse
  } catch {
    return { tenants: [] }
  }
}

export function registerTenantErrorMessage(bodyText: string): string {
  try {
    const o = JSON.parse(bodyText) as { errors?: Array<{ message?: string; description?: string }> }
    const e = o.errors?.[0]
    if (e?.message) return e.description ? `${e.message}: ${e.description}` : e.message
  } catch {
    /* ignore */
  }
  return bodyText.slice(0, 500)
}

export async function digitJson<T>(
  service: DigitService,
  path: string,
  init: RequestInit & { headers?: Record<string, string> } = {},
): Promise<T> {
  const r = await digitFetch(service, path, init)
  const text = await r.text()
  if (r.status >= 400) {
    throw new ApiError(`${service} ${r.status}`, r.status, text)
  }
  try {
    return JSON.parse(text) as T
  } catch {
    return { _raw: text } as unknown as T
  }
}

export function useDigitHeaders(): Record<string, string> {
  const auth = useAuth()
  const sel = useSelection()
  const token = auth.token
  const tenantId = sel.selectedAccountId || auth.tenantId
  const clientId = auth.profile?.username || 'demo-ui'
  return useMemo(
    () => ({
      Authorization: token ? `Bearer ${token}` : '',
      'X-Tenant-ID': tenantId || '',
      'X-Client-ID': clientId,
      'X-Client-Id': clientId,
    }),
    [token, tenantId, clientId],
  )
}

