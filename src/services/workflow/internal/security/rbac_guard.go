package security

import (
	"errors"
	"fmt"
)

// RBACGuard provides a default implementation of the Guard interface
// based on roles, assignees, and attributes.
type RBACGuard struct{}

// NewRBACGuard creates a new RBACGuard.
func NewRBACGuard() Guard {
	return &RBACGuard{}
}

// CanTransition checks if a user is permitted to perform a state transition.
func (g *RBACGuard) CanTransition(ctx GuardContext) (bool, error) {
	fmt.Printf("🛡️ GUARD DEBUG: Starting validation\n")
	fmt.Printf("  Action: %s\n", ctx.Action.Name)
	fmt.Printf("  UserID: %s\n", ctx.UserID)
	fmt.Printf("  RequestAttributes: %+v\n", ctx.RequestAttributes)

	// 1. Assignee Check
	if ctx.Action.AttributeValidation != nil && ctx.Action.AttributeValidation.AssigneeCheck {
		if !isAssignee(ctx.UserID, ctx.ProcessInstance.Assignees) {
			return false, errors.New("user is not an assignee for this instance")
		}
	}

	// 2. Attribute Check - Check if request attributes match action requirements
	if ctx.Action.AttributeValidation != nil && ctx.Action.AttributeValidation.Attributes != nil {
		fmt.Printf("🔍 Starting attribute validation\n")
		if !hasRequiredAttributes(ctx.RequestAttributes, ctx.Action.AttributeValidation.Attributes) {
			return false, errors.New("request attributes do not match action requirements")
		}
	}

	fmt.Printf("✅ GUARD: All validations passed!\n")
	return true, nil
}

// hasRequiredRole checks if the user has at least one of the roles required by the action.
func hasRequiredRole(userRoles, requiredRoles []string) bool {
	if len(requiredRoles) == 0 {
		return true // No specific roles required
	}
	userRoleSet := make(map[string]struct{}, len(userRoles))
	for _, role := range userRoles {
		userRoleSet[role] = struct{}{}
	}
	for _, requiredRole := range requiredRoles {
		if _, ok := userRoleSet[requiredRole]; ok {
			return true
		}
	}
	return false
}

// isAssignee checks if the user is in the list of assignees for the process instance.
func isAssignee(userID string, assignees []string) bool {
	for _, assignee := range assignees {
		if assignee == userID {
			return true
		}
	}
	return false
}

// hasRequiredAttributes checks if the request attributes match the action requirements.
func hasRequiredAttributes(requestAttributes, requiredAttributes map[string][]string) bool {
	fmt.Printf("🔍 ATTR DEBUG: Checking attributes\n")
	fmt.Printf("  Required: %+v\n", requiredAttributes)
	fmt.Printf("  Request:  %+v\n", requestAttributes)

	for key, requiredValues := range requiredAttributes {
		requestValues, ok := requestAttributes[key]
		if !ok {
			fmt.Printf("❌ Missing required attribute key: %s\n", key)
			return false // Required attribute key is missing from the request
		}

		// Check if at least one of the request's values for this key is in the required list
		foundMatch := false
		for _, requiredVal := range requiredValues {
			for _, requestVal := range requestValues {
				if requestVal == requiredVal {
					foundMatch = true
					break
				}
			}
			if foundMatch {
				break
			}
		}

		if !foundMatch {
			fmt.Printf("❌ Attribute mismatch for key '%s'. Required one of %v, but request has %v\n", key, requiredValues, requestValues)
			return false // No matching value found for this attribute key
		}
		fmt.Printf("✅ Attribute match for key '%s': %v\n", key, requestValues)
	}
	fmt.Printf("✅ All attributes matched!\n")
	return true
}
