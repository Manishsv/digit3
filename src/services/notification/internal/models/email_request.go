package models

// EmailRequest represents the API request model for sending emails
type EmailRequest struct {
	TemplateID  string                 `json:"templateId" binding:"required"`
	Version     string                 `json:"version"`
	TenantID    string                 `json:"tenantId"`
	EmailIds    []string               `json:"emailIds" binding:"required,dive,email"`
	Enrich      bool                   `json:"enrich"`
	Payload     map[string]interface{} `json:"payload,omitempty"`
	Attachments []string               `json:"attachments,omitempty"`
}
