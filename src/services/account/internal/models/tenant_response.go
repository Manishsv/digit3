package models

type TenantRequest struct {
	Tenant Tenant `json:"tenant"`
}

type TenantResponse struct {
	Tenants []Tenant `json:"tenants"`
} 