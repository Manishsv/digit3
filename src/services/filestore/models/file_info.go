package models

type FileInfo struct {
	ContentType  string       `json:"contentType"` // Match Java field name in JSON
	FileLocation FileLocation `json:"fileLocation"`
	TenantID     string       `json:"tenantId"`
}
