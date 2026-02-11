package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// TemplateConfigDB is the database model that matches the table schema
type TemplateConfigDB struct {
	ID               uuid.UUID      `gorm:"column:id;type:uuid;primary_key"`
	TemplateID       string         `gorm:"column:templateid;not null"`
	Version          int            `gorm:"column:version;not null"`
	TenantID         string         `gorm:"column:tenantid;not null"`
	FieldMapping     FieldMapping   `gorm:"column:fieldmapping;type:jsonb"`
	APIMapping       APIMappingList `gorm:"column:apimapping;type:jsonb"`
	CreatedBy        string         `gorm:"column:createdby"`
	LastModifiedBy   string         `gorm:"column:lastmodifiedby"`
	CreatedTime      int64          `gorm:"column:createdtime"`
	LastModifiedTime int64          `gorm:"column:lastmodifiedtime"`
}

func (TemplateConfigDB) TableName() string {
	return "template_config"
}

// Implement custom scanner and valuer for []APIMapping

// Value implements the driver.Valuer interface
func (a APIMappingList) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan implements the sql.Scanner interface
type APIMappingList []APIMapping

func (a *APIMappingList) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	// Try to unmarshal as an array
	var arr []APIMapping
	if err := json.Unmarshal(bytes, &arr); err == nil {
		*a = arr
		return nil
	}

	// Fallback: try to unmarshal as single object
	var single APIMapping
	if err := json.Unmarshal(bytes, &single); err == nil {
		*a = []APIMapping{single}
		return nil
	}

	return errors.New("failed to unmarshal APIMappingList")
}

type FieldMapping map[string]string

func (f *FieldMapping) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed for FieldMapping")
	}
	return json.Unmarshal(bytes, f)
}

func (f FieldMapping) Value() (driver.Value, error) {
	return json.Marshal(f)
}

// ToDTO converts TemplateConfigDB to TemplateConfigWithVersion (DB to API)
func (tc *TemplateConfigDB) ToDTO() TemplateConfigWithVersion {
	return TemplateConfigWithVersion{
		ID:           tc.ID,
		TemplateID:   tc.TemplateID,
		TenantID:     tc.TenantID,
		Version:      fmt.Sprintf("v%d", tc.Version),
		FieldMapping: tc.FieldMapping,
		APIMapping:   tc.APIMapping,
		AuditDetails: AuditDetails{
			CreatedBy:        tc.CreatedBy,
			CreatedTime:      tc.CreatedTime,
			LastModifiedBy:   tc.LastModifiedBy,
			LastModifiedTime: tc.LastModifiedTime,
		},
	}
}
