package handlers

import (
	"net/http"
	"strings"
	"template-config/internal/models"
	"template-config/internal/service"
	"template-config/internal/validation"

	"github.com/gin-gonic/gin"
)

type TemplateConfigHandler struct {
	service   *service.TemplateConfigService
	validator *validation.TemplateValidator
}

func NewTemplateConfigHandler(service *service.TemplateConfigService) *TemplateConfigHandler {
	return &TemplateConfigHandler{service: service, validator: validation.NewTemplateValidator()}
}

func getTenantIDFromHeader(c *gin.Context) string {
	return c.GetHeader("X-Tenant-ID")
}

func getClientIDFromHeader(c *gin.Context) string {
	return c.GetHeader("X-Client-ID")
}

// CreateTemplateConfig handles POST /template-config (creates version v1)
func (h *TemplateConfigHandler) CreateTemplateConfig(c *gin.Context) {
	var config models.TemplateConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:        "BAD_REQUEST",
			Message:     "Invalid request body",
			Description: err.Error(),
		})
		return
	}

	config.TenantID = getTenantIDFromHeader(c)
	if config.TenantID == "" {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:        "BAD_REQUEST",
			Message:     "Missing required tenantId",
			Description: "Request must include tenantId in headers",
		})
		return
	}

	config.AuditDetails.CreatedBy = getClientIDFromHeader(c)

	if err := h.validator.ValidateTemplateConfig(&config); err != nil {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:        "BAD_REQUEST",
			Message:     "Invalid template config",
			Description: err.Error(),
		})
		return
	}

	dbConfig, err := h.service.Create(&config)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, models.Error{
				Code:        "CONFLICT",
				Message:     "Template config already exists",
				Description: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:        "INTERNAL_SERVER_ERROR",
			Message:     "Failed to create template config",
			Description: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, dbConfig.ToDTO())
}

// UpdateTemplateConfig handles PUT /template-config (creates new version)
func (h *TemplateConfigHandler) UpdateTemplateConfig(c *gin.Context) {
	var config models.TemplateConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:        "BAD_REQUEST",
			Message:     "Invalid request body",
			Description: err.Error(),
		})
		return
	}

	config.TenantID = getTenantIDFromHeader(c)
	if config.TenantID == "" {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:        "BAD_REQUEST",
			Message:     "Missing required tenantId",
			Description: "Request must include tenantId in headers",
		})
		return
	}

	config.AuditDetails.LastModifiedBy = getClientIDFromHeader(c)

	if err := h.validator.ValidateTemplateConfig(&config); err != nil {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:        "BAD_REQUEST",
			Message:     "Invalid template config",
			Description: err.Error(),
		})
		return
	}

	// Update creates a new version
	dbConfig, err := h.service.Update(&config)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, models.Error{
				Code:        "NOT_FOUND",
				Message:     "Template config not found",
				Description: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:        "INTERNAL_SERVER_ERROR",
			Message:     "Failed to update template config",
			Description: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dbConfig.ToDTO())
}

// SearchTemplateConfigs - version optional in query
func (h *TemplateConfigHandler) SearchTemplateConfigs(c *gin.Context) {
	var search models.TemplateConfigSearch
	if err := c.ShouldBindQuery(&search); err != nil {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:        "BAD_REQUEST",
			Message:     "Invalid query parameters",
			Description: err.Error(),
		})
		return
	}

	if idsStr := c.Query("ids"); idsStr != "" {
		search.IDs = strings.Split(idsStr, ",")
	}

	search.TenantID = getTenantIDFromHeader(c)
	if search.TenantID == "" {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:    "BAD_REQUEST",
			Message: "Missing required tenantId",
		})
		return
	}

	configDBList, err := h.service.Search(&search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:        "INTERNAL_SERVER_ERROR",
			Message:     "Failed to search template configs",
			Description: err.Error(),
		})
		return
	}

	configList := make([]models.TemplateConfigWithVersion, 0, len(configDBList))
	for _, config := range configDBList {
		configList = append(configList, config.ToDTO())
	}
	c.JSON(http.StatusOK, configList)
}

// DeleteTemplateConfig handles DELETE /template-config
func (h *TemplateConfigHandler) DeleteTemplateConfig(c *gin.Context) {
	var deleteReq models.TemplateConfigDelete
	if err := c.ShouldBindQuery(&deleteReq); err != nil {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:        "BAD_REQUEST",
			Message:     "Invalid query parameters",
			Description: err.Error(),
		})
		return
	}
	deleteReq.TenantID = getTenantIDFromHeader(c)
	if deleteReq.TenantID == "" {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:        "BAD_REQUEST",
			Message:     "Missing required tenantId",
			Description: "Request must include tenantId in headers",
		})
		return
	}

	if err := h.service.Delete(deleteReq.TemplateID, deleteReq.TenantID, deleteReq.Version); err != nil {
		if strings.Contains(err.Error(), "record not found") {
			c.JSON(http.StatusNotFound, models.Error{
				Code:        "NOT_FOUND",
				Message:     "Template config not found",
				Description: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:        "INTERNAL_SERVER_ERROR",
			Message:     "Failed to delete template config",
			Description: err.Error(),
		})
		return
	}
	c.Status(http.StatusOK)
}

// RenderTemplateConfig handles POST /template-config/render
func (h *TemplateConfigHandler) RenderTemplateConfig(c *gin.Context) {
	var request models.RenderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:        "BAD_REQUEST",
			Message:     "Invalid request body",
			Description: err.Error(),
		})
		return
	}
	request.TenantID = getTenantIDFromHeader(c)
	if request.TenantID == "" {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:        "BAD_REQUEST",
			Message:     "Missing required tenantId",
			Description: "Request must include tenantId in headers",
		})
		return
	}

	response, errors := h.service.Render(&request)
	if len(errors) > 0 {
		c.JSON(http.StatusUnprocessableEntity, errors)
		return
	}
	c.JSON(http.StatusOK, response)
}
