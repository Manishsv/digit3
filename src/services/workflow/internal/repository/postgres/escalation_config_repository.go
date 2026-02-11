package postgres

import (
	"context"
	"errors"
	"fmt"

	"digit.org/workflow/internal/models"
	"digit.org/workflow/internal/repository"
	"gorm.io/gorm"
)

type escalationConfigRepository struct {
	db *gorm.DB
}

// NewEscalationConfigRepository creates a new instance of EscalationConfigRepository.
func NewEscalationConfigRepository(db *gorm.DB) repository.EscalationConfigRepository {
	return &escalationConfigRepository{db: db}
}

// CreateEscalationConfig creates a new escalation configuration.
func (r *escalationConfigRepository) CreateEscalationConfig(ctx context.Context, config *models.EscalationConfig) (*models.EscalationConfig, error) {
	err := r.db.WithContext(ctx).Create(config).Error
	if err != nil {
		return nil, fmt.Errorf("failed to create escalation config: %w", err)
	}
	return config, nil
}

// GetEscalationConfigByID retrieves an escalation configuration by ID.
func (r *escalationConfigRepository) GetEscalationConfigByID(ctx context.Context, tenantID, id string) (*models.EscalationConfig, error) {
	var config models.EscalationConfig
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		First(&config).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("escalation config not found")
		}
		return nil, fmt.Errorf("failed to get escalation config: %w", err)
	}
	return &config, nil
}

// GetEscalationConfigsByProcessID retrieves escalation configurations for a process.
func (r *escalationConfigRepository) GetEscalationConfigsByProcessID(ctx context.Context, tenantID, processID string, stateCode string, isActive *bool) ([]*models.EscalationConfig, error) {
	var configs []*models.EscalationConfig

	query := r.db.WithContext(ctx).
		Where("tenant_id = ? AND process_id = ?", tenantID, processID)

	if stateCode != "" {
		query = query.Where("state_code = ?", stateCode)
	}

	// Note: isActive parameter is ignored since we removed the column
	// We keep the parameter for API compatibility but don't use it

	err := query.Order("state_code, escalation_action").Find(&configs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get escalation configs: %w", err)
	}

	return configs, nil
}

// UpdateEscalationConfig updates an existing escalation configuration.
func (r *escalationConfigRepository) UpdateEscalationConfig(ctx context.Context, config *models.EscalationConfig) (*models.EscalationConfig, error) {
	// First, fetch the existing config to preserve unchanged fields
	existing, err := r.GetEscalationConfigByID(ctx, config.TenantID, config.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch existing escalation config: %w", err)
	}

	// Merge non-empty fields from request into existing config
	if config.StateCode != "" {
		existing.StateCode = config.StateCode
	}
	if config.EscalationAction != "" {
		existing.EscalationAction = config.EscalationAction
	}
	if config.StateSlaMinutes != nil {
		existing.StateSlaMinutes = config.StateSlaMinutes
	}
	if config.ProcessSlaMinutes != nil {
		existing.ProcessSlaMinutes = config.ProcessSlaMinutes
	}
	// Update audit details
	existing.AuditDetail.ModifiedBy = config.AuditDetail.ModifiedBy
	existing.AuditDetail.ModifiedTime = config.AuditDetail.ModifiedTime

	err = r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", existing.TenantID, existing.ID).
		Updates(existing).Error
	if err != nil {
		return nil, fmt.Errorf("failed to update escalation config: %w", err)
	}

	return existing, nil
}

// DeleteEscalationConfig deletes an escalation configuration.
func (r *escalationConfigRepository) DeleteEscalationConfig(ctx context.Context, tenantID, id string) error {
	result := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		Delete(&models.EscalationConfig{})

	if result.Error != nil {
		return fmt.Errorf("failed to delete escalation config: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.New("escalation config not found")
	}

	return nil
}

// GetActiveEscalationConfigs retrieves all escalation configurations for a tenant.
// Note: Method name kept for compatibility, but returns all configs since isActive was removed.
func (r *escalationConfigRepository) GetActiveEscalationConfigs(ctx context.Context, tenantID string) ([]*models.EscalationConfig, error) {
	var configs []*models.EscalationConfig
	err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("process_id, state_code").
		Find(&configs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get escalation configs: %w", err)
	}

	return configs, nil
}
