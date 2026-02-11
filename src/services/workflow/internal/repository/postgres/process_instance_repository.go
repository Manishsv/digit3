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

type processInstanceRepository struct {
	db *gorm.DB
}

func NewProcessInstanceRepository(db *gorm.DB) repository.ProcessInstanceRepository {
	return &processInstanceRepository{db: db}
}

func generateUUID() string {
	return uuid.New().String()
}

func (r *processInstanceRepository) CreateProcessInstance(ctx context.Context, instance *models.ProcessInstance) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if instance.ID == "" {
			instance.ID = uuid.New().String()
		}

		// Mark previous latest instance as not latest before inserting the new record
		if instance.IsParallelBranch {
			if instance.BranchID != nil {
				if err := tx.Model(&models.ProcessInstance{}).
					Where("tenant_id = ? AND entity_id = ? AND process_id = ? AND is_parallel_branch = TRUE AND branch_id = ? AND is_latest = TRUE",
						instance.TenantID, instance.EntityID, instance.ProcessID, *instance.BranchID).
					Update("is_latest", false).Error; err != nil {
					return err
				}
			}
		} else {
			if err := tx.Model(&models.ProcessInstance{}).
				Where("tenant_id = ? AND entity_id = ? AND process_id = ? AND (is_parallel_branch = FALSE OR is_parallel_branch IS NULL) AND is_latest = TRUE",
					instance.TenantID, instance.EntityID, instance.ProcessID).
				Update("is_latest", false).Error; err != nil {
				return err
			}
		}

		instance.IsLatest = true
		return tx.Create(instance).Error
	})
}

func (r *processInstanceRepository) GetProcessInstanceByID(ctx context.Context, tenantID, id string) (*models.ProcessInstance, error) {
	var instance models.ProcessInstance
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		First(&instance).Error
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

func (r *processInstanceRepository) GetProcessInstanceByEntityID(ctx context.Context, tenantID, entityID, processID string) (*models.ProcessInstance, error) {
	var instance models.ProcessInstance
	fmt.Printf("🔍 SQL[GetProcessInstanceByEntityID]: SELECT * FROM process_instances WHERE tenant_id = $1 AND entity_id = $2 AND process_id = $3 AND is_latest = TRUE AND (is_parallel_branch = FALSE OR is_parallel_branch IS NULL); ARGS: [%s %s %s]\n", tenantID, entityID, processID)
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND entity_id = ? AND process_id = ? AND is_latest = TRUE AND (is_parallel_branch = FALSE OR is_parallel_branch IS NULL)",
			tenantID, entityID, processID).
		First(&instance).Error
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

func (r *processInstanceRepository) GetLatestProcessInstanceByEntityID(ctx context.Context, tenantID, entityID, processID string) (*models.ProcessInstance, error) {
	var instance models.ProcessInstance
	fmt.Printf("🔍 REPO DEBUG: Getting latest instance for entityID=%s, processID=%s\n", entityID, processID)
	fmt.Printf("🔍 SQL[GetLatestProcessInstanceByEntityID]: SELECT * FROM process_instances WHERE tenant_id = $1 AND entity_id = $2 AND process_id = $3 AND is_latest = TRUE AND (is_parallel_branch = FALSE OR is_parallel_branch IS NULL) LIMIT 1; ARGS: [%s %s %s]\n", tenantID, entityID, processID)

	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND entity_id = ? AND process_id = ? AND is_latest = TRUE AND (is_parallel_branch = FALSE OR is_parallel_branch IS NULL)",
			tenantID, entityID, processID).
		First(&instance).Error

	if err == nil {
		fmt.Printf("✅ REPO DEBUG: Found latest instance ID=%s, currentState=%s, action=%s, createdAt=%d\n",
			instance.ID, instance.CurrentState, instance.Action, instance.AuditDetails.CreatedTime)
	} else {
		fmt.Printf("❌ REPO DEBUG: Error getting latest instance: %v\n", err)
	}

	if err != nil {
		return nil, err
	}
	return &instance, nil
}

func (r *processInstanceRepository) GetProcessInstancesByEntityID(ctx context.Context, tenantID, entityID, processID string, history bool) ([]*models.ProcessInstance, error) {
	var instances []*models.ProcessInstance

	query := r.db.WithContext(ctx).
		Where("tenant_id = ? AND entity_id = ? AND process_id = ?", tenantID, entityID, processID)

	if history {
		// Return all records ordered by created_at (oldest first for chronological order)
		fmt.Printf("🔍 SQL[GetProcessInstancesByEntityID-history]: SELECT * FROM process_instances WHERE tenant_id = $1 AND entity_id = $2 AND process_id = $3 ORDER BY created_at ASC; ARGS: [%s %s %s]\n", tenantID, entityID, processID)
		query = query.Order("created_at ASC")
	} else {
		// Return only the latest linear instance
		fmt.Printf("🔍 SQL[GetProcessInstancesByEntityID-latest]: SELECT * FROM process_instances WHERE tenant_id = $1 AND entity_id = $2 AND process_id = $3 AND is_latest = TRUE AND (is_parallel_branch = FALSE OR is_parallel_branch IS NULL); ARGS: [%s %s %s]\n", tenantID, entityID, processID)
		query = query.
			Where("is_latest = TRUE AND (is_parallel_branch = FALSE OR is_parallel_branch IS NULL)")
	}

	err := query.Find(&instances).Error
	return instances, err
}

