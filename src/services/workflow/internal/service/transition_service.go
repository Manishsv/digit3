package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"digit.org/workflow/internal/models"
	"digit.org/workflow/internal/repository"
	"digit.org/workflow/internal/security"
	"github.com/looplab/fsm"
)

type transitionService struct {
	instanceRepo repository.ProcessInstanceRepository
	stateRepo    repository.StateRepository
	actionRepo   repository.ActionRepository
	processRepo  repository.ProcessRepository
	parallelRepo repository.ParallelExecutionRepository
	guard        security.Guard
}

// NewTransitionService creates a new instance of TransitionService.
func NewTransitionService(
	instanceRepo repository.ProcessInstanceRepository,
	stateRepo repository.StateRepository,
	actionRepo repository.ActionRepository,
	processRepo repository.ProcessRepository,
	parallelRepo repository.ParallelExecutionRepository,
	guard security.Guard,
) TransitionService {
	return &transitionService{
		instanceRepo: instanceRepo,
		stateRepo:    stateRepo,
		actionRepo:   actionRepo,
		processRepo:  processRepo,
		parallelRepo: parallelRepo,
		guard:        guard,
	}
}

func (s *transitionService) Transition(ctx context.Context, processInstanceID *string, processID, entityID, action string, init *bool, status *string, currentState *string, comment *string, documents []string, assigner *string, assignees *[]string, attributes map[string][]string, tenantID string) (*models.ProcessInstance, error) {
	// Convert documents []string to []models.Document
	var docModels []models.Document
	for _, fileStoreID := range documents {
		docModels = append(docModels, models.Document{
			FileStoreID: fileStoreID,
		})
	}

	// Build ProcessInstance from parameters
	instance := &models.ProcessInstance{
		ProcessID:  processID, // Use provided process ID directly
		EntityID:   entityID,
		Action:     action,
		Comment:    comment,
		Documents:  docModels,
		Assigner:   assigner,
		TenantID:   tenantID,
		Attributes: attributes, // User attributes for validation
	}

	if processInstanceID != nil {
		instance.ID = *processInstanceID
	}

	if status != nil {
		instance.Status = *status
	}

	if assignees != nil {
		instance.Assignees = *assignees
	}

	// Use the original getOrCreateInstance logic
	existingInstance, err := s.getOrCreateInstance(ctx, tenantID, instance, currentState)
	if err != nil {
		return nil, err
	}

	// If no action is provided, return the instance (creation case)
	if action == "" {
		return existingInstance, nil
	}

	// Find the action by getting all actions for current state and matching by name
	actions, err := s.actionRepo.GetActionsByStateID(ctx, tenantID, existingInstance.CurrentState, nil)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve actions for current state '%s': %w", existingInstance.CurrentState, err)
	}

	var targetAction *models.Action
	for _, act := range actions {
		if act.Name == action {
			targetAction = act
			break
		}
	}

	if targetAction == nil {
		return nil, fmt.Errorf("action '%s' is not valid for current state '%s'", action, existingInstance.CurrentState)
	}

	// Authorization Guard Check
	userID := "anonymous"
	var userRoles []string

	// Extract user information from context (for testing)
	if uid := ctx.Value("userID"); uid != nil {
		if uidStr, ok := uid.(string); ok {
			userID = uidStr
		}
	}
	if roles := ctx.Value("userRoles"); roles != nil {
		if rolesSlice, ok := roles.([]string); ok {
			userRoles = rolesSlice
		}
	}

	guardCtx := security.GuardContext{
		UserRoles:         userRoles,
		UserID:            userID,
		ProcessInstance:   existingInstance,
		Action:            targetAction,
		RequestAttributes: instance.Attributes,
	}

	can, err := s.guard.CanTransition(guardCtx)
	if err != nil {
		return nil, fmt.Errorf("guard check failed: %w", err)
	}
	if !can {
		return nil, fmt.Errorf("user is not authorized to perform action '%s'", action)
	}

	// Find target state to determine transition type
	targetState, err := s.stateRepo.GetStateByID(ctx, tenantID, targetAction.NextState)
	if err != nil {
		return nil, fmt.Errorf("could not find target state for action '%s': %w", action, err)
	}

	// Determine transition type and delegate
	if targetState.IsParallel {
		return s.handleParallelTransition(ctx, tenantID, existingInstance, targetAction, targetState, instance)
	} else if targetState.IsJoin {
		return s.handleJoinTransition(ctx, tenantID, existingInstance, targetAction, targetState, instance)
	} else {
		return s.handleLinearTransition(ctx, tenantID, existingInstance, targetAction, instance)
	}
}

