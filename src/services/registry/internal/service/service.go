package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"registry-service/internal/audit"
	"registry-service/internal/clients"
	"registry-service/internal/idgen"
	"registry-service/internal/models"
	"registry-service/internal/repository"
	"registry-service/internal/utils"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	// ErrAuditLogNotFound indicates that no audit logs exist for the record being verified.
	ErrAuditLogNotFound = errors.New("audit log not found")
	// ErrSignatureVerificationUnavailable indicates signing has not been configured so verification cannot proceed.
	ErrSignatureVerificationUnavailable = errors.New("signature verification unavailable")
)

type RegistryService interface {
	// Schema operations
	CreateSchema(request *models.SchemaRequest, clientID string) (*models.Schema, error)
	GetSchema(tenantID, schemaCode string) (*models.Schema, error)
	GetSchemaVersion(tenantID, schemaCode string, version int) (*models.Schema, error)
	UpdateSchema(tenantID, schemaCode string, request *models.SchemaRequest, clientID string) (*models.Schema, error)
	DeleteSchema(tenantID, schemaCode, clientID string) error
	ListSchemas(tenantID string) ([]models.Schema, error)

	// Data operations
	CreateData(schemaCode string, request *models.DataRequest, clientID string, callback *models.CallbackConfig) (*models.RegistryData, error)
	GetData(tenantID, schemaCode, id string) (*models.RegistryData, error)
	UpdateData(tenantID, schemaCode, id string, request *models.DataRequest, clientID string, callback *models.CallbackConfig) (*models.RegistryData, error)
	DeleteData(tenantID, schemaCode, id, clientID string, callback *models.CallbackConfig) error
	SearchData(request *models.SearchRequest) ([]models.RegistryData, error)
	DataExists(tenantID, schemaCode, id string) (bool, error)
	GetRegistryData(tenantID, schemaCode, registryID string, history bool) ([]models.RegistryData, error)
	VerifyDataSignature(tenantID, schemaCode, identifier string) (bool, error)
	FieldExists(tenantID, schemaCode, field, value string) (bool, error)
}

type registryService struct {
	repo             repository.RegistryRepository
	validator        *utils.SchemaValidator
	callbackManager  *utils.CallbackManager
	auditor          audit.Auditor
	idGenerator      idgen.Generator
	externalResolver clients.ExternalRegistryResolver
}

func NewRegistryService(repo repository.RegistryRepository, auditor audit.Auditor, generator idgen.Generator, resolver clients.ExternalRegistryResolver) RegistryService {
	if auditor == nil {
		auditor = audit.NewNoopAuditor()
	}
	if generator == nil {
		generator = idgen.FallbackGenerator{}
	}
	return &registryService{
		repo:             repo,
		validator:        utils.NewSchemaValidator(),
		callbackManager:  utils.NewCallbackManager(),
		auditor:          auditor,
		idGenerator:      generator,
		externalResolver: resolver,
	}
}

// Schema operations
func (s *registryService) CreateSchema(request *models.SchemaRequest, clientID string) (*models.Schema, error) {
	// Validate schema definition
	if err := s.validator.ValidateSchema(request.Definition); err != nil {
		return nil, fmt.Errorf("schema validation failed: %w", err)
	}

	existing, err := s.repo.GetSchema(request.TenantID, request.SchemaCode)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("schema with code '%s' already exists for tenant '%s'", request.SchemaCode, request.TenantID)
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to verify existing schema: %w", err)
	}

	cleanDefinition, uniqueConstraints, refConstraints, indexDefinitions, webhookConfig, err := resolveSchemaConfig(request.Definition, request.XUnique, request.XRefSchema, request.XIndexes, request.Webhook)
	if err != nil {
		return nil, fmt.Errorf("failed to process schema configuration: %w", err)
	}

	schema := &models.Schema{
		ID:         uuid.New(),
		TenantID:   request.TenantID,
		SchemaCode: request.SchemaCode,
		Version:    1,
		Definition: cleanDefinition,
		XUnique:    uniqueConstraints,
		XRefSchema: refConstraints,
		XIndexes:   indexDefinitions,
		Webhook:    webhookConfig,
		IsLatest:   true,
		IsActive:   true,
		CreatedBy:  clientID,
		UpdatedBy:  clientID,
	}

	if err := s.repo.CreateSchema(schema); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	if _, err := s.repo.EnsureDataTable(schema.TenantID, schema.SchemaCode, schema.XIndexes); err != nil {
		return nil, fmt.Errorf("failed to provision data table: %w", err)
	}

	if err := s.auditor.LogSchemaEvent(context.Background(), audit.SchemaEvent{
		TenantID:      schema.TenantID,
		SchemaCode:    schema.SchemaCode,
		SchemaVersion: schema.Version,
		Operation:     audit.OperationCreate,
		Actor:         clientID,
		Definition:    schema.Definition,
		Unique:        schema.XUnique,
		References:    schema.XRefSchema,
		Webhook:       schema.Webhook,
		Timestamp:     schema.CreatedAt,
	}); err != nil {
		return nil, err
	}

	s.schemaWithAuditDetails(schema)
	return schema, nil
}

