package service

import (
	"context"
	"fmt"
	"log"

	"digit.org/workflow/internal/models"
	"digit.org/workflow/internal/repository"
)

type autoEscalationService struct {
	escalationConfigRepo repository.EscalationConfigRepository
	processInstanceRepo  repository.ProcessInstanceRepository
	processRepo          repository.ProcessRepository
	stateRepo            repository.StateRepository
	transitionService    TransitionService
}

// NewAutoEscalationService creates a new instance of AutoEscalationService.
func NewAutoEscalationService(
	escalationConfigRepo repository.EscalationConfigRepository,
	processInstanceRepo repository.ProcessInstanceRepository,
	processRepo repository.ProcessRepository,
	stateRepo repository.StateRepository,
	transitionService TransitionService,
) AutoEscalationService {
	return &autoEscalationService{
		escalationConfigRepo: escalationConfigRepo,
		processInstanceRepo:  processInstanceRepo,
		processRepo:          processRepo,
		stateRepo:            stateRepo,
		transitionService:    transitionService,
	}
}

// EscalateApplications triggers auto-escalation for a specific process based on SLA breaches.
func (s *autoEscalationService) EscalateApplications(ctx context.Context, tenantID, processCode string, attributes map[string][]string, userID string) (*models.EscalationResult, error) {
	// Get process by code
	process, err := s.processRepo.GetProcessByCode(ctx, tenantID, processCode)
	if err != nil {
		return nil, fmt.Errorf("process with code '%s' not found: %w", processCode, err)
	}

	// Get all escalation configs for this process
	escalationConfigs, err := s.escalationConfigRepo.GetEscalationConfigsByProcessID(ctx, tenantID, process.ID, "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get escalation configs: %w", err)
	}

	if len(escalationConfigs) == 0 {
		return &models.EscalationResult{
			TotalFound:         0,
			TotalEscalated:     0,
			EscalatedInstances: []*models.ProcessInstance{},
		}, nil
	}

	result := &models.EscalationResult{
		EscalatedInstances: []*models.ProcessInstance{},
		Errors:             []string{},
	}

	// Process each escalation config
	for _, config := range escalationConfigs {
		log.Printf("Processing escalation config: state=%s, action=%s", config.StateCode, config.EscalationAction)

		// Get the state ID from state code
		state, err := s.stateRepo.GetStateByCodeAndProcess(ctx, tenantID, process.ID, config.StateCode)
		if err != nil {
			errorMsg := fmt.Sprintf("failed to find state '%s' in process: %v", config.StateCode, err)
			result.Errors = append(result.Errors, errorMsg)
			continue
		}

		// Find SLA-breached instances for this state
		breachedInstances, err := s.processInstanceRepo.GetSLABreachedInstances(
			ctx, tenantID, process.ID, state.ID,
			config.StateSlaMinutes, config.ProcessSlaMinutes,
		)
		if err != nil {
			errorMsg := fmt.Sprintf("failed to get SLA breached instances for state %s: %v", config.StateCode, err)
			result.Errors = append(result.Errors, errorMsg)
			continue
		}

		result.TotalFound += len(breachedInstances)
		log.Printf("Found %d SLA-breached instances for state %s", len(breachedInstances), config.StateCode)

		// Escalate each instance
		for _, instance := range breachedInstances {
			escalatedInstance, err := s.escalateInstance(ctx, instance, config, attributes, userID)
			if err != nil {
				errorMsg := fmt.Sprintf("failed to escalate instance %s: %v", instance.EntityID, err)
				result.Errors = append(result.Errors, errorMsg)
				log.Printf("Error escalating instance %s: %v", instance.EntityID, err)
				continue
			}

			if escalatedInstance != nil {
				result.EscalatedInstances = append(result.EscalatedInstances, escalatedInstance)
				result.TotalEscalated++
				log.Printf("Successfully escalated instance %s from state %s", instance.EntityID, config.StateCode)
			}
		}
	}

	log.Printf("Auto-escalation completed: found=%d, escalated=%d, errors=%d",
		result.TotalFound, result.TotalEscalated, len(result.Errors))

	return result, nil
}

// escalateInstance performs the actual escalation transition for a single instance.
func (s *autoEscalationService) escalateInstance(ctx context.Context, instance *models.ProcessInstance, config *models.EscalationConfig, attributes map[string][]string, userID string) (*models.ProcessInstance, error) {
	// Create escalation comment
	comment := fmt.Sprintf("Auto-escalated from state %s due to SLA breach", config.StateCode)

	// Perform the transition using the existing transition service
	escalatedInstance, err := s.transitionService.Transition(
		ctx,
		nil, // processInstanceID - let it create new instance
		instance.ProcessID,
		instance.EntityID,
		config.EscalationAction,
		nil, // init - escalations are actions, not initialization
		nil, // status - let service handle it
		nil, // currentState - auto-escalation doesn't validate current state
		&comment,
		[]string{}, // empty documents (fileStoreIds)
		nil,        // assigner
		nil,        // assignees
		attributes,
		instance.TenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("transition failed: %w", err)
	}

	return escalatedInstance, nil
}

// SearchEscalatedApplications retrieves process instances that have been auto-escalated.
func (s *autoEscalationService) SearchEscalatedApplications(ctx context.Context, tenantID, processID string, limit, offset int) ([]*models.ProcessInstance, error) {
	return s.processInstanceRepo.GetEscalatedInstances(ctx, tenantID, processID, limit, offset)
}
