import { useMemo, useState } from 'react'
import { ApiError, digitJson, useDigitHeaders } from '../lib/digitApi'

type PublishedRuleset = {
  rulesetId: string
  code: string
  version: string
  registryRecordId?: string
  digest?: { alg: string; value: string }
}

const RS_KEY = 'digit.demoUi.lastPublishedRulesets.v1'

function saveRulesets(r: PublishedRuleset[]) {
  localStorage.setItem(RS_KEY, JSON.stringify(r))
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

export function StudioConfiguratorPage() {
  const headers = useDigitHeaders()
  const [serviceCode, setServiceCode] = useState('PGR67')
  const [serviceName, setServiceName] = useState('Demo Service')
  const [factsContractCode, setFactsContractCode] = useState('SBL_DEFAULT')
  const [factsContractVersion, setFactsContractVersion] = useState('1')
  const [rulesYaml, setRulesYaml] = useState(
    `ruleset:\n  code: DEMO_SBL\n  version: \"1.0\"\ninputs: {}\nrules:\n  - id: r1\n    predicate: eq\n    args:\n      path: application.status\n      value: SUBMITTED\n    outcome:\n      status: ELIGIBLE\n    reason: submitted\n`,
  )
  const [out, setOut] = useState<Record<string, unknown> | null>(null)
  const [err, setErr] = useState<string | null>(null)
  const [published, setPublished] = useState<PublishedRuleset[]>(() => loadRulesets())

  const rulesetItem = useMemo(
    () => ({ yamlText: rulesYaml, issuerAuthorityId: 'REG-DEMO' }),
    [rulesYaml],
  )

  async function run() {
    setErr(null)
    setOut(null)
    try {
      await digitJson('studio', '/studio/v1/services', {
        method: 'POST',
        headers: { ...headers, 'Content-Type': 'application/json' },
        body: JSON.stringify({
          serviceCode,
          name: serviceName,
          moduleType: 'SERVICE',
          status: 'ENABLED',
          metadata: {},
        }),
      })

      const bundle = await digitJson<{ bundleId: string; registryRecordId?: string }>('studio', '/studio/v1/bundles', {
        method: 'POST',
        headers: { ...headers, 'Content-Type': 'application/json' },
        body: JSON.stringify({
          serviceCode,
          version: '1',
          status: 'DRAFT',
          factsContractCode,
          factsContractVersion,
          rulesets: [rulesetItem],
          metadata: {},
        }),
      })

      const job = await digitJson<{ status: string; results?: { publishedRulesets?: PublishedRuleset[] } }>(
        'studio',
        '/studio/v1/jobs',
        {
          method: 'POST',
          headers: { ...headers, 'Content-Type': 'application/json' },
          body: JSON.stringify({ serviceCode, bundleId: bundle.bundleId, action: 'APPLY_RULESETS' }),
        },
      )

      const pub = job.results?.publishedRulesets || []
      setPublished(pub)
      saveRulesets(pub)
      setOut({ bundle, job })
    } catch (e: unknown) {
      if (e instanceof ApiError) setErr(`${e.message}\n${e.bodyText}`)
      else setErr(String(e))
    }
  }

  return (
    <div>
      <h2 style={{ marginTop: 0 }}>Regulator (rules)</h2>
      <p style={{ color: '#374151' }}>Specify/publish rulesets (policy). Operator will apply decisions using published rules.</p>

      <div style={{ display: 'grid', gap: 10, maxWidth: 900 }}>
        <div style={{ display: 'grid', gap: 6 }}>
          <label>serviceCode</label>
          <input value={serviceCode} onChange={(e) => setServiceCode(e.target.value)} />
        </div>
        <div style={{ display: 'grid', gap: 6 }}>
          <label>serviceName</label>
          <input value={serviceName} onChange={(e) => setServiceName(e.target.value)} />
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
          <label>rules YAML</label>
          <textarea
            value={rulesYaml}
            onChange={(e) => setRulesYaml(e.target.value)}
            rows={14}
            style={{ fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace' }}
          />
        </div>

        <div style={{ display: 'flex', gap: 10 }}>
          <button onClick={run}>Create + apply</button>
        </div>

        {published.length > 0 && (
          <div style={{ padding: 12, border: '1px solid #e5e7eb', borderRadius: 12 }}>
            <div style={{ fontWeight: 700, marginBottom: 6 }}>publishedRulesets</div>
            <pre style={{ margin: 0, overflow: 'auto' }}>{JSON.stringify(published, null, 2)}</pre>
          </div>
        )}

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

