package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Schema struct {
	ID           uuid.UUID         `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID     string            `json:"-" gorm:"not null;uniqueIndex:idx_tenant_schema_version" binding:"required"`
	SchemaCode   string            `json:"schemaCode" gorm:"not null;uniqueIndex:idx_tenant_schema_version" binding:"required"`
	Version      int               `json:"version" gorm:"not null;uniqueIndex:idx_tenant_schema_version"`
	Definition   json.RawMessage   `json:"definition" gorm:"type:jsonb;not null" binding:"required"`
	XUnique      UniqueConstraints `json:"x-unique,omitempty" gorm:"type:jsonb;serializer:json"`
	XRefSchema   []RefSchema       `json:"x-ref-schema,omitempty" gorm:"type:jsonb;serializer:json"`
	XIndexes     []SchemaIndex     `json:"x-indexes,omitempty" gorm:"type:jsonb;serializer:json"`
	Webhook      *WebhookConfig    `json:"webhook,omitempty" gorm:"type:jsonb;serializer:json"`
	IsLatest     bool              `json:"isLatest" gorm:"default:false"`
	IsActive     bool              `json:"isActive" gorm:"default:true"`
	AuditDetails AuditDetails      `json:"auditDetails" gorm:"-"`
	CreatedAt    time.Time         `json:"-" gorm:"autoCreateTime"`
	UpdatedAt    time.Time         `json:"-" gorm:"autoUpdateTime"`
	CreatedBy    string            `json:"-" gorm:"column:created_by"`
	UpdatedBy    string            `json:"-" gorm:"column:updated_by"`
}

type RegistryData struct {
	ID            uuid.UUID       `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	RegistryID    string          `json:"registryId" gorm:"not null;index"`
	TenantID      string          `json:"-" gorm:"not null;index" binding:"required"`
	SchemaCode    string          `json:"schemaCode" gorm:"not null;index" binding:"required"`
	SchemaVersion int             `json:"schemaVersion" gorm:"not null"`
	Version       int             `json:"version" gorm:"not null"`
	Data          json.RawMessage `json:"data" gorm:"type:jsonb;not null" binding:"required"`
	IsActive      bool            `json:"isActive" gorm:"default:true"`
	EffectiveFrom time.Time       `json:"effectiveFrom" gorm:"not null"`
	EffectiveTo   *time.Time      `json:"effectiveTo,omitempty"`
	AuditDetails  AuditDetails    `json:"auditDetails" gorm:"-"`
	CreatedAt     time.Time       `json:"-" gorm:"autoCreateTime"`
	UpdatedAt     time.Time       `json:"-" gorm:"autoUpdateTime"`
	CreatedBy     string          `json:"-" gorm:"column:created_by"`
	UpdatedBy     string          `json:"-" gorm:"column:updated_by"`
}

type SchemaRequest struct {
	TenantID   string            `json:"-"`
	SchemaCode string            `json:"schemaCode" binding:"required"`
	Definition json.RawMessage   `json:"definition" binding:"required"`
	XUnique    UniqueConstraints `json:"x-unique,omitempty"`
	XRefSchema []RefSchema       `json:"x-ref-schema,omitempty"`
	XIndexes   []SchemaIndex     `json:"x-indexes,omitempty"`
	Webhook    *WebhookConfig    `json:"webhook,omitempty"`
}

type SchemaIndex struct {
	Name      string `json:"name,omitempty"`
	FieldPath string `json:"fieldPath"`
	Method    string `json:"method,omitempty"`
}

type RefSchema struct {
	FieldPath  string `json:"fieldPath"`
	SchemaCode string `json:"schemaCode"`
	RefField   string `json:"refField,omitempty"`
	External   bool   `json:"external,omitempty"`
	Registry   string `json:"registry,omitempty"`
}

type IsExistRequest struct {
	TenantID string `json:"tenantId"`
	Field    string `json:"field"`
	Value    string `json:"value"`
}

type UniqueConstraint []string

type UniqueConstraints []UniqueConstraint

func (uc *UniqueConstraints) UnmarshalJSON(data []byte) error {
	if uc == nil {
		return fmt.Errorf("unique constraints: nil receiver")
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*uc = nil
		return nil
	}

	var multi [][]string
	if err := json.Unmarshal(trimmed, &multi); err == nil {
		*uc = buildUniqueConstraintsFromMulti(multi)
		return nil
	}

	var simple []string
	if err := json.Unmarshal(trimmed, &simple); err == nil {
		*uc = buildUniqueConstraintsFromSimple(simple)
		return nil
	}

	var objects []struct {
		Fields []string `json:"fields"`
	}
	if err := json.Unmarshal(trimmed, &objects); err == nil {
		constraints := make(UniqueConstraints, 0, len(objects))
		for _, obj := range objects {
			fields := normalizeConstraintFields(obj.Fields)
			if len(fields) == 0 {
				continue
			}
			constraints = append(constraints, UniqueConstraint(fields))
		}
		*uc = constraints
		return nil
	}

	return fmt.Errorf("invalid x-unique format")
}