func (s *registryService) GetSchema(tenantID, schemaCode string) (*models.Schema, error) {
	schema, err := s.repo.GetSchema(tenantID, schemaCode)
	if err != nil {
		return nil, err
	}
	s.schemaWithAuditDetails(schema)
	return schema, nil
}

func (s *registryService) GetSchemaVersion(tenantID, schemaCode string, version int) (*models.Schema, error) {
	schema, err := s.repo.GetSchemaByVersion(tenantID, schemaCode, version)
	if err != nil {
		return nil, err
	}
	s.schemaWithAuditDetails(schema)
	return schema, nil
}

func (s *registryService) UpdateSchema(tenantID, schemaCode string, request *models.SchemaRequest, clientID string) (*models.Schema, error) {
	// Validate schema definition
	if err := s.validator.ValidateSchema(request.Definition); err != nil {
		return nil, fmt.Errorf("schema validation failed: %w", err)
	}

	// Get existing latest schema
	currentSchema, err := s.repo.GetSchema(tenantID, schemaCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("schema not found: %w", err)
		}
		return nil, fmt.Errorf("failed to retrieve schema: %w", err)
	}

	cleanDefinition, uniqueConstraints, refConstraints, indexDefinitions, webhookConfig, err := resolveSchemaConfig(request.Definition, request.XUnique, request.XRefSchema, request.XIndexes, request.Webhook)
	if err != nil {
		return nil, fmt.Errorf("failed to process schema configuration: %w", err)
	}

	sameDefinition, err := schemasEqual(currentSchema.Definition, cleanDefinition)
	if err != nil {
		return nil, fmt.Errorf("failed to compare schema definitions: %w", err)
	}
	if sameDefinition && uniqueConstraintsEqual(currentSchema.XUnique, uniqueConstraints) && refConstraintsEqual(currentSchema.XRefSchema, refConstraints) && indexesEqual(currentSchema.XIndexes, indexDefinitions) && webhookEqual(currentSchema.Webhook, webhookConfig) {
		return currentSchema, nil
	}

	newVersion, err := s.nextSchemaVersion(tenantID, schemaCode, currentSchema.Version)
	if err != nil {
		return nil, err
	}

	if err := s.repo.MarkSchemasNotLatest(tenantID, schemaCode); err != nil {
		return nil, fmt.Errorf("failed to mark existing schema versions as not latest: %w", err)
	}

	newSchema := &models.Schema{
		ID:         uuid.New(),
		TenantID:   tenantID,
		SchemaCode: schemaCode,
		Version:    newVersion,
		Definition: cleanDefinition,
		XUnique:    uniqueConstraints,
		XRefSchema: refConstraints,
		XIndexes:   indexDefinitions,
		Webhook:    webhookConfig,
		IsLatest:   true,
		IsActive:   true,
		CreatedBy:  clientID,
		UpdatedBy:  clientID,
	}

	if err := s.repo.CreateSchema(newSchema); err != nil {
		return nil, fmt.Errorf("failed to create new schema version: %w", err)
	}

	if _, err := s.repo.EnsureDataTable(newSchema.TenantID, newSchema.SchemaCode, newSchema.XIndexes); err != nil {
		return nil, fmt.Errorf("failed to ensure data table: %w", err)
	}

	if err := s.auditor.LogSchemaEvent(context.Background(), audit.SchemaEvent{
		TenantID:      newSchema.TenantID,
		SchemaCode:    newSchema.SchemaCode,
		SchemaVersion: newSchema.Version,
		Operation:     audit.OperationUpdate,
		Actor:         clientID,
		Definition:    newSchema.Definition,
		Unique:        newSchema.XUnique,
		References:    newSchema.XRefSchema,
		Webhook:       newSchema.Webhook,
		Timestamp:     newSchema.CreatedAt,
	}); err != nil {
		return nil, err
	}

	s.schemaWithAuditDetails(newSchema)
	return newSchema, nil
}

