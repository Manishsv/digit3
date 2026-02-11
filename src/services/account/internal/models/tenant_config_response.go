package models

type TenantConfigRequest struct {
	TenantConfig TenantConfig `json:"tenantConfig"`
}

type TenantConfigResponse struct {
	TenantConfigs []TenantConfig `json:"tenantConfigs"`
} 