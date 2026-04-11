package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"registry-service/internal/models"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RegistryRepository interface {
	// Schema operations
	CreateSchema(schema *models.Schema) error
	GetSchema(tenantID, schemaCode string) (*models.Schema, error)
	GetSchemaByVersion(tenantID, schemaCode string, version int) (*models.Schema, error)
	MarkSchemasNotLatest(tenantID, schemaCode string) error
	UpdateSchema(schema *models.Schema) error
	DeleteSchema(tenantID, schemaCode string) error
	ListSchemas(tenantID string) ([]models.Schema, error)
	EnsureDataTable(tenantID, schemaCode string, indexes []models.SchemaIndex) (string, error)

	// Data operations
	CreateData(data *models.RegistryData) error
	CreateDataVersion(previous *models.RegistryData, next *models.RegistryData) error
	GetData(tenantID, schemaCode, id string) (*models.RegistryData, error)
	DeleteData(tenantID, schemaCode, id string) error
	SearchData(request *models.SearchRequest) ([]models.RegistryData, error)
	ExistsMatchingFields(tenantID, schemaCode string, fields map[string]string, excludeID *uuid.UUID) (bool, error)
	ExistsData(tenantID, schemaCode, id string) (bool, error)
	FieldExists(tenantID, schemaCode, field, value string) (bool, error)
	ListRegistryData(tenantID, schemaCode, registryID string, includeHistory bool) ([]models.RegistryData, error)
	CreateAuditLog(entry *models.AuditLog) error
	GetLatestAuditLog(tenantID, subjectType, schemaCode string, recordID *uuid.UUID) (*models.AuditLog, error)
}

type registryRepository struct {
	db *gorm.DB
}

func NewRegistryRepository(db *gorm.DB) RegistryRepository {
	return &registryRepository{db: db}
}

