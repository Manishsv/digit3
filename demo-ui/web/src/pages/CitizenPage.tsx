import { useState } from 'react'
import { ApiError, digitJson, useDigitHeaders } from '../lib/digitApi'
import { useSelection } from '../state/selection'

type WorkflowProcess = { id: string; code: string }

type CaseRef = { accountId?: string; serviceCode: string; caseId: string; registryId?: string }
const CASES_KEY = 'digit.demoUi.cases.v1'

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

function saveCases(c: CaseRef[]) {
  localStorage.setItem(CASES_KEY, JSON.stringify(c))
}

export function CitizenPage() {
  const headers = useDigitHeaders()
  const sel = useSelection()
  const [serviceCode, setServiceCode] = useState(sel.selectedServiceCode || 'PGR67')
  const [caseId, setCaseId] = useState(`CS-${Date.now()}`)
  const [desc, setDesc] = useState('Complaint submission (demo)')
  const [out, setOut] = useState<Record<string, unknown> | null>(null)
  const [err, setErr] = useState<string | null>(null)

  async function submit() {
    setErr(null)
    setOut(null)
    try {
      const procs = await digitJson<WorkflowProcess[]>('workflow', `/workflow/v1/process?code=${encodeURIComponent(serviceCode)}`, {
        headers,
      })
      const pid = procs?.[0]?.id
      if (!pid) throw new Error(`No workflow process for code=${serviceCode}`)

      const inst = await digitJson<Record<string, unknown>>('workflow', '/workflow/v1/transition', {
        method: 'POST',
        headers: { ...headers, 'Content-Type': 'application/json' },
        body: JSON.stringify({
          processId: pid,
          entityId: caseId,
          action: 'APPLY',
          init: true,
          comment: desc,
          attributes: { roles: ['CITIZEN'] },
        }),
      })

      const reg = await digitJson<Record<string, unknown>>('registry', '/registry/v1/schema/complaints.case/data', {
        method: 'POST',
        headers: { ...headers, 'Content-Type': 'application/json' },
        body: JSON.stringify({
          data: {
            serviceRequestId: caseId,
            tenantId: headers['X-Tenant-ID'],
            serviceCode,
            processId: pid,
            workflowInstanceId: (inst as any)?.id,
            description: desc,
            applicationStatus: 'SUBMITTED',
          },
        }),
      })

      const rid =
        (reg as any)?.registryId ||
        (reg as any)?.data?.registryId ||
        (reg as any)?.data?.registry_id ||
        (reg as any)?.result?.registryId

      const next = [
        { accountId: sel.selectedAccountId, serviceCode, caseId, registryId: typeof rid === 'string' ? rid : undefined },
        ...loadCases().filter((c) => !(c.caseId === caseId && c.serviceCode === serviceCode && c.accountId === sel.selectedAccountId)),
      ].slice(0, 200)
      saveCases(next)

      setOut({ workflowInstance: inst, registry: reg })
    } catch (e: unknown) {
      setErr(e instanceof ApiError ? `${e.message}\n${e.bodyText}` : String(e))
    }
  }

  async function loadCaseByRegistryId(registryId: string) {
    setErr(null)
    setOut(null)
    try {
      const r = await digitJson<Record<string, unknown>>(
        'registry',
        `/registry/v1/schema/complaints.case/data/_registry?registryId=${encodeURIComponent(registryId)}`,
        { headers },
      )
      setOut({ registry: r })
    } catch (e: unknown) {
      setErr(e instanceof ApiError ? `${e.message}\n${e.bodyText}` : String(e))
    }
  }

  return (
    <div>
      <h2 style={{ marginTop: 0 }}>Citizen (requester)</h2>
      <p style={{ color: '#374151' }}>Initiate a case (workflow APPLY + registry row) and view the stored payload.</p>

      <div style={{ display: 'grid', gap: 10, maxWidth: 900 }}>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
          <div style={{ display: 'grid', gap: 6 }}>
            <label>serviceCode</label>
            <input
              value={serviceCode}
              onChange={(e) => {
                const v = e.target.value
                setServiceCode(v)
                sel.setSelectedServiceCode(v.trim() || undefined)
              }}
            />
          </div>
          <div style={{ display: 'grid', gap: 6 }}>
            <label>caseId / serviceRequestId</label>
            <input value={caseId} onChange={(e) => setCaseId(e.target.value)} />
          </div>
        </div>

        <div style={{ display: 'grid', gap: 6 }}>
          <label>description</label>
          <input value={desc} onChange={(e) => setDesc(e.target.value)} />
        </div>

        <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap' }}>
          <button onClick={submit}>Submit (APPLY)</button>
          <button
            onClick={() => {
              const v = prompt('Registry registryId (e.g. REGISTRY-YYYYMMDD-....)')
              if (v) void loadCaseByRegistryId(v)
            }}
          >
            Load registry row by registryId…
          </button>
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


