package models

// TemplateConfigDelete represents delete parameters
type TemplateConfigDelete struct {
	TemplateID string `form:"templateId" binding:"required"`
	TenantID   string `form:"tenantId"`
	Version    string `form:"version" binding:"required"`
}
