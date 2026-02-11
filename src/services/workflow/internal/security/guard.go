package security

import "digit.org/workflow/internal/models"

// GuardContext holds all the necessary information to make an authorization decision.
type GuardContext struct {
	UserRoles         []string
	UserID            string
	ProcessInstance   *models.ProcessInstance
	Action            *models.Action
	RequestAttributes map[string][]string
}

// Guard is the interface for pluggable authorization logic.
// It allows different strategies (RBAC, ABAC, etc.) to be used to protect state transitions.
type Guard interface {
	CanTransition(ctx GuardContext) (bool, error)
}
