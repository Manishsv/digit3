import { useMemo, useState } from 'react'
import { ApiError, digitJson, useDigitHeaders } from '../lib/digitApi'

const LAST_DECISION_KEY = 'digit.demoUi.lastDecision.v1'

function loadLastDecision(): any | null {
  const raw = localStorage.getItem(LAST_DECISION_KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw)
  } catch {
    return null
  }
}

export function AuditorPage() {
  const headers = useDigitHeaders()
  const last = useMemo(() => loadLastDecision(), [])
  const [tab, setTab] = useState<'artifacts' | 'raw'>('artifacts')
  const [receiptId, setReceiptId] = useState<string>(last?.response?.receiptId || '')
  const [decisionId, setDecisionId] = useState<string>(last?.response?.decisionId || '')
  const [out, setOut] = useState<Record<string, unknown> | null>(null)
  const [err, setErr] = useState<string | null>(null)

  async function fetchReceipt() {
    setErr(null)
    setOut(null)
    try {
      const resp = await digitJson<Record<string, unknown>>(
        'governance',
        `/governance/v1/receipts/${encodeURIComponent(receiptId)}`,
        { headers },
      )
      setOut(resp)
    } catch (e: unknown) {
      setErr(e instanceof ApiError ? `${e.message}\n${e.bodyText}` : String(e))
    }
  }

  async function fetchTrace() {
    setErr(null)
    setOut(null)
    try {
      const resp = await digitJson<Record<string, unknown>>(
        'governance',
        `/governance/v1/decisions/${encodeURIComponent(decisionId)}/trace`,
        { headers },
      )
      setOut(resp)
    } catch (e: unknown) {
      setErr(e instanceof ApiError ? `${e.message}\n${e.bodyText}` : String(e))
    }
  }

  return (
    <div style={{ maxWidth: 980 }}>
      <h2 style={{ marginTop: 0 }}>Auditor</h2>
      <p style={{ color: '#374151' }}>Read-only: inspect decision artifacts (receipt/trace) and logs.</p>

      <div style={{ display: 'grid', gap: 10 }}>
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
          <button onClick={() => setTab('artifacts')} disabled={tab === 'artifacts'}>
            Artifacts
          </button>
          <button onClick={() => setTab('raw')} disabled={tab === 'raw'}>
            Raw (last decision)
          </button>
        </div>

        {tab === 'raw' ? (
          <pre style={{ whiteSpace: 'pre-wrap' }}>{JSON.stringify(last, null, 2)}</pre>
        ) : null}

        {tab === 'artifacts' ? (
          <>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
              <div style={{ display: 'grid', gap: 6 }}>
                <label>receiptId</label>
                <input value={receiptId} onChange={(e) => setReceiptId(e.target.value)} />
              </div>
              <div style={{ display: 'grid', gap: 6 }}>
                <label>decisionId</label>
                <input value={decisionId} onChange={(e) => setDecisionId(e.target.value)} />
              </div>
            </div>

            <div style={{ display: 'flex', gap: 10, alignItems: 'center' }}>
              <button onClick={fetchReceipt} disabled={!receiptId.trim()}>
                Fetch receipt
              </button>
              <button onClick={fetchTrace} disabled={!decisionId.trim()}>
                Fetch trace
              </button>
            </div>
          </>
        ) : null}

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

