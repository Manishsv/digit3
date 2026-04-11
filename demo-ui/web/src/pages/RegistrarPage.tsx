import { useState } from 'react'
import { ApiError, digitJson, useDigitHeaders } from '../lib/digitApi'

export function RegistrarPage() {
  const headers = useDigitHeaders()
  const [schemaCode, setSchemaCode] = useState('complaints.case')
  const [registryId, setRegistryId] = useState('')
  const [out, setOut] = useState<Record<string, unknown> | null>(null)
  const [err, setErr] = useState<string | null>(null)

  async function loadByRegistryId() {
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

  return (
    <div style={{ maxWidth: 980 }}>
      <h2 style={{ marginTop: 0 }}>Registrar (record authority)</h2>
      <p style={{ color: '#374151' }}>
        Establishes/maintains authoritative facts or statuses. This page is a minimal read helper for registry-backed facts.
      </p>

      <div style={{ display: 'grid', gap: 10 }}>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 2fr', gap: 10 }}>
          <div style={{ display: 'grid', gap: 6 }}>
            <label>schemaCode</label>
            <input value={schemaCode} onChange={(e) => setSchemaCode(e.target.value)} />
          </div>
          <div style={{ display: 'grid', gap: 6 }}>
            <label>registryId</label>
            <input value={registryId} onChange={(e) => setRegistryId(e.target.value)} placeholder="e.g. registry UUID" />
          </div>
        </div>

        <div style={{ display: 'flex', gap: 10, alignItems: 'center' }}>
          <button onClick={loadByRegistryId} disabled={!registryId.trim()}>
            Load authoritative record
          </button>
        </div>

        {err ? (
          <pre style={{ whiteSpace: 'pre-wrap', color: '#b91c1c', background: '#fff1f2', padding: 12, borderRadius: 8 }}>
            {err}
          </pre>
        ) : null}
        {out ? (
          <pre style={{ whiteSpace: 'pre-wrap', background: '#f8fafc', padding: 12, borderRadius: 8 }}>
            {JSON.stringify(out, null, 2)}
          </pre>
        ) : null}
      </div>
    </div>
  )
}

