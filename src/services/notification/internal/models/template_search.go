package models

// TemplateSearch represents search parameters
type TemplateSearch struct {
	IDs        []string `form:"ids"`
	TemplateID string   `form:"templateId"`
	TenantID   string   `form:"tenantId"`
	Version    string   `form:"version"`
	VersionInt int
	Type       TemplateType `form:"type" binding:"omitempty,oneof=EMAIL SMS"`
	IsHTML     bool         `form:"isHTML"`
}
