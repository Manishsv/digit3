package models

type AuditDetails struct {
	CreatedBy        string `json:"createdBy,omitempty"`
	LastModifiedBy   string `json:"lastModifiedBy,omitempty"`
	CreatedTime      int64  `json:"createdTime,omitempty"`
	LastModifiedTime int64  `json:"lastModifiedTime,omitempty"`
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