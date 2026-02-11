package postgres

import (
	"context"
	"fmt"
	"time"

	"digit.org/workflow/internal/models"
	"digit.org/workflow/internal/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type parallelExecutionRepository struct {
	db *gorm.DB
}

func NewParallelExecutionRepository(db *gorm.DB) repository.ParallelExecutionRepository {
	return &parallelExecutionRepository{db: db}
}

func (r *parallelExecutionRepository) CreateParallelExecution(ctx context.Context, execution *models.ParallelExecution) error {
	if execution.ID == "" {
		execution.ID = uuid.New().String()
	}

	return r.db.WithContext(ctx).Create(execution).Error
}

func (r *parallelExecutionRepository) GetParallelExecution(ctx context.Context, tenantID, entityID, processID, parallelStateID string) (*models.ParallelExecution, error) {
	var execution models.ParallelExecution
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND entity_id = ? AND process_id = ? AND parallel_state_id = ?",
			tenantID, entityID, processID, parallelStateID).
		First(&execution).Error
	if err != nil {
		return nil, err
	}
	return &execution, nil
}

func (r *parallelExecutionRepository) UpdateParallelExecution(ctx context.Context, execution *models.ParallelExecution) error {
	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", execution.TenantID, execution.ID).
		Updates(execution).Error
}

func (r *parallelExecutionRepository) MarkBranchCompleted(ctx context.Context, tenantID, entityID, processID, branchID string) error {
	// Use raw SQL for complex JSONB operations - GORM doesn't handle this well
	query := `UPDATE parallel_executions SET 
		completed_branches = completed_branches || ?::jsonb,
		modified_at = ?
		WHERE tenant_id = ? AND entity_id = ? AND process_id = ? AND status = 'ACTIVE'
		AND NOT (completed_branches ? ?)`

	branchJSON := fmt.Sprintf(`"%s"`, branchID) // JSON string format

	return r.db.WithContext(ctx).
		Exec(query, branchJSON, time.Now().UnixMilli(), tenantID, entityID, processID, branchID).Error
}

func (r *parallelExecutionRepository) GetActiveParallelExecutions(ctx context.Context, tenantID, entityID, processID string) ([]*models.ParallelExecution, error) {
	var executions []*models.ParallelExecution
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND entity_id = ? AND process_id = ? AND status = ?",
			tenantID, entityID, processID, "ACTIVE").
		Find(&executions).Error
	return executions, err
}
