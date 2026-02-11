package postgres

import (
	"context"
	"time"

	"digit.org/workflow/internal/models"
	"digit.org/workflow/internal/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type stateRepository struct {
	db *gorm.DB
}

// NewStateRepository creates a new instance of StateRepository.
func NewStateRepository(db *gorm.DB) repository.StateRepository {
	return &stateRepository{db: db}
}

// CreateState inserts a new state record.
func (r *stateRepository) CreateState(ctx context.Context, state *models.State) error {
	state.ID = uuid.New().String()
	// Audit details should be set by handlers, only set time if not already set
	now := time.Now().UnixMilli()
	if state.AuditDetail.CreatedTime == 0 {
		state.AuditDetail.CreatedTime = now
	}
	if state.AuditDetail.ModifiedTime == 0 {
		state.AuditDetail.ModifiedTime = now
	}

	return r.db.WithContext(ctx).Create(state).Error
}

// GetStatesByProcessID retrieves all states for a given process.
func (r *stateRepository) GetStatesByProcessID(ctx context.Context, tenantID, processID string, codes []string) ([]*models.State, error) {
	var states []*models.State
	query := r.db.WithContext(ctx).
		Where("tenant_id = ? AND process_id = ?", tenantID, processID)

	if len(codes) > 0 {
		query = query.Where("code IN ?", codes)
	}

	err := query.
		Order("created_at").
		Find(&states).Error
	return states, err
}

// GetStateByID retrieves a single state by its ID.
func (r *stateRepository) GetStateByID(ctx context.Context, tenantID, id string) (*models.State, error) {
	var state models.State
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		First(&state).Error
	if err != nil {
		return nil, err
	}
	return &state, nil
}

// UpdateState updates an existing state record.
func (r *stateRepository) UpdateState(ctx context.Context, state *models.State) error {
	now := time.Now().UnixMilli()
	state.AuditDetail.ModifiedTime = now

	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", state.TenantID, state.ID).
		Updates(state).Error
}

// GetStateByCodeAndProcess finds a state by code within a specific process.
func (r *stateRepository) GetStateByCodeAndProcess(ctx context.Context, tenantID, processID, code string) (*models.State, error) {
	var state models.State
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND process_id = ? AND code = ?", tenantID, processID, code).
		First(&state).Error
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (r *stateRepository) DeleteState(ctx context.Context, tenantID, id string) error {
	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		Delete(&models.State{}).Error
}
