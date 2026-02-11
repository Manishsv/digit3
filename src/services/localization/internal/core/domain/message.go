package domain

import (
	"time"
)

// Message represents a localization message entry
type Message struct {
	ID               int64     `json:"-" gorm:"column:id;primaryKey;autoIncrement"`
	UUID             string    `json:"uuid" gorm:"column:uuid;type:uuid;uniqueIndex;not null"`
	TenantID         string    `json:"tenantId" gorm:"column:tenant_id;type:varchar(256);not null;uniqueIndex:idx_unique_message,priority:1"`
	Module           string    `json:"module" gorm:"column:module;type:varchar(256);not null;uniqueIndex:idx_unique_message,priority:3"`
	Locale           string    `json:"locale" gorm:"column:locale;type:varchar(256);not null;uniqueIndex:idx_unique_message,priority:2"`
	Code             string    `json:"code" gorm:"column:code;type:varchar(256);not null;uniqueIndex:idx_unique_message,priority:4"`
	Message          string    `json:"message" gorm:"column:message;type:text;not null"`
	CreatedBy        string    `json:"created_by,omitempty" gorm:"column:created_by;type:varchar(256)"`
	CreatedDate      time.Time `json:"created_date,omitempty" gorm:"column:created_date;type:timestamptz;not null"`
	LastModifiedBy   string    `json:"last_modified_by,omitempty" gorm:"column:last_modified_by;type:varchar(256)"`
	LastModifiedDate time.Time `json:"last_modified_date,omitempty" gorm:"column:last_modified_date;type:timestamptz;not null"`
}

// TableName specifies the table name for GORM
func (Message) TableName() string {
	return "localisation"
}
