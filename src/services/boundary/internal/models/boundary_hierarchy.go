package models

import (
	commonmodels "boundary/internal/common/models"
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// BoundaryHierarchyRequest represents a request to create or update a boundary hierarchy
type BoundaryHierarchyRequest struct {
	BoundaryHierarchy BoundaryHierarchy `json:"boundaryHierarchy"`
}

// BoundaryHierarchyList is a custom type for JSONB serialization
type BoundaryHierarchyList []BoundaryTypeHierarchy

// Value implements the driver.Valuer interface for database storage
func (bhl BoundaryHierarchyList) Value() (driver.Value, error) {
	if bhl == nil {
		return nil, nil
	}
	return json.Marshal(bhl)
}

// Scan implements the sql.Scanner interface for database retrieval
func (bhl *BoundaryHierarchyList) Scan(value interface{}) error {
	if value == nil {
		*bhl = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into BoundaryHierarchyList", value)
	}

	return json.Unmarshal(bytes, bhl)
}

// BoundaryHierarchy represents a boundary hierarchy entity
type BoundaryHierarchy struct {
	ID                string                     `json:"id" gorm:"column:id;primaryKey"`
	TenantID          string                     `json:"tenantId" gorm:"column:tenantid"`
	HierarchyType     string                     `json:"hierarchyType" gorm:"column:hierarchytype"`
	BoundaryHierarchy BoundaryHierarchyList      `json:"boundaryHierarchy" gorm:"column:boundaryhierarchy;type:jsonb"`
	AuditDetails      *commonmodels.AuditDetails `json:"auditDetails,omitempty" gorm:"embedded"`
}

// TableName specifies the table name for GORM
func (BoundaryHierarchy) TableName() string {
	return "boundary_hierarchy_v1"
}

// BoundaryHierarchySearchCriteria represents search criteria for boundary hierarchies
type BoundaryHierarchySearchCriteria struct {
	TenantID      string `json:"tenantId"`
	HierarchyType string `json:"hierarchyType"`
}

// BoundaryHierarchyResponse represents a response containing boundary hierarchy information
type BoundaryHierarchyResponse struct {
	Hierarchy *BoundaryHierarchy `json:"hierarchy"`
}

// BoundaryTypeHierarchySearchRequest represents a request to search boundary hierarchies
type BoundaryTypeHierarchySearchRequest struct {
	TenantID      string `json:"tenantId"`
	HierarchyType string `json:"hierarchyType"`
}

// BoundaryTypeHierarchyResponse represents a response containing boundary hierarchies
type BoundaryTypeHierarchyResponse struct {
	Hierarchy []BoundaryHierarchy `json:"hierarchy"`
}
