package models

type GenerateIDResponse struct {
	TenantID     string `json:"tenantId"`
	TemplateCode string `json:"templateCode"`
	Version      string `form:"version"`
	ID           string `json:"id"`
}