// handleLinearTransition processes standard non-parallel transitions
func (s *transitionService) handleLinearTransition(ctx context.Context, tenantID string, existingInstance *models.ProcessInstance, targetAction *models.Action, instance *models.ProcessInstance) (*models.ProcessInstance, error) {
	// Construct FSM and transition
	actions, err := s.actionRepo.GetActionsByStateID(ctx, tenantID, existingInstance.CurrentState, nil)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve actions for FSM: %w", err)
	}

	events := eventsForState(actions)
	fmt.Printf("🔍 DEBUG: Building FSM with %d events for current state %s\n", len(events), existingInstance.CurrentState)
	for i, event := range events {
		fmt.Printf("   Event %d: %s from %v to %s\n", i, event.Name, event.Src, event.Dst)
	}

	fsm, err := s.buildFSM(ctx, tenantID, existingInstance, events)
	if err != nil {
		return nil, err
	}

	fmt.Printf("🚀 DEBUG: Executing FSM event '%s' from current state: %s\n", instance.Action, fsm.Current())
	err = fsm.Event(ctx, instance.Action)
	if err != nil {
		return nil, fmt.Errorf("invalid state transition: %w", err)
	}
	fmt.Printf("✅ DEBUG: FSM transitioned to new state: %s\n", fsm.Current())

	// Create new process instance record for this transition
	newInstance := &models.ProcessInstance{
		ProcessID:        existingInstance.ProcessID,
		EntityID:         existingInstance.EntityID,
		Action:           instance.Action,
		Status:           existingInstance.Status,
		Comment:          instance.Comment,
		Documents:        instance.Documents,
		Assignees:        instance.Assignees,
		CurrentState:     fsm.Current(),
		StateSLA:         existingInstance.StateSLA,
		ProcessSLA:       existingInstance.ProcessSLA,
		Attributes:       instance.Attributes,
		ParentInstanceID: existingInstance.ParentInstanceID,
		BranchID:         existingInstance.BranchID,
		IsParallelBranch: existingInstance.IsParallelBranch,
		TenantID:         tenantID,
		// Set escalated flag based on whether this is an auto-escalation action
		// Following Java service pattern: escalated = true for auto-escalation actions
		Escalated: s.isEscalationAction(instance.Action, instance.Comment),
	}

	// Set audit details
	newInstance.AuditDetails.SetAuditDetailsForCreate("system")

	if err := s.instanceRepo.CreateProcessInstance(ctx, newInstance); err != nil {
		return nil, fmt.Errorf("failed to create new process instance record: %w", err)
	}

	// Populate NextActions
	nextActions, err := s.actionRepo.GetActionsByStateID(ctx, tenantID, newInstance.CurrentState, nil)
	if err != nil {
		newInstance.NextActions = []string{}
	} else {
		newInstance.NextActions = make([]string, len(nextActions))
		for i, action := range nextActions {
			newInstance.NextActions[i] = action.Name
		}
	}

	return newInstance, nil
}

