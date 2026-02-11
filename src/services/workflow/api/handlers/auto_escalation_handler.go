package handlers

import (
	"net/http"
	"strconv"

	"digit.org/workflow/internal/models"
	"digit.org/workflow/internal/service"
	"github.com/gin-gonic/gin"
)

// AutoEscalationHandler handles auto-escalation related HTTP requests.
type AutoEscalationHandler struct {
	service service.AutoEscalationService
}

// NewAutoEscalationHandler creates a new AutoEscalationHandler.
func NewAutoEscalationHandler(s service.AutoEscalationService) *AutoEscalationHandler {
	return &AutoEscalationHandler{service: s}
}

// EscalationRequest represents the request body for triggering auto-escalation.
type EscalationRequest struct {
	Attributes map[string][]string `json:"attributes,omitempty"`
}

// EscalateApplications handles the auto-escalation trigger API.
// POST /workflow/v1/auto/:processCode/_escalate
func (h *AutoEscalationHandler) EscalateApplications(c *gin.Context) {
	// Extract tenant ID from header (required for multi-tenancy)
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, models.Error{Code: "BadRequest", Message: "X-Tenant-ID header is required"})
		return
	}

	// Extract process code from URL path
	processCode := c.Param("processCode")
	if processCode == "" {
		c.JSON(http.StatusBadRequest, models.Error{Code: "BadRequest", Message: "Process code is required"})
		return
	}

	// Extract user ID from X-Client-Id header
	userID := models.GetUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusBadRequest, models.Error{Code: "BadRequest", Message: "X-Client-Id header is required"})
		return
	}

	// Parse request body for attributes (optional)
	var req EscalationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// If no body or invalid JSON, use empty attributes
		req.Attributes = make(map[string][]string)
	}

	// Trigger auto-escalation
	result, err := h.service.EscalateApplications(c.Request.Context(), tenantID, processCode, req.Attributes, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Error{Code: "BadRequest", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// SearchEscalatedApplications handles the search for auto-escalated instances.
// GET /workflow/v1/auto/_search
func (h *AutoEscalationHandler) SearchEscalatedApplications(c *gin.Context) {
	// Extract tenant ID from header (required for multi-tenancy)
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, models.Error{Code: "BadRequest", Message: "X-Tenant-ID header is required"})
		return
	}

	// Extract query parameters
	processID := c.Query("processId")

	// Parse pagination parameters
	limit := 100 // default limit
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := 0 // default offset
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Search escalated instances
	instances, err := h.service.SearchEscalatedApplications(c.Request.Context(), tenantID, processID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Error{Code: "InternalServerError", Message: err.Error()})
		return
	}

	// Return the instances
	if instances == nil {
		instances = []*models.ProcessInstance{}
	}

	c.JSON(http.StatusOK, instances)
}
