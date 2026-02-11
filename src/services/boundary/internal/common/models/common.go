package models

// AuditDetails represents the audit information for entities
// Use string for createdBy/lastModifiedBy and int64 for createdTime/lastModifiedTime
type AuditDetails struct {
	CreatedBy        string `json:"createdBy,omitempty" gorm:"column:createdby"`
	LastModifiedBy   string `json:"lastModifiedBy,omitempty" gorm:"column:lastmodifiedby"`
	CreatedTime      int64  `json:"createdTime,omitempty" gorm:"column:createdtime"`
	LastModifiedTime int64  `json:"lastModifiedTime,omitempty" gorm:"column:lastmodifiedtime"`
}


type Error struct {
	Code        string   `json:"code"`
	Message     string   `json:"message"`
	Description string   `json:"description"`
	Params      []string `json:"params"`
}

type ErrorResponse struct {
	Errors []Error `json:"Errors"`
} 