// handleParallelTransition creates parallel branches when transitioning to a parallel state
func (s *transitionService) handleParallelTransition(ctx context.Context, tenantID string, existingInstance *models.ProcessInstance, targetAction *models.Action, parallelState *models.State, instance *models.ProcessInstance) (*models.ProcessInstance, error) {
	fmt.Printf("🔀 DEBUG: Starting parallel transition to state %s with branches: %v\n", parallelState.Code, parallelState.BranchStates)

	// 1. Find the join state for this parallel execution
	joinState, err := s.findJoinStateForParallel(ctx, tenantID, parallelState)
	if err != nil {
		return nil, fmt.Errorf("could not find join state: %w", err)
	}

	// 2. Create parallel execution record
	parallelExec := &models.ParallelExecution{
		EntityID:          existingInstance.EntityID,
		ProcessID:         existingInstance.ProcessID,
		ParallelStateID:   parallelState.ID,
		JoinStateID:       joinState.ID,
		ActiveBranches:    parallelState.BranchStates,
		CompletedBranches: []string{},
		Status:            "ACTIVE",
		TenantID:          tenantID,
	}
	parallelExec.AuditDetail.SetAuditDetailsForCreate("system")

	if err := s.parallelRepo.CreateParallelExecution(ctx, parallelExec); err != nil {
		return nil, fmt.Errorf("could not create parallel execution: %w", err)
	}

	// 3. Create process instances for each parallel branch
	var branchInstances []*models.ProcessInstance
	for _, branchStateCode := range parallelState.BranchStates {
		branchState, err := s.stateRepo.GetStateByCodeAndProcess(ctx, tenantID, existingInstance.ProcessID, branchStateCode)
		if err != nil {
			return nil, fmt.Errorf("could not find branch state %s: %w", branchStateCode, err)
		}

		branchInstance := &models.ProcessInstance{
			ProcessID:        existingInstance.ProcessID,
			EntityID:         existingInstance.EntityID,
			Action:           instance.Action,
			Status:           "ACTIVE",
			Comment:          instance.Comment,
			Documents:        instance.Documents,
			Assignees:        instance.Assignees,
			CurrentState:     branchState.ID,
			ParentInstanceID: &existingInstance.ID,
			BranchID:         &branchStateCode,
			IsParallelBranch: true,
			Attributes:       instance.Attributes,
			TenantID:         tenantID,
			Escalated:        s.isEscalationAction(instance.Action, instance.Comment),
		}
		branchInstance.AuditDetails.SetAuditDetailsForCreate("system")

		if err := s.instanceRepo.CreateProcessInstance(ctx, branchInstance); err != nil {
			return nil, fmt.Errorf("could not create branch instance for %s: %w", branchStateCode, err)
		}
		branchInstances = append(branchInstances, branchInstance)
		fmt.Printf("✅ DEBUG: Created parallel branch instance for %s (ID: %s)\n", branchStateCode, branchInstance.ID)
	}

	// 4. Return first branch instance with parallel execution info
	result := branchInstances[0]

	// Add next actions for this branch
	nextActions, err := s.actionRepo.GetActionsByStateID(ctx, tenantID, result.CurrentState, nil)
	if err == nil {
		result.NextActions = make([]string, len(nextActions))
		for i, action := range nextActions {
			result.NextActions[i] = action.Name
		}
	}

	fmt.Printf("🎉 DEBUG: Parallel transition completed. Created %d branch instances\n", len(branchInstances))
	return result, nil
}

// handleJoinTransition manages joining parallel branches back to linear workflow
func (s *transitionService) handleJoinTransition(ctx context.Context, tenantID string, existingInstance *models.ProcessInstance, targetAction *models.Action, joinState *models.State, instance *models.ProcessInstance) (*models.ProcessInstance, error) {
	fmt.Printf("🔗 DEBUG: Processing join transition from branch %s to join state %s\n", *existingInstance.BranchID, joinState.Code)

	// 1. Find the parallel execution this branch belongs to
	parallelExec, err := s.findParallelExecutionForBranch(ctx, tenantID, existingInstance)
	if err != nil {
		return nil, fmt.Errorf("could not find parallel execution: %w", err)
	}

	// 2. Mark this branch as completed
	branchID := *existingInstance.BranchID
	if err := s.parallelRepo.MarkBranchCompleted(ctx, tenantID, existingInstance.EntityID, existingInstance.ProcessID, branchID); err != nil {
		return nil, fmt.Errorf("could not mark branch completed: %w", err)
	}

	// 3. Get updated parallel execution to check completion status
	updatedExec, err := s.parallelRepo.GetParallelExecution(ctx, tenantID, existingInstance.EntityID, existingInstance.ProcessID, parallelExec.ParallelStateID)
	if err != nil {
		return nil, fmt.Errorf("could not get updated parallel execution: %w", err)
	}

	fmt.Printf("🔍 DEBUG: Branch completion status: %d/%d completed\n", len(updatedExec.CompletedBranches), len(updatedExec.ActiveBranches))

	// 4. If not all branches complete, create instance in WAITING state
	if len(updatedExec.CompletedBranches) < len(updatedExec.ActiveBranches) {
		waitingInstance := &models.ProcessInstance{
			ProcessID:        existingInstance.ProcessID,
			EntityID:         existingInstance.EntityID,
			Action:           instance.Action,
			Status:           "WAITING_FOR_JOIN",
			Comment:          instance.Comment,
			Documents:        instance.Documents,
			Assignees:        existingInstance.Assignees,
			CurrentState:     joinState.ID,
			ParentInstanceID: existingInstance.ParentInstanceID,
			BranchID:         existingInstance.BranchID,
			IsParallelBranch: true,
			Attributes:       instance.Attributes,
			TenantID:         tenantID,
			Escalated:        s.isEscalationAction(instance.Action, instance.Comment),
		}
		waitingInstance.AuditDetails.SetAuditDetailsForCreate("system")

		if err := s.instanceRepo.CreateProcessInstance(ctx, waitingInstance); err != nil {
			return nil, fmt.Errorf("could not create waiting instance: %w", err)
		}

		fmt.Printf("⏳ DEBUG: Branch waiting for other branches to complete\n")
		return waitingInstance, nil
	}

	// 5. All branches complete - merge and continue as linear workflow
	fmt.Printf("🎯 DEBUG: All branches completed, merging parallel execution\n")
	return s.mergeParallelBranches(ctx, tenantID, updatedExec, targetAction, joinState, instance)
}

