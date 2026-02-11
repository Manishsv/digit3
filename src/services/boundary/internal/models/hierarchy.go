package models

import (
	commonmodels "boundary/internal/common/models"
)

// BoundaryTypeHierarchy represents a hierarchy level in boundary type
type BoundaryTypeHierarchy struct {
	ID                 string  `json:"id,omitempty"`
	BoundaryType       string  `json:"boundaryType,omitempty"`
	ParentBoundaryType *string `json:"parentBoundaryType"`
	Active             bool    `json:"active"`
}

// BoundaryTypeHierarchyDefinition represents a complete hierarchy definition
type BoundaryTypeHierarchyDefinition struct {
	ID                string                     `json:"id,omitempty"`
	TenantID          string                     `json:"tenantId,omitempty"`
	HierarchyType     string                     `json:"hierarchyType,omitempty"`
	BoundaryHierarchy []BoundaryTypeHierarchy    `json:"boundaryHierarchy,omitempty"`
	AuditDetails      *commonmodels.AuditDetails `json:"auditDetails,omitempty"`
}
