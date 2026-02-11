package models

// RenderRequest represents the request for rendering
type RenderRequest struct {
	TemplateID string         `json:"templateId" binding:"required"`
	TenantID   string         `json:"tenantId"`
	Version    string         `json:"version"`
	Payload    map[string]any `json:"payload"`
}
