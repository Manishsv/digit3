package handlers

import (
	"net/http"

	"digit.org/workflow/internal/models"
	"digit.org/workflow/internal/service"
	"github.com/gin-gonic/gin"
)

// EscalationConfigHandler handles escalation configuration related HTTP requests.
type EscalationConfigHandler struct {
	service service.EscalationConfigService
}

// NewEscalationConfigHandler creates a new EscalationConfigHandler.
func NewEscalationConfigHandler(s service.EscalationConfigService) *EscalationConfigHandler {
	return &EscalationConfigHandler{service: s}
}

// CreateEscalationConfig handles the creation of a new escalation configuration.
func (h *EscalationConfigHandler) CreateEscalationConfig(c *gin.Context) {
	var config models.EscalationConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, models.Error{Code: "BadRequest", Message: err.Error()})
		return
	}

	// Extract tenant ID from header (required for multi-tenancy)
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, models.Error{Code: "BadRequest", Message: "X-Tenant-ID header is required"})
		return
	}

	// Extract process ID from URL path
	processID := c.Param("id")
	if processID == "" {
		c.JSON(http.StatusBadRequest, models.Error{Code: "BadRequest", Message: "Process ID is required"})
		return
	}

	// Extract user ID from X-Client-Id header and set audit details
	userID := models.GetUserIDFromContext(c)
	config.TenantID = tenantID
	config.ProcessID = processID
	config.AuditDetail.SetAuditDetailsForCreate(userID)

	createdConfig, err := h.service.CreateEscalationConfig(c.Request.Context(), &config)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Error{Code: "BadRequest", Message: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, createdConfig)
}

// GetEscalationConfigs handles retrieving escalation configurations for a process.
func (h *EscalationConfigHandler) GetEscalationConfigs(c *gin.Context) {
	// Extract tenant ID from header (required for multi-tenancy)
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, models.Error{Code: "BadRequest", Message: "X-Tenant-ID header is required"})
		return
	}

	// Extract process ID from URL path
	processID := c.Param("id")
	if processID == "" {
		c.JSON(http.StatusBadRequest, models.Error{Code: "BadRequest", Message: "Process ID is required"})
		return
	}

	// Parse query parameters
	stateCode := c.Query("stateCode")

	// Note: isActive query parameter is no longer supported since the column was removed
	// All escalation configs are returned, filtered only by stateCode if provided

	configs, err := h.service.GetEscalationConfigsByProcessID(c.Request.Context(), tenantID, processID, stateCode, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Error{Code: "InternalServerError", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, configs)
}

// GetEscalationConfig handles retrieving a single escalation configuration by ID.
func (h *EscalationConfigHandler) GetEscalationConfig(c *gin.Context) {
	// Extract tenant ID from header (required for multi-tenancy)
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, models.Error{Code: "BadRequest", Message: "X-Tenant-ID header is required"})
		return
	}

	// Extract config ID from URL path
	configID := c.Param("id")
	if configID == "" {
		c.JSON(http.StatusBadRequest, models.Error{Code: "BadRequest", Message: "Escalation config ID is required"})
		return
	}

	config, err := h.service.GetEscalationConfigByID(c.Request.Context(), tenantID, configID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.Error{Code: "NotFound", Message: "Escalation config not found"})
		return
	}

	c.JSON(http.StatusOK, config)
}

// UpdateEscalationConfig handles updating an escalation configuration.
func (h *EscalationConfigHandler) UpdateEscalationConfig(c *gin.Context) {
	var config models.EscalationConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, models.Error{Code: "BadRequest", Message: err.Error()})
		return
	}

	// Extract tenant ID from header (required for multi-tenancy)
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, models.Error{Code: "BadRequest", Message: "X-Tenant-ID header is required"})
		return
	}

	// Extract config ID from URL path
	configID := c.Param("id")
	if configID == "" {
		c.JSON(http.StatusBadRequest, models.Error{Code: "BadRequest", Message: "Escalation config ID is required"})
		return
	}

	// Extract user ID from X-Client-Id header and set audit details
	userID := models.GetUserIDFromContext(c)
	config.TenantID = tenantID
	config.ID = configID
	config.AuditDetail.SetAuditDetailsForUpdate(userID)

	updatedConfig, err := h.service.UpdateEscalationConfig(c.Request.Context(), &config)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Error{Code: "BadRequest", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedConfig)
}

// DeleteEscalationConfig handles deleting an escalation configuration.
func (h *EscalationConfigHandler) DeleteEscalationConfig(c *gin.Context) {
	// Extract tenant ID from header (required for multi-tenancy)
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, models.Error{Code: "BadRequest", Message: "X-Tenant-ID header is required"})
		return
	}

	// Extract config ID from URL path
	configID := c.Param("id")
	if configID == "" {
		c.JSON(http.StatusBadRequest, models.Error{Code: "BadRequest", Message: "Escalation config ID is required"})
		return
	}

	err := h.service.DeleteEscalationConfig(c.Request.Context(), tenantID, configID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.Error{Code: "NotFound", Message: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
