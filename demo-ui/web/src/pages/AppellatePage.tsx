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

export function AppellatePage() {
  const headers = useDigitHeaders()
  const last = useMemo(() => loadLastDecision(), [])
  const [tab, setTab] = useState<'appeal' | 'order' | 'recompute' | 'artifacts'>('appeal')
  const [decisionId, setDecisionId] = useState<string>(last?.response?.decisionId || '')
  const [receiptId, setReceiptId] = useState<string>(last?.response?.receiptId || '')
  const [caseId, setCaseId] = useState<string>(last?.caseId || 'CS-DEMO-001')
  const [appealId, setAppealId] = useState<string>('')
  const [orderId, setOrderId] = useState<string>('')
  const [parentDecisionId, setParentDecisionId] = useState<string>(decisionId)
  const [factsJson, setFactsJson] = useState<string>('{"application":{"status":"SUBMITTED"}}')
  const [out, setOut] = useState<Record<string, unknown> | null>(null)
  const [err, setErr] = useState<string | null>(null)

  const ruleset = last?.ruleset

  async function createAppeal() {
    setErr(null)
    setOut(null)
    try {
      const resp = await digitJson<{ appealId: string; registryRecordId?: string }>('governance', '/governance/v1/appeals', {
        method: 'POST',
        headers: { ...headers, 'Content-Type': 'application/json' },
        body: JSON.stringify({
          receiptId,
          decisionId,
          filedBy: headers['X-Client-ID'] || 'user',
          grounds: 'Request review (demo)',
          status: 'FILED',
          metadata: {},
        }),
      })
      setAppealId(resp.appealId)
      setOut(resp as any)
    } catch (e: unknown) {
      if (e instanceof ApiError) setErr(`${e.message}\n${e.bodyText}`)
      else setErr(String(e))
    }
  }

  async function issueOrder() {
    setErr(null)
    setOut(null)
    try {
      const resp = await digitJson<{ orderId: string; registryRecordId?: string }>('governance', '/governance/v1/orders', {
        method: 'POST',
        headers: { ...headers, 'Content-Type': 'application/json' },
        body: JSON.stringify({
          appealId,
          decisionId,
          receiptId,
          issuedBy: headers['X-Client-ID'] || 'authority',
          outcome: 'REMAND',
          instructions: 'Recompute with corrected facts (demo)',
          metadata: {},
        }),
      })
      setOrderId(resp.orderId)
      setOut(resp as any)
    } catch (e: unknown) {
      if (e instanceof ApiError) setErr(`${e.message}\n${e.bodyText}`)
      else setErr(String(e))
    }
  }

  async function recompute() {
    setErr(null)
    setOut(null)
    if (!ruleset?.rulesetId || !ruleset?.version) {
      setErr('Missing ruleset context. Run a decision from Regulator tab first.')
      return
    }
    let facts: unknown
    try {
      facts = JSON.parse(factsJson)
    } catch (e) {
      setErr(`Invalid JSON: ${String(e)}`)
      return
    }
    try {
      const resp = await digitJson<Record<string, unknown>>('governance', '/governance/v1/decisions:recompute', {
        method: 'POST',
        headers: { ...headers, 'Content-Type': 'application/json' },
        body: JSON.stringify({
          decisionType: 'SBL_LICENSE',
          correlationId: `ui-${Date.now()}`,
          requestId: `ui-${Date.now()}`,
          channel: 'web',
          parentDecisionId,
          appealId: appealId || undefined,
          orderId: orderId || undefined,
          caseRef: { system: 'coordination', entityType: 'Case', entityId: caseId },
          ruleset: {
            rulesetId: ruleset.rulesetId,
            version: ruleset.version,
            registryRecordId: ruleset.registryRecordId,
          },
          factsSnapshot: facts,
          mdmsFactChecks: [],
          factsContractCode: last?.factsContractCode || 'SBL_DEFAULT',
          factsContractVersion: last?.factsContractVersion || '1',
        }),
      })
      setOut(resp)
    } catch (e: unknown) {
      if (e instanceof ApiError) setErr(`${e.message}\n${e.bodyText}`)
      else setErr(String(e))
    }
  }

  async function fetchReceipt() {
    setErr(null)
    setOut(null)
    try {
      const resp = await digitJson<Record<string, unknown>>('governance', `/governance/v1/decisions/${encodeURIComponent(receiptId)}`, {
        headers,
      })
      setOut(resp)
    } catch (e: unknown) {
      if (e instanceof ApiError) setErr(`${e.message}\n${e.bodyText}`)
      else setErr(String(e))
    }
  }

  return (
    <div>
      <h2 style={{ marginTop: 0 }}>Appellate authority (dispute resolution)</h2>
      <p style={{ color: '#374151' }}>Resolve disputes: appeal → order → recompute (lineage) + receipt fetch.</p>

      <div style={{ display: 'grid', gap: 10, maxWidth: 900 }}>
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
          <button onClick={() => setTab('appeal')} disabled={tab === 'appeal'}>
            Appeal
          </button>
          <button onClick={() => setTab('order')} disabled={tab === 'order'}>
            Order
          </button>
          <button onClick={() => setTab('recompute')} disabled={tab === 'recompute'}>
            Recompute
          </button>
          <button onClick={() => setTab('artifacts')} disabled={tab === 'artifacts'}>
            Artifacts
          </button>
        </div>

        <div style={{ display: 'grid', gap: 6 }}>
          <label>caseId</label>
          <input value={caseId} onChange={(e) => setCaseId(e.target.value)} />
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
          <div style={{ display: 'grid', gap: 6 }}>
            <label>decisionId</label>
            <input value={decisionId} onChange={(e) => setDecisionId(e.target.value)} />
          </div>
          <div style={{ display: 'grid', gap: 6 }}>
            <label>receiptId</label>
            <input value={receiptId} onChange={(e) => setReceiptId(e.target.value)} />
          </div>
        </div>

        {tab === 'appeal' ? (
          <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap' }}>
            <button onClick={createAppeal} disabled={!decisionId || !receiptId}>
              Create appeal
            </button>
          </div>
        ) : null}
        {tab === 'order' ? (
          <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap' }}>
            <button onClick={issueOrder} disabled={!appealId}>
              Issue order (REMAND)
            </button>
          </div>
        ) : null}
        {tab === 'artifacts' ? (
          <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap' }}>
            <button onClick={fetchReceipt} disabled={!receiptId}>
              Fetch receipt (audit)
            </button>
          </div>
        ) : null}

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
          <div style={{ display: 'grid', gap: 6 }}>
            <label>appealId</label>
            <input value={appealId} onChange={(e) => setAppealId(e.target.value)} />
          </div>
          <div style={{ display: 'grid', gap: 6 }}>
            <label>orderId</label>
            <input value={orderId} onChange={(e) => setOrderId(e.target.value)} />
          </div>
        </div>

        {tab === 'recompute' ? (
          <>
            <div style={{ display: 'grid', gap: 6 }}>
              <label>parentDecisionId (for recompute)</label>
              <input value={parentDecisionId} onChange={(e) => setParentDecisionId(e.target.value)} />
            </div>

            <div style={{ display: 'grid', gap: 6 }}>
              <label>factsSnapshot (JSON)</label>
              <textarea
                value={factsJson}
                onChange={(e) => setFactsJson(e.target.value)}
                rows={10}
                style={{ fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace' }}
              />
            </div>

            <button onClick={recompute} disabled={!parentDecisionId}>
              Recompute
            </button>
          </>
        ) : null}

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