func (r *processInstanceRepository) GetLatestInstancesByState(ctx context.Context, tenantID, currentStateID string, processID *string) ([]*models.ProcessInstance, error) {
	var instances []*models.ProcessInstance
	if processID != nil && *processID != "" {
		fmt.Printf("🔍 SQL[GetLatestInstancesByState]: SELECT * FROM process_instances WHERE tenant_id = $1 AND current_state_id = $2 AND process_id = $3 AND is_latest = TRUE AND (is_parallel_branch = FALSE OR is_parallel_branch IS NULL) ORDER BY created_at DESC; ARGS: [%s %s %s]\n", tenantID, currentStateID, *processID)
	} else {
		fmt.Printf("🔍 SQL[GetLatestInstancesByState]: SELECT * FROM process_instances WHERE tenant_id = $1 AND current_state_id = $2 AND is_latest = TRUE AND (is_parallel_branch = FALSE OR is_parallel_branch IS NULL) ORDER BY created_at DESC; ARGS: [%s %s]\n", tenantID, currentStateID)
	}

	query := r.db.WithContext(ctx).
		Where(
			"tenant_id = ? AND current_state_id = ? AND is_latest = TRUE AND (is_parallel_branch = FALSE OR is_parallel_branch IS NULL)",
			tenantID, currentStateID,
		)

	if processID != nil && *processID != "" {
		query = query.Where("process_id = ?", *processID)
	}

	err := query.Order("created_at DESC").Find(&instances).Error
	return instances, err
}

func (r *processInstanceRepository) GetLatestInstancesByAssignee(ctx context.Context, tenantID, assignee string, processID *string) ([]*models.ProcessInstance, error) {
	var instances []*models.ProcessInstance
	if processID != nil && *processID != "" {
		fmt.Printf("🔍 SQL[GetLatestInstancesByAssignee]: SELECT * FROM process_instances WHERE tenant_id = $1 AND process_id = $2 AND assignees @> to_jsonb(array[$3]::text[]) AND is_latest = TRUE AND (is_parallel_branch = FALSE OR is_parallel_branch IS NULL) ORDER BY created_at DESC; ARGS: [%s %s %s]\n", tenantID, *processID, assignee)
	} else {
		fmt.Printf("🔍 SQL[GetLatestInstancesByAssignee]: SELECT * FROM process_instances WHERE tenant_id = $1 AND assignees @> to_jsonb(array[$2]::text[]) AND is_latest = TRUE AND (is_parallel_branch = FALSE OR is_parallel_branch IS NULL) ORDER BY created_at DESC; ARGS: [%s %s]\n", tenantID, assignee)
	}

	query := r.db.WithContext(ctx).
		Where(
			"tenant_id = ? AND is_latest = TRUE AND (is_parallel_branch = FALSE OR is_parallel_branch IS NULL)",
			tenantID,
		).
		Where("assignees @> to_jsonb(array[?]::text[])", assignee)

	if processID != nil && *processID != "" {
		query = query.Where("process_id = ?", *processID)
	}

	err := query.Order("created_at DESC").Find(&instances).Error
	return instances, err
}

func (r *processInstanceRepository) UpdateProcessInstance(ctx context.Context, instance *models.ProcessInstance) error {
	// Update audit details
	now := time.Now().UnixMilli()
	instance.AuditDetails.ModifiedTime = now
	if instance.AuditDetails.ModifiedBy == "" {
		instance.AuditDetails.ModifiedBy = "system"
	}

	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", instance.TenantID, instance.ID).
		Updates(instance).Error
}

// GetActiveParallelInstances returns all active parallel branch instances for an entity
func (r *processInstanceRepository) GetActiveParallelInstances(ctx context.Context, tenantID, entityID, processID string) ([]*models.ProcessInstance, error) {
	var instances []*models.ProcessInstance
	fmt.Printf("🔍 SQL[GetActiveParallelInstances]: SELECT * FROM process_instances WHERE tenant_id = $1 AND entity_id = $2 AND process_id = $3 AND is_parallel_branch = TRUE AND status != 'COMPLETED' AND is_latest = TRUE ORDER BY created_at DESC; ARGS: [%s %s %s]\n", tenantID, entityID, processID)
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND entity_id = ? AND process_id = ? AND is_parallel_branch = true AND status != ? AND is_latest = TRUE",
			tenantID, entityID, processID, "COMPLETED").
		Order("created_at DESC").
		Find(&instances).Error
	return instances, err
}

