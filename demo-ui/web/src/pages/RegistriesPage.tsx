import { useMemo, useState } from 'react'
import { ApiError, digitJson, useDigitHeaders } from '../lib/digitApi'
import { useSelection } from '../state/selection'

type RegistrySearchResp = Record<string, unknown>

export function RegistriesPage() {
  const headers = useDigitHeaders()
  const sel = useSelection()
  const [tab, setTab] = useState<'schemas' | 'data'>('schemas')

  const [schemaCode, setSchemaCode] = useState('complaints.case')
  const [registryId, setRegistryId] = useState('')
  const [serviceRequestId, setServiceRequestId] = useState('')
  const [out, setOut] = useState<Record<string, unknown> | null>(null)
  const [err, setErr] = useState<string | null>(null)

  const scopedHelp = useMemo(() => {
    const a = sel.selectedAccountId ? `account=${sel.selectedAccountId}` : 'account=—'
    const s = sel.selectedServiceCode ? `service=${sel.selectedServiceCode}` : 'service=—'
    return `${a}, ${s}`
  }, [sel.selectedAccountId, sel.selectedServiceCode])

  async function fetchSchema() {
    setErr(null)
    setOut(null)
    try {
      // Commonly supported in DIGIT Registry.
      const r = await digitJson<Record<string, unknown>>(
        'registry',
        `/registry/v1/schema/${encodeURIComponent(schemaCode)}`,
        { headers },
      )
      setOut(r)
    } catch (e: unknown) {
      setErr(e instanceof ApiError ? `${e.message}\n${e.bodyText}` : String(e))
    }
  }

  async function fetchByRegistryId() {
    setErr(null)
    setOut(null)
    try {
      const r = await digitJson<Record<string, unknown>>(
        'registry',
        `/registry/v1/schema/${encodeURIComponent(schemaCode)}/data/_registry?registryId=${encodeURIComponent(registryId)}`,
        { headers },
      )
      setOut(r)
    } catch (e: unknown) {
      setErr(e instanceof ApiError ? `${e.message}\n${e.bodyText}` : String(e))
    }
  }

  async function searchByServiceRequestId() {
    setErr(null)
    setOut(null)
    try {
      // If the registry supports _search, this will work; otherwise we show the error verbatim.
      const r = await digitJson<RegistrySearchResp>(
        'registry',
        `/registry/v1/schema/${encodeURIComponent(schemaCode)}/data/_search`,
        {
          method: 'POST',
          headers: { ...headers, 'Content-Type': 'application/json' },
          body: JSON.stringify({
            query: {
              // Conservative: many registry search APIs accept a "query" object; backend may ignore unknown fields.
              serviceRequestId,
              serviceCode: sel.selectedServiceCode,
            },
          }),
        },
      )
      setOut(r as any)
    } catch (e: unknown) {
      setErr(e instanceof ApiError ? `${e.message}\n${e.bodyText}` : String(e))
    }
  }

  return (
    <div style={{ maxWidth: 980 }}>
      <h2 style={{ marginTop: 0 }}>Registries</h2>
      <p style={{ color: '#6b7280' }}>Manage schemas and data for the selected scope ({scopedHelp}).</p>

      <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 10 }}>
        <button onClick={() => setTab('schemas')} disabled={tab === 'schemas'}>
          Schemas
        </button>
        <button onClick={() => setTab('data')} disabled={tab === 'data'}>
          Data
        </button>
      </div>

      <div style={{ display: 'grid', gap: 10 }}>
        <div style={{ display: 'grid', gap: 6 }}>
          <label>schemaCode</label>
          <input value={schemaCode} onChange={(e) => setSchemaCode(e.target.value)} />
        </div>

        {tab === 'schemas' ? (
          <div style={{ display: 'flex', gap: 10, alignItems: 'center', flexWrap: 'wrap' }}>
            <button onClick={fetchSchema}>View schema</button>
          </div>
        ) : (
          <div style={{ display: 'grid', gap: 10 }}>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 2fr auto', gap: 10, alignItems: 'end' }}>
              <div style={{ display: 'grid', gap: 6 }}>
                <label>registryId</label>
                <input value={registryId} onChange={(e) => setRegistryId(e.target.value)} placeholder="registry UUID" />
              </div>
              <div style={{ display: 'grid', gap: 6 }}>
                <label>serviceRequestId (caseId)</label>
                <input
                  value={serviceRequestId}
                  onChange={(e) => setServiceRequestId(e.target.value)}
                  placeholder="CS-..."
                />
              </div>
              <div style={{ display: 'flex', gap: 10 }}>
                <button onClick={fetchByRegistryId} disabled={!registryId.trim()}>
                  Get by registryId
                </button>
                <button onClick={searchByServiceRequestId} disabled={!serviceRequestId.trim()}>
                  Search
                </button>
              </div>
            </div>
          </div>
        )}

        {err ? <pre style={{ whiteSpace: 'pre-wrap' }}>{err}</pre> : null}
        {out ? <pre style={{ whiteSpace: 'pre-wrap' }}>{JSON.stringify(out, null, 2)}</pre> : null}
      </div>
    </div>
  )
}

