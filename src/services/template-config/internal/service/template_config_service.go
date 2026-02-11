package service

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"template-config/internal/models"
	"template-config/internal/repository"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"github.com/oliveagle/jsonpath"
	"gorm.io/gorm"
)

type TemplateConfigService struct {
	repo       *repository.TemplateConfigRepository
	httpClient *resty.Client
}

func NewTemplateConfigService(repo *repository.TemplateConfigRepository) *TemplateConfigService {
	return &TemplateConfigService{
		repo:       repo,
		httpClient: resty.New().SetTimeout(30 * time.Second),
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
func (s *TemplateConfigService) Create(config *models.TemplateConfig) (*models.TemplateConfigDB, error) {
	// Check if template already exists
	existing, err := s.repo.GetLatestTemplateConfig(config.TemplateID, config.TenantID)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	if existing != nil {
		return nil, fmt.Errorf("template config already exists for templateId: %s, tenantId: %s. Use update to create new version",
			config.TemplateID, config.TenantID)
	}

	now := time.Now().Unix()
	dbConfig := &models.TemplateConfigDB{
		ID:               uuid.New(),
		TemplateID:       config.TemplateID,
		TenantID:         config.TenantID,
		Version:          1, // Start with version 1
		FieldMapping:     config.FieldMapping,
		APIMapping:       config.APIMapping,
		CreatedBy:        config.AuditDetails.CreatedBy,
		CreatedTime:      now,
		LastModifiedBy:   config.AuditDetails.CreatedBy,
		LastModifiedTime: now,
	}

	if err := s.repo.Create(dbConfig); err != nil {
		return nil, err
	}

	return dbConfig, nil
}

// Update creates a new version (incremented) instead of updating existing
func (s *TemplateConfigService) Update(config *models.TemplateConfig) (*models.TemplateConfigDB, error) {
	// Get the latest version
	existing, err := s.repo.GetLatestTemplateConfig(config.TemplateID, config.TenantID)
	if err != nil {
		return nil, err
	}

	if existing == nil {
		return nil, fmt.Errorf("template config not found for templateId: %s, tenantId: %s. Use create first",
			config.TemplateID, config.TenantID)
	}

	// Increment version
	newVersion := existing.Version + 1

	now := time.Now().Unix()
	dbConfig := &models.TemplateConfigDB{
		ID:               uuid.New(), // New UUID for new version
		TemplateID:       config.TemplateID,
		TenantID:         config.TenantID,
		Version:          newVersion,
		FieldMapping:     config.FieldMapping,
		APIMapping:       config.APIMapping,
		CreatedBy:        existing.CreatedBy,
		CreatedTime:      existing.CreatedTime,
		LastModifiedBy:   config.AuditDetails.LastModifiedBy,
		LastModifiedTime: now,
	}

	// Insert as new record
	if err := s.repo.Create(dbConfig); err != nil {
		return nil, err
	}

	return dbConfig, nil
}

func (s *TemplateConfigService) Search(search *models.TemplateConfigSearch) ([]models.TemplateConfigDB, error) {
	if search.Version != "" {
		search.VersionInt = parseVersion(search.Version)
		if search.VersionInt == 0 {
			return nil, fmt.Errorf("invalid version: %s", search.Version)
		}
	}

	// Validation rule: version should always be mentioned with templateId
	if search.VersionInt > 0 && search.TemplateID == "" {
		return nil, fmt.Errorf("invalid search: version filter requires templateId")
	}

	return s.repo.Search(search)
}

func (s *TemplateConfigService) Delete(templateID, tenantID, versionStr string) error {
	version := parseVersion(versionStr)
	if version == 0 {
		return fmt.Errorf("invalid version: %s", versionStr)
	}

	// Verify it exists
	if _, err := s.repo.GetByTemplateIDAndVersion(templateID, tenantID, version); err != nil {
		return err
	}
	return s.repo.Delete(templateID, tenantID, version)
}

func (s *TemplateConfigService) Render(request *models.RenderRequest) (*models.RenderResponse, []models.Error) {
	var config *models.TemplateConfigDB
	var err error

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
		config, err = s.repo.GetByTemplateIDAndVersion(request.TemplateID, request.TenantID, version)
	} else {
		// Use latest version
		config, err = s.repo.GetLatestTemplateConfig(request.TemplateID, request.TenantID)
	}

	if err != nil {
		return nil, []models.Error{{
			Code:        "NOT_FOUND",
			Message:     "Template config not found",
			Description: err.Error(),
		}}
	}

	response := &models.RenderResponse{
		TemplateID: request.TemplateID,
		TenantID:   request.TenantID,
		Version:    fmt.Sprintf("v%d", config.Version), // Return the version that was used
		Data:       make(map[string]any),
	}

	for field, jsonPath := range config.FieldMapping {
		if value, err := jsonpath.JsonPathLookup(request.Payload, jsonPath); err == nil {
			response.Data[field] = value
			log.Printf("[FieldMapping] %s: %v", field, value)
		} else {
			log.Printf("[FieldMapping] Failed for %s (%s): %v", field, jsonPath, err)
		}
	}

	if len(config.APIMapping) > 0 {
		if errors := s.executeAPIMappings(config.APIMapping, request.Payload, request.TenantID, response); len(errors) > 0 {
			return nil, errors
		}
	}

	return response, nil
}

func (s *TemplateConfigService) executeAPIMappings(apiMappings []models.APIMapping, payload map[string]interface{}, tenantID string, response *models.RenderResponse) []models.Error {
	var (
		wg        sync.WaitGroup
		errorChan = make(chan models.Error, len(apiMappings))
	)

	for _, mapping := range apiMappings {
		wg.Add(1)
		go func(mapping models.APIMapping) {
			defer wg.Done()
			url := s.buildURL(mapping.Endpoint, payload)
			log.Printf("[APIMapping] Calling: %s", url)

			resp, err := s.httpClient.R().
				SetHeader("Content-Type", "application/json").
				SetHeader("X-Tenant-ID", tenantID).
				Get(url)

			// Log the raw response body if it exists
			if resp != nil {
				log.Printf("[APIMapping] Response from %s: %s", url, resp.String())
			}

			if err != nil || resp == nil || resp.StatusCode() != http.StatusOK {
				var errDesc string
				if err != nil {
					errDesc = err.Error()
				}
				if resp != nil {
					errDesc = fmt.Sprintf("HTTP %d: %s", resp.StatusCode(), resp.String())
				}
				errorChan <- models.Error{
					Code:        "API_CALL_FAILED",
					Message:     "External API call failed",
					Description: errDesc,
					Params:      []string{url, mapping.Method},
				}
				return
			}

			var apiResp interface{}
			if err := json.Unmarshal(resp.Body(), &apiResp); err != nil {
				errorChan <- models.Error{
					Code:        "INVALID_JSON",
					Message:     "Failed to parse API response",
					Description: err.Error(),
					Params:      []string{url},
				}
				return
			}

			for field, jsonPath := range mapping.ResponseMapping {
				if value, err := jsonpath.JsonPathLookup(apiResp, jsonPath); err == nil {
					response.Data[field] = value
					log.Printf("[APIResponseMapping] %s: %v", field, value)
				} else {
					log.Printf("[APIResponseMapping] Failed for %s (%s): %v", field, jsonPath, err)
				}
			}
		}(mapping)
	}

	wg.Wait()
	close(errorChan)

	errors := make([]models.Error, 0, len(apiMappings))
	for err := range errorChan {
		errors = append(errors, err)
	}
	return errors
}

func (s *TemplateConfigService) buildURL(endpoint models.EndpointConfig, payload map[string]interface{}) string {
	url := endpoint.Base + endpoint.Path

	for param, path := range endpoint.PathParams {
		if value, err := jsonpath.JsonPathLookup(payload, path); err == nil {
			url = strings.ReplaceAll(url, "{{"+param+"}}", fmt.Sprintf("%v", value))
		}
	}

	if len(endpoint.QueryParams) > 0 {
		var query []string
		for key, path := range endpoint.QueryParams {
			if value, err := jsonpath.JsonPathLookup(payload, path); err == nil {
				query = append(query, fmt.Sprintf("%s=%v", key, value))
			}
		}
		if len(query) > 0 {
			url += "?" + strings.Join(query, "&")
		}
	}
	return url
}
