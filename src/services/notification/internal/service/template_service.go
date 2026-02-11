package service

import (
	"context"
	"fmt"
	"log"
	"notification/internal/config"
	"notification/internal/models"
	"notification/internal/repository"
	"notification/internal/utils"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TemplateService struct {
	repo             *repository.TemplateRepository
	enrichmentClient *utils.EnrichmentClient
	templateRenderer *TemplateRenderer
	config           *config.Config
}

func NewTemplateService(repo *repository.TemplateRepository, cfg *config.Config) *TemplateService {
	// Create enrichment client with base path and endpoint from config
	enrichmentClient := utils.NewEnrichmentClient(cfg.TemplateConfigHost, cfg.TemplateConfigPath)

	return &TemplateService{
		repo:             repo,
		enrichmentClient: enrichmentClient,
		templateRenderer: NewTemplateRenderer(),
		config:           cfg,
	}
}

func parseVersion(versionStr string) int {
	if versionStr == "" {
		return 0
	}
	// Remove leading "v" or "V"
	versionStr = strings.TrimPrefix(strings.ToLower(versionStr), "v")

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		log.Printf("[WARN] Invalid version format: %s", versionStr)
		return 0
	}
	return version
}

// Create creates the first version of a template (version v1)
func (s *TemplateService) Create(template *models.Template) (*models.TemplateDB, error) {
	// Check if template already exists
	existing, err := s.repo.GetLatestTemplate(template.TemplateID, template.TenantID)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	if existing != nil {
		return nil, fmt.Errorf("template already exists for templateId: %s, tenantId: %s. Use update to create new version",
			template.TemplateID, template.TenantID)
	}

	now := time.Now().Unix()
	templateDB := &models.TemplateDB{
		ID:               uuid.New(),
		TemplateID:       template.TemplateID,
		TenantID:         template.TenantID,
		Version:          1, // Start with version 1
		Type:             string(template.Type),
		Subject:          template.Subject,
		Content:          template.Content,
		IsHTML:           template.IsHTML,
		CreatedBy:        template.AuditDetails.CreatedBy,
		CreatedTime:      now,
		LastModifiedBy:   template.AuditDetails.CreatedBy,
		LastModifiedTime: now,
	}
	if err := s.repo.Create(templateDB); err != nil {
		return nil, err
	}

	return templateDB, nil
}

// Update creates a new version (incremented) instead of updating existing
func (s *TemplateService) Update(template *models.Template) (*models.TemplateDB, error) {
	// Get the latest version
	existing, err := s.repo.GetLatestTemplate(template.TemplateID, template.TenantID)
	if err != nil {
		return nil, err
	}

	if existing == nil {
		return nil, fmt.Errorf("template not found for templateId: %s, tenantId: %s. Use create first",
			template.TemplateID, template.TenantID)
	}

	// Increment version
	newVersion := existing.Version + 1

	now := time.Now().Unix()

	templateDB := &models.TemplateDB{
		ID:               uuid.New(),
		TemplateID:       template.TemplateID,
		TenantID:         template.TenantID,
		Version:          newVersion,
		Type:             string(template.Type),
		Subject:          template.Subject,
		Content:          template.Content,
		IsHTML:           template.IsHTML,
		CreatedBy:        existing.CreatedBy,
		CreatedTime:      existing.CreatedTime,
		LastModifiedBy:   template.AuditDetails.LastModifiedBy,
		LastModifiedTime: now,
	}

	// Insert as new record
	if err := s.repo.Create(templateDB); err != nil {
		return nil, err
	}

	return templateDB, nil
}

func (s *TemplateService) Search(searchReq *models.TemplateSearch) ([]models.TemplateDB, error) {
	if searchReq.Version != "" {
		searchReq.VersionInt = parseVersion(searchReq.Version)
		if searchReq.VersionInt == 0 {
			return nil, fmt.Errorf("invalid version: %s", searchReq.Version)
		}
	}

	// Validation rule: version should always be mentioned with templateId
	if searchReq.VersionInt > 0 && searchReq.TemplateID == "" {
		return nil, fmt.Errorf("invalid search: version filter requires templateId")
	}

	return s.repo.Search(searchReq)
}