func (s *registryService) DeleteSchema(tenantID, schemaCode, clientID string) error {
	schema, err := s.repo.GetSchema(tenantID, schemaCode)
	if err != nil {
		return err
	}

	if err := s.repo.DeleteSchema(tenantID, schemaCode); err != nil {
		return err
	}

	schema.IsActive = false

	if err := s.auditor.LogSchemaEvent(context.Background(), audit.SchemaEvent{
		TenantID:      schema.TenantID,
		SchemaCode:    schema.SchemaCode,
		SchemaVersion: schema.Version,
		Operation:     audit.OperationDelete,
		Actor:         clientID,
		Definition:    schema.Definition,
		Unique:        schema.XUnique,
		References:    schema.XRefSchema,
		Webhook:       schema.Webhook,
		Timestamp:     time.Now().UTC(),
	}); err != nil {
		return err
	}

	return nil
}

func (s *registryService) ListSchemas(tenantID string) ([]models.Schema, error) {
	schemas, err := s.repo.ListSchemas(tenantID)
	if err != nil {
		return nil, err
	}
	for i := range schemas {
		s.schemaWithAuditDetails(&schemas[i])
	}
	return schemas, nil
}

// Data operations
func (s *registryService) CreateData(schemaCode string, request *models.DataRequest, clientID string, callback *models.CallbackConfig) (*models.RegistryData, error) {
	// Get schema for validation
	schema, err := s.repo.GetSchema(request.TenantID, schemaCode)
	if err != nil {
		return nil, fmt.Errorf("schema not found: %w", err)
	}

	// Validate data against schema
	if err := s.validator.ValidateData(request.Data, schema.Definition); err != nil {
		return nil, fmt.Errorf("data validation failed: %w", err)
	}

	dataMap, err := rawToMap(request.Data)
	if err != nil {
		return nil, fmt.Errorf("invalid JSON payload: %w", err)
	}

	if err := s.validateReferenceSchemas(schema, dataMap, request.TenantID); err != nil {
		return nil, err
	}

	if err := s.validateUniqueConstraints(schema, dataMap, request.TenantID, schemaCode, nil); err != nil {
		return nil, err
	}

	registryID, err := s.idGenerator.Generate(context.Background(), request.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate registry id: %w", err)
	}

	now := time.Now().UTC()
	data := &models.RegistryData{
		ID:            uuid.New(),
		RegistryID:    registryID,
		TenantID:      request.TenantID,
		SchemaCode:    schemaCode,
		SchemaVersion: schema.Version,
		Version:       1,
		Data:          request.Data,
		IsActive:      true,
		EffectiveFrom: now,
		CreatedBy:     clientID,
		UpdatedBy:     clientID,
	}

	if err := s.repo.CreateData(data); err != nil {
		return nil, fmt.Errorf("failed to create data: %w", err)
	}

	if err := s.auditor.LogDataEvent(context.Background(), audit.DataEvent{
		TenantID:      request.TenantID,
		SchemaCode:    schemaCode,
		SchemaVersion: data.SchemaVersion,
		RecordID:      data.ID,
		Version:       data.Version,
		Operation:     audit.OperationCreate,
		Actor:         clientID,
		Data:          data.Data,
		Timestamp:     data.CreatedAt,
	}); err != nil {
		return nil, err
	}

	s.recordWithAuditDetails(data)

	// Execute callback if configured (schema-level overrides request)
	effectiveCallback := chooseCallback(callback, schema.Webhook)
	if effectiveCallback != nil {
		payload := map[string]interface{}{
			"action":     "CREATE",
			"schemaCode": schemaCode,
			"tenantId":   request.TenantID,
			"data":       data,
		}
		go s.callbackManager.ExecuteCallback(*effectiveCallback, payload)
	}

	return data, nil
}

func (s *registryService) GetData(tenantID, schemaCode, id string) (*models.RegistryData, error) {
	record, err := s.repo.GetData(tenantID, schemaCode, id)
	if err != nil {
		return nil, err
	}
	s.recordWithAuditDetails(record)
	return record, nil
}

