/**
 * v1.0 service definition — aligned with Registry schemas `studio.service` / `studio.bundle`
 * and Studio API `serviceDefinition` on POST /studio/v1/services and /studio/v1/bundles.
 */
export type ServiceDefinitionV1 = {
  schemaVersion?: '1.0'
  documentVersion?: string
  caseModel?: {
    caseIdPattern?: string
    factsContractCode?: string
    factsContractVersion?: string
    intakeFormRef?: string
    evidenceTypes?: Array<{
      evidenceType: string
      maxFiles?: number
      mimeTypes?: string[]
    }>
    registryPulls?: Array<{
      schemaCode: string
      lookupKey: string
      bindToFactsPath: string
    }>
  }
  governance?: {
    rulesetBindings?: Array<{
      rulesetId: string
      version: string
      registryRecordId?: string
      precedence?: number
    }>
  }
  workflow?: {
    verificationProcessCode?: string
    fulfillmentProcessCode?: string
  }
  rbac?: { roleMatrixRef?: string }
  notifications?: Array<{
    event: string
    channel?: 'email' | 'sms' | 'push'
    templateRef?: string
  }>
  outputs?: { certificateTemplateRef?: string }
  payments?: {
    model?: 'NONE' | 'FLAT' | 'RULE_REF' | 'EXTERNAL'
    pricingRulesetRef?: string
  }
  audit?: {
    traceProfile?: string
    coordinationTraceEnabled?: boolean
    governanceReceiptEnabled?: boolean
  }
  appeals?: {
    appellateAuthorityRef?: string
    appealWindowDays?: number
  }
  bindings?: { registrySchemasUsed?: string[] }
}
