package models

// IDGenTemplateDelete represents delete parameters
type IDGenTemplateDelete struct {
	TemplateCode string `form:"templateCode" binding:"required"`
	TenantID     string `form:"tenantId"`
	Version      string `form:"version" binding:"required"`
}
