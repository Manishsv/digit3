package models

type GenerateIDRequest struct {
	TenantID     string            `json:"tenantId"`
	TemplateCode string            `json:"templateCode" binding:"required"`
	Variables    map[string]string `json:"variables"`
}
