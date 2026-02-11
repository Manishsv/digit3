package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type AuditDetails struct {
	CreatedBy    string `json:"createdBy"`
	CreatedTime  string `json:"createdTime"`
	ModifiedTime string `json:"modifiedTime"`
	ModifiedBy   string `json:"modifiedBy"`
}

// Scan implements the sql.Scanner interface
func (a *AuditDetails) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, a)
}

// Value implements the driver.Valuer interface
func (a AuditDetails) Value() (driver.Value, error) {
	return json.Marshal(a)
}
