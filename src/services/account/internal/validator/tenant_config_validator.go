package validator

import (
	"account/internal/models"
)

func ValidateTenantConfig(config *models.TenantConfig, existing []*models.TenantConfig) error {
	for _, t := range existing {
		if t.Code == config.Code {
			return ErrDuplicateTenantConfigCode
		}
	}
	return nil
}

var ErrDuplicateTenantConfigCode = &ValidationError{"DUPLICATE_RECORD", "TenantConfig with this code already exists"}
