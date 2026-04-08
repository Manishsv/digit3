package models

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Matches DB columns created_by, modified_by (VARCHAR(64) in Flyway initial schema).
const maxAuditUserIDLength = 64

// AuditDetail represents audit information for database records.
type AuditDetail struct {
	CreatedBy    string `json:"createdBy,omitempty" db:"created_by" gorm:"column:created_by"`
	CreatedTime  int64  `json:"createdTime,omitempty" db:"created_at" gorm:"column:created_at"`
	ModifiedBy   string `json:"modifiedBy,omitempty" db:"modified_by" gorm:"column:modified_by"`
	ModifiedTime int64  `json:"modifiedTime,omitempty" db:"modified_at" gorm:"column:modified_at"`
}

// ClampAuditUserID trims and truncates to fit VARCHAR(64) audit columns.
func ClampAuditUserID(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > maxAuditUserIDLength {
		return s[:maxAuditUserIDLength]
	}
	return s
}

// GetUserIDFromContext extracts user ID from X-Client-Id / X-Client-ID and truncates to fit VARCHAR(64).
func GetUserIDFromContext(c *gin.Context) string {
	clientID := strings.TrimSpace(c.GetHeader("X-Client-Id"))
	if clientID == "" {
		clientID = strings.TrimSpace(c.GetHeader("X-Client-ID"))
	}
	if clientID == "" {
		return "system"
	}
	return ClampAuditUserID(clientID)
}

// SetAuditDetailsForCreate sets audit details for a new record creation
func (a *AuditDetail) SetAuditDetailsForCreate(userID string) {
	now := time.Now().UnixMilli()
	u := strings.TrimSpace(userID)
	if u == "" {
		u = "system"
	} else {
		u = ClampAuditUserID(u)
	}
	a.CreatedBy = u
	a.CreatedTime = now
	a.ModifiedBy = u
	a.ModifiedTime = now
}

// SetAuditDetailsForUpdate sets audit details for record update
func (a *AuditDetail) SetAuditDetailsForUpdate(userID string) {
	now := time.Now().UnixMilli()
	u := strings.TrimSpace(userID)
	if u == "" {
		u = "system"
	} else {
		u = ClampAuditUserID(u)
	}
	a.ModifiedBy = u
	a.ModifiedTime = now
}

// Document represents a document attachment.
type Document struct {
	DocumentType      string                 `json:"documentType,omitempty" db:"document_type" gorm:"column:document_type"`
	FileStoreID       string                 `json:"fileStoreId,omitempty" db:"file_store_id" gorm:"column:file_store_id"`
	DocumentUID       string                 `json:"documentUid,omitempty" db:"document_uid" gorm:"column:document_uid"`
	AdditionalDetails map[string]interface{} `json:"additionalDetails,omitempty" db:"additional_details" gorm:"column:additional_details;type:jsonb"`
}

// Error represents an API error response.
type Error struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	Description string `json:"description,omitempty"`
}
