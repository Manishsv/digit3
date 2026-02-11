package handlers

import (
	"net/http"
	"strings"

	"digit.org/workflow/internal/models"
	"digit.org/workflow/internal/service"
	"github.com/gin-gonic/gin"
)

// ActionHandler handles action-related HTTP requests.
type ActionHandler struct {
	actionService service.ActionService
	stateService  service.StateService
}

// NewActionHandler creates a new ActionHandler.
func NewActionHandler(actionService service.ActionService, stateService service.StateService) *ActionHandler {
	return &ActionHandler{
		actionService: actionService,
		stateService:  stateService,
	}
}

// CreateAction handles the creation of a new action.
func (h *ActionHandler) CreateAction(c *gin.Context) {
	var actionRequest struct {
		Name                string                      `json:"name"`
		Label               *string                     `json:"label,omitempty"`
		NextState           string                      `json:"nextState"`
		AttributeValidation *models.AttributeValidation `json:"attributeValidation,omitempty"`
	}

	// Extract tenant ID from header (required for multi-tenancy)
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	// Extract current state ID from URL path
	currentStateID := c.Param("id")
	if currentStateID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "State ID is required"})
		return
	}

	if err := c.ShouldBindJSON(&actionRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Debug logging
	if actionRequest.NextState == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "NextState is empty in request"})
		return
	}

	// Get the current state to find the process ID
	currentState, err := h.stateService.GetStateByID(c.Request.Context(), tenantID, currentStateID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Current state not found: " + err.Error()})
		return
	}

	// Convert next state code to state ID
	var nextStateID string
	if actionRequest.NextState != "" {
		// Check if it's already a UUID (for backward compatibility)
		if len(actionRequest.NextState) == 36 && actionRequest.NextState[8] == '-' {
			nextStateID = actionRequest.NextState // It's already a UUID
		} else {
			// Try to convert state code to UUID
			nextState, err := h.stateService.GetStateByCodeAndProcess(c.Request.Context(), tenantID, currentState.ProcessID, actionRequest.NextState)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Next state not found for code '" + actionRequest.NextState + "': " + err.Error()})
				return
			}
			nextStateID = nextState.ID
		}
	}

	// More debug logging
	if nextStateID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "NextStateID is empty after conversion"})
		return
	}

	// Extract user ID from X-Client-Id header and set audit details
	userID := models.GetUserIDFromContext(c)
	action := models.Action{
		TenantID:            tenantID,
		Name:                actionRequest.Name,
		Label:               actionRequest.Label,
		CurrentState:        currentStateID, // Use state UUID from URL
		NextState:           nextStateID,    // Converted from code to UUID
		AttributeValidation: actionRequest.AttributeValidation,
	}
	action.AuditDetail.SetAuditDetailsForCreate(userID)

	createdAction, err := h.actionService.CreateAction(c.Request.Context(), &action)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, createdAction)
}

// GetActions handles retrieving actions by state ID.
func (h *ActionHandler) GetActions(c *gin.Context) {
	// Extract tenant ID from header (required for multi-tenancy)
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	stateID := c.Param("id")
	// Validate UUID format
	if err := models.ValidateUUID(stateID, "stateId"); err != nil {
		c.JSON(http.StatusBadRequest, models.Error{Code: "ValidationError", Message: err.Error()})
		return
	}

	codes := c.QueryArray("code")

	actions, err := h.actionService.GetActionsByStateID(c.Request.Context(), tenantID, stateID, codes)
	if err != nil {
		// Check for database constraint violations and return proper error codes
		if models.IsDatabaseConstraintError(err) {
			c.JSON(http.StatusBadRequest, models.Error{Code: "ValidationError", Message: models.GetConstraintErrorMessage(err)})
			return
		}
		c.JSON(http.StatusInternalServerError, models.Error{Code: "InternalServerError", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, actions)
}

// GetAction handles retrieving a single action by ID.
func (h *ActionHandler) GetAction(c *gin.Context) {
	// Extract tenant ID from header (required for multi-tenancy)
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	actionID := c.Param("id")
	action, err := h.actionService.GetActionByID(c.Request.Context(), tenantID, actionID)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Action not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, action)
}

// UpdateAction handles updating an action.
func (h *ActionHandler) UpdateAction(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	// First, get the existing action
	existingAction, err := h.actionService.GetActionByID(c.Request.Context(), tenantID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.Error{Code: "NotFound", Message: "Action not found"})
		return
	}

	var updateRequest struct {
		Name                string                      `json:"name,omitempty"`
		Label               *string                     `json:"label,omitempty"`
		NextState           string                      `json:"nextState,omitempty"`
		AttributeValidation *models.AttributeValidation `json:"attributeValidation,omitempty"`
	}

	if err := c.ShouldBindJSON(&updateRequest); err != nil {
		c.JSON(http.StatusBadRequest, models.Error{Code: "BadRequest", Message: err.Error()})
		return
	}

	// Merge update request with existing action, only updating provided fields
	actionToUpdate := *existingAction
	if updateRequest.Name != "" {
		actionToUpdate.Name = updateRequest.Name
	}
	if updateRequest.Label != nil {
		actionToUpdate.Label = updateRequest.Label
	}
	if updateRequest.NextState != "" {
		// Handle nextState conversion like in CreateAction
		if len(updateRequest.NextState) == 36 && updateRequest.NextState[8] == '-' {
			actionToUpdate.NextState = updateRequest.NextState // It's already a UUID
		} else {
			// Try to convert state code to UUID - need current state for context
			currentState, err := h.stateService.GetStateByID(c.Request.Context(), tenantID, actionToUpdate.CurrentState)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Current state not found: " + err.Error()})
				return
			}
			nextState, err := h.stateService.GetStateByCodeAndProcess(c.Request.Context(), tenantID, currentState.ProcessID, updateRequest.NextState)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Next state not found for code '" + updateRequest.NextState + "': " + err.Error()})
				return
			}
			actionToUpdate.NextState = nextState.ID
		}
	}
	if updateRequest.AttributeValidation != nil {
		actionToUpdate.AttributeValidation = updateRequest.AttributeValidation
	}

	// Extract user ID from X-Client-Id header and set audit details for update
	userID := models.GetUserIDFromContext(c)
	actionToUpdate.AuditDetail.SetAuditDetailsForUpdate(userID)

	updatedAction, err := h.actionService.UpdateAction(c.Request.Context(), &actionToUpdate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedAction)
}

// DeleteAction handles deleting an action.
func (h *ActionHandler) DeleteAction(c *gin.Context) {
	// Extract tenant ID from header (required for multi-tenancy)
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	actionID := c.Param("id")
	err := h.actionService.DeleteAction(c.Request.Context(), tenantID, actionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