// mergeParallelBranches combines parallel branch data and creates a linear instance
func (s *transitionService) mergeParallelBranches(ctx context.Context, tenantID string, parallelExec *models.ParallelExecution, targetAction *models.Action, joinState *models.State, instance *models.ProcessInstance) (*models.ProcessInstance, error) {
	// 1. Get all parallel branch instances that reached the join
	parallelInstances, err := s.instanceRepo.GetActiveParallelInstances(ctx, tenantID, parallelExec.EntityID, parallelExec.ProcessID)
	if err != nil {
		return nil, fmt.Errorf("could not get parallel instances: %w", err)
	}

	// 2. Merge attributes, documents, assignees from all branches
	mergedAttributes := make(map[string][]string)
	var mergedDocuments []models.Document
	var mergedAssignees []string

	for _, inst := range parallelInstances {
		// Merge attributes
		for key, values := range inst.Attributes {
			mergedAttributes[key] = append(mergedAttributes[key], values...)
		}
		// Merge documents
		mergedDocuments = append(mergedDocuments, inst.Documents...)
		// Merge assignees
		mergedAssignees = append(mergedAssignees, inst.Assignees...)
	}

	// 3. Create merged linear instance
	mergedComment := fmt.Sprintf("Merged from parallel branches: %s", strings.Join(parallelExec.ActiveBranches, ", "))
	mergedInstance := &models.ProcessInstance{
		ProcessID:        parallelExec.ProcessID,
		EntityID:         parallelExec.EntityID,
		Action:           instance.Action,
		Status:           "ACTIVE",
		Comment:          &mergedComment,
		Documents:        mergedDocuments,
		Assignees:        removeDuplicates(mergedAssignees),
		CurrentState:     joinState.ID,
		ParentInstanceID: nil, // Back to linear execution
		BranchID:         nil,
		IsParallelBranch: false,
		Attributes:       mergedAttributes,
		TenantID:         tenantID,
		Escalated:        s.isEscalationAction(instance.Action, instance.Comment),
	}
	mergedInstance.AuditDetails.SetAuditDetailsForCreate("system")

	if err := s.instanceRepo.CreateProcessInstance(ctx, mergedInstance); err != nil {
		return nil, fmt.Errorf("could not create merged instance: %w", err)
	}

	// 4. Mark parallel execution as completed
	parallelExec.Status = "COMPLETED"
	parallelExec.AuditDetail.SetAuditDetailsForUpdate("system")
	if err := s.parallelRepo.UpdateParallelExecution(ctx, parallelExec); err != nil {
		return nil, fmt.Errorf("could not update parallel execution status: %w", err)
	}

	// 5. Add next actions
	nextActions, err := s.actionRepo.GetActionsByStateID(ctx, tenantID, mergedInstance.CurrentState, nil)
	if err == nil {
		mergedInstance.NextActions = make([]string, len(nextActions))
		for i, action := range nextActions {
			mergedInstance.NextActions[i] = action.Name
		}
	}

	fmt.Printf("🎉 DEBUG: Parallel branches merged successfully\n")
	return mergedInstance, nil
}

