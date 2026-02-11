package models

import "io"

type Resource struct {
	ContentType string    `json:"contentType"` // MIME type of the file
	FileName    string    `json:"fileName"`    // Name of the file
	Resource    io.Reader `json:"-"`           // The file content (stream) - typically not marshalled to JSON
	TenantID    string    `json:"tenantId"`    // Tenant identifier
	FileSize    string    `json:"fileSize"`    // File size as string (could use int64 for bytes if preferred)
}
