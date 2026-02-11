package models

import (
	"github.com/google/uuid"
)

type IDGenSequenceReset struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantID     string    `gorm:"column:tenantid;index"`
	TemplateCode string    `gorm:"column:templatecode;index"`
	ScopeKey     string    `gorm:"column:scopekey"`
	LastValue    int64     `gorm:"column:lastvalue"`
}

func (IDGenSequenceReset) TableName() string {
	return "idgen_sequence_resets"
}