// Helper functions for parallel workflow

func (s *transitionService) findJoinStateForParallel(ctx context.Context, tenantID string, parallelState *models.State) (*models.State, error) {
	// For simplicity, assume the join state is the next state after parallel branches complete
	// In a more complex implementation, this could be configured or derived from process definition
	states, err := s.stateRepo.GetStatesByProcessID(ctx, tenantID, parallelState.ProcessID, nil)
	if err != nil {
		return nil, err
	}

	for _, state := range states {
		if state.IsJoin {
			return state, nil
		}
	}
	return nil, errors.New("no join state found for parallel state")
}

func (s *transitionService) findParallelExecutionForBranch(ctx context.Context, tenantID string, instance *models.ProcessInstance) (*models.ParallelExecution, error) {
	executions, err := s.parallelRepo.GetActiveParallelExecutions(ctx, tenantID, instance.EntityID, instance.ProcessID)
	if err != nil {
		return nil, err
	}

	if len(executions) == 0 {
		return nil, errors.New("no active parallel execution found for branch")
	}

	// Return the first active parallel execution (in practice there should only be one active at a time)
	return executions[0], nil
}

func removeDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	var result []string
	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}
	return result
}

func (s *transitionService) GetTransitions(ctx context.Context, tenantID string, criteria *models.ProcessInstanceSearchCriteria) ([]*models.ProcessInstance, error) {
	if criteria == nil {
		return nil, errors.New("search criteria is required")
	}

	filtersProvided := 0
	if criteria.EntityID != nil && *criteria.EntityID != "" {
		filtersProvided++
	}
	if criteria.CurrentStateID != nil && *criteria.CurrentStateID != "" {
		filtersProvided++
	}
	if criteria.AssigneeID != nil && *criteria.AssigneeID != "" {
		filtersProvided++
	}

	if filtersProvided == 0 {
		return nil, errors.New("provide either entityId, currentStateId, or assigneeId for transition search")
	}
	if filtersProvided > 1 {
		return nil, errors.New("specify only one of entityId, currentStateId, or assigneeId per request")
	}

	if criteria.EntityID != nil && *criteria.EntityID != "" {
		if criteria.ProcessID == nil || *criteria.ProcessID == "" {
			return nil, errors.New("processId is required when searching by entityId")
		}
		return s.instanceRepo.GetProcessInstancesByEntityID(ctx, tenantID, *criteria.EntityID, *criteria.ProcessID, criteria.History)
	}

	if criteria.CurrentStateID != nil && *criteria.CurrentStateID != "" {
		return s.instanceRepo.GetLatestInstancesByState(ctx, tenantID, *criteria.CurrentStateID, criteria.ProcessID)
	}

	if criteria.AssigneeID != nil && *criteria.AssigneeID != "" {
		return s.instanceRepo.GetLatestInstancesByAssignee(ctx, tenantID, *criteria.AssigneeID, criteria.ProcessID)
	}

	return nil, errors.New("unsupported search criteria")
}

