package models

// RenderResponse represents the response from rendering
type RenderResponse struct {
	TemplateID string         `json:"templateId"`
	TenantID   string         `json:"tenantId"`
	Version    string         `json:"version"`
	Data       map[string]any `json:"data"`
}