func (s *registryService) UpdateData(tenantID, schemaCode, id string, request *models.DataRequest, clientID string, callback *models.CallbackConfig) (*models.RegistryData, error) {
	// Get existing data
	existingData, err := s.repo.GetData(tenantID, schemaCode, id)
	if err != nil {
		return nil, fmt.Errorf("data not found: %w", err)
	}

	if request == nil || request.Version <= 0 {
		return nil, fmt.Errorf("version must be provided for update")
	}
	if request.Version != existingData.Version {
		return nil, fmt.Errorf("version mismatch: expected %d", existingData.Version)
	}

	// Get schema for validation
	schema, err := s.repo.GetSchemaByVersion(tenantID, schemaCode, existingData.SchemaVersion)
	if err != nil {
		return nil, fmt.Errorf("schema version not found for validation: %w", err)
	}

	// Validate new data against schema
	if err := s.validator.ValidateData(request.Data, schema.Definition); err != nil {
		return nil, fmt.Errorf("data validation failed: %w", err)
	}

	dataMap, err := rawToMap(request.Data)
	if err != nil {
		return nil, fmt.Errorf("invalid JSON payload: %w", err)
	}

	if err := s.validateReferenceSchemas(schema, dataMap, tenantID); err != nil {
		return nil, err
	}

	if err := s.validateUniqueConstraints(schema, dataMap, tenantID, schemaCode, &existingData.ID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	newVersion := &models.RegistryData{
		ID:            uuid.New(),
		RegistryID:    existingData.RegistryID,
		TenantID:      tenantID,
		SchemaCode:    schemaCode,
		SchemaVersion: schema.Version,
		Version:       existingData.Version + 1,
		Data:          request.Data,
		IsActive:      true,
		EffectiveFrom: now,
		CreatedBy:     clientID,
		UpdatedBy:     clientID,
	}

	if err := s.repo.CreateDataVersion(existingData, newVersion); err != nil {
		return nil, fmt.Errorf("failed to create new version: %w", err)
	}

	metadata := map[string]interface{}{
		"previousVersion": existingData.Version,
	}

	if err := s.auditor.LogDataEvent(context.Background(), audit.DataEvent{
		TenantID:      tenantID,
		SchemaCode:    schemaCode,
		SchemaVersion: newVersion.SchemaVersion,
		RecordID:      newVersion.ID,
		Version:       newVersion.Version,
		Operation:     audit.OperationUpdate,
		Actor:         clientID,
		Data:          newVersion.Data,
		PreviousData:  existingData.Data,
		Metadata:      metadata,
		Timestamp:     newVersion.EffectiveFrom,
	}); err != nil {
		return nil, err
	}

	s.recordWithAuditDetails(newVersion)

	// Execute callback if configured
	effectiveCallback := chooseCallback(callback, schema.Webhook)
	if effectiveCallback != nil {
		payload := map[string]interface{}{
			"action":     "UPDATE",
			"schemaCode": schemaCode,
			"tenantId":   tenantID,
			"data":       newVersion,
		}
		go s.callbackManager.ExecuteCallback(*effectiveCallback, payload)
	}

	return newVersion, nil
}

func (s *registryService) DeleteData(tenantID, schemaCode, id, clientID string, callback *models.CallbackConfig) error {
	existingData, err := s.repo.GetData(tenantID, schemaCode, id)
	if err != nil {
		return fmt.Errorf("data not found: %w", err)
	}

	schema, err := s.repo.GetSchemaByVersion(tenantID, schemaCode, existingData.SchemaVersion)
	if err != nil {
		return fmt.Errorf("schema version not found for callback: %w", err)
	}

	if err := s.repo.DeleteData(tenantID, schemaCode, id); err != nil {
		return fmt.Errorf("failed to delete data: %w", err)
	}

	existingData.IsActive = false

	effectiveCallback := chooseCallback(callback, schema.Webhook)
	if effectiveCallback != nil {
		payload := map[string]interface{}{
			"action":     "DELETE",
			"schemaCode": schemaCode,
			"tenantId":   tenantID,
			"data":       existingData,
		}
		go s.callbackManager.ExecuteCallback(*effectiveCallback, payload)
	}

	if err := s.auditor.LogDataEvent(context.Background(), audit.DataEvent{
		TenantID:      tenantID,
		SchemaCode:    schemaCode,
		SchemaVersion: existingData.SchemaVersion,
		RecordID:      existingData.ID,
		Version:       existingData.Version,
		Operation:     audit.OperationDelete,
		Actor:         clientID,
		Data:          existingData.Data,
		Timestamp:     time.Now().UTC(),
		Metadata: map[string]interface{}{
			"softDelete": true,
		},
	}); err != nil {
		return err
	}

	return nil
}

func (s *registryService) SearchData(request *models.SearchRequest) ([]models.RegistryData, error) {
	records, err := s.repo.SearchData(request)
	if err != nil {
		return nil, err
	}
	for i := range records {
		s.recordWithAuditDetails(&records[i])
	}
	return records, nil
}

func (s *registryService) DataExists(tenantID, schemaCode, id string) (bool, error) {
	if strings.TrimSpace(id) == "" {
		return false, fmt.Errorf("id must be provided")
	}
	return s.repo.ExistsData(tenantID, schemaCode, id)
}

func (s *registryService) FieldExists(tenantID, schemaCode, field, value string) (bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return false, fmt.Errorf("value must be provided")
	}
	return s.repo.FieldExists(tenantID, schemaCode, field, value)
}