func (r *registryRepository) createVersionedRecord(data *models.RegistryData, previousID *uuid.UUID) error {
	tableName, err := r.tableNameForSchema(data.TenantID, data.SchemaCode)
	if err != nil {
		return err
	}

	tx := r.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	if previousID != nil {
		if err := tx.Table(tableName).
			Where("id = ?", *previousID).
			Updates(map[string]interface{}{
				"is_active":    false,
				"effective_to": data.EffectiveFrom,
				"updated_at":   time.Now().UTC(),
			}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Table(tableName).Create(data).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// Schema operations
func (r *registryRepository) CreateSchema(schema *models.Schema) error {
	return r.db.Create(schema).Error
}

func (r *registryRepository) GetSchema(tenantID, schemaCode string) (*models.Schema, error) {
	var schema models.Schema
	err := r.db.Where("tenant_id = ? AND schema_code = ? AND is_active = ? AND is_latest = ?",
		tenantID, schemaCode, true, true).
		Order("created_at DESC").First(&schema).Error
	if err != nil {
		return nil, err
	}
	return &schema, nil
}

func (r *registryRepository) GetSchemaByVersion(tenantID, schemaCode string, version int) (*models.Schema, error) {
	var schema models.Schema
	err := r.db.Where("tenant_id = ? AND schema_code = ? AND version = ? AND is_active = ?",
		tenantID, schemaCode, version, true).First(&schema).Error
	if err != nil {
		return nil, err
	}
	return &schema, nil
}

func (r *registryRepository) MarkSchemasNotLatest(tenantID, schemaCode string) error {
	return r.db.Model(&models.Schema{}).
		Where("tenant_id = ? AND schema_code = ?", tenantID, schemaCode).
		Update("is_latest", false).Error
}

func (r *registryRepository) UpdateSchema(schema *models.Schema) error {
	return r.db.Save(schema).Error
}

func (r *registryRepository) DeleteSchema(tenantID, schemaCode string) error {
	return r.db.Model(&models.Schema{}).
		Where("tenant_id = ? AND schema_code = ?", tenantID, schemaCode).
		Updates(map[string]interface{}{
			"is_active": false,
			"is_latest": false,
		}).Error
}

func (r *registryRepository) ListSchemas(tenantID string) ([]models.Schema, error) {
	var schemas []models.Schema
	err := r.db.Where("tenant_id = ? AND is_active = ?", tenantID, true).Order("schema_code, created_at DESC").Find(&schemas).Error
	return schemas, err
}

// Data operations
func (r *registryRepository) CreateData(data *models.RegistryData) error {
	return r.createVersionedRecord(data, nil)
}

func (r *registryRepository) CreateDataVersion(previous *models.RegistryData, next *models.RegistryData) error {
	if previous == nil {
		return fmt.Errorf("previous version is required")
	}
	return r.createVersionedRecord(next, &previous.ID)
}

func (r *registryRepository) GetData(tenantID, schemaCode, id string) (*models.RegistryData, error) {
	var data models.RegistryData
	var err error

	tableName, tblErr := r.tableNameForSchema(tenantID, schemaCode)
	if tblErr != nil {
		return nil, tblErr
	}

	query := r.db.Table(tableName)
	if _, parseErr := uuid.Parse(id); parseErr == nil {
		err = query.Where("tenant_id = ? AND schema_code = ? AND id = ?",
			tenantID, schemaCode, id).First(&data).Error
	} else {
		err = query.Where("tenant_id = ? AND schema_code = ? AND registry_id = ? AND is_active = ?",
			tenantID, schemaCode, id, true).
			Order("version DESC").
			First(&data).Error
	}

	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *registryRepository) DeleteData(tenantID, schemaCode, id string) error {
	tableName, err := r.tableNameForSchema(tenantID, schemaCode)
	if err != nil {
		return err
	}

	query := r.db.Table(tableName).
		Where("tenant_id = ? AND schema_code = ? AND is_active = ?", tenantID, schemaCode, true)

	if _, parseErr := uuid.Parse(id); parseErr == nil {
		query = query.Where("id = ?", id)
	} else {
		query = query.Where("registry_id = ?", id)
	}

	now := time.Now().UTC()
	return query.Updates(map[string]interface{}{
		"is_active":    false,
		"effective_to": now,
		"updated_at":   now,
	}).Error
}

func (r *registryRepository) SearchData(request *models.SearchRequest) ([]models.RegistryData, error) {
	tableName, err := r.tableNameForSchema(request.TenantID, request.SchemaCode)
	if err != nil {
		return nil, err
	}

	query := r.db.Table(tableName).Where("tenant_id = ? AND schema_code = ? AND is_active = ?",
		request.TenantID, request.SchemaCode, true)

	// Apply JSON filters
	if len(request.Filters) > 0 {
		for key, value := range request.Filters {
			jsonPath := fmt.Sprintf("data->>'%s'", key)
			if valueStr, ok := value.(string); ok {
				query = query.Where(fmt.Sprintf("%s = ?", jsonPath), valueStr)
			} else {
				// For non-string values, convert to JSON and use JSON operators
				valueBytes, _ := json.Marshal(value)
				query = query.Where(fmt.Sprintf("data->'%s' = ?", key), string(valueBytes))
			}
		}
	}

	if len(request.Contains) > 0 {
		containsBytes, err := json.Marshal(request.Contains)
		if err != nil {
			return nil, err
		}
		query = query.Where("data @> ?", containsBytes)
	}

	// Apply pagination
	if request.Limit > 0 {
		query = query.Limit(request.Limit)
	}
	if request.Offset > 0 {
		query = query.Offset(request.Offset)
	}

	var data []models.RegistryData
	if err := query.Find(&data).Error; err != nil {
		return nil, err
	}
	return data, nil
}

func (r *registryRepository) ExistsMatchingFields(tenantID, schemaCode string, fields map[string]string, excludeID *uuid.UUID) (bool, error) {
	if len(fields) == 0 {
		return false, nil
	}

	tableName, err := r.tableNameForSchema(tenantID, schemaCode)
	if err != nil {
		return false, err
	}

	query := r.db.Table(tableName).
		Where("tenant_id = ? AND schema_code = ? AND is_active = ?", tenantID, schemaCode, true)

	for path, value := range fields {
		selector := jsonSelector(path)
		query = query.Where(fmt.Sprintf("%s = ?", selector), value)
	}

	if excludeID != nil {
		query = query.Where("id <> ?", *excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *registryRepository) ExistsData(tenantID, schemaCode, id string) (bool, error) {
	tableName, err := r.tableNameForSchema(tenantID, schemaCode)
	if err != nil {
		return false, err
	}

	query := r.db.Table(tableName).
		Where("tenant_id = ? AND schema_code = ?", tenantID, schemaCode)
	if _, parseErr := uuid.Parse(id); parseErr == nil {
		query = query.Where("id = ? AND is_active = ?", id, true)
	} else {
		query = query.Where("registry_id = ? AND is_active = ?", id, true)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *registryRepository) FieldExists(tenantID, schemaCode, field, value string) (bool, error) {
	tableName, err := r.tableNameForSchema(tenantID, schemaCode)
	if err != nil {
		return false, err
	}

	query := r.db.Table(tableName).
		Where("tenant_id = ? AND schema_code = ? AND is_active = ?", tenantID, schemaCode, true)

	switch strings.TrimSpace(field) {
	case "", "registryId":
		query = query.Where("registry_id = ?", value)
	case "id":
		query = query.Where("id = ?", value)
	default:
		selector := jsonSelector(field)
		query = query.Where(fmt.Sprintf("%s = ?", selector), value)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *registryRepository) CreateAuditLog(entry *models.AuditLog) error {
	return r.db.Create(entry).Error
}

func (r *registryRepository) ListRegistryData(tenantID, schemaCode, registryID string, includeHistory bool) ([]models.RegistryData, error) {
	tableName, err := r.tableNameForSchema(tenantID, schemaCode)
	if err != nil {
		return nil, err
	}

	query := r.db.Table(tableName).
		Where("tenant_id = ? AND schema_code = ? AND registry_id = ?", tenantID, schemaCode, registryID).
		Order("version DESC")
	if !includeHistory {
		query = query.Where("is_active = ?", true).Limit(1)
	}

	var data []models.RegistryData
	if err := query.Find(&data).Error; err != nil {
		return nil, err
	}
	return data, nil
}

func (r *registryRepository) GetLatestAuditLog(tenantID, subjectType, schemaCode string, recordID *uuid.UUID) (*models.AuditLog, error) {
	query := r.db.Where("tenant_id = ? AND subject_type = ?", tenantID, subjectType)

	if strings.TrimSpace(schemaCode) != "" {
		query = query.Where("schema_code = ?", schemaCode)
	}

	if recordID != nil {
		query = query.Where("record_id = ?", *recordID)
	}

	var log models.AuditLog
	if err := query.Order("created_at DESC").First(&log).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &log, nil
}

func jsonSelector(path string) string {
	if strings.TrimSpace(path) == "" {
		return "data::text"
	}
	parts := strings.Split(path, ".")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return fmt.Sprintf("data #>> '{%s}'", strings.Join(parts, ","))
}