func (s *transitionService) getOrCreateInstance(ctx context.Context, tenantID string, instance *models.ProcessInstance, currentState *string) (*models.ProcessInstance, error) {
	// First, check if we have an action that needs to be performed
	// If we have an action, check if it belongs to a parallel branch state
	if instance.Action != "" {
		// Get all active parallel instances to see if this action belongs to a parallel branch
		parallelInstances, err := s.instanceRepo.GetActiveParallelInstances(ctx, tenantID, instance.EntityID, instance.ProcessID)
		if err == nil && len(parallelInstances) > 0 {
			// Check each parallel instance to see if the action is valid for its current state
			for _, parallelInstance := range parallelInstances {
				actions, err := s.actionRepo.GetActionsByStateID(ctx, tenantID, parallelInstance.CurrentState, nil)
				if err == nil {
					for _, action := range actions {
						if action.Name == instance.Action {
							fmt.Printf("🎯 DEBUG: Found matching parallel branch instance for action '%s' in branch '%s'\n", instance.Action, *parallelInstance.BranchID)
							return parallelInstance, nil
						}
					}
				}
			}
		}
	}

	// Try to find latest existing non-parallel instance (linear workflow)
	existingInstance, err := s.instanceRepo.GetLatestProcessInstanceByEntityID(ctx, tenantID, instance.EntityID, instance.ProcessID)
	if err == nil {
		// Found latest instance, validate current state if provided
		if currentState != nil && existingInstance.CurrentState != *currentState {
			return nil, fmt.Errorf("current state mismatch: expected '%s', but entity is in state '%s'", *currentState, existingInstance.CurrentState)
		}
		// Found latest instance, return it as-is (we'll create new record for transition)
		return existingInstance, nil
	}

	// Entity doesn't exist - create a new instance in initial state
	// This allows both initialization and first transition in one API call

	// Get the initial state to create the instance
	states, err := s.stateRepo.GetStatesByProcessID(ctx, tenantID, instance.ProcessID, nil)
	if err != nil || len(states) == 0 {
		return nil, errors.New("cannot find any states for this process definition")
	}
	var initialState *models.State
	for _, state := range states {
		if state.IsInitial {
			initialState = state
			break
		}
	}
	if initialState == nil {
		return nil, errors.New("no initial state configured for this process")
	}

	// Create new instance in initial state (ready for transition)
	newInstance := &models.ProcessInstance{
		ProcessID:    instance.ProcessID,
		EntityID:     instance.EntityID,
		CurrentState: initialState.ID,
		Status:       "ACTIVE",
		TenantID:     tenantID,
		Attributes:   instance.Attributes, // Include attributes for validation
		Comment:      instance.Comment,    // Include comment
		Documents:    instance.Documents,  // Include documents
		Assigner:     instance.Assigner,   // Include assigner
		Assignees:    instance.Assignees,  // Include assignees
	}

	// Set audit details for new instance
	newInstance.AuditDetails.SetAuditDetailsForCreate("system")
	if err := s.instanceRepo.CreateProcessInstance(ctx, newInstance); err != nil {
		return nil, fmt.Errorf("failed to create new process instance: %w", err)
	}

	fmt.Printf("🆕 DEBUG: Created new process instance in initial state %s for entity %s\n", initialState.ID, instance.EntityID)
	return newInstance, nil
}

func (s *transitionService) buildFSM(ctx context.Context, tenantID string, instance *models.ProcessInstance, events fsm.Events) (*fsm.FSM, error) {
	return fsm.NewFSM(instance.CurrentState, events, fsm.Callbacks{}), nil
}

func eventsForState(actions []*models.Action) fsm.Events {
	var events fsm.Events
	for _, action := range actions {
		events = append(events, fsm.EventDesc{
			Name: action.Name,
			Src:  []string{action.CurrentState},
			Dst:  action.NextState,
		})
	}
	return events
}

// isEscalationAction determines if an action represents an auto-escalation
// Following the Java service pattern, we detect escalation by checking:
// 1. Action name contains "escalat" (case-insensitive)
// 2. Comment contains "auto-escalat" (case-insensitive)
func (s *transitionService) isEscalationAction(action string, comment *string) bool {
	// Check if action name indicates escalation
	actionLower := strings.ToLower(action)
	if strings.Contains(actionLower, "escalat") {
		return true
	}

	// Check if comment indicates auto-escalation
	if comment != nil {
		commentLower := strings.ToLower(*comment)
		if strings.Contains(commentLower, "auto-escalat") {
			return true
		}
	}

	return false
}

