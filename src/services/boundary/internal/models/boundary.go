package models

import (
	commonmodels "boundary/internal/common/models"
	"encoding/json"
)

// Boundary represents a boundary entity
type Boundary struct {
	ID                string                     `json:"id" gorm:"column:id;primaryKey"`
	TenantID          string                     `json:"tenantId" gorm:"column:tenantid"`
	Code              string                     `json:"code" binding:"required" gorm:"column:code"`
	Geometry          json.RawMessage            `json:"geometry" binding:"required" gorm:"column:geometry;type:jsonb"`
	AdditionalDetails json.RawMessage            `json:"additionalDetails" gorm:"column:additionaldetails;type:jsonb"`
	AuditDetails      *commonmodels.AuditDetails `json:"auditDetails,omitempty" gorm:"embedded"`
}

// TableName specifies the table name for GORM
func (Boundary) TableName() string {
	return "boundary_v1"
}

// BoundaryRequest represents a request to create or update boundaries
type BoundaryRequest struct {
	Boundary []Boundary `json:"boundary" binding:"required,min=1"`
}

// BoundarySearchCriteria represents search criteria for boundaries
type BoundarySearchCriteria struct {
	TenantID string   `json:"tenantId"`
	Codes    []string `json:"codes,omitempty"`
	Limit    int      `json:"limit,omitempty"`
	Offset   int      `json:"offset,omitempty"`
}

// BoundaryResponse represents a response containing boundaries
type BoundaryResponse struct {
	Boundary []Boundary `json:"boundary"`
}
