package models

import "encoding/json"

type TenantConfig struct {
	ID                   string          `json:"id,omitempty"`
	DefaultLoginType     string          `json:"defaultLoginType,omitempty"`
	OtpLength            string          `json:"otpLength,omitempty"`
	Code                 string          `json:"code,omitempty"`
	Name                 string          `json:"name,omitempty"`
	EnableUserBasedLogin bool            `json:"enableUserBasedLogin"`
	AdditionalAttributes json.RawMessage `json:"additionalAttributes,omitempty"`
	Documents            []Document      `json:"documents,omitempty"`
	IsActive             bool            `json:"isActive"`
	Languages            []string        `json:"languages,omitempty"`
	AuditDetails         AuditDetails    `json:"auditDetails,omitempty"`
}

type Document struct {
	ID             string       `json:"id"`
	TenantID       string       `json:"tenantId"`
	TenantConfigID string       `json:"tenantConfigId"`
	Type           string       `json:"type"`
	FileStoreID    string       `json:"fileStoreId"`
	URL            string       `json:"url"`
	IsActive       bool         `json:"isActive"`
	AuditDetails   AuditDetails `json:"auditDetails"`
}
