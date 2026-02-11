package security

import (
	"fmt"
	"strings"
)

// AttributeValidator defines the interface for validating specific attribute types
type AttributeValidator interface {
	ValidatorName() string
	Validate(userAttrs, requiredAttrs []string) (bool, error)
}

// ValidatorRegistry manages available attribute validators
type ValidatorRegistry struct {
	validators map[string]AttributeValidator
}

// NewValidatorRegistry creates a new validator registry with built-in validators
func NewValidatorRegistry() *ValidatorRegistry {
	registry := &ValidatorRegistry{
		validators: make(map[string]AttributeValidator),
	}

	// Register built-in validators
	registry.Register(&RolesValidator{})
	registry.Register(&JurisdictionValidator{})

	return registry
}

// Register adds a validator to the registry
func (r *ValidatorRegistry) Register(validator AttributeValidator) {
	r.validators[validator.ValidatorName()] = validator
}

// GetValidator retrieves a validator by name
func (r *ValidatorRegistry) GetValidator(name string) (AttributeValidator, bool) {
	validator, exists := r.validators[name]
	return validator, exists
}

// GetAvailableValidators returns list of all registered validator names
func (r *ValidatorRegistry) GetAvailableValidators() []string {
	names := make([]string, 0, len(r.validators))
	for name := range r.validators {
		names = append(names, name)
	}
	return names
}

// ValidateEnabledValidators checks if all enabled validators are registered
func (r *ValidatorRegistry) ValidateEnabledValidators(enabledValidators []string) error {
	var invalidValidators []string

	for _, validatorName := range enabledValidators {
		if _, exists := r.validators[validatorName]; !exists {
			invalidValidators = append(invalidValidators, validatorName)
		}
	}

	if len(invalidValidators) > 0 {
		available := strings.Join(r.GetAvailableValidators(), ", ")
		return fmt.Errorf("invalid validators configured: [%s]. Available validators: [%s]",
			strings.Join(invalidValidators, ", "), available)
	}

	return nil
}

// hasAnyValue checks if userAttrs contains any value from requiredAttrs
func hasAnyValue(userAttrs, requiredAttrs []string) bool {
	for _, userAttr := range userAttrs {
		for _, requiredAttr := range requiredAttrs {
			if strings.TrimSpace(userAttr) == strings.TrimSpace(requiredAttr) {
				return true
			}
		}
	}
	return false
}