func (s *registryService) GetRegistryData(tenantID, schemaCode, registryID string, history bool) ([]models.RegistryData, error) {
	if strings.TrimSpace(registryID) == "" {
		return nil, fmt.Errorf("registry id must be provided")
	}
	records, err := s.repo.ListRegistryData(tenantID, schemaCode, registryID, history)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("record not found")
	}
	for i := range records {
		s.recordWithAuditDetails(&records[i])
	}
	return records, nil
}

func (s *registryService) VerifyDataSignature(tenantID, schemaCode, identifier string) (bool, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return false, fmt.Errorf("record identifier is required")
	}

	record, err := s.repo.GetData(tenantID, schemaCode, identifier)
	if err != nil {
		return false, err
	}

	logEntry, err := s.repo.GetLatestAuditLog(tenantID, audit.SubjectTypeData, schemaCode, &record.ID)
	if err != nil {
		return false, err
	}
	if logEntry == nil {
		return false, ErrAuditLogNotFound
	}

	if strings.TrimSpace(logEntry.Signature) == "" {
		return false, fmt.Errorf("audit log signature missing")
	}

	digest, hash, err := audit.ComputeDataPayloadDigest(logEntry.PreviousHash, logEntry.Payload)
	if err != nil {
		return false, err
	}
	if !strings.EqualFold(hash, logEntry.PayloadHash) {
		return false, fmt.Errorf("payload hash mismatch")
	}

	verifier, ok := s.auditor.(audit.SignatureVerifier)
	if !ok {
		return false, ErrSignatureVerificationUnavailable
	}

	valid, err := verifier.VerifySignature(context.Background(), logEntry.TenantID, digest, logEntry.Signature)
	if err != nil {
		return false, err
	}
	return valid, nil
}

func (s *registryService) validateUniqueConstraints(schema *models.Schema, data map[string]interface{}, tenantID, schemaCode string, excludeID *uuid.UUID) error {
	if schema == nil || len(schema.XUnique) == 0 {
		return nil
	}

	for _, constraint := range schema.XUnique {
		if len(constraint) == 0 {
			continue
		}
		conditions := make(map[string]string, len(constraint))
		missing := make([]string, 0)
		empty := make([]string, 0)
		for _, path := range constraint {
			value, ok := extractValueFromPath(data, path)
			if !ok {
				missing = append(missing, path)
				continue
			}
			valueStr := fmt.Sprint(value)
			if strings.TrimSpace(valueStr) == "" {
				empty = append(empty, path)
				continue
			}
			conditions[path] = valueStr
		}

		if len(missing) > 0 {
			return fmt.Errorf("fields %s are required for unique constraint", strings.Join(missing, ", "))
		}
		if len(empty) > 0 {
			return fmt.Errorf("fields %s cannot be empty for unique constraint", strings.Join(empty, ", "))
		}
		if len(conditions) == 0 {
			continue
		}

		exists, err := s.repo.ExistsMatchingFields(tenantID, schemaCode, conditions, excludeID)
		if err != nil {
			return fmt.Errorf("failed to evaluate unique constraint: %w", err)
		}
		if exists {
			return fmt.Errorf("unique constraint violated for fields %s", strings.Join(sortedKeys(conditions), ", "))
		}
	}

	return nil
}

