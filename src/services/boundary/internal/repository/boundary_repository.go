package repository

import (
	"boundary/internal/models"
	"context"
)

// BoundaryRepository defines the interface for boundary data access
type BoundaryRepository interface {
	// Create creates new boundaries (batch)
	Create(ctx context.Context, request *models.BoundaryRequest) error

	// Search searches for boundaries based on criteria
	Search(ctx context.Context, criteria *models.BoundarySearchCriteria) ([]models.Boundary, error)

	// Update updates an existing boundary
	Update(ctx context.Context, request *models.BoundaryRequest) error

	// GetByID fetches a boundary by ID and tenantId
	GetByID(ctx context.Context, id, tenantId string) (*models.Boundary, error)

	// Check if a boundary with the given code and tenantId exists
	ExistsByCode(ctx context.Context, tenantId, code string) (bool, error)
}