func buildUniqueConstraintsFromMulti(groups [][]string) UniqueConstraints {
	constraints := make(UniqueConstraints, 0, len(groups))
	for _, group := range groups {
		fields := normalizeConstraintFields(group)
		if len(fields) == 0 {
			continue
		}
		constraints = append(constraints, UniqueConstraint(fields))
	}
	if len(constraints) == 0 {
		return nil
	}
	return constraints
}

func buildUniqueConstraintsFromSimple(fields []string) UniqueConstraints {
	constraints := make(UniqueConstraints, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		constraints = append(constraints, UniqueConstraint{field})
	}
	if len(constraints) == 0 {
		return nil
	}
	return constraints
}

func normalizeConstraintFields(fields []string) []string {
	if len(fields) == 0 {
		return nil
	}
	clean := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		clean = append(clean, field)
	}
	if len(clean) == 0 {
		return nil
	}
	slice := make([]string, len(clean))
	copy(slice, clean)
	return slice
}

type WebhookConfig struct {
	URL     string            `json:"url"`
	Method  string            `json:"method,omitempty"`
	ApiKey  string            `json:"apiKey,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Active  bool              `json:"active"`
}

type DataRequest struct {
	TenantID   string          `json:"-"`
	SchemaCode string          `json:"-"`
	Version    int             `json:"version,omitempty"`
	Data       json.RawMessage `json:"data" binding:"required"`
}

type SearchRequest struct {
	TenantID   string                 `json:"-"`
	SchemaCode string                 `json:"-"`
	Filters    map[string]interface{} `json:"filters,omitempty"`
	Contains   map[string]interface{} `json:"contains,omitempty"`
	Limit      int                    `json:"limit,omitempty"`
	Offset     int                    `json:"offset,omitempty"`
}

type CallbackConfig struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers,omitempty"`
}

type AuditDetails struct {
	CreatedBy    string `json:"createdBy"`
	CreatedTime  int64  `json:"createdTime"`
	ModifiedBy   string `json:"modifiedBy"`
	ModifiedTime int64  `json:"modifiedTime"`
}

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

type AuditLog struct {
	ID             uuid.UUID       `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID       string          `json:"tenantId" gorm:"index"`
	SubjectType    string          `json:"subjectType" gorm:"index"`
	SchemaCode     string          `json:"schemaCode,omitempty" gorm:"index"`
	SchemaVersion  int             `json:"schemaVersion,omitempty"`
	RecordID       *uuid.UUID      `json:"recordId,omitempty" gorm:"type:uuid"`
	Version        *int            `json:"version,omitempty" gorm:"column:record_version"`
	Operation      string          `json:"operation" gorm:"index"`
	Actor          string          `json:"actor"`
	EventTimestamp time.Time       `json:"eventTimestamp"`
	Payload        json.RawMessage `json:"payload" gorm:"type:jsonb"`
	PayloadHash    string          `json:"payloadHash"`
	PreviousHash   string          `json:"previousHash,omitempty"`
	Signature      string          `json:"signature"`
	SignatureAlgo  string          `json:"signatureAlgo"`
	KeyVersion     int             `json:"keyVersion"`
	CreatedAt      time.Time       `json:"createdAt" gorm:"autoCreateTime"`
}

func (s *Schema) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

func (r *RegistryData) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	if r.Version == 0 {
		r.Version = 1
	}
	if r.EffectiveFrom.IsZero() {
		r.EffectiveFrom = time.Now().UTC()
	}
	return nil
}

func (s *Schema) PopulateAuditDetails() {
	s.AuditDetails = AuditDetails{
		CreatedBy:    s.CreatedBy,
		CreatedTime:  unixMillis(s.CreatedAt),
		ModifiedBy:   s.UpdatedBy,
		ModifiedTime: unixMillis(s.UpdatedAt),
	}
}

func (r *RegistryData) PopulateAuditDetails() {
	r.AuditDetails = AuditDetails{
		CreatedBy:    r.CreatedBy,
		CreatedTime:  unixMillis(r.CreatedAt),
		ModifiedBy:   r.UpdatedBy,
		ModifiedTime: unixMillis(r.UpdatedAt),
	}
}

func unixMillis(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixNano() / int64(time.Millisecond)
}
