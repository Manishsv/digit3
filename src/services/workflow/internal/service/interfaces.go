package service

import (
	"context"

	"digit.org/workflow/internal/models"
)

// ProcessService defines the interface for process business logic.
type ProcessService interface {
	CreateProcess(ctx context.Context, process *models.Process) (*models.Process, error)
	GetProcessByID(ctx context.Context, tenantID, id string) (*models.Process, error)
	GetProcessByCode(ctx context.Context, tenantID, code string) (*models.Process, error)
	GetProcesses(ctx context.Context, tenantID string, ids []string, names []string, codes []string) ([]*models.Process, error)
	GetProcessDefinitions(ctx context.Context, tenantID string, ids []string, names []string) ([]*models.ProcessDefinitionDetail, error)
	UpdateProcess(ctx context.Context, process *models.Process) (*models.Process, error)
	DeleteProcess(ctx context.Context, tenantID, id string) error
}

// StateService defines the interface for state business logic.
type StateService interface {
	CreateState(ctx context.Context, state *models.State) (*models.State, error)
	GetStatesByProcessID(ctx context.Context, tenantID, processID string, codes []string) ([]*models.State, error)
	GetStateByID(ctx context.Context, tenantID, id string) (*models.State, error)
	GetStateByCodeAndProcess(ctx context.Context, tenantID, processID, code string) (*models.State, error)
	UpdateState(ctx context.Context, state *models.State) (*models.State, error)
	DeleteState(ctx context.Context, tenantID, id string) error
}

// ActionService defines the interface for action business logic.
type ActionService interface {
	CreateAction(ctx context.Context, action *models.Action) (*models.Action, error)
	GetActionsByStateID(ctx context.Context, tenantID, stateID string, codes []string) ([]*models.Action, error)
	GetActionByID(ctx context.Context, tenantID, id string) (*models.Action, error)
	UpdateAction(ctx context.Context, action *models.Action) (*models.Action, error)
	DeleteAction(ctx context.Context, tenantID, id string) error
}

// TransitionService handles business logic for process instance transitions.
type TransitionService interface {
	Transition(ctx context.Context, processInstanceID *string, processID, entityID, action string, init *bool, status *string, currentState *string, comment *string, documents []string, assigner *string, assignees *[]string, attributes map[string][]string, tenantID string) (*models.ProcessInstance, error)
	GetTransitions(ctx context.Context, tenantID string, criteria *models.ProcessInstanceSearchCriteria) ([]*models.ProcessInstance, error)
}

// EscalationConfigService handles business logic for escalation configurations.
type EscalationConfigService interface {
	CreateEscalationConfig(ctx context.Context, config *models.EscalationConfig) (*models.EscalationConfig, error)
	GetEscalationConfigByID(ctx context.Context, tenantID, id string) (*models.EscalationConfig, error)
	GetEscalationConfigsByProcessID(ctx context.Context, tenantID, processID string, stateCode string, isActive *bool) ([]*models.EscalationConfig, error)
	UpdateEscalationConfig(ctx context.Context, config *models.EscalationConfig) (*models.EscalationConfig, error)
	DeleteEscalationConfig(ctx context.Context, tenantID, id string) error
}

// AutoEscalationService handles auto-escalation business logic.
type AutoEscalationService interface {
	EscalateApplications(ctx context.Context, tenantID, processCode string, attributes map[string][]string, userID string) (*models.EscalationResult, error)
	SearchEscalatedApplications(ctx context.Context, tenantID, processID string, limit, offset int) ([]*models.ProcessInstance, error)
}

// More service interfaces will be added here later.