// GetInstancesByBranch returns instances for a specific parallel branch
func (r *processInstanceRepository) GetInstancesByBranch(ctx context.Context, tenantID, entityID, processID, branchID string) ([]*models.ProcessInstance, error) {
	var instances []*models.ProcessInstance
	fmt.Printf("🔍 SQL[GetInstancesByBranch]: SELECT * FROM process_instances WHERE tenant_id = $1 AND entity_id = $2 AND process_id = $3 AND branch_id = $4 ORDER BY created_at DESC; ARGS: [%s %s %s %s]\n", tenantID, entityID, processID, branchID)
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND entity_id = ? AND process_id = ? AND branch_id = ?",
			tenantID, entityID, processID, branchID).
		Order("created_at DESC").
		Find(&instances).Error
	return instances, err
}

// GetSLABreachedInstances retrieves process instances that have breached SLA thresholds
func (r *processInstanceRepository) GetSLABreachedInstances(ctx context.Context, tenantID, processID, stateCode string, stateSlaMinutes, processSlaMinutes *int) ([]*models.ProcessInstance, error) {
	// Use raw SQL for complex subquery with window functions - GORM doesn't handle this well
	currentTimeMillis := time.Now().UnixMilli()

	query := `
		SELECT pi.id, pi.tenant_id, pi.process_id, pi.entity_id, pi.action, pi.status,
			   pi.current_state_id, pi.documents, pi.assignees, pi.attributes, pi.comment,
			   pi.state_sla, pi.process_sla, pi.parent_instance_id, pi.branch_id, pi.is_parallel_branch,
			   pi.escalated, pi.created_by, pi.created_at, pi.modified_by, pi.modified_at
		FROM process_instances pi
		WHERE pi.tenant_id = ?
			AND pi.process_id = ?
			AND pi.current_state_id = ?
			AND pi.is_latest = TRUE
			AND (pi.is_parallel_branch = FALSE OR pi.is_parallel_branch IS NULL)`

	args := []interface{}{tenantID, processID, stateCode}

	// Add SLA breach conditions
	if stateSlaMinutes != nil && *stateSlaMinutes > 0 {
		query += fmt.Sprintf(" AND (%d - pi.created_at) > (? * 60 * 1000)", currentTimeMillis)
		args = append(args, *stateSlaMinutes)
	}

	if processSlaMinutes != nil && *processSlaMinutes > 0 {
		query += fmt.Sprintf(" AND (%d - pi.created_at) > (? * 60 * 1000)", currentTimeMillis)
		args = append(args, *processSlaMinutes)
	}

	query += " ORDER BY pi.created_at DESC"
	fmt.Printf("🔍 SQL[GetSLABreachedInstances]: %s; ARGS: %v\n", query, args)

	var instances []*models.ProcessInstance
	err := r.db.WithContext(ctx).Raw(query, args...).Scan(&instances).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query SLA breached instances: %w", err)
	}

	return instances, nil
}

// GetEscalatedInstances retrieves process instances that have been auto-escalated
// Following the Java service pattern - this searches for instances with escalated = true
func (r *processInstanceRepository) GetEscalatedInstances(ctx context.Context, tenantID, processID string, limit, offset int) ([]*models.ProcessInstance, error) {
	// Use raw SQL for complex window function query - GORM doesn't handle this well
	query := `
		SELECT pi.id, pi.tenant_id, pi.process_id, pi.entity_id, pi.action, pi.status,
			   pi.current_state_id, pi.documents, pi.assignees, pi.attributes, pi.comment,
			   pi.state_sla, pi.process_sla, pi.parent_instance_id, pi.branch_id, pi.is_parallel_branch,
			   pi.escalated, pi.created_by, pi.created_at, pi.modified_by, pi.modified_at
		FROM process_instances pi
		WHERE pi.tenant_id = ?
			AND pi.escalated = TRUE
			AND pi.is_latest = TRUE
			AND (pi.is_parallel_branch = FALSE OR pi.is_parallel_branch IS NULL)`

	args := []interface{}{tenantID}

	if processID != "" {
		query += " AND pi.process_id = ?"
		args = append(args, processID)
	}

	query += " ORDER BY pi.created_at DESC"
	fmt.Printf("🔍 SQL[GetEscalatedInstances]: %s; ARGS: %v\n", query, args)

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	if offset > 0 {
		query += " OFFSET ?"
		args = append(args, offset)
	}

	var instances []*models.ProcessInstance
	err := r.db.WithContext(ctx).Raw(query, args...).Scan(&instances).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query escalated instances: %w", err)
	}

	return instances, nil
}
