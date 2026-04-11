import { useState } from 'react'
import { ApiError, digitJson, useDigitHeaders } from '../lib/digitApi'

type WorkflowProcess = { id: string; code: string }

export function CsrPage() {
  const headers = useDigitHeaders()
  const [serviceCode, setServiceCode] = useState('PGR67')
  const [caseId, setCaseId] = useState('CS-DEMO-001')
  const [action, setAction] = useState('ASSIGN')
  const [comment, setComment] = useState('CSR action (demo)')
  const [out, setOut] = useState<Record<string, unknown> | null>(null)
  const [err, setErr] = useState<string | null>(null)

  async function transition() {
    setErr(null)
    setOut(null)
    try {
      const procs = await digitJson<WorkflowProcess[]>(
        'workflow',
        `/workflow/v1/process?code=${encodeURIComponent(serviceCode)}`,
        { headers },
      )
      const pid = procs?.[0]?.id
      if (!pid) throw new Error(`No workflow process for code=${serviceCode}`)
      const resp = await digitJson<Record<string, unknown>>('workflow', '/workflow/v1/transition', {
        method: 'POST',
        headers: { ...headers, 'Content-Type': 'application/json' },
        body: JSON.stringify({
          processId: pid,
          entityId: caseId,
          action,
          init: false,
          comment,
          attributes: { roles: ['CSR'] },
        }),
      })
      setOut(resp)
    } catch (e: unknown) {
      setErr(e instanceof ApiError ? `${e.message}\n${e.bodyText}` : String(e))
    }
  }

  async function fetchHistory() {
    setErr(null)
    setOut(null)
    try {
      const procs = await digitJson<WorkflowProcess[]>(
        'workflow',
        `/workflow/v1/process?code=${encodeURIComponent(serviceCode)}`,
        { headers },
      )
      const pid = procs?.[0]?.id
      if (!pid) throw new Error(`No workflow process for code=${serviceCode}`)
      const url = `/workflow/v1/transition?entityId=${encodeURIComponent(caseId)}&processId=${encodeURIComponent(pid)}&history=true`
      const resp = await digitJson<Record<string, unknown>>('workflow', url, { headers })
      setOut(resp)
    } catch (e: unknown) {
      setErr(e instanceof ApiError ? `${e.message}\n${e.bodyText}` : String(e))
    }
  }

  return (
    <div>
      <h2 style={{ marginTop: 0 }}>Operator</h2>
      <p style={{ color: '#374151' }}>Execute the service process: workflow transitions for an existing caseId.</p>

      <div style={{ display: 'grid', gap: 10, maxWidth: 900 }}>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
          <div style={{ display: 'grid', gap: 6 }}>
            <label>serviceCode</label>
            <input value={serviceCode} onChange={(e) => setServiceCode(e.target.value)} />
          </div>
          <div style={{ display: 'grid', gap: 6 }}>
            <label>caseId (entityId)</label>
            <input value={caseId} onChange={(e) => setCaseId(e.target.value)} />
          </div>
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
          <div style={{ display: 'grid', gap: 6 }}>
            <label>action</label>
            <input value={action} onChange={(e) => setAction(e.target.value)} />
          </div>
          <div style={{ display: 'grid', gap: 6 }}>
            <label>comment</label>
            <input value={comment} onChange={(e) => setComment(e.target.value)} />
          </div>
        </div>

        <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap' }}>
          <button onClick={transition}>Submit transition</button>
          <button onClick={fetchHistory}>Fetch transition history</button>
        </div>

        {err && (
          <pre style={{ background: '#fee2e2', padding: 12, borderRadius: 12, overflow: 'auto' }}>{err}</pre>
        )}
        {out && (
          <pre style={{ background: '#f3f4f6', padding: 12, borderRadius: 12, overflow: 'auto' }}>
            {JSON.stringify(out, null, 2)}
          </pre>
        )}
      </div>
    </div>
  )
}


