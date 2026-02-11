package models

import "encoding/json"

type Tenant struct {
	ID                  string          `json:"id" binding:"omitempty,min=2,max=128"`
	Code                string          `json:"code" binding:"omitempty,min=1,max=50"`
	Name                string          `json:"name" binding:"required,min=1,max=50"`
	Email               string          `json:"email" binding:"required,email,max=64"`
	AdditionalAttributes json.RawMessage `json:"additionalAttributes,omitempty"`
	IsActive            bool            `json:"isActive"`
	TenantID            string          `json:"tenantId,omitempty"`
	CreatedBy           string          `json:"createdBy,omitempty"`
	LastModifiedBy      string          `json:"lastModifiedBy,omitempty"`
	CreatedTime         int64           `json:"createdTime,omitempty"`
	LastModifiedTime    int64           `json:"lastModifiedTime,omitempty"`
}
