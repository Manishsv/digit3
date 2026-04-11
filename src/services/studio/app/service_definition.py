"""Shared service-definition shape for Registry `studio.service` and optional `studio.bundle` rows.

Aligned with provision examples `studio-service.schema.yaml` / `studio-bundle.schema.yaml`.
Version 1.0 — references-heavy; large artefacts (YAML rules, PDF templates) use *Ref fields.
"""

from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, ConfigDict, Field


class EvidenceTypeSpec(BaseModel):
    model_config = ConfigDict(extra="allow")

    evidenceType: str
    maxFiles: int | None = None
    mimeTypes: list[str] = Field(default_factory=list)


class RegistryPullSpec(BaseModel):
    model_config = ConfigDict(extra="allow")

    schemaCode: str
    lookupKey: str
    bindToFactsPath: str


class CaseModelSpec(BaseModel):
    model_config = ConfigDict(extra="allow")

    caseIdPattern: str | None = None
    factsContractCode: str | None = None
    factsContractVersion: str = "1"
    intakeFormRef: str | None = None
    evidenceTypes: list[EvidenceTypeSpec] = Field(default_factory=list)
    registryPulls: list[RegistryPullSpec] = Field(default_factory=list)


class RulesetBinding(BaseModel):
    model_config = ConfigDict(extra="allow")

    rulesetId: str
    version: str
    registryRecordId: str | None = None
    precedence: int = 0


class GovernanceSpec(BaseModel):
    model_config = ConfigDict(extra="allow")

    rulesetBindings: list[RulesetBinding] = Field(default_factory=list)


class WorkflowSpec(BaseModel):
    model_config = ConfigDict(extra="allow")

    verificationProcessCode: str | None = None
    fulfillmentProcessCode: str | None = None


class NotificationSpec(BaseModel):
    model_config = ConfigDict(extra="allow")

    event: str
    channel: Literal["email", "sms", "push"] = "email"
    templateRef: str | None = None


class OutputsSpec(BaseModel):
    model_config = ConfigDict(extra="allow")

    certificateTemplateRef: str | None = None


class PaymentsSpec(BaseModel):
    model_config = ConfigDict(extra="allow")

    model: Literal["NONE", "FLAT", "RULE_REF", "EXTERNAL"] = "NONE"
    pricingRulesetRef: str | None = None


class AuditSpec(BaseModel):
    model_config = ConfigDict(extra="allow")

    traceProfile: str | None = None
    coordinationTraceEnabled: bool | None = True
    governanceReceiptEnabled: bool | None = True


class AppealsSpec(BaseModel):
    model_config = ConfigDict(extra="allow")

    appellateAuthorityRef: str | None = None
    appealWindowDays: int | None = None


class RbacSpec(BaseModel):
    model_config = ConfigDict(extra="allow")

    roleMatrixRef: str | None = None


class BindingsSpec(BaseModel):
    model_config = ConfigDict(extra="allow")

    registrySchemasUsed: list[str] = Field(default_factory=list)


class ServiceDefinitionDoc(BaseModel):
    """Tenant-scoped service definition; persisted with studio.service / studio.bundle."""

    model_config = ConfigDict(extra="allow")

    schemaVersion: Literal["1.0"] = "1.0"
    documentVersion: str = Field(
        default="1",
        pattern=r"^[0-9A-Za-z.\-]{1,32}$",
        description="Version of this definition document (semver-ish).",
    )
    caseModel: CaseModelSpec | None = None
    governance: GovernanceSpec | None = None
    workflow: WorkflowSpec | None = None
    rbac: RbacSpec | None = None
    notifications: list[NotificationSpec] = Field(default_factory=list)
    outputs: OutputsSpec | None = None
    payments: PaymentsSpec | None = None
    audit: AuditSpec | None = None
    appeals: AppealsSpec | None = None
    bindings: BindingsSpec | None = None