func (s *registryService) validateReferenceSchemas(schema *models.Schema, data map[string]interface{}, tenantID string) error {
	ctx := context.Background()
	if schema == nil || len(schema.XRefSchema) == 0 {
		return nil
	}

	for _, ref := range schema.XRefSchema {
		if ref.FieldPath == "" || ref.SchemaCode == "" {
			continue
		}

		value, ok := extractValueFromPath(data, ref.FieldPath)
		if !ok {
			return fmt.Errorf("field '%s' is required for reference schema validation", ref.FieldPath)
		}
		valueStr := fmt.Sprint(value)
		if strings.TrimSpace(valueStr) == "" {
			return fmt.Errorf("field '%s' cannot be empty for reference schema validation", ref.FieldPath)
		}

		fieldName := strings.TrimSpace(ref.RefField)
		if fieldName == "" {
			fieldName = "registryId"
		}

		if ref.External {
			if s.externalResolver == nil {
				return fmt.Errorf("external reference validation failed for field '%s': resolver not configured", ref.FieldPath)
			}
			registryKey := ref.Registry
			if strings.TrimSpace(registryKey) == "" {
				registryKey = ref.SchemaCode
			}
			exists, err := s.externalResolver.Exists(ctx, registryKey, ref.SchemaCode, tenantID, fieldName, valueStr)
			if err != nil {
				return fmt.Errorf("external reference validation failed for field '%s': %w", ref.FieldPath, err)
			}
			if !exists {
				return fmt.Errorf("external reference validation failed for field '%s': referenced record not found", ref.FieldPath)
			}
			continue
		}

		exists, err := s.repo.FieldExists(tenantID, ref.SchemaCode, fieldName, valueStr)
		if err != nil {
			return fmt.Errorf("reference validation failed for field '%s': %w", ref.FieldPath, err)
		}
		if !exists {
			return fmt.Errorf("reference validation failed for field '%s': referenced record not found", ref.FieldPath)
		}
	}

	return nil
}

func rawToMap(raw json.RawMessage) (map[string]interface{}, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func extractValueFromPath(payload map[string]interface{}, path string) (interface{}, bool) {
	if payload == nil {
		return nil, false
	}
	if path == "" {
		return nil, false
	}
	parts := strings.Split(path, ".")
	var current interface{} = payload
	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		val, exists := m[part]
		if !exists {
			return nil, false
		}
		current = val
	}
	return current, true
}

func resolveSchemaConfig(definition json.RawMessage, reqUnique models.UniqueConstraints, reqRefs []models.RefSchema, reqIndexes []models.SchemaIndex, reqWebhook *models.WebhookConfig) (json.RawMessage, models.UniqueConstraints, []models.RefSchema, []models.SchemaIndex, *models.WebhookConfig, error) {
	cleanDefinition, defUnique, defRefs, defIndexes, defWebhook, err := sanitizeSchemaDefinition(definition)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	uniqueConstraints := reqUnique
	if len(uniqueConstraints) == 0 {
		uniqueConstraints = defUnique
	}

	refConstraints := reqRefs
	if len(refConstraints) == 0 {
		refConstraints = defRefs
	}

	indexDefinitions := reqIndexes
	if len(indexDefinitions) == 0 {
		indexDefinitions = defIndexes
	}

	var webhookConfig *models.WebhookConfig
	if reqWebhook != nil {
		copyCfg := *reqWebhook
		webhookConfig = &copyCfg
	} else if defWebhook != nil {
		copyCfg := *defWebhook
		webhookConfig = &copyCfg
	}

	return cleanDefinition, uniqueConstraints, refConstraints, indexDefinitions, webhookConfig, nil
}

