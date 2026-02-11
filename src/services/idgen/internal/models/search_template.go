package models

// IDGenTemplateSearch represents search parameters
type IDGenTemplateSearch struct {
	IDs          []string `form:"ids"`
	TemplateCode string   `form:"templateCode"`
	TenantID     string   `form:"tenantId"`
	Version      string   `form:"version"`
	VersionInt   int
}
