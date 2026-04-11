import type { ServiceDefinitionV1 } from './serviceDefinition'

/** Row from Studio `GET /studio/v1/services` (shape varies; we keep common fields). */
export type StudioServiceRow = {
  tenant_id: string
  service_code: string
  registry_id?: string | null
  status?: string | null
  updated_at?: string | null
  payload_json?: string | null
  payload?: {
    id?: string
    tenantId?: string
    serviceCode?: string
    name?: string
    moduleType?: string
    status?: string
    metadata?: Record<string, unknown>
    serviceDefinition?: ServiceDefinitionV1
  }
}
