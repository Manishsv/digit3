package handlers

import (
	"net/http"
	"notification/internal/models"
	"notification/internal/service"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	emailService *service.EmailService
	smsService   *service.SMSService
}

func NewNotificationHandler(emailService *service.EmailService, smsService *service.SMSService) *NotificationHandler {
	return &NotificationHandler{emailService: emailService, smsService: smsService}
}

// Send email handle POST /email/send
func (h *NotificationHandler) SendEmail(c *gin.Context) {
	var emailReq models.EmailRequest
	if err := c.ShouldBindJSON(&emailReq); err != nil {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:        "BAD_REQUEST",
			Message:     "Invalid request body",
			Description: err.Error(),
		})
		return
	}

	emailReq.TenantID = getTenantIDFromHeader(c)
	if emailReq.TenantID == "" {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:        "BAD_REQUEST",
			Message:     "Missing required tenantId",
			Description: "Request must include tenantId in headers",
		})
		return
	}

	if errs := h.emailService.SendEmail(c.Request.Context(), &emailReq); errs != nil {
		c.JSON(http.StatusInternalServerError, errs)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "EMAIL sent successfully"})
}

// Send SMS handle POST /sms/send
func (h *NotificationHandler) SendSMS(c *gin.Context) {
	var smsReq models.SMSRequest
	if err := c.ShouldBindJSON(&smsReq); err != nil {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:        "BAD_REQUEST",
			Message:     "Invalid request body",
			Description: err.Error(),
		})
		return
	}

	smsReq.TenantID = getTenantIDFromHeader(c)
	if smsReq.TenantID == "" {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:        "BAD_REQUEST",
			Message:     "Missing required tenantId",
			Description: "Request must include tenantId in headers",
		})
		return
	}

	if errs := h.smsService.SendSMS(c.Request.Context(), &smsReq); errs != nil {
		c.JSON(http.StatusInternalServerError, errs)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "SMS sent successfully"})
}