func (s *TemplateService) Delete(deleteReq *models.TemplateDelete) error {
	version := parseVersion(deleteReq.Version)
	if version == 0 {
		return fmt.Errorf("invalid version: %s", deleteReq.Version)
	}

	// Verify it exists
	if _, err := s.repo.GetByTemplateIDAndVersion(deleteReq.TemplateID, deleteReq.TenantID, version); err != nil {
		return err
	}
	return s.repo.Delete(deleteReq.TemplateID, deleteReq.TenantID, version)
}

func (s *TemplateService) Preview(request *models.TemplatePreviewRequest) (*models.TemplatePreviewResponse, []models.Error) {
	var templateDB *models.TemplateDB
	var err error

	// Step 1: Template Fetching
	if request.Version != "" {
		// Use specific version
		version := parseVersion(request.Version)
		if version == 0 {
			return nil, []models.Error{{
				Code:        "INVALID_VERSION",
				Message:     "Invalid version",
				Description: fmt.Sprintf("Invalid version: %s", request.Version),
			}}
		}
		templateDB, err = s.repo.GetByTemplateIDAndVersion(request.TemplateID, request.TenantID, version)
	} else {
		// Use latest version
		templateDB, err = s.repo.GetLatestTemplate(request.TemplateID, request.TenantID)
	}

	if err != nil {
		return nil, []models.Error{{
			Code:        "NOT_FOUND",
			Message:     "Template not found",
			Description: err.Error(),
		}}
	}

	// Step 2: Payload Enrichment (Conditional)
	var templateData map[string]interface{}
	if request.Enrich {
		// Create context for enrichment
		ctx := context.Background()

		// Enrich payload using the template-config service
		enrichedData, err := s.enrichmentClient.EnrichPayload(ctx, request.TemplateID, request.TenantID, request.Version, request.Payload)
		if err != nil {
			log.Printf("Failed to enrich payload: %v", err)
			return nil, []models.Error{{
				Code:        "ENRICHMENT_FAILED",
				Message:     "Failed to enrich payload",
				Description: err.Error(),
			}}
		}
		templateData = enrichedData
	} else {
		// Use raw payload for template rendering
		templateData = request.Payload
	}

	// Step 3: Template Rendering
	var renderedSubject, renderedContent string
	var errors []models.Error

	// Render subject if it exists
	if templateDB.Type == string(models.TemplateTypeEmail) && templateDB.Subject != "" {
		renderedSubject, err = s.templateRenderer.RenderSubject(templateDB.Subject, templateData)
		if err != nil {
			errors = append(errors, models.Error{
				Code:        "TEMPLATE_RENDER_ERROR",
				Message:     "Failed to render subject template",
				Description: err.Error(),
			})
		}
	}

	// Render content
	renderedContent, err = s.templateRenderer.RenderContent(templateDB.Content, templateData, templateDB.IsHTML)
	if err != nil {
		errors = append(errors, models.Error{
			Code:        "TEMPLATE_RENDER_ERROR",
			Message:     "Failed to render content template",
			Description: err.Error(),
		})
	}

	// If there were rendering errors, return them
	if len(errors) > 0 {
		return nil, errors
	}

	response := &models.TemplatePreviewResponse{
		TemplateID:      templateDB.TemplateID,
		TenantID:        templateDB.TenantID,
		Version:         fmt.Sprintf("v%d", templateDB.Version),
		Type:            models.TemplateType(templateDB.Type),
		IsHTML:          templateDB.IsHTML,
		RenderedSubject: renderedSubject,
		RenderedContent: renderedContent,
	}

	return response, nil
}
