package models

import "github.com/google/uuid"

type IDGenTemplate struct {
	ID           uuid.UUID           `json:"id"`
	TenantID     string              `json:"tenantId"`
	TemplateCode string              `json:"templateCode" binding:"required"`
	Config       IDGenTemplateConfig `json:"config" binding:"required"`
	AuditDetails AuditDetails        `json:"auditDetails"`
}

type IDGenTemplateWithVersion struct {
	ID           uuid.UUID           `json:"id"`
	TenantID     string              `json:"tenantId"`
	TemplateCode string              `json:"templateCode" binding:"required"`
	Version      string              `json:"version"`
	Config       IDGenTemplateConfig `json:"config" binding:"required"`
	AuditDetails AuditDetails        `json:"auditDetails"`
}

type IDGenTemplateConfig struct {
	Template string         `json:"template" binding:"required"`
	Sequence SequenceConfig `json:"sequence"`
	Random   RandomConfig   `json:"random"`
}

type SequenceConfig struct {
	Scope   string        `json:"scope" default:"global"`
	Start   int           `json:"start" default:"1"`
	Padding PaddingConfig `json:"padding"`
}

type PaddingConfig struct {
	Length int    `json:"length"`
	Char   string `json:"char" default:"0"`
}

type RandomConfig struct {
	Length  int    `json:"length" default:"0"`
	Charset string `json:"charset"`
}
