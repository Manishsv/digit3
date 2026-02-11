package models

// SMSCategory represents the classification of SMS notifications
type SMSCategory string

const (
	SMSCategoryOTP          SMSCategory = "OTP"
	SMSCategoryTransaction  SMSCategory = "TRANSACTION"
	SMSCategoryPromotion    SMSCategory = "PROMOTION"
	SMSCategoryNotification SMSCategory = "NOTIFICATION"
	SMSCategoryOthers       SMSCategory = "OTHERS"
)

// SMSRequest represents the API request model for sending SMS
type SMSRequest struct {
	TemplateID    string                 `json:"templateId" binding:"required"`
	Version       string                 `json:"version"`
	TenantID      string                 `json:"tenantId"`
	MobileNumbers []string               `json:"mobileNumbers" binding:"required,dive,mobile_number"`
	Enrich        bool                   `json:"enrich"`
	Payload       map[string]interface{} `json:"payload,omitempty"`
	Category      SMSCategory            `json:"category,omitempty" binding:"omitempty,oneof=OTP TRANSACTION PROMOTION NOTIFICATION OTHERS"`
}