func sanitizeSchemaDefinition(definition json.RawMessage) (json.RawMessage, models.UniqueConstraints, []models.RefSchema, []models.SchemaIndex, *models.WebhookConfig, error) {
	if len(definition) == 0 {
		return definition, nil, nil, nil, nil, nil
	}

	var schemaMap map[string]interface{}
	if err := json.Unmarshal(definition, &schemaMap); err != nil {
		return nil, nil, nil, nil, nil, err
	}

	var uniqueConstraints models.UniqueConstraints
	if raw, ok := schemaMap["x-unique"]; ok {
		bytes, err := json.Marshal(raw)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
		if err := json.Unmarshal(bytes, &uniqueConstraints); err != nil {
			return nil, nil, nil, nil, nil, err
		}
		delete(schemaMap, "x-unique")
	}

	var refConstraints []models.RefSchema
	if raw, ok := schemaMap["x-ref-schema"]; ok {
		bytes, err := json.Marshal(raw)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
		if err := json.Unmarshal(bytes, &refConstraints); err != nil {
			return nil, nil, nil, nil, nil, err
		}
		delete(schemaMap, "x-ref-schema")
	}

	var indexes []models.SchemaIndex
	if raw, ok := schemaMap["x-indexes"]; ok {
		switch v := raw.(type) {
		case []interface{}:
			for _, entry := range v {
				switch val := entry.(type) {
				case string:
					indexes = append(indexes, models.SchemaIndex{FieldPath: val})
				case map[string]interface{}:
					bytes, err := json.Marshal(val)
					if err != nil {
						return nil, nil, nil, nil, nil, err
					}
					var idx models.SchemaIndex
					if err := json.Unmarshal(bytes, &idx); err != nil {
						return nil, nil, nil, nil, nil, err
					}
					indexes = append(indexes, idx)
				}
			}
		case []string:
			for _, path := range v {
				indexes = append(indexes, models.SchemaIndex{FieldPath: path})
			}
		default:
			bytes, err := json.Marshal(raw)
			if err != nil {
				return nil, nil, nil, nil, nil, err
			}
			if err := json.Unmarshal(bytes, &indexes); err != nil {
				return nil, nil, nil, nil, nil, err
			}
		}
		delete(schemaMap, "x-indexes")
	}

	var webhook *models.WebhookConfig
	if raw, ok := schemaMap["webhook"]; ok {
		bytes, err := json.Marshal(raw)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
		var cfg models.WebhookConfig
		if err := json.Unmarshal(bytes, &cfg); err != nil {
			return nil, nil, nil, nil, nil, err
		}
		webhook = &cfg
		delete(schemaMap, "webhook")
	}

	cleanDefinition, err := json.Marshal(schemaMap)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	return cleanDefinition, uniqueConstraints, refConstraints, indexes, webhook, nil
}

func uniqueConstraintsEqual(a, b models.UniqueConstraints) bool {
	na := normalizeUniqueConstraints(a)
	nb := normalizeUniqueConstraints(b)
	if len(na) != len(nb) {
		return false
	}
	for i := range na {
		if len(na[i]) != len(nb[i]) {
			return false
		}
		for j := range na[i] {
			if na[i][j] != nb[i][j] {
				return false
			}
		}
	}
	return true
}

func normalizeUniqueConstraints(constraints models.UniqueConstraints) models.UniqueConstraints {
	if len(constraints) == 0 {
		return nil
	}
	normalized := make(models.UniqueConstraints, 0, len(constraints))
	for _, group := range constraints {
		if len(group) == 0 {
			continue
		}
		clean := make([]string, 0, len(group))
		for _, field := range group {
			field = strings.TrimSpace(field)
			if field == "" {
				continue
			}
			clean = append(clean, field)
		}
		if len(clean) == 0 {
			continue
		}
		sort.Strings(clean)
		copyGroup := make([]string, len(clean))
		copy(copyGroup, clean)
		normalized = append(normalized, models.UniqueConstraint(copyGroup))
	}
	if len(normalized) == 0 {
		return nil
	}
	sort.Slice(normalized, func(i, j int) bool {
		left := strings.Join(normalized[i], "|")
		right := strings.Join(normalized[j], "|")
		return left < right
	})
	return normalized
}

func refConstraintsEqual(a, b []models.RefSchema) bool {
	na := normalizeRefConstraints(a)
	nb := normalizeRefConstraints(b)
	if len(na) != len(nb) {
		return false
	}
	for i := range na {
		if na[i] != nb[i] {
			return false
		}
	}
	return true
}

func normalizeRefConstraints(constraints []models.RefSchema) []models.RefSchema {
	if len(constraints) == 0 {
		return nil
	}
	out := make([]models.RefSchema, len(constraints))
	for i, constraint := range constraints {
		out[i] = constraint
	}
	sort.Slice(out, func(i, j int) bool {
		left := fmt.Sprintf("%s|%s|%s", out[i].FieldPath, out[i].SchemaCode, out[i].RefField)
		right := fmt.Sprintf("%s|%s|%s", out[j].FieldPath, out[j].SchemaCode, out[j].RefField)
		return left < right
	})
	return out
}

func indexesEqual(a, b []models.SchemaIndex) bool {
	na := normalizeIndexes(a)
	nb := normalizeIndexes(b)
	if len(na) != len(nb) {
		return false
	}
	for i := range na {
		if na[i] != nb[i] {
			return false
		}
	}
	return true
}

