package models

type FileLocation struct {
	FileStoreID string `json:"fileStoreId"` // Match Java field name in JSON
	Module      string `json:"module"`
	Tag         string `json:"tag,omitempty"` // omitempty for optional fields
	TenantID    string `json:"tenantId"`
	FileName    string `json:"fileName"`
	FileSource  string `json:"fileSource"`
}
