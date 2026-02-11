package service

import (
	"context"

	"digit.org/workflow/internal/models"
	"digit.org/workflow/internal/repository"
)

type processService struct {
	repo       repository.ProcessRepository
	stateRepo  repository.StateRepository
	actionRepo repository.ActionRepository
}

// NewProcessService creates a new instance of ProcessService.
func NewProcessService(repo repository.ProcessRepository, stateRepo repository.StateRepository, actionRepo repository.ActionRepository) ProcessService {
	return &processService{
		repo:       repo,
		stateRepo:  stateRepo,
		actionRepo: actionRepo,
	}
}

// CreateProcess handles the business logic for creating a new process.
func (s *processService) CreateProcess(ctx context.Context, process *models.Process) (*models.Process, error) {
	// Here you could add validation, enrichment, or other business logic.
	err := s.repo.CreateProcess(ctx, process)
	if err != nil {
		return nil, err
	}
	return process, nil
}

// GetProcessByID handles the business logic for retrieving a single process.
func (s *processService) GetProcessByID(ctx context.Context, tenantID, id string) (*models.Process, error) {
	return s.repo.GetProcessByID(ctx, tenantID, id)
}

// GetProcessByCode handles retrieving a single process by its code.
func (s *processService) GetProcessByCode(ctx context.Context, tenantID, code string) (*models.Process, error) {
	return s.repo.GetProcessByCode(ctx, tenantID, code)
}

// GetProcesses handles the business logic for searching for processes.
func (s *processService) GetProcesses(ctx context.Context, tenantID string, ids []string, names []string, codes []string) ([]*models.Process, error) {
	return s.repo.GetProcesses(ctx, tenantID, ids, names, codes)
}

func (s *processService) GetProcessDefinitions(ctx context.Context, tenantID string, ids []string, names []string) ([]*models.ProcessDefinitionDetail, error) {
	processes, err := s.repo.GetProcesses(ctx, tenantID, ids, names, nil)
	if err != nil {
		return nil, err
	}

	details := make([]*models.ProcessDefinitionDetail, 0, len(processes))
	for _, p := range processes {
		states, err := s.stateRepo.GetStatesByProcessID(ctx, tenantID, p.ID, nil)
		if err != nil {
			return nil, err
		}

		stateDetails := make([]models.StateDetail, 0, len(states))
		for _, st := range states {
			actions, err := s.actionRepo.GetActionsByStateID(ctx, tenantID, st.ID, nil)
			if err != nil {
				return nil, err
			}
			sd := models.StateDetail{
				State:   *st,
				Actions: make([]models.Action, 0, len(actions)),
			}
			for _, a := range actions {
				sd.Actions = append(sd.Actions, *a)
			}
			stateDetails = append(stateDetails, sd)
		}

		details = append(details, &models.ProcessDefinitionDetail{
			Process: *p,
			States:  stateDetails,
		})
	}
	return details, nil
}

// UpdateProcess handles the business logic for updating a process.
func (s *processService) UpdateProcess(ctx context.Context, process *models.Process) (*models.Process, error) {
	err := s.repo.UpdateProcess(ctx, process)
	if err != nil {
		return nil, err
	}
	return s.repo.GetProcessByID(ctx, process.TenantID, process.ID)
}

// DeleteProcess handles the business logic for deleting a process.
func (s *processService) DeleteProcess(ctx context.Context, tenantID, id string) error {
	return s.repo.DeleteProcess(ctx, tenantID, id)
}
