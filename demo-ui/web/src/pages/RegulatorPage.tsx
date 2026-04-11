import { useMemo, useState } from 'react'
import { ApiError, digitJson, useDigitHeaders } from '../lib/digitApi'

type PublishedRuleset = {
  rulesetId: string
  code: string
  version: string
  registryRecordId?: string
}

const RS_KEY = 'digit.demoUi.lastPublishedRulesets.v1'
const LAST_DECISION_KEY = 'digit.demoUi.lastDecision.v1'

function loadRulesets(): PublishedRuleset[] {
  const raw = localStorage.getItem(RS_KEY)
  if (!raw) return []
  try {
    const parsed = JSON.parse(raw)
    return Array.isArray(parsed) ? (parsed as PublishedRuleset[]) : []
  } catch {
    return []
  }
}

export function RegulatorPage() {
  const headers = useDigitHeaders()
  const [caseId, setCaseId] = useState('CS-DEMO-001')
  const rulesets = useMemo(() => loadRulesets(), [])
  const [idx, setIdx] = useState(0)
  const [factsContractCode, setFactsContractCode] = useState('SBL_DEFAULT')
  const [factsContractVersion, setFactsContractVersion] = useState('1')
  const [factsJson, setFactsJson] = useState('{"application":{"status":"SUBMITTED"}}')
  const [out, setOut] = useState<Record<string, unknown> | null>(null)
  const [err, setErr] = useState<string | null>(null)

  const selected = rulesets[idx]

  async function decide() {
    setErr(null)
    setOut(null)
    if (!selected) {
      setErr('No published rulesets found. Publish one from the Studio tab first.')
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
      const body: Record<string, unknown> = {
        correlationId: `ui-${Date.now()}`,
        requestId: `ui-${Date.now()}`,
        channel: 'web',
        rulesetId: selected.rulesetId,
        rulesetVersion: selected.version,
        registryRecordId: selected.registryRecordId,
        factsContractCode,
        factsContractVersion,
        factsSnapshot: facts,
        mdmsFactChecks: [],
      }
      const resp = await digitJson<Record<string, unknown>>(
        'coordination',
        `/coordination/v1/cases/${encodeURIComponent(caseId)}/governance:decide`,
        {
          method: 'POST',
          headers: { ...headers, 'Content-Type': 'application/json' },
          body: JSON.stringify(body),
        },
      )
      setOut(resp)
      localStorage.setItem(
        LAST_DECISION_KEY,
        JSON.stringify({
          caseId,
          ruleset: selected,
          factsContractCode,
          factsContractVersion,
          factsSnapshot: facts,
          response: resp,
        }),
      )
    } catch (e: unknown) {
      if (e instanceof ApiError) setErr(`${e.message}\n${e.bodyText}`)
      else setErr(String(e))
    }
  }

  return (
    <div>
      <h2 style={{ marginTop: 0 }}>Regulator (policy authority)</h2>
      <p style={{ color: '#374151' }}>
        Policy + decision console. Uses published rulesets to evaluate a case (calls Coordination → Governance).
      </p>

      <div style={{ display: 'grid', gap: 10, maxWidth: 900 }}>
        <div style={{ display: 'grid', gap: 6 }}>
          <label>caseId</label>
          <input value={caseId} onChange={(e) => setCaseId(e.target.value)} />
        </div>

        <div style={{ display: 'grid', gap: 6 }}>
          <label>published ruleset</label>
          <select value={String(idx)} onChange={(e) => setIdx(Number(e.target.value))}>
            {rulesets.length === 0 ? (
              <option value="0">No published rulesets yet</option>
            ) : (
              rulesets.map((r, i) => (
                <option key={r.rulesetId} value={String(i)}>
                  {r.code} {r.version} ({r.rulesetId})
                </option>
              ))
            )}
          </select>
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
          <div style={{ display: 'grid', gap: 6 }}>
            <label>factsContractCode</label>
            <input value={factsContractCode} onChange={(e) => setFactsContractCode(e.target.value)} />
          </div>
          <div style={{ display: 'grid', gap: 6 }}>
            <label>factsContractVersion</label>
            <input value={factsContractVersion} onChange={(e) => setFactsContractVersion(e.target.value)} />
          </div>
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

        <div style={{ display: 'flex', gap: 10 }}>
          <button onClick={decide}>Decide</button>
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

