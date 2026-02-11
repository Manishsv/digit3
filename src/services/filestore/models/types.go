package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type StringArray []string

// Value marshals the array into a JSON-encoded []byte for storing in DB
func (a StringArray) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan unmarshals a JSON-encoded value from the DB into the array
func (a *StringArray) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, a)
}
