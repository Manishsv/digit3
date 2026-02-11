package security

import (
	"fmt"
	"strings"
)

// ConfigurableGuard implements Guard interface with configurable attribute validators
type ConfigurableGuard struct {
	registry          *ValidatorRegistry
	enabledValidators []string
}

// NewConfigurableGuard creates a new configurable guard with specified validators
func NewConfigurableGuard(enabledValidators []string) (*ConfigurableGuard, error) {
	registry := NewValidatorRegistry()

	// Validate that all enabled validators are available
	if err := registry.ValidateEnabledValidators(enabledValidators); err != nil {
		return nil, err
	}

	return &ConfigurableGuard{
		registry:          registry,
		enabledValidators: enabledValidators,
	}, nil
}

// CanTransition validates if a user can perform a transition based on configured validators
func (g *ConfigurableGuard) CanTransition(ctx GuardContext) (bool, error) {
	var validationErrors []string

	// 1. Check assignee validation only if required by the action
	shouldCheckAssignee := false
	if ctx.Action.AttributeValidation != nil && ctx.Action.AttributeValidation.AssigneeCheck {
		shouldCheckAssignee = true
	}

	if shouldCheckAssignee && !isAssignee(ctx.UserID, ctx.ProcessInstance.Assignees) {
		validationErrors = append(validationErrors, "user is not assigned to this process instance")
	}

	// 2. Check all enabled attribute validators with AND logic
	for _, validatorName := range g.enabledValidators {
		if valid, err := g.validateAttribute(validatorName, ctx); !valid {
			validationErrors = append(validationErrors, fmt.Sprintf("%s validation failed: %s", validatorName, err.Error()))
		}
	}

	// 3. Return combined error if any validation failed
	if len(validationErrors) > 0 {
		return false, fmt.Errorf("validation failed: %s", strings.Join(validationErrors, "; "))
	}

	return true, nil
}

// validateAttribute validates a specific attribute type
func (g *ConfigurableGuard) validateAttribute(validatorName string, ctx GuardContext) (bool, error) {
	// Get the validator for this attribute type
	validator, exists := g.registry.GetValidator(validatorName)
	if !exists {
		return false, fmt.Errorf("validator '%s' not found", validatorName)
	}

	// Get user attributes from request
	userAttrs, userAttrsExists := ctx.RequestAttributes[validatorName]
	if !userAttrsExists || len(userAttrs) == 0 {
		return false, fmt.Errorf("required attribute '%s' is missing in request", validatorName)
	}

	// Get required attributes from action's AttributeValidation.Attributes
	var requiredAttrs []string
	if ctx.Action.AttributeValidation != nil {
		if attrs, exists := ctx.Action.AttributeValidation.Attributes[validatorName]; exists {
			requiredAttrs = attrs
		}
	}

	// If action doesn't specify requirements for this attribute, allow it
	if len(requiredAttrs) == 0 {
		return true, nil
	}

	// Validate user attributes against required attributes
	return validator.Validate(userAttrs, requiredAttrs)
}
