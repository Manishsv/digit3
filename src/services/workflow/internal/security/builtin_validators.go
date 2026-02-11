package security

import (
	"fmt"
)

// RolesValidator validates user roles against required roles
type RolesValidator struct{}

func (v *RolesValidator) ValidatorName() string {
	return "roles"
}

func (v *RolesValidator) Validate(userAttrs, requiredAttrs []string) (bool, error) {
	if len(userAttrs) == 0 {
		return false, fmt.Errorf("user has no roles assigned")
	}

	if len(requiredAttrs) == 0 {
		return true, nil // No role restrictions
	}

	if hasAnyValue(userAttrs, requiredAttrs) {
		return true, nil
	}

	return false, fmt.Errorf("user roles %v not in allowed roles %v", userAttrs, requiredAttrs)
}

// JurisdictionValidator validates user jurisdiction against required jurisdictions
type JurisdictionValidator struct{}

func (v *JurisdictionValidator) ValidatorName() string {
	return "jurisdiction"
}

func (v *JurisdictionValidator) Validate(userAttrs, requiredAttrs []string) (bool, error) {
	if len(userAttrs) == 0 {
		return false, fmt.Errorf("user has no jurisdiction assigned")
	}

	if len(requiredAttrs) == 0 {
		return true, nil // No jurisdiction restrictions
	}

	if hasAnyValue(userAttrs, requiredAttrs) {
		return true, nil
	}

	return false, fmt.Errorf("user jurisdiction %v not in allowed jurisdictions %v", userAttrs, requiredAttrs)
}
