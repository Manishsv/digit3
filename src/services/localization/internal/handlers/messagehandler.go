package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"localization/internal/core/domain"
	"localization/internal/core/ports"
	"localization/pkg/dtos"
)

// MessageHandler is the handler for localization message related APIs
type MessageHandler struct {
	service ports.MessageService
}

// NewMessageHandler creates a new MessageHandler
func NewMessageHandler(service ports.MessageService) *MessageHandler {
	return &MessageHandler{
		service: service,
	}
}

// Helper to get user ID from header
func getUserIDFromHeader(c *gin.Context) string {
	return c.GetHeader("X-User-ID")
}

// SearchMessages handles the messages search API endpoint
func (h *MessageHandler) SearchMessages(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	locale := c.Query("locale")
	module := c.Query("module")
	codes := c.QueryArray("codes")

	var messages []domain.Message
	var err error

	// Handle pagination parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if len(codes) > 0 {
		messages, err = h.service.SearchMessagesByCodes(c.Request.Context(), tenantID, locale, codes)
	} else {
		messages, err = h.service.SearchMessages(c.Request.Context(), tenantID, module, locale, limit, offset)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert domain messages to response DTOs
	responseMessages := make([]dtos.MessageResponse, len(messages))
	for i, msg := range messages {
		responseMessages[i] = dtos.MessageResponse{
			UUID:    msg.UUID,
			Code:    msg.Code,
			Message: msg.Message,
			Module:  msg.Module,
			Locale:  msg.Locale,
		}
	}

	c.JSON(http.StatusOK, gin.H{"messages": responseMessages})
}

// CreateMessages handles the creation of multiple new localization messages
func (h *MessageHandler) CreateMessages(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	var req dtos.CreateMessagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := getUserIDFromHeader(c)

	domainMessages := make([]domain.Message, len(req.Messages))
	for i, msg := range req.Messages {
		domainMessages[i] = domain.Message{
			Code:    msg.Code,
			Message: msg.Message,
			Module:  msg.Module,
			Locale:  msg.Locale,
		}
	}

	createdMessages, err := h.service.CreateMessages(c.Request.Context(), tenantID, userID, domainMessages)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert domain messages to response DTOs
	responseMessages := make([]dtos.MessageResponse, len(createdMessages))
	for i, msg := range createdMessages {
		responseMessages[i] = dtos.MessageResponse{
			UUID:    msg.UUID,
			Code:    msg.Code,
			Message: msg.Message,
			Module:  msg.Module,
			Locale:  msg.Locale,
		}
	}

	c.JSON(http.StatusCreated, gin.H{"messages": responseMessages})
}

// UpdateMessages handles the update of multiple existing localization messages
func (h *MessageHandler) UpdateMessages(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	var req dtos.UpdateMessagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := getUserIDFromHeader(c)

	domainMessages := make([]domain.Message, len(req.Messages))
	for i, msg := range req.Messages {
		domainMessages[i] = domain.Message{
			UUID:    msg.UUID,
			Message: msg.Message,
		}
	}

	updatedMessages, err := h.service.UpdateMessages(c.Request.Context(), tenantID, userID, domainMessages)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert domain messages to response DTOs
	responseMessages := make([]dtos.MessageResponse, len(updatedMessages))
	for i, msg := range updatedMessages {
		responseMessages[i] = dtos.MessageResponse{
			UUID:    msg.UUID,
			Code:    msg.Code,
			Message: msg.Message,
			Module:  msg.Module,
			Locale:  msg.Locale,
		}
	}

	c.JSON(http.StatusOK, gin.H{"messages": responseMessages})
}

// UpsertMessages handles the upsert of multiple localization messages
func (h *MessageHandler) UpsertMessages(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	var req dtos.UpsertMessagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := getUserIDFromHeader(c)

	domainMessages := make([]domain.Message, len(req.Messages))
	for i, msg := range req.Messages {
		domainMessages[i] = domain.Message{
			UUID:    msg.UUID,
			Code:    msg.Code,
			Message: msg.Message,
			Module:  msg.Module,
			Locale:  msg.Locale,
		}
	}

	upsertedMessages, err := h.service.UpsertMessages(c.Request.Context(), tenantID, userID, domainMessages)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert domain messages to response DTOs
	responseMessages := make([]dtos.MessageResponse, len(upsertedMessages))
	for i, msg := range upsertedMessages {
		responseMessages[i] = dtos.MessageResponse{
			UUID:    msg.UUID,
			Code:    msg.Code,
			Message: msg.Message,
			Module:  msg.Module,
			Locale:  msg.Locale,
		}
	}

	c.JSON(http.StatusOK, gin.H{"messages": responseMessages})
}

// DeleteMessages handles the deletion of multiple localization messages by UUID
func (h *MessageHandler) DeleteMessages(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	// Support both single 'uuid' and multiple 'uuids' parameters
	var uuids []string

	// Check for single uuid parameter
	if uuid := c.Query("uuid"); uuid != "" {
		uuids = append(uuids, uuid)
	}

	// Check for multiple uuids parameters (array form)
	uuidsArray := c.QueryArray("uuids")
	uuids = append(uuids, uuidsArray...)

	// Remove duplicates if both forms are used
	uniqueUUIDs := make(map[string]bool)
	var finalUUIDs []string
	for _, uuid := range uuids {
		if uuid != "" && !uniqueUUIDs[uuid] {
			uniqueUUIDs[uuid] = true
			finalUUIDs = append(finalUUIDs, uuid)
		}
	}

	if len(finalUUIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one uuid is required in query params"})
		return
	}

	err := h.service.DeleteMessages(c.Request.Context(), tenantID, finalUUIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// FindMissingMessages handles the API endpoint for finding missing localization messages
func (h *MessageHandler) FindMissingMessages(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}
	module := c.Query("module")

	missingMessages, err := h.service.FindMissingMessages(c.Request.Context(), tenantID, module)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if missingMessages == nil {
		c.JSON(http.StatusOK, gin.H{})
		return
	}

	c.JSON(http.StatusOK, missingMessages)
}

// BustCache handles clearing the cache for a tenant, with optional module and locale
func (h *MessageHandler) BustCache(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	module := c.Query("module")
	locale := c.Query("locale")

	err := h.service.BustCache(c.Request.Context(), tenantID, module, locale)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cache bust operation completed successfully", "success": true})
}

// RegisterRoutes registers the API routes on the given router group
func (h *MessageHandler) RegisterRoutes(router *gin.RouterGroup) {
	// Main messages routes
	messagesGroup := router.Group("/v1/messages")
	{
		messagesGroup.GET("", h.SearchMessages)                // GET /messages
		messagesGroup.POST("", h.CreateMessages)               // POST /messages
		messagesGroup.PUT("", h.UpdateMessages)                // PUT /messages
		messagesGroup.DELETE("", h.DeleteMessages)             // DELETE /messages
		messagesGroup.PUT("/_upsert", h.UpsertMessages)        // PUT /messages/_upsert
		messagesGroup.POST("/_missing", h.FindMissingMessages) // POST /messages/_missing
	}

	// Cache bust route
	cacheGroup := router.Group("/v1/cache")
	{
		cacheGroup.DELETE("/_bust", h.BustCache) // DELETE /cache/_bust
	}
}