// createNewInstance creates a new process instance in the initial state
func (s *transitionService) createNewInstance(ctx context.Context, tenantID string, instance *models.ProcessInstance) (*models.ProcessInstance, error) {
	// Check if instance already exists - init should fail if entity already exists
	_, err := s.instanceRepo.GetLatestProcessInstanceByEntityID(ctx, tenantID, instance.EntityID, instance.ProcessID)
	if err == nil {
		return nil, fmt.Errorf("process instance already exists for entity '%s'. Use action-based transitions instead of init=true", instance.EntityID)
	}

	// Get the initial state
	states, err := s.stateRepo.GetStatesByProcessID(ctx, tenantID, instance.ProcessID, nil)
	if err != nil || len(states) == 0 {
		return nil, errors.New("cannot find any states for this process definition")
	}

	var initialState *models.State
	for _, state := range states {
		if state.IsInitial {
			initialState = state
			break
		}
	}
	if initialState == nil {
		return nil, errors.New("no initial state configured for this process")
	}

	// Create new instance in initial state
	newInstance := &models.ProcessInstance{
		ProcessID:    instance.ProcessID,
		EntityID:     instance.EntityID,
		CurrentState: initialState.ID,
		Status:       "ACTIVE",
		Comment:      instance.Comment,
		Documents:    instance.Documents,
		Assigner:     instance.Assigner,
		Assignees:    instance.Assignees,
		Attributes:   instance.Attributes,
		TenantID:     tenantID,
	}

	// Set audit details for new instance
	newInstance.AuditDetails.SetAuditDetailsForCreate("system")
	if err := s.instanceRepo.CreateProcessInstance(ctx, newInstance); err != nil {
		return nil, fmt.Errorf("failed to create new process instance: %w", err)
	}

	// Populate NextActions for the response
	nextActions, err := s.actionRepo.GetActionsByStateID(ctx, tenantID, newInstance.CurrentState, nil)
	if err != nil {
		newInstance.NextActions = []string{}
	} else {
		newInstance.NextActions = make([]string, len(nextActions))
		for i, actionItem := range nextActions {
			newInstance.NextActions[i] = actionItem.Name
		}
	}

	fmt.Printf("🆕 DEBUG: Created new process instance in initial state %s for entity %s\n", initialState.ID, instance.EntityID)
	return newInstance, nil
}

// performTransition executes an action on an existing process instance
func (s *transitionService) performTransition(ctx context.Context, tenantID string, instance *models.ProcessInstance, currentState *string) (*models.ProcessInstance, error) {
	// Find existing instance
	existingInstance, err := s.getOrCreateInstance(ctx, tenantID, instance, currentState)
	if err != nil {
		return nil, err
	}

	// Find the action being performed
	actions, err := s.actionRepo.GetActionsByStateID(ctx, tenantID, existingInstance.CurrentState, nil)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve actions for current state: %w", err)
	}

	var targetAction *models.Action
	for _, a := range actions {
		if a.Name == instance.Action {
			targetAction = a
			break
		}
	}
	if targetAction == nil {
		return nil, fmt.Errorf("action '%s' is not valid for the current state", instance.Action)
	}

	// Authorization Guard Check
	userID := "anonymous"
	var userRoles []string

	// Extract user information from context (for testing)
	if uid := ctx.Value("userID"); uid != nil {
		if uidStr, ok := uid.(string); ok {
			userID = uidStr
		}
	}
	if roles := ctx.Value("userRoles"); roles != nil {
		if rolesSlice, ok := roles.([]string); ok {
			userRoles = rolesSlice
		}
	}

	guardCtx := security.GuardContext{
		UserRoles:         userRoles,
		UserID:            userID,
		ProcessInstance:   existingInstance,
		Action:            targetAction,
		RequestAttributes: instance.Attributes,
	}

	can, err := s.guard.CanTransition(guardCtx)
	if err != nil {
		return nil, fmt.Errorf("guard check failed: %w", err)
	}
	if !can {
		return nil, errors.New("transition not permitted by guard")
	}

	// Check if transitioning TO a parallel or join state
	nextState, err := s.stateRepo.GetStateByID(ctx, tenantID, targetAction.NextState)
	if err != nil {
		return nil, fmt.Errorf("could not get next state: %w", err)
	}

	// Handle parallel workflow transitions
	if nextState.IsParallel {
		return s.handleParallelTransition(ctx, tenantID, existingInstance, targetAction, nextState, instance)
	}

	if nextState.IsJoin {
		return s.handleJoinTransition(ctx, tenantID, existingInstance, targetAction, nextState, instance)
	}

	// Standard linear transition logic
	return s.handleLinearTransition(ctx, tenantID, existingInstance, targetAction, instance)
}
