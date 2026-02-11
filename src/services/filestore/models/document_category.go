package models

type DocumentCategory struct {
	ID             uint64       `gorm:"column:id;type:bigint;primaryKey;autoIncrement"`
	Type           string       `json:"type" binding:"required" gorm:"column:type"`
	TenantId       string       `gorm:"column:tenantid"`
	Code           string       `json:"code" binding:"required" gorm:"column:code"`
	AllowedFormats StringArray  `json:"allowedFormats" gorm:"column:allowedformats;type:json" binding:"required"`
	MinSize        string       `json:"minSize" gorm:"column:minsize"`
	MaxSize        string       `json:"maxSize" gorm:"column:maxsize"`
	IsSensitive    bool         `json:"isSensitive" gorm:"column:issensitive"`
	Description    string       `json:"description" gorm:"column:description"`
	IsActive       bool         `json:"isActive" gorm:"column:isactive"`
	AuditDetails   AuditDetails `json:"auditDetails" gorm:"type:json;column:auditdetail"`
}

func (*DocumentCategory) TableName() string {
	return "eg_doc_metadata_v2"
}
