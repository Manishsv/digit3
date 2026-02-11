package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"digit.org/workflow/internal/models"
	"digit.org/workflow/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TransitionHandler struct {
	transitionService service.TransitionService
}

func NewTransitionHandler(transitionService service.TransitionService) *TransitionHandler {
	return &TransitionHandler{
		transitionService: transitionService,
	}
}

type TransitionRequest struct {
	ProcessInstanceID *string             `json:"processInstanceId,omitempty"`
	ProcessID         string              `json:"processId" binding:"required"`
	EntityID          string              `json:"entityId" binding:"required"`
	Init              *bool               `json:"init,omitempty"` // NEW: Flag to create new instance in initial state
	Action            string              `json:"action"`
	Status            *string             `json:"status,omitempty"`
	CurrentState      *string             `json:"currentState,omitempty"` // Expected current state for validation
	Comment           *string             `json:"comment,omitempty"`
	Documents         []string            `json:"documents,omitempty"`
	Assigner          *string             `json:"assigner,omitempty"`
	Assignees         *[]string           `json:"assignees,omitempty"`
	Attributes        map[string][]string `json:"attributes,omitempty"` // User attributes for validation
}

type TransitionResponse struct {
	ID           string              `json:"id"`
	ProcessID    string              `json:"processId"`
	EntityID     string              `json:"entityId"`
	Action       string              `json:"action"`
	Status       string              `json:"status"`
	Comment      string              `json:"comment"`
	Documents    []string            `json:"documents"`
	Assigner     string              `json:"assigner"`
	Assignees    []string            `json:"assignees"`
	CurrentState string              `json:"currentState"`
	StateSla     int64               `json:"stateSla"`
	ProcessSla   int64               `json:"processSla"`
	Attributes   map[string][]string `json:"attributes"`
	AuditDetails models.AuditDetail  `json:"auditDetails"`
}

func (h *TransitionHandler) Transition(c *gin.Context) {
	var req TransitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.Error{Code: "BadRequest", Message: err.Error()})
		return
	}

	// No validation needed - the old logic handles both creation and transition cases

	// Extract tenant ID from header (required for multi-tenancy)
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, models.Error{Code: "ValidationError", Message: "X-Tenant-ID header is required"})
		return
	}

	// Extract user ID from X-Client-Id header for audit tracking
	userID := models.GetUserIDFromContext(c)

	// Add user information to context for the service layer
	ctx := context.WithValue(c.Request.Context(), "userID", userID)
	ctx = context.WithValue(ctx, "tenantID", tenantID)

	// Call the transition service with init parameter
	result, err := h.transitionService.Transition(ctx, req.ProcessInstanceID, req.ProcessID, req.EntityID, req.Action, req.Init, req.Status, req.CurrentState, req.Comment, req.Documents, req.Assigner, req.Assignees, req.Attributes, tenantID)
	if err != nil {
		statusCode, apiErr := mapTransitionError(err)
		c.JSON(statusCode, apiErr)
		return
	}

	// Convert documents from []models.Document to []string
	var docStrings []string
	for _, doc := range result.Documents {
		docStrings = append(docStrings, doc.FileStoreID)
	}

	// Handle optional fields with defaults
	comment := ""
	if req.Comment != nil {
		comment = *req.Comment
	}

	assigner := ""
	if result.Assigner != nil {
		assigner = *result.Assigner
	}

	stateSla := int64(0)
	if result.StateSLA != nil {
		stateSla = *result.StateSLA
	}

	processSla := int64(0)
	if result.ProcessSLA != nil {
		processSla = *result.ProcessSLA
	}

	// Ensure attributes is not nil
	attributes := result.Attributes
	if attributes == nil {
		attributes = make(map[string][]string)
	}

	response := TransitionResponse{
		ID:           result.ID,
		ProcessID:    result.ProcessID,
		EntityID:     result.EntityID,
		Action:       req.Action,
		Status:       result.Status,
		Comment:      comment,
		Documents:    docStrings,
		Assigner:     assigner,
		Assignees:    result.Assignees,
		CurrentState: result.CurrentState,
		StateSla:     stateSla,
		ProcessSla:   processSla,
		Attributes:   attributes,
		AuditDetails: result.AuditDetails,
	}

	c.JSON(http.StatusOK, response)
}

