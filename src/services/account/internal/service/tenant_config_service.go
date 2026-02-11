package service

import (
	"account/internal/enrichment"
	"account/internal/models"
	"account/internal/repository"
	"account/internal/validator"
	"context"
	"errors"
	"fmt"
)

type TenantConfigService struct {
	Repo          *repository.TenantConfigRepository
	TenantService *TenantService
	DocumentRepo  *repository.DocumentRepository
}

func NewTenantConfigService(repo *repository.TenantConfigRepository, tenantService *TenantService, documentRepo *repository.DocumentRepository) *TenantConfigService {
	return &TenantConfigService{
		Repo:          repo,
		TenantService: tenantService,
		DocumentRepo:  documentRepo,
	}
}

func (s *TenantConfigService) CreateTenantConfig(ctx context.Context, config *models.TenantConfig) error {
	return s.Repo.CreateTenantConfig(ctx, config)
}

func (s *TenantConfigService) CreateTenantConfigWithValidation(ctx context.Context, config *models.TenantConfig, clientID string) error {
	// 1. Validate tenant config
	existing, err := s.Repo.ListTenantConfigs(ctx)
	if err != nil {
		return err
	}
	if err := validator.ValidateTenantConfig(config, existing); err != nil {
		return err
	}

	// 2. Check if tenant exists
	tenants, err := s.TenantService.ListTenants(ctx)
	if err != nil {
		return err
	}
	tenantExists := false
	for _, t := range tenants {
		if t.Code == config.Code {
			tenantExists = true
			break
		}
	}
	if !tenantExists {
		return errors.New("TENANT_NOT_FOUND")
	}

	// 3. Enrich tenant config
	enrichment.EnrichTenantConfig(config, clientID)

	// 4. Create tenant config
	return s.Repo.CreateTenantConfig(ctx, config)
}

func (s *TenantConfigService) UpdateTenantConfig(ctx context.Context, config *models.TenantConfig) error {
	// First fetch the existing record to ensure it exists and preserve audit details
	existing, err := s.Repo.GetTenantConfigByID(ctx, config.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("RECORD_NOT_FOUND")
	}

	// Preserve the original audit details (createdBy and createdTime)
	config.AuditDetails.CreatedBy = existing.AuditDetails.CreatedBy
	config.AuditDetails.CreatedTime = existing.AuditDetails.CreatedTime

	// Update the record
	return s.Repo.UpdateTenantConfig(ctx, config)
}

func (s *TenantConfigService) UpdateTenantConfigWithValidation(ctx context.Context, updateRequest *models.TenantConfig, clientID string) (*models.TenantConfig, error) {
	// 1. Fetch existing record
	existing, err := s.Repo.GetTenantConfigByID(ctx, updateRequest.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, errors.New("RECORD_NOT_FOUND")
	}

	// 2. Validate the update request
	if err := s.validateUpdateRequest(updateRequest, existing); err != nil {
		return nil, err
	}

	// 3. Merge changes: preserve existing data, update allowed fields
	updatedConfig := *existing // Copy existing config

	// Update all fields except code, createdBy, and createdTime
	if updateRequest.DefaultLoginType != "" {
		updatedConfig.DefaultLoginType = updateRequest.DefaultLoginType
	}
	if updateRequest.OtpLength != "" {
		updatedConfig.OtpLength = updateRequest.OtpLength
	}
	if updateRequest.Name != "" {
		updatedConfig.Name = updateRequest.Name
	}
	updatedConfig.EnableUserBasedLogin = updateRequest.EnableUserBasedLogin
	if updateRequest.AdditionalAttributes != nil {
		updatedConfig.AdditionalAttributes = updateRequest.AdditionalAttributes
	}
	updatedConfig.IsActive = updateRequest.IsActive
	if updateRequest.Languages != nil {
		updatedConfig.Languages = updateRequest.Languages
	}

	// 4. Handle document updates
	if err := s.handleDocumentUpdates(ctx, &updatedConfig, updateRequest, existing, clientID); err != nil {
		return nil, err
	}

	// 5. Enrich with audit details (only lastModifiedBy and lastModifiedTime)
	enrichment.EnrichTenantConfigUpdate(&updatedConfig, clientID)

	// 6. Update the record
	if err := s.Repo.UpdateTenantConfig(ctx, &updatedConfig); err != nil {
		return nil, err
	}

	return &updatedConfig, nil
}

