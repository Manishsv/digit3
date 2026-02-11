package models

// TemplateConfigSearch represents search parameters
type TemplateConfigSearch struct {
	IDs        []string `form:"ids"`
	TemplateID string   `form:"templateId"`
	TenantID   string   `form:"tenantId"`
	Version    string   `form:"version"`
	VersionInt int
}
