package validator

import (
	"account/internal/models"
)

func ValidateTenant(tenant *models.Tenant, existing []*models.Tenant) error {
	for _, t := range existing {
		if t.Code == tenant.Code {
			return ErrDuplicateTenantCode
		}
	}
	return nil
}

var ErrDuplicateTenantCode = &ValidationError{"DUPLICATE_RECORD", "Tenant with this code already exists"}
