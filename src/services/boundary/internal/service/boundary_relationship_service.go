package service

import (
	"boundary/internal/models"
	"context"
)

// BoundaryRelationshipService defines the interface for boundary relationship operations
type BoundaryRelationshipService interface {
	// Create creates a new boundary relationship
	Create(ctx context.Context, request *models.BoundaryRelationshipRequest, tenantID, clientID string) error
	// Search searches for boundary relationships based on criteria
	Search(ctx context.Context, criteria *models.BoundaryRelationshipSearchCriteria) (*models.BoundarySearchResponse, error)
	// Update updates an existing boundary relationship
	Update(ctx context.Context, request *models.BoundaryRelationshipRequest, tenantID, clientID string) error
}