func (h *TransitionHandler) GetTransitions(c *gin.Context) {
	// Extract tenant ID from header (required for multi-tenancy)
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, models.Error{Code: "ValidationError", Message: "X-Tenant-ID header is required"})
		return
	}

	entityID := strings.TrimSpace(c.Query("entityId"))
	processID := strings.TrimSpace(c.Query("processId"))
	currentStateID := strings.TrimSpace(c.Query("currentStateId"))
	assigneeID := strings.TrimSpace(c.Query("assigneeId"))

	historyParam, historyProvided := c.GetQuery("history")
	if !historyProvided {
		historyParam = "false"
	}

	filtersProvided := 0
	if entityID != "" {
		filtersProvided++
	}
	if currentStateID != "" {
		filtersProvided++
	}
	if assigneeID != "" {
		filtersProvided++
	}

	if filtersProvided == 0 {
		c.JSON(http.StatusBadRequest, models.Error{Code: "ValidationError", Message: "provide either entityId, currentStateId, or assigneeId as a query parameter"})
		return
	}
	if filtersProvided > 1 {
		c.JSON(http.StatusBadRequest, models.Error{Code: "ValidationError", Message: "specify only one of entityId, currentStateId, or assigneeId per request"})
		return
	}

	criteria := &models.ProcessInstanceSearchCriteria{}

	if processID != "" {
		criteria.ProcessID = &processID
	}

	switch {
	case entityID != "":
		if criteria.ProcessID == nil || *criteria.ProcessID == "" {
			c.JSON(http.StatusBadRequest, models.Error{Code: "ValidationError", Message: "processId query parameter is required when filtering by entityId"})
			return
		}
		criteria.EntityID = &entityID
		criteria.History = strings.EqualFold(historyParam, "true")

	case currentStateID != "":
		if err := models.ValidateUUID(currentStateID, "currentStateId"); err != nil {
			c.JSON(http.StatusBadRequest, models.Error{Code: "ValidationError", Message: err.Error()})
			return
		}
		if historyProvided && !strings.EqualFold(historyParam, "false") && historyParam != "" {
			c.JSON(http.StatusBadRequest, models.Error{Code: "ValidationError", Message: "history parameter is only supported when filtering by entityId"})
			return
		}
		criteria.CurrentStateID = &currentStateID

	case assigneeID != "":
		if historyProvided && !strings.EqualFold(historyParam, "false") && historyParam != "" {
			c.JSON(http.StatusBadRequest, models.Error{Code: "ValidationError", Message: "history parameter is only supported when filtering by entityId"})
			return
		}
		criteria.AssigneeID = &assigneeID
	}

	// Add tenant information to context
	ctx := context.WithValue(c.Request.Context(), "tenantID", tenantID)

	// Call the service
	instances, err := h.transitionService.GetTransitions(ctx, tenantID, criteria)
	if err != nil {
		statusCode, apiErr := mapTransitionError(err)
		c.JSON(statusCode, apiErr)
		return
	}

	// Return the response
	c.JSON(http.StatusOK, gin.H{
		"processInstances": instances,
		"totalCount":       len(instances),
	})
}

func mapTransitionError(err error) (int, models.Error) {
	if err == nil {
		return http.StatusInternalServerError, models.Error{Code: "InternalServerError", Message: "unexpected error while processing transition"}
	}

	errMsg := err.Error()
	lower := strings.ToLower(errMsg)

	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return http.StatusNotFound, models.Error{Code: "NotFound", Message: "Process instance not found"}
	case strings.Contains(lower, "provide either entityid"):
		return http.StatusBadRequest, models.Error{Code: "ValidationError", Message: "provide either entityId, currentStateId, or assigneeId"}
	case strings.Contains(lower, "specify only one of entityid"):
		return http.StatusBadRequest, models.Error{Code: "ValidationError", Message: "specify only one of entityId, currentStateId, or assigneeId"}
	case strings.Contains(lower, "processid is required when searching by entityid"):
		return http.StatusBadRequest, models.Error{Code: "ValidationError", Message: "processId is required when searching by entityId"}
	case strings.Contains(lower, "search criteria is required"):
		return http.StatusBadRequest, models.Error{Code: "ValidationError", Message: "search criteria is required"}
	case strings.Contains(lower, "unsupported search criteria"):
		return http.StatusBadRequest, models.Error{Code: "ValidationError", Message: "unsupported search criteria"}
	case strings.Contains(lower, "process instance already exists"):
		return http.StatusConflict, models.Error{Code: "Conflict", Message: "Process instance already exists for this entity"}
	case strings.Contains(lower, "current state mismatch"):
		return http.StatusConflict, models.Error{Code: "Conflict", Message: "Process instance is no longer in the expected state"}
	case strings.Contains(lower, "invalid state transition"):
		return http.StatusConflict, models.Error{Code: "Conflict", Message: "Process instance is no longer in the expected state"}
	case strings.Contains(lower, "no active parallel execution found for branch"):
		return http.StatusConflict, models.Error{Code: "Conflict", Message: "Parallel execution has already completed"}
	case strings.Contains(lower, "user is not authorized") || strings.Contains(lower, "transition not permitted by guard"):
		return http.StatusForbidden, models.Error{Code: "Forbidden", Message: "User is not authorized to perform this action"}
	case strings.Contains(lower, "guard check failed"):
		message := strings.TrimPrefix(errMsg, "guard check failed: ")
		return http.StatusForbidden, models.Error{Code: "Forbidden", Message: message}
	case strings.Contains(lower, "action '") && strings.Contains(lower, "is not valid for"):
		return http.StatusBadRequest, models.Error{Code: "ValidationError", Message: errMsg}
	default:
		return http.StatusInternalServerError, models.Error{Code: "InternalServerError", Message: "unexpected error while processing transition"}
	}
}
