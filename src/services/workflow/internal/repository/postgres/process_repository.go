package postgres

import (
	"context"
	"time"

	"digit.org/workflow/internal/models"
	"digit.org/workflow/internal/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type processRepository struct {
	db *gorm.DB
}

// NewProcessRepository creates a new instance of ProcessRepository.
func NewProcessRepository(db *gorm.DB) repository.ProcessRepository {
	return &processRepository{db: db}
}

// CreateProcess inserts a new process record into the database.
func (r *processRepository) CreateProcess(ctx context.Context, process *models.Process) error {
	process.ID = uuid.New().String()
	// Audit details should be set by handlers, only set time if not already set
	now := time.Now().UnixMilli()
	if process.AuditDetail.CreatedTime == 0 {
		process.AuditDetail.CreatedTime = now
	}
	if process.AuditDetail.ModifiedTime == 0 {
		process.AuditDetail.ModifiedTime = now
	}

	return r.db.WithContext(ctx).Create(process).Error
}

// GetProcessByID retrieves a single process by its ID.
func (r *processRepository) GetProcessByID(ctx context.Context, tenantID, id string) (*models.Process, error) {
	var process models.Process
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&process).Error
	if err != nil {
		return nil, err
	}
	return &process, nil
}

// GetProcessByCode retrieves a single process by its code.
func (r *processRepository) GetProcessByCode(ctx context.Context, tenantID, code string) (*models.Process, error) {
	var process models.Process
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND code = ?", tenantID, code).First(&process).Error
	if err != nil {
		return nil, err
	}
	return &process, nil
}

// GetProcesses retrieves a list of processes based on filter criteria.
func (r *processRepository) GetProcesses(ctx context.Context, tenantID string, ids []string, names []string, codes []string) ([]*models.Process, error) {
	var processes []*models.Process

	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)

	if len(ids) > 0 {
		query = query.Where("id IN ?", ids)
	}
	if len(names) > 0 {
		query = query.Where("name IN ?", names)
	}
	if len(codes) > 0 {
		query = query.Where("code IN ?", codes)
	}

	err := query.Find(&processes).Error
	return processes, err
}

// UpdateProcess updates an existing process record in the database.
func (r *processRepository) UpdateProcess(ctx context.Context, process *models.Process) error {
	process.AuditDetail.ModifiedTime = time.Now().UnixMilli()

	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", process.TenantID, process.ID).
		Updates(process).Error
}

// DeleteProcess removes a process record from the database.
func (r *processRepository) DeleteProcess(ctx context.Context, tenantID, id string) error {
	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		Delete(&models.Process{}).Error
}