func normalizeIndexes(indexes []models.SchemaIndex) []models.SchemaIndex {
	if len(indexes) == 0 {
		return nil
	}
	out := make([]models.SchemaIndex, len(indexes))
	for i, idx := range indexes {
		out[i] = models.SchemaIndex{
			Name:      strings.TrimSpace(idx.Name),
			FieldPath: strings.TrimSpace(idx.FieldPath),
			Method:    strings.ToLower(strings.TrimSpace(idx.Method)),
		}
	}
	sort.Slice(out, func(i, j int) bool {
		left := fmt.Sprintf("%s|%s|%s", out[i].FieldPath, out[i].Method, out[i].Name)
		right := fmt.Sprintf("%s|%s|%s", out[j].FieldPath, out[j].Method, out[j].Name)
		return left < right
	})
	return out
}

func buildCallbackConfig(webhook *models.WebhookConfig) *models.CallbackConfig {
	if webhook == nil || !webhook.Active || strings.TrimSpace(webhook.URL) == "" {
		return nil
	}
	method := normalizeMethod(webhook.Method)
	if method == "" {
		method = "POST"
	}
	headers := map[string]string{}
	for k, v := range webhook.Headers {
		if strings.TrimSpace(k) == "" {
			continue
		}
		headers[k] = v
	}
	if webhook.ApiKey != "" {
		if headers == nil {
			headers = map[string]string{}
		}
		headers["X-API-Key"] = webhook.ApiKey
	}
	return &models.CallbackConfig{
		URL:     webhook.URL,
		Method:  method,
		Headers: headers,
	}
}

func chooseCallback(requestCallback *models.CallbackConfig, schemaWebhook *models.WebhookConfig) *models.CallbackConfig {
	if requestCallback != nil {
		return requestCallback
	}
	return buildCallbackConfig(schemaWebhook)
}

func webhookEqual(a, b *models.WebhookConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if strings.TrimSpace(a.URL) != strings.TrimSpace(b.URL) {
		return false
	}
	if normalizeMethod(a.Method) != normalizeMethod(b.Method) {
		return false
	}
	if a.ApiKey != b.ApiKey {
		return false
	}
	if a.Active != b.Active {
		return false
	}
	if !headersEqual(a.Headers, b.Headers) {
		return false
	}
	return true
}

func sortedKeys(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func normalizeMethod(method string) string {
	if strings.TrimSpace(method) == "" {
		return "POST"
	}
	return strings.ToUpper(method)
}

func headersEqual(a, b map[string]string) bool {
	na := normalizeHeaders(a)
	nb := normalizeHeaders(b)
	if len(na) != len(nb) {
		return false
	}
	for i := range na {
		if na[i] != nb[i] {
			return false
		}
	}
	return true
}

func normalizeHeaders(headers map[string]string) []string {
	if len(headers) == 0 {
		return nil
	}
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	result := make([]string, len(keys))
	for i, k := range keys {
		result[i] = fmt.Sprintf("%s=%s", strings.ToLower(k), headers[k])
	}
	return result
}

func schemasEqual(existing, incoming json.RawMessage) (bool, error) {
	normalizedExisting, err := normalizeJSON(existing)
	if err != nil {
		return false, err
	}

	normalizedIncoming, err := normalizeJSON(incoming)
	if err != nil {
		return false, err
	}

	return bytes.Equal(normalizedExisting, normalizedIncoming), nil
}

func normalizeJSON(raw json.RawMessage) ([]byte, error) {
	if len(raw) == 0 {
		return raw, nil
	}
	var payload interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return json.Marshal(payload)
}

func (s *registryService) nextSchemaVersion(tenantID, schemaCode string, currentVersion int) (int, error) {
	version := currentVersion + 1
	for {
		_, err := s.repo.GetSchemaByVersion(tenantID, schemaCode, version)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return version, nil
			}
			return 0, fmt.Errorf("failed to check schema version availability: %w", err)
		}
		version++
	}
}

func (s *registryService) schemaWithAuditDetails(schema *models.Schema) {
	if schema == nil {
		return
	}
	schema.PopulateAuditDetails()
}

func (s *registryService) recordWithAuditDetails(record *models.RegistryData) {
	if record == nil {
		return
	}
	record.PopulateAuditDetails()
}
