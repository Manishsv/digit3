package security

import (
	"errors"
	"fmt"
	"strings"
)

// AttributeGuard provides simple attribute-based validation
// It compares request attributes with action requirements using direct key-value matching
type AttributeGuard struct{}

// NewAttributeGuard creates a new AttributeGuard
func NewAttributeGuard() Guard {
	return &AttributeGuard{}
}

// CanTransition validates if a user can perform a transition based on attribute matching
func (g *AttributeGuard) CanTransition(ctx GuardContext) (bool, error) {
	if ctx.Action.AttributeValidation == nil || len(ctx.Action.AttributeValidation.Attributes) == 0 {
		return true, nil
	}

	valid, err := g.validateAttributes(ctx.RequestAttributes, ctx.Action.AttributeValidation.Attributes)
	if !valid {
		return false, fmt.Errorf("validation failed: %s", err.Error())
	}

	return true, nil
}

// validateAttributes compares request attributes with action requirements
// requestAttrs: attributes from the transition request
// requiredAttrs: attributes required by the action
func (g *AttributeGuard) validateAttributes(requestAttrs, requiredAttrs map[string][]string) (bool, error) {
	var missingKeys []string
	var mismatchedValues []string

	// Iterate through all required attributes from the action
	for requiredKey, requiredValues := range requiredAttrs {
		// Check if the request contains this required key
		requestValues, keyExists := requestAttrs[requiredKey]
		if !keyExists || len(requestValues) == 0 {
			missingKeys = append(missingKeys, requiredKey)
			continue
		}

		// Check if at least one request value matches one of the required values
		hasMatch := false
		for _, reqVal := range requestValues {
			for _, requiredVal := range requiredValues {
				if strings.TrimSpace(reqVal) == strings.TrimSpace(requiredVal) {
					hasMatch = true
					break
				}
			}
			if hasMatch {
				break
			}
		}

		if !hasMatch {
			mismatchedValues = append(mismatchedValues, fmt.Sprintf("key '%s': request has %v, but requires one of %v", requiredKey, requestValues, requiredValues))
		}
	}

	// Build error message
	var errorParts []string
	if len(missingKeys) > 0 {
		errorParts = append(errorParts, fmt.Sprintf("missing required attributes: %v", missingKeys))
	}
	if len(mismatchedValues) > 0 {
		errorParts = append(errorParts, fmt.Sprintf("attribute mismatches: [%s]", strings.Join(mismatchedValues, ", ")))
	}

	if len(errorParts) > 0 {
		return false, errors.New(strings.Join(errorParts, "; "))
	}

	return true, nil
}
