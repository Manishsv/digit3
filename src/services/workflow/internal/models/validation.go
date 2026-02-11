package models

import (
	"fmt"
	"regexp"
	"strings"
)

// UUIDRegex is a regular expression for validating UUIDs
var UUIDRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrors represents multiple validation errors
type ValidationErrors struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Errors  []ValidationError `json:"errors,omitempty"`
}

// IsValidUUID checks if a string is a valid UUID
func IsValidUUID(uuid string) bool {
	return UUIDRegex.MatchString(strings.ToLower(uuid))
}

// ValidateUUID validates a UUID and returns an error if invalid
func ValidateUUID(uuid, fieldName string) error {
	if !IsValidUUID(uuid) {
		return fmt.Errorf("invalid %s: '%s' is not a valid UUID", fieldName, uuid)
	}
	return nil
}

// ValidateProcessCreate validates a process for creation
func ValidateProcessCreate(process *Process) *ValidationErrors {
	var errors []ValidationError

	// Required field validations
	if strings.TrimSpace(process.Name) == "" {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "name is required and cannot be empty",
		})
	}

	if strings.TrimSpace(process.Code) == "" {
		errors = append(errors, ValidationError{
			Field:   "code",
			Message: "code is required and cannot be empty",
		})
	}

	// Length validations
	if len(process.Name) > 128 {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "name cannot exceed 128 characters",
		})
	}

	if len(process.Code) > 128 {
		errors = append(errors, ValidationError{
			Field:   "code",
			Message: "code cannot exceed 128 characters",
		})
	}

	if process.Description != nil && len(*process.Description) > 512 {
		errors = append(errors, ValidationError{
			Field:   "description",
			Message: "description cannot exceed 512 characters",
		})
	}

	if process.Version != nil && len(*process.Version) > 32 {
		errors = append(errors, ValidationError{
			Field:   "version",
			Message: "version cannot exceed 32 characters",
		})
	}

	if len(errors) > 0 {
		return &ValidationErrors{
			Code:    "ValidationError",
			Message: "Invalid input data",
			Errors:  errors,
		}
	}

	return nil
}

// ValidateStateCreate validates a state for creation
func ValidateStateCreate(state *State) *ValidationErrors {
	var errors []ValidationError

	// Required field validations
	if strings.TrimSpace(state.Code) == "" {
		errors = append(errors, ValidationError{
			Field:   "code",
			Message: "code is required and cannot be empty",
		})
	}

	if strings.TrimSpace(state.Name) == "" {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "name is required and cannot be empty",
		})
	}

	// Length validations
	if len(state.Code) > 64 {
		errors = append(errors, ValidationError{
			Field:   "code",
			Message: "code cannot exceed 64 characters",
		})
	}

	if len(state.Name) > 128 {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "name cannot exceed 128 characters",
		})
	}

	if state.Description != nil && len(*state.Description) > 512 {
		errors = append(errors, ValidationError{
			Field:   "description",
			Message: "description cannot exceed 512 characters",
		})
	}

	if len(errors) > 0 {
		return &ValidationErrors{
			Code:    "ValidationError",
			Message: "Invalid input data",
			Errors:  errors,
		}
	}

	return nil
}

// ValidateActionCreate validates an action for creation
func ValidateActionCreate(action *Action) *ValidationErrors {
	var errors []ValidationError

	// Required field validations
	if strings.TrimSpace(action.Name) == "" {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "name is required and cannot be empty",
		})
	}

	if strings.TrimSpace(action.CurrentState) == "" {
		errors = append(errors, ValidationError{
			Field:   "currentState",
			Message: "currentState is required and cannot be empty",
		})
	}

	if strings.TrimSpace(action.NextState) == "" {
		errors = append(errors, ValidationError{
			Field:   "nextState",
			Message: "nextState is required and cannot be empty",
		})
	}

	// Length validations
	if len(action.Name) > 64 {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "name cannot exceed 64 characters",
		})
	}

	if action.Label != nil && len(*action.Label) > 128 {
		errors = append(errors, ValidationError{
			Field:   "label",
			Message: "label cannot exceed 128 characters",
		})
	}

	// UUID validations
	if action.CurrentState != "" && !IsValidUUID(action.CurrentState) {
		errors = append(errors, ValidationError{
			Field:   "currentState",
			Message: "currentState must be a valid UUID",
		})
	}

	if action.NextState != "" && !IsValidUUID(action.NextState) {
		errors = append(errors, ValidationError{
			Field:   "nextState",
			Message: "nextState must be a valid UUID",
		})
	}

	if len(errors) > 0 {
		return &ValidationErrors{
			Code:    "ValidationError",
			Message: "Invalid input data",
			Errors:  errors,
		}
	}

	return nil
}

// IsDatabaseConstraintError checks if an error is a database constraint violation
func IsDatabaseConstraintError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return IsDuplicateKeyError(err) ||
		strings.Contains(errMsg, "SQLSTATE 22001") || // value too long
		strings.Contains(errMsg, "SQLSTATE 22P02") || // invalid input syntax
		strings.Contains(errMsg, "value too long") ||
		strings.Contains(errMsg, "invalid input syntax") ||
		IsForeignKeyViolation(err)
}

// IsDuplicateKeyError returns true if the error is a duplicate/unique constraint violation.
func IsDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "SQLSTATE 23505") ||
		strings.Contains(errMsg, "duplicate key value")
}

// IsForeignKeyViolation returns true if the error signifies a FK constraint failure.
func IsForeignKeyViolation(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "SQLSTATE 23503") ||
		strings.Contains(errMsg, "violates foreign key constraint")
}

// GetConstraintErrorMessage returns a user-friendly error message for database constraint violations
func GetConstraintErrorMessage(err error) string {
	if err == nil {
		return "Unknown database error"
	}

	errMsg := err.Error()

	if strings.Contains(errMsg, "duplicate key value") {
		if strings.Contains(errMsg, "processes_tenant_id_code_key") {
			return "A process with this code already exists"
		}
		return "This record already exists"
	}

	if strings.Contains(errMsg, "value too long") {
		if strings.Contains(errMsg, "character varying(128)") {
			return "Input value exceeds maximum length of 128 characters"
		}
		if strings.Contains(errMsg, "character varying(64)") {
			return "Input value exceeds maximum length of 64 characters"
		}
		if strings.Contains(errMsg, "character varying(512)") {
			return "Input value exceeds maximum length of 512 characters"
		}
		return "Input value is too long"
	}

	if strings.Contains(errMsg, "invalid input syntax") && strings.Contains(errMsg, "uuid") {
		return "Invalid UUID format provided"
	}

	if IsForeignKeyViolation(err) {
		return "Record cannot be modified because it is referenced by other resources"
	}

	return "Invalid input data"
}
