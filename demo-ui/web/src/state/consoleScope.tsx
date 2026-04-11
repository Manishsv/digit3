import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { ApiError, digitJson, useDigitHeaders } from '../lib/digitApi'
import type { StudioServiceRow } from '../types/studio'
import { useSelection } from './selection'

type ConsoleScopeValue = {
  services: StudioServiceRow[]
  servicesLoading: boolean
  servicesError: string | null
  refreshServices: () => void
  /** Services for the currently selected account (empty if none selected). */
  scopedServices: StudioServiceRow[]
}

const Ctx = createContext<ConsoleScopeValue | null>(null)

export function ConsoleScopeProvider(props: { children: React.ReactNode }) {
  const headers = useDigitHeaders()
  const sel = useSelection()
  const [services, setServices] = useState<StudioServiceRow[]>([])
  const [servicesLoading, setServicesLoading] = useState(false)
  const [servicesError, setServicesError] = useState<string | null>(null)

  const refreshServices = useCallback(() => {
    setServicesLoading(true)
    setServicesError(null)
    digitJson<{ services: StudioServiceRow[] }>('studio', '/studio/v1/services', { headers })
      .then((r) => setServices(r.services || []))
      .catch((e: unknown) =>
        setServicesError(e instanceof ApiError ? `${e.message}\n${e.bodyText}` : String(e)),
      )
      .finally(() => setServicesLoading(false))
  }, [headers])

  useEffect(() => {
    refreshServices()
  }, [refreshServices])

  const value = useMemo<ConsoleScopeValue>(() => {
    const scopedServices = sel.selectedAccountId
      ? services.filter((s) => String(s.tenant_id) === String(sel.selectedAccountId))
      : services
    return {
      services,
      servicesLoading,
      servicesError,
      refreshServices,
      scopedServices,
    }
  }, [services, servicesLoading, servicesError, refreshServices, sel.selectedAccountId])

  return <Ctx.Provider value={value}>{props.children}</Ctx.Provider>
}

export function useConsoleScope(): ConsoleScopeValue {
  const v = useContext(Ctx)
  if (!v) throw new Error('useConsoleScope must be used within ConsoleScopeProvider')
  return v
}
