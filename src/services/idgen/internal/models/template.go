package models

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type IDGenTemplateDB struct {
	ID               uuid.UUID      `gorm:"column:id;type:uuid;primaryKey"`
	TenantID         string         `gorm:"column:tenantid;not null;size:64"`
	TemplateCode     string         `gorm:"column:templatecode;not null;size:64"`
	Version          int            `gorm:"column:version;not null"`
	Config           datatypes.JSON `gorm:"column:config;type:jsonb;not null"`
	CreatedTime      int64          `gorm:"column:createdtime"`
	CreatedBy        string         `gorm:"column:createdby;size:64"`
	LastModifiedTime int64          `gorm:"column:lastmodifiedtime"`
	LastModifiedBy   string         `gorm:"column:lastmodifiedby;size:64"`
}

// TableName overrides the default (id_gen_templates) to match your table
func (IDGenTemplateDB) TableName() string {
	return "idgen_templates"
}

// ToDTO converts TemplateDB to Template (DB to API)
func (t *IDGenTemplateDB) ToDTO() (IDGenTemplateWithVersion, error) {
	var cfg IDGenTemplateConfig
	if err := json.Unmarshal(t.Config, &cfg); err != nil {
		return IDGenTemplateWithVersion{}, err
	}

	return IDGenTemplateWithVersion{
		ID:           t.ID,
		TemplateCode: t.TemplateCode,
		TenantID:     t.TenantID,
		Version:      fmt.Sprintf("v%d", t.Version),
		Config:       cfg,
		AuditDetails: AuditDetails{
			CreatedBy:        t.CreatedBy,
			CreatedTime:      t.CreatedTime,
			LastModifiedBy:   t.LastModifiedBy,
			LastModifiedTime: t.LastModifiedTime,
		},
	}, nil
}
