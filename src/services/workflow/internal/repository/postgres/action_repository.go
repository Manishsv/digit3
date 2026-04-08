package postgres

import (
	"context"
	"fmt"
	"time"

	"digit.org/workflow/internal/models"
	"digit.org/workflow/internal/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type actionRepository struct {
	db                      *gorm.DB
	attributeValidationRepo repository.AttributeValidationRepository
}

// NewActionRepository creates a new instance of ActionRepository.
func NewActionRepository(db *gorm.DB, attributeValidationRepo repository.AttributeValidationRepository) repository.ActionRepository {
	return &actionRepository{
		db:                      db,
		attributeValidationRepo: attributeValidationRepo,
	}
}

// CreateAction creates a new action in the database.
func (r *actionRepository) CreateAction(ctx context.Context, action *models.Action) (*models.Action, error) {
	// Generate UUID if not provided
	if action.ID == "" {
		action.ID = uuid.New().String()
	}

	now := time.Now().UnixMilli()
	createdBy := action.AuditDetail.CreatedBy
	if createdBy == "" {
		createdBy = "system"
	} else {
		createdBy = models.ClampAuditUserID(createdBy)
	}

	// Set audit details
	action.AuditDetail.CreatedBy = createdBy
	action.AuditDetail.CreatedTime = now
	action.AuditDetail.ModifiedBy = createdBy
	action.AuditDetail.ModifiedTime = now

	// Handle AttributeValidation if provided
	if action.AttributeValidation != nil {
		action.AttributeValidation.TenantID = action.TenantID
		err := r.attributeValidationRepo.CreateAttributeValidation(ctx, action.AttributeValidation)
		if err != nil {
			return nil, fmt.Errorf("failed to create attribute validation: %w", err)
		}
		action.AttributeValidationID = &action.AttributeValidation.ID
	}

	err := r.db.WithContext(ctx).Create(action).Error
	if err != nil {
		return nil, fmt.Errorf("failed to create action: %w", err)
	}

	return action, nil
}

// GetActionsByStateID retrieves all actions for a given state.
func (r *actionRepository) GetActionsByStateID(ctx context.Context, tenantID, stateID string, codes []string) ([]*models.Action, error) {
	var actions []*models.Action
	query := r.db.WithContext(ctx).
		Where("tenant_id = ? AND current_state_id = ?", tenantID, stateID)

	if len(codes) > 0 {
		query = query.Where("name IN ?", codes)
	}

	err := query.Find(&actions).Error
	if err != nil {
		return nil, err
	}

	// Load AttributeValidation for each action if ID exists
	for _, action := range actions {
		if action.AttributeValidationID != nil {
			attributeValidation, err := r.attributeValidationRepo.GetAttributeValidationByID(ctx, tenantID, *action.AttributeValidationID)
			if err == nil {
				action.AttributeValidation = attributeValidation
			}
		}
	}

	return actions, nil
}

// GetActionByID retrieves a single action by its ID.
func (r *actionRepository) GetActionByID(ctx context.Context, tenantID, id string) (*models.Action, error) {
	var action models.Action
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		First(&action).Error
	if err != nil {
		return nil, err
	}

	// Load AttributeValidation if ID exists
	if action.AttributeValidationID != nil {
		attributeValidation, err := r.attributeValidationRepo.GetAttributeValidationByID(ctx, tenantID, *action.AttributeValidationID)
		if err == nil {
			action.AttributeValidation = attributeValidation
		}
	}

	return &action, nil
}

// UpdateAction updates an existing action in the database.
func (r *actionRepository) UpdateAction(ctx context.Context, action *models.Action) error {
	action.AuditDetail.ModifiedTime = time.Now().UnixMilli()

	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", action.TenantID, action.ID).
		Updates(action).Error
}

// DeleteAction removes an action record from the database.
func (r *actionRepository) DeleteAction(ctx context.Context, tenantID, id string) error {
	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		Delete(&models.Action{}).Error
}
