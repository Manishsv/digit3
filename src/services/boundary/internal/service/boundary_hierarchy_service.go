package service

import (
	"boundary/internal/models"
	"context"
)

// BoundaryHierarchyService defines the interface for boundary hierarchy operations
type BoundaryHierarchyService interface {
	// Create creates a new boundary hierarchy
	Create(ctx context.Context, request *models.BoundaryHierarchyRequest, tenantID, clientID string) error
	// Search searches for boundary hierarchies based on criteria
	Search(ctx context.Context, criteria *models.BoundaryHierarchySearchCriteria) ([]models.BoundaryHierarchy, error)
	// Update updates an existing boundary hierarchy
	Update(ctx context.Context, request *models.BoundaryHierarchyRequest, tenantID, clientID string) error
}
