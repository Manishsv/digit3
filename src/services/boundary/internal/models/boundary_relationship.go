package models

import (
	commonmodels "boundary/internal/common/models"
)

// BoundaryRelationship represents a boundary relationship entity
type BoundaryRelationship struct {
	ID                        string                     `json:"id" gorm:"column:id;primaryKey"`
	TenantID                  string                     `json:"tenantId" gorm:"column:tenantid"`
	Code                      string                     `json:"code" binding:"required" gorm:"column:code"`
	HierarchyType             string                     `json:"hierarchyType" binding:"required" gorm:"column:hierarchytype"`
	BoundaryType              string                     `json:"boundaryType" binding:"required" gorm:"column:boundarytype"`
	Parent                    string                     `json:"parent" gorm:"column:parent"`
	AncestralMaterializedPath string                     `json:"ancestralMaterializedPath" gorm:"column:ancestralmaterializedpath"`
	AuditDetails              *commonmodels.AuditDetails `json:"auditDetails,omitempty" gorm:"embedded"`
}

// TableName specifies the table name for GORM
func (BoundaryRelationship) TableName() string {
	return "boundary_relationship_v1"
}

// BoundaryRelationshipRequest represents a request to create or update a boundary relationship
type BoundaryRelationshipRequest struct {
	BoundaryRelationship BoundaryRelationship `json:"boundaryRelationship"`
}

// BoundaryRelationshipSearchCriteria represents search criteria for boundary relationships
type BoundaryRelationshipSearchCriteria struct {
	TenantID        string   `json:"tenantId"`
	HierarchyType   string   `json:"hierarchyType"`
	BoundaryType    string   `json:"boundaryType,omitempty"`
	Codes           []string `json:"codes,omitempty"`
	Parent          string   `json:"parent,omitempty"`
	IncludeChildren bool     `json:"includeChildren,omitempty"`
	IncludeParents  bool     `json:"includeParents,omitempty"`
	Limit           int      `json:"limit,omitempty"`
	Offset          int      `json:"offset,omitempty"`
	// Internal parameters (similar to Java implementation)
	CurrentBoundaryCodes []string `json:"-"` // Internal use only, not exposed in JSON
	IsSearchForRootNode  bool     `json:"-"` // Internal flag for root node search
}

// BoundaryRelationshipResponse represents a response containing boundary relationships
type BoundaryRelationshipResponse struct {
	Relationship []BoundaryRelationship `json:"relationship"`
}

// EnrichedBoundary represents a node in the hierarchical response
// Matches Java's EnrichedBoundary
// Children is recursive
// Parent is omitted from JSON (used for tree building)
type EnrichedBoundary struct {
	ID           string                     `json:"id"`
	Code         string                     `json:"code"`
	BoundaryType string                     `json:"boundaryType"`
	Children     []EnrichedBoundary         `json:"children,omitempty"`
	AuditDetails *commonmodels.AuditDetails `json:"auditDetails,omitempty"`
	Parent       string                     `json:"-"`
}

// HierarchyRelation matches Java's HierarchyRelation
// Contains the root(s) of the boundary tree
type HierarchyRelation struct {
	TenantID      string             `json:"tenantId"`
	HierarchyType string             `json:"hierarchyType"`
	Boundary      []EnrichedBoundary `json:"boundary"`
}

// BoundarySearchResponse matches Java's BoundarySearchResponse
type BoundarySearchResponse struct {
	TenantBoundary []HierarchyRelation `json:"tenantBoundary"`
}
