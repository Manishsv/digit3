package models

import (
	"fmt"

	"github.com/google/uuid"
)

// TemplateDB is the database model that matches the table schema
type TemplateDB struct {
	ID               uuid.UUID `gorm:"column:id;type:uuid;primary_key"`
	TemplateID       string    `gorm:"column:templateid;not null"`
	Version          int       `gorm:"column:version;not null"`
	TenantID         string    `gorm:"column:tenantid;not null"`
	Type             string    `gorm:"column:type;not null"`
	Subject          string    `gorm:"column:subject;type:text"`
	Content          string    `gorm:"column:content;type:text;not null"`
	IsHTML           bool      `gorm:"column:ishtml"`
	CreatedBy        string    `gorm:"column:createdby"`
	LastModifiedBy   string    `gorm:"column:lastmodifiedby"`
	CreatedTime      int64     `gorm:"column:createdtime"`
	LastModifiedTime int64     `gorm:"column:lastmodifiedtime"`
}

func (TemplateDB) TableName() string {
	return "notification_template"
}

// ToDTO converts TemplateDB to Template (DB to API)
func (t *TemplateDB) ToDTO() TemplateWithVersion {
	return TemplateWithVersion{
		ID:         t.ID,
		TemplateID: t.TemplateID,
		TenantID:   t.TenantID,
		Version:    fmt.Sprintf("v%d", t.Version),
		Type:       TemplateType(t.Type),
		Subject:    t.Subject,
		Content:    t.Content,
		IsHTML:     t.IsHTML,
		AuditDetails: AuditDetails{
			CreatedBy:        t.CreatedBy,
			CreatedTime:      t.CreatedTime,
			LastModifiedBy:   t.LastModifiedBy,
			LastModifiedTime: t.LastModifiedTime,
		},
	}
}
