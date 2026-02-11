package models

import (
	"github.com/google/uuid"
)

type IDGenSequenceLookup struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	SeqName      string    `gorm:"column:seqname;uniqueIndex"`
	TenantID     string    `gorm:"column:tenantid;index"`
	TemplateCode string    `gorm:"column:templatecode;index"`
}

func (IDGenSequenceLookup) TableName() string {
	return "idgen_sequence_lookup"
}
