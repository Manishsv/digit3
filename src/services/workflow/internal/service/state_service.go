package service

import (
	"context"

	"digit.org/workflow/internal/models"
	"digit.org/workflow/internal/repository"
)

type stateService struct {
	repo repository.StateRepository
}

// NewStateService creates a new instance of StateService.
func NewStateService(repo repository.StateRepository) StateService {
	return &stateService{repo: repo}
}

// CreateState handles the business logic for creating a new state.
func (s *stateService) CreateState(ctx context.Context, state *models.State) (*models.State, error) {
	err := s.repo.CreateState(ctx, state)
	if err != nil {
		return nil, err
	}
	return state, nil
}

// GetStatesByProcessID handles the business logic for retrieving all states for a process.
func (s *stateService) GetStatesByProcessID(ctx context.Context, tenantID, processID string, codes []string) ([]*models.State, error) {
	return s.repo.GetStatesByProcessID(ctx, tenantID, processID, codes)
}

// GetStateByID handles the business logic for retrieving a single state.
func (s *stateService) GetStateByID(ctx context.Context, tenantID, id string) (*models.State, error) {
	return s.repo.GetStateByID(ctx, tenantID, id)
}

// UpdateState handles the business logic for updating a state.
func (s *stateService) UpdateState(ctx context.Context, state *models.State) (*models.State, error) {
	err := s.repo.UpdateState(ctx, state)
	if err != nil {
		return nil, err
	}
	return s.repo.GetStateByID(ctx, state.TenantID, state.ID)
}

// GetStateByCodeAndProcess handles the business logic for finding a state by code within a process.
func (s *stateService) GetStateByCodeAndProcess(ctx context.Context, tenantID, processID, code string) (*models.State, error) {
	return s.repo.GetStateByCodeAndProcess(ctx, tenantID, processID, code)
}

// DeleteState handles the business logic for deleting a state.
func (s *stateService) DeleteState(ctx context.Context, tenantID, id string) error {
	return s.repo.DeleteState(ctx, tenantID, id)
}
