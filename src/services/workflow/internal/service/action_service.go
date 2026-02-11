package service

import (
	"context"

	"digit.org/workflow/internal/models"
	"digit.org/workflow/internal/repository"
)

type actionService struct {
	repo repository.ActionRepository
}

// NewActionService creates a new instance of ActionService.
func NewActionService(repo repository.ActionRepository) ActionService {
	return &actionService{repo: repo}
}

// CreateAction handles the business logic for creating a new action.
func (s *actionService) CreateAction(ctx context.Context, action *models.Action) (*models.Action, error) {
	createdAction, err := s.repo.CreateAction(ctx, action)
	if err != nil {
		return nil, err
	}
	return createdAction, nil
}

// GetActionsByStateID handles the business logic for retrieving all actions for a state.
func (s *actionService) GetActionsByStateID(ctx context.Context, tenantID, stateID string, codes []string) ([]*models.Action, error) {
	return s.repo.GetActionsByStateID(ctx, tenantID, stateID, codes)
}

// GetActionByID handles the business logic for retrieving a single action.
func (s *actionService) GetActionByID(ctx context.Context, tenantID, id string) (*models.Action, error) {
	return s.repo.GetActionByID(ctx, tenantID, id)
}

// UpdateAction handles the business logic for updating an action.
func (s *actionService) UpdateAction(ctx context.Context, action *models.Action) (*models.Action, error) {
	err := s.repo.UpdateAction(ctx, action)
	if err != nil {
		return nil, err
	}
	return s.repo.GetActionByID(ctx, action.TenantID, action.ID)
}

// DeleteAction handles the business logic for deleting an action.
func (s *actionService) DeleteAction(ctx context.Context, tenantID, id string) error {
	return s.repo.DeleteAction(ctx, tenantID, id)
}
