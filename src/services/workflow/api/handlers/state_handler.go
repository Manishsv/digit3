package handlers

import (
	"net/http"

	"digit.org/workflow/internal/models"
	"digit.org/workflow/internal/service"
	"github.com/gin-gonic/gin"
)

type StateHandler struct {
	service service.StateService
}

func NewStateHandler(s service.StateService) *StateHandler {
	return &StateHandler{service: s}
}

// CreateState handles the API request to create a new state for a process.
func (h *StateHandler) CreateState(c *gin.Context) {
	var state models.State
	if err := c.ShouldBindJSON(&state); err != nil {
		c.JSON(http.StatusBadRequest, models.Error{Code: "BadRequest", Message: err.Error()})
		return
	}

	// Extract tenant ID from header (required for multi-tenancy)
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	// Validate processId UUID format
	processID := c.Param("id")
	if err := models.ValidateUUID(processID, "processId"); err != nil {
		c.JSON(http.StatusBadRequest, models.Error{Code: "ValidationError", Message: err.Error()})
		return
	}

	// Validate input
	if validationErrors := models.ValidateStateCreate(&state); validationErrors != nil {
		c.JSON(http.StatusBadRequest, validationErrors)
		return
	}

	// Extract user ID from X-Client-Id header and set audit details
	userID := models.GetUserIDFromContext(c)
	state.ProcessID = processID
	state.TenantID = tenantID
	state.AuditDetail.SetAuditDetailsForCreate(userID)

	createdState, err := h.service.CreateState(c.Request.Context(), &state)
	if err != nil {
		// Check for database constraint violations and return proper error codes
		if models.IsDatabaseConstraintError(err) {
			c.JSON(http.StatusBadRequest, models.Error{Code: "ValidationError", Message: models.GetConstraintErrorMessage(err)})
			return
		}
		c.JSON(http.StatusInternalServerError, models.Error{Code: "InternalServerError", Message: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, createdState)
}

// GetStates handles the API request to list all states for a process.
func (h *StateHandler) GetStates(c *gin.Context) {
	processID := c.Param("id") // Changed from "processId"
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	codes := c.QueryArray("code")

	states, err := h.service.GetStatesByProcessID(c.Request.Context(), tenantID, processID, codes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Error{Code: "InternalServerError", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, states)
}

// GetState handles the API request to get a single state by its ID.
func (h *StateHandler) GetState(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	state, err := h.service.GetStateByID(c.Request.Context(), tenantID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.Error{Code: "NotFound", Message: "State not found"})
		return
	}

	c.JSON(http.StatusOK, state)
}

// UpdateState handles the API request to update a state.
func (h *StateHandler) UpdateState(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	// First, get the existing state
	existingState, err := h.service.GetStateByID(c.Request.Context(), tenantID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.Error{Code: "NotFound", Message: "State not found"})
		return
	}

	var updateRequest models.State
	if err := c.ShouldBindJSON(&updateRequest); err != nil {
		c.JSON(http.StatusBadRequest, models.Error{Code: "BadRequest", Message: err.Error()})
		return
	}

	// Extract user ID from X-Client-Id header and set audit details for update
	userID := models.GetUserIDFromContext(c)

	// Merge update request with existing state, only updating provided fields
	stateToUpdate := *existingState
	if updateRequest.Name != "" {
		stateToUpdate.Name = updateRequest.Name
	}
	if updateRequest.Code != "" {
		stateToUpdate.Code = updateRequest.Code
	}
	if updateRequest.Description != nil && *updateRequest.Description != "" {
		stateToUpdate.Description = updateRequest.Description
	}
	if updateRequest.SLA != nil {
		stateToUpdate.SLA = updateRequest.SLA
	}
	// Note: Boolean fields need special handling since false is a valid value
	// We'll preserve existing boolean values unless explicitly provided
	// For now, keep existing boolean logic - this could be enhanced to detect if field was actually provided

	// Set audit details for update
	stateToUpdate.AuditDetail.SetAuditDetailsForUpdate(userID)

	updatedState, err := h.service.UpdateState(c.Request.Context(), &stateToUpdate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Error{Code: "InternalServerError", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedState)
}

// DeleteState handles the API request to delete a state.
func (h *StateHandler) DeleteState(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	err := h.service.DeleteState(c.Request.Context(), tenantID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Error{Code: "InternalServerError", Message: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
