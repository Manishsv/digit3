import { useMemo, useState } from 'react'
import { ApiError, digitJson, useDigitHeaders } from '../lib/digitApi'
import { useSelection } from '../state/selection'

type WorkflowProcess = { id: string; code: string }

type PublishedRuleset = {
  rulesetId: string
  code: string
  version: string
  registryRecordId?: string
}

const RS_KEY = 'digit.demoUi.lastPublishedRulesets.v1'
const LAST_DECISION_KEY = 'digit.demoUi.lastDecision.v1'
const CASES_KEY = 'digit.demoUi.cases.v1'

type CaseRef = { accountId?: string; serviceCode: string; caseId: string; registryId?: string }

function loadCases(): CaseRef[] {
  const raw = localStorage.getItem(CASES_KEY)
  if (!raw) return []
  try {
    const parsed = JSON.parse(raw)
    return Array.isArray(parsed) ? (parsed as CaseRef[]) : []
  } catch {
    return []
  }
}

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

export function OperatorPage() {
  const headers = useDigitHeaders()
  const sel = useSelection()
  const [tab, setTab] = useState<'cases' | 'decision' | 'workflow'>('cases')

  const [channel, setChannel] = useState('web')
  const [serviceCode, setServiceCode] = useState(sel.selectedServiceCode || 'PGR67')
  const [caseId, setCaseId] = useState('CS-DEMO-001')

  const rulesets = useMemo(() => loadRulesets(), [])
  const [idx, setIdx] = useState(0)
  const [factsContractCode, setFactsContractCode] = useState('SBL_DEFAULT')
  const [factsContractVersion, setFactsContractVersion] = useState('1')
  const [factsJson, setFactsJson] = useState('{"application":{"status":"SUBMITTED"}}')

  const [action, setAction] = useState('ASSIGN')
  const [comment, setComment] = useState('Operator action (demo)')

  const [out, setOut] = useState<Record<string, unknown> | null>(null)
  const [err, setErr] = useState<string | null>(null)

  const selected = rulesets[idx]
  const cases = useMemo(() => {
    const all = loadCases()
    return all.filter((c) => {
      if (sel.selectedAccountId && c.accountId !== sel.selectedAccountId) return false
      if (sel.selectedServiceCode && c.serviceCode !== sel.selectedServiceCode) return false
      return true
    })
  }, [sel.selectedAccountId, sel.selectedServiceCode])

  async function decide() {
    setErr(null)
    setOut(null)
    if (!selected) {
      setErr('No published rulesets found. Publish one from Regulator (rules) first.')
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
        channel,
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
          response: resp,
        }),
      )
    } catch (e: unknown) {
      if (e instanceof ApiError) setErr(`${e.message}\n${e.bodyText}`)
      else setErr(String(e))
    }
  }

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
          attributes: { roles: ['OPERATOR'] },
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
    <div style={{ maxWidth: 980 }}>
      <h2 style={{ marginTop: 0 }}>Operator</h2>
      <p style={{ color: '#374151' }}>
        Receives cases from channels (web/frontline/etc), pulls facts (case file and/or registry), and applies decisions using
        regulator-defined rules.
      </p>

      <div style={{ display: 'grid', gap: 12 }}>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
          <div style={{ display: 'grid', gap: 6 }}>
            <label>selected account</label>
            <select value={sel.selectedAccountId || ''} onChange={(e) => sel.setSelectedAccountId(e.target.value || undefined)}>
              <option value="">— select —</option>
              {sel.accounts.map((a) => (
                <option key={a.id} value={a.id}>
                  {a.name || a.id}
                </option>
              ))}
            </select>
          </div>
          <div style={{ display: 'grid', gap: 6 }}>
            <label>selected service</label>
            <input
              value={serviceCode}
              onChange={(e) => {
                const v = e.target.value
                setServiceCode(v)
                sel.setSelectedServiceCode(v.trim() || undefined)
              }}
              placeholder="serviceCode"
            />
          </div>
        </div>

        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
          <button onClick={() => setTab('cases')} disabled={tab === 'cases'}>
            Cases
          </button>
          <button onClick={() => setTab('decision')} disabled={tab === 'decision'}>
            Decision
          </button>
          <button onClick={() => setTab('workflow')} disabled={tab === 'workflow'}>
            Workflow
          </button>
        </div>

        {tab === 'cases' ? (
          <section style={{ border: '1px solid #e5e7eb', borderRadius: 12, padding: 12 }}>
            <div style={{ fontWeight: 700, marginBottom: 6 }}>Cases + status</div>
            <div style={{ color: '#6b7280', marginBottom: 10 }}>
              Uses locally saved caseIds from Citizen intake; select one to operate/decide.
            </div>
            {cases.length === 0 ? (
              <div style={{ color: '#6b7280' }}>No cases found for current selection.</div>
            ) : (
              <div style={{ display: 'grid', gap: 8 }}>
                {cases.slice(0, 50).map((c) => (
                  <button
                    key={`${c.accountId || '—'}:${c.serviceCode}:${c.caseId}`}
                    onClick={() => setCaseId(c.caseId)}
                    style={{ textAlign: 'left' }}
                  >
                    <div style={{ fontWeight: 600 }}>{c.caseId}</div>
                    <div style={{ fontSize: 12, color: '#6b7280' }}>
                      service={c.serviceCode} {c.registryId ? `registryId=${c.registryId}` : ''}
                    </div>
                  </button>
                ))}
              </div>
            )}
          </section>
        ) : null}

        {tab === 'decision' ? (
          <section style={{ border: '1px solid #e5e7eb', borderRadius: 12, padding: 12 }}>
            <div style={{ fontWeight: 700, marginBottom: 8 }}>Decision</div>

            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
              <div style={{ display: 'grid', gap: 6 }}>
                <label>channel</label>
                <input value={channel} onChange={(e) => setChannel(e.target.value)} placeholder="web / frontline / ..." />
              </div>
              <div style={{ display: 'grid', gap: 6 }}>
                <label>caseId</label>
                <input value={caseId} onChange={(e) => setCaseId(e.target.value)} />
              </div>
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
                rows={6}
                style={{ fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace' }}
              />
            </div>

            <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap' }}>
              <button onClick={decide}>Apply decision</button>
            </div>
          </section>
        ) : null}

        {tab === 'workflow' ? (
          <section style={{ border: '1px solid #e5e7eb', borderRadius: 12, padding: 12 }}>
            <div style={{ fontWeight: 700, marginBottom: 8 }}>Workflow operations</div>

          <div style={{ display: 'grid', gap: 10 }}>
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
          </div>
        </section>
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

