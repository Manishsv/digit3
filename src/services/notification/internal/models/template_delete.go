package models

// TemplateDelete represents delete parameters
type TemplateDelete struct {
	TemplateID string `form:"templateId" binding:"required"`
	TenantID   string `form:"tenantId"`
	Version    string `form:"version" binding:"required"`
}
