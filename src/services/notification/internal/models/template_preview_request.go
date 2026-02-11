package models

// TemplatePreviewRequest represents the API request model for template preview
type TemplatePreviewRequest struct {
	TemplateID string                 `json:"templateId" binding:"required"`
	Version    string                 `json:"version"`
	TenantID   string                 `json:"tenantId"`
	Enrich     bool                   `json:"enrich"`
	Payload    map[string]interface{} `json:"payload,omitempty"`
}