func (s *TenantConfigService) validateUpdateRequest(updateRequest *models.TenantConfig, existing *models.TenantConfig) error {
	// Validate that all existing documents are included in the update request
	if updateRequest.Documents == nil {
		return errors.New("DOCUMENTS_REQUIRED")
	}

	// Create a map of existing document IDs for quick lookup
	existingDocMap := make(map[string]bool)
	for _, doc := range existing.Documents {
		existingDocMap[doc.ID] = true
	}

	// Check if all existing documents are included
	for _, existingDoc := range existing.Documents {
		found := false
		for _, updateDoc := range updateRequest.Documents {
			if updateDoc.ID == existingDoc.ID {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("MISSING_DOCUMENT: Document with ID %s must be included in update request", existingDoc.ID)
		}
	}

	return nil
}

func (s *TenantConfigService) handleDocumentUpdates(ctx context.Context, updatedConfig *models.TenantConfig, updateRequest *models.TenantConfig, existing *models.TenantConfig, clientID string) error {
	// Process each document in the update request
	var updatedDocuments []models.Document

	for _, updateDoc := range updateRequest.Documents {
		if updateDoc.ID == "" {
			// New document - create it
			newDoc := updateDoc
			newDoc.TenantConfigID = updatedConfig.ID
			newDoc.TenantID = existing.Code // Assuming tenant ID is the same as code

			// Enrich the new document
			enrichment.EnrichDocument(&newDoc, clientID)

			// Create in database
			if err := s.DocumentRepo.CreateDocument(ctx, &newDoc); err != nil {
				return fmt.Errorf("failed to create new document: %v", err)
			}

			updatedDocuments = append(updatedDocuments, newDoc)
		} else {
			// Existing document - update it
			existingDoc := s.findExistingDocument(existing.Documents, updateDoc.ID)
			if existingDoc == nil {
				return fmt.Errorf("DOCUMENT_NOT_FOUND: Document with ID %s not found in existing config", updateDoc.ID)
			}

			// Update document fields
			updatedDoc := *existingDoc
			updatedDoc.Type = updateDoc.Type
			updatedDoc.FileStoreID = updateDoc.FileStoreID
			updatedDoc.URL = updateDoc.URL
			updatedDoc.IsActive = updateDoc.IsActive

			// Enrich with update audit details
			enrichment.EnrichDocumentUpdate(&updatedDoc, clientID)

			// Update in database
			if err := s.DocumentRepo.UpdateDocument(ctx, &updatedDoc); err != nil {
				return fmt.Errorf("failed to update document %s: %v", updateDoc.ID, err)
			}

			updatedDocuments = append(updatedDocuments, updatedDoc)
		}
	}

	updatedConfig.Documents = updatedDocuments
	return nil
}

func (s *TenantConfigService) findExistingDocument(documents []models.Document, id string) *models.Document {
	for _, doc := range documents {
		if doc.ID == id {
			return &doc
		}
	}
	return nil
}

func (s *TenantConfigService) ListTenantConfigs(ctx context.Context) ([]*models.TenantConfig, error) {
	return s.Repo.ListTenantConfigs(ctx)
}

func (s *TenantConfigService) GetTenantConfigByID(ctx context.Context, id string) (*models.TenantConfig, error) {
	return s.Repo.GetTenantConfigByID(ctx, id)
}

func (s *TenantConfigService) DeleteTenantConfig(ctx context.Context, id string) error {
	return s.Repo.DeleteTenantConfig(ctx, id)
}
