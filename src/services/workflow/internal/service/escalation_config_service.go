package service

import (
	"context"
	"fmt"

	"digit.org/workflow/internal/models"
	"digit.org/workflow/internal/repository"
)

type escalationConfigService struct {
	repo        repository.EscalationConfigRepository
	processRepo repository.ProcessRepository
	stateRepo   repository.StateRepository
}

// NewEscalationConfigService creates a new instance of EscalationConfigService.
func NewEscalationConfigService(repo repository.EscalationConfigRepository, processRepo repository.ProcessRepository, stateRepo repository.StateRepository) EscalationConfigService {
	return &escalationConfigService{
		repo:        repo,
		processRepo: processRepo,
		stateRepo:   stateRepo,
	}
}

// CreateEscalationConfig creates a new escalation configuration with validation.
func (s *escalationConfigService) CreateEscalationConfig(ctx context.Context, config *models.EscalationConfig) (*models.EscalationConfig, error) {
	// Validate that the process exists
	_, err := s.processRepo.GetProcessByID(ctx, config.TenantID, config.ProcessID)
	if err != nil {
		return nil, fmt.Errorf("process not found: %w", err)
	}

	// Validate that the state exists for this process
	_, err = s.stateRepo.GetStateByCodeAndProcess(ctx, config.TenantID, config.ProcessID, config.StateCode)
	if err != nil {
		return nil, fmt.Errorf("state '%s' not found in process: %w", config.StateCode, err)
	}

	// Validate that at least one SLA is specified
	if config.StateSlaMinutes == nil && config.ProcessSlaMinutes == nil {
		return nil, fmt.Errorf("at least one of stateSlaMinutes or processSlaMinutes must be specified")
	}

	// Validate SLA values are positive
	if config.StateSlaMinutes != nil && *config.StateSlaMinutes <= 0 {
		return nil, fmt.Errorf("stateSlaMinutes must be positive")
	}
	if config.ProcessSlaMinutes != nil && *config.ProcessSlaMinutes <= 0 {
		return nil, fmt.Errorf("processSlaMinutes must be positive")
	}

	return s.repo.CreateEscalationConfig(ctx, config)
}

// GetEscalationConfigByID retrieves an escalation configuration by ID.
func (s *escalationConfigService) GetEscalationConfigByID(ctx context.Context, tenantID, id string) (*models.EscalationConfig, error) {
	return s.repo.GetEscalationConfigByID(ctx, tenantID, id)
}

// GetEscalationConfigsByProcessID retrieves escalation configurations for a process.
func (s *escalationConfigService) GetEscalationConfigsByProcessID(ctx context.Context, tenantID, processID string, stateCode string, isActive *bool) ([]*models.EscalationConfig, error) {
	// Validate that the process exists
	_, err := s.processRepo.GetProcessByID(ctx, tenantID, processID)
	if err != nil {
		return nil, fmt.Errorf("process not found: %w", err)
	}

	return s.repo.GetEscalationConfigsByProcessID(ctx, tenantID, processID, stateCode, isActive)
}

// UpdateEscalationConfig updates an existing escalation configuration.
func (s *escalationConfigService) UpdateEscalationConfig(ctx context.Context, config *models.EscalationConfig) (*models.EscalationConfig, error) {
	// Validate the escalation config exists
	existing, err := s.repo.GetEscalationConfigByID(ctx, config.TenantID, config.ID)
	if err != nil {
		return nil, fmt.Errorf("escalation config not found: %w", err)
	}

	// If state code is being changed, validate it exists
	if config.StateCode != "" && config.StateCode != existing.StateCode {
		_, err = s.stateRepo.GetStateByCodeAndProcess(ctx, config.TenantID, existing.ProcessID, config.StateCode)
		if err != nil {
			return nil, fmt.Errorf("state '%s' not found in process: %w", config.StateCode, err)
		}
	}

	// Validate SLA values if provided
	if config.StateSlaMinutes != nil && *config.StateSlaMinutes <= 0 {
		return nil, fmt.Errorf("stateSlaMinutes must be positive")
	}
	if config.ProcessSlaMinutes != nil && *config.ProcessSlaMinutes <= 0 {
		return nil, fmt.Errorf("processSlaMinutes must be positive")
	}

	return s.repo.UpdateEscalationConfig(ctx, config)
}

// DeleteEscalationConfig deletes an escalation configuration.
func (s *escalationConfigService) DeleteEscalationConfig(ctx context.Context, tenantID, id string) error {
	return s.repo.DeleteEscalationConfig(ctx, tenantID, id)
}
