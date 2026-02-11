package repository

import (
	hierarchymodels "boundary/internal/models"
	"context"
)

// BoundaryHierarchyRepository defines the interface for boundary hierarchy data access
type BoundaryHierarchyRepository interface {
	// Create creates a new boundary hierarchy
	Create(ctx context.Context, request *hierarchymodels.BoundaryHierarchyRequest) error

	// Search searches for boundary hierarchies based on criteria
	Search(ctx context.Context, criteria *hierarchymodels.BoundaryHierarchySearchCriteria) ([]hierarchymodels.BoundaryHierarchy, error)

	// Update updates an existing boundary hierarchy
	Update(ctx context.Context, request *hierarchymodels.BoundaryHierarchyRequest) error

	// Check if a hierarchy with the given type and tenantId exists
	ExistsByType(ctx context.Context, tenantId, hierarchyType string) (bool, error)
}
