package service

import (
	"boundary/internal/models"
	"context"
)

// BoundaryService defines the interface for boundary-related operations
type BoundaryService interface {
	// Create creates new boundaries (batch)
	Create(ctx context.Context, request *models.BoundaryRequest, tenantID, clientID string) error
	// Search searches for boundaries based on criteria
	Search(ctx context.Context, criteria *models.BoundarySearchCriteria) ([]models.Boundary, error)
	// Update updates an existing boundary
	Update(ctx context.Context, request *models.BoundaryRequest, tenantID, clientID string) error
}
