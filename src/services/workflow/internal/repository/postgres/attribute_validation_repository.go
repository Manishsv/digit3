package postgres

import (
	"context"
	"time"

	"digit.org/workflow/internal/models"
	"digit.org/workflow/internal/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type attributeValidationRepository struct {
	db *gorm.DB
}

// NewAttributeValidationRepository creates a new instance of AttributeValidationRepository.
func NewAttributeValidationRepository(db *gorm.DB) repository.AttributeValidationRepository {
	return &attributeValidationRepository{db: db}
}

// CreateAttributeValidation inserts a new attribute validation record.
func (r *attributeValidationRepository) CreateAttributeValidation(ctx context.Context, validation *models.AttributeValidation) error {
	validation.ID = uuid.New().String()

	// Set default audit values if not provided
	if validation.AuditDetail.CreatedBy == "" {
		validation.AuditDetail.CreatedBy = "system"
	}
	if validation.AuditDetail.ModifiedBy == "" {
		validation.AuditDetail.ModifiedBy = "system"
	}

	now := time.Now().UnixMilli()
	validation.AuditDetail.CreatedTime = now
	validation.AuditDetail.ModifiedTime = now

	return r.db.WithContext(ctx).Create(validation).Error
}

// GetAttributeValidationByID retrieves a single attribute validation by its ID.
func (r *attributeValidationRepository) GetAttributeValidationByID(ctx context.Context, tenantID, id string) (*models.AttributeValidation, error) {
	var validation models.AttributeValidation
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		First(&validation).Error
	if err != nil {
		return nil, err
	}
	return &validation, nil
}
