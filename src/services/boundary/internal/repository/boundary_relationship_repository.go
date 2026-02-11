package repository

import (
	"boundary/internal/models"
	"context"
)

// BoundaryRelationshipRepository defines the interface for boundary relationship data access
type BoundaryRelationshipRepository interface {
	// Create creates a new boundary relationship
	Create(ctx context.Context, request *models.BoundaryRelationshipRequest) error

	// Search searches for boundary relationships based on criteria
	Search(ctx context.Context, criteria *models.BoundaryRelationshipSearchCriteria) ([]models.BoundaryRelationship, error)

	// Update updates an existing boundary relationship
	Update(ctx context.Context, request *models.BoundaryRelationshipRequest) error

	// GetByID fetches a boundary relationship by ID and tenantId
	GetByID(ctx context.Context, id, tenantId string) (*models.BoundaryRelationship, error)
	// Check if a relationship with the given code, tenantId, and hierarchyType exists
	ExistsByCode(ctx context.Context, tenantId, code, hierarchyType string) (bool, error)
	// Check if a parent relationship exists
	ParentExists(ctx context.Context, tenantId, parent, hierarchyType string) (bool, error)
	// Check if a boundary entity exists for the given code
	BoundaryExists(ctx context.Context, tenantId, boundaryCode string) (bool, error)

	// SearchWithMaterializedPath searches using materialized path for efficient hierarchy queries
	SearchWithMaterializedPath(ctx context.Context, criteria *models.BoundaryRelationshipSearchCriteria) ([]models.BoundaryRelationship, error)
}
