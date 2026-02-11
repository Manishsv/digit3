package models

import (
	"github.com/google/uuid"
)

// TemplateConfig - API model (Version REMOVED from API)
type TemplateConfig struct {
	ID           uuid.UUID         `json:"id"`
	TemplateID   string            `json:"templateId" binding:"required"`
	TenantID     string            `json:"tenantId"`
	FieldMapping map[string]string `json:"fieldMapping"`
	APIMapping   []APIMapping      `json:"apiMapping"`
	AuditDetails AuditDetails      `json:"auditDetails"`
}

// TemplateConfigWithVersion - API response that includes version
type TemplateConfigWithVersion struct {
	ID           uuid.UUID         `json:"id"`
	TemplateID   string            `json:"templateId"`
	TenantID     string            `json:"tenantId"`
	Version      string            `json:"version"` // Returned in response
	FieldMapping map[string]string `json:"fieldMapping"`
	APIMapping   []APIMapping      `json:"apiMapping"`
	AuditDetails AuditDetails      `json:"auditDetails"`
}
