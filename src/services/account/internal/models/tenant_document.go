package models

type TenantDocument struct {
	ID             string `json:"id"`
	TenantConfigID string `json:"tenantConfigId"`
	TenantID       string `json:"tenantId"`
	Type           string `json:"type"`
	FileStoreID    string `json:"fileStoreId"`
	URL            string `json:"url"`
	IsActive       bool   `json:"isActive"`
	CreatedBy      string `json:"createdBy"`
	LastModifiedBy string `json:"lastModifiedBy"`
	CreatedTime    int64  `json:"createdTime"`
	LastModifiedTime int64 `json:"lastModifiedTime"`
}
