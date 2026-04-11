package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"registry-service/internal/models"
)

const (
	SubjectTypeData   = "DATA"
	SubjectTypeSchema = "SCHEMA"

	OperationCreate   = "CREATE"
	OperationUpdate   = "UPDATE"
	OperationDelete   = "DELETE"
	OperationRollback = "ROLLBACK"
)

type Auditor interface {
	LogDataEvent(ctx context.Context, event DataEvent) error
	LogSchemaEvent(ctx context.Context, event SchemaEvent) error
}

// SignatureVerifier allows verification of signed audit digests.
type SignatureVerifier interface {
	VerifySignature(ctx context.Context, tenantID string, digest []byte, signature string) (bool, error)
}

type Signer interface {
	Sign(ctx context.Context, tenantID string, digest []byte) (signature string, keyVersion int, algorithm string, err error)
	Verify(ctx context.Context, tenantID string, digest []byte, signature string) (bool, error)
}

type auditRepository interface {
	CreateAuditLog(entry *models.AuditLog) error
	GetLatestAuditLog(tenantID, subjectType, schemaCode string, recordID *uuid.UUID) (*models.AuditLog, error)
}

type Manager struct {
	repo   auditRepository
	signer Signer
}

func NewManager(repo auditRepository, signer Signer) Auditor {
	if repo == nil || signer == nil {
		return &noopAuditor{}
	}
	return &Manager{repo: repo, signer: signer}
}

type DataEvent struct {
	TenantID      string
	SchemaCode    string
	SchemaVersion int
	RecordID      uuid.UUID
	Version       int
	Operation     string
	Actor         string
	Data          json.RawMessage
	PreviousData  json.RawMessage
	Metadata      map[string]interface{}
	Timestamp     time.Time
}

type SchemaEvent struct {
	TenantID      string
	SchemaCode    string
	SchemaVersion int
	Operation     string
	Actor         string
	Definition    json.RawMessage
	Unique        models.UniqueConstraints
	References    []models.RefSchema
	Webhook       *models.WebhookConfig
	Timestamp     time.Time
}

func (m *Manager) LogDataEvent(ctx context.Context, event DataEvent) error {
	if m == nil || m.signer == nil || m.repo == nil {
		return nil
	}

	ts := event.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	payload := dataAuditPayload{
		SubjectType:   SubjectTypeData,
		Operation:     event.Operation,
		TenantID:      event.TenantID,
		SchemaCode:    event.SchemaCode,
		SchemaVersion: event.SchemaVersion,
		RecordID:      event.RecordID.String(),
		Version:       event.Version,
		Actor:         event.Actor,
		Timestamp:     ts,
		Data:          event.Data,
		PreviousData:  event.PreviousData,
		Metadata:      normalizeMetadata(event.Metadata),
	}

	// Normalize raw JSON fields to ensure consistent hashing irrespective of storage reformatting.
	payload.Data = normalizeRawJSON(payload.Data)
	payload.PreviousData = normalizeRawJSON(payload.PreviousData)

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal data audit payload: %w", err)
	}

	previousHash, err := m.previousAuditHash(event.TenantID, SubjectTypeData, event.SchemaCode, event.Operation, &event.RecordID)
	if err != nil {
		return fmt.Errorf("failed to load previous data audit log: %w", err)
	}

	digestInput := buildDigestInput(previousHash, payloadBytes)
	digest := sha256.Sum256(digestInput)
	signature, keyVersion, signatureAlgo, err := m.signer.Sign(ctx, event.TenantID, digest[:])
	if err != nil {
		return fmt.Errorf("failed to sign data audit payload: %w", err)
	}

	recordID := event.RecordID
	recordVersion := event.Version

	logEntry := &models.AuditLog{
		TenantID:       event.TenantID,
		SubjectType:    SubjectTypeData,
		SchemaCode:     event.SchemaCode,
		SchemaVersion:  event.SchemaVersion,
		RecordID:       &recordID,
		Version:        &recordVersion,
		Operation:      event.Operation,
		Actor:          event.Actor,
		EventTimestamp: ts,
		Payload:        payloadBytes,
		PayloadHash:    hex.EncodeToString(digest[:]),
		PreviousHash:   previousHash,
		Signature:      signature,
		SignatureAlgo:  signatureAlgo,
		KeyVersion:     keyVersion,
	}

	if err := m.repo.CreateAuditLog(logEntry); err != nil {
		return fmt.Errorf("failed to persist data audit log: %w", err)
	}

	return nil
}

func (m *Manager) LogSchemaEvent(ctx context.Context, event SchemaEvent) error {
	if m == nil || m.signer == nil || m.repo == nil {
		return nil
	}

	ts := event.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	payload := schemaAuditPayload{
		SubjectType:   SubjectTypeSchema,
		Operation:     event.Operation,
		TenantID:      event.TenantID,
		SchemaCode:    event.SchemaCode,
		SchemaVersion: event.SchemaVersion,
		Actor:         event.Actor,
		Timestamp:     ts,
		Definition:    event.Definition,
		Unique:        event.Unique,
		References:    event.References,
		Webhook:       event.Webhook,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal schema audit payload: %w", err)
	}

	previousHash, err := m.previousAuditHash(event.TenantID, SubjectTypeSchema, event.SchemaCode, event.Operation, nil)
	if err != nil {
		return fmt.Errorf("failed to load previous schema audit log: %w", err)
	}

	digestInput := buildDigestInput(previousHash, payloadBytes)
	digest := sha256.Sum256(digestInput)
	signature, keyVersion, signatureAlgo, err := m.signer.Sign(ctx, event.TenantID, digest[:])
	if err != nil {
		return fmt.Errorf("failed to sign schema audit payload: %w", err)
	}

	logEntry := &models.AuditLog{
		TenantID:       event.TenantID,
		SubjectType:    SubjectTypeSchema,
		SchemaCode:     event.SchemaCode,
		SchemaVersion:  event.SchemaVersion,
		Operation:      event.Operation,
		Actor:          event.Actor,
		EventTimestamp: ts,
		Payload:        payloadBytes,
		PayloadHash:    hex.EncodeToString(digest[:]),
		PreviousHash:   previousHash,
		Signature:      signature,
		SignatureAlgo:  signatureAlgo,
		KeyVersion:     keyVersion,
	}

	if err := m.repo.CreateAuditLog(logEntry); err != nil {
		return fmt.Errorf("failed to persist schema audit log: %w", err)
	}

	return nil
}

func (m *Manager) previousAuditHash(tenantID, subjectType, schemaCode, operation string, recordID *uuid.UUID) (string, error) {
	if operation == OperationCreate || m.repo == nil {
		return "", nil
	}

	prev, err := m.repo.GetLatestAuditLog(tenantID, subjectType, schemaCode, recordID)
	if err != nil {
		return "", err
	}
	if prev == nil {
		return "", nil
	}
	return prev.PayloadHash, nil
}

func buildDigestInput(previousHash string, payload []byte) []byte {
	if previousHash == "" {
		return payload
	}

	prevBytes, err := hex.DecodeString(previousHash)
	if err != nil {
		prevBytes = []byte(previousHash)
	}

	digestInput := make([]byte, 0, len(prevBytes)+len(payload))
	digestInput = append(digestInput, prevBytes...)
	digestInput = append(digestInput, payload...)
	return digestInput
}

// ComputePayloadDigest returns the digest bytes and hex string for the provided payload.
func ComputePayloadDigest(previousHash string, payload []byte) ([]byte, string) {
	digestInput := buildDigestInput(previousHash, payload)
	sum := sha256.Sum256(digestInput)
	encoded := hex.EncodeToString(sum[:])
	digest := make([]byte, len(sum))
	copy(digest, sum[:])
	return digest, encoded
}

// ComputeDataPayloadDigest normalizes a data audit payload before hashing so the
// digest matches the originally signed bytes even if the JSON storage reorders fields.
func ComputeDataPayloadDigest(previousHash string, payload []byte) ([]byte, string, error) {
	var parsed dataAuditPayload
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return nil, "", fmt.Errorf("failed to parse data audit payload: %w", err)
	}
	parsed.Data = normalizeRawJSON(parsed.Data)
	parsed.PreviousData = normalizeRawJSON(parsed.PreviousData)
	normalized, err := json.Marshal(parsed)
	if err != nil {
		return nil, "", fmt.Errorf("failed to normalize data audit payload: %w", err)
	}
	digest, hash := ComputePayloadDigest(previousHash, normalized)
	return digest, hash, nil
}

func normalizeRawJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return raw
	}
	var val interface{}
	if err := json.Unmarshal(raw, &val); err != nil {
		return raw
	}
	normalized, err := json.Marshal(val)
	if err != nil {
		return raw
	}
	return normalized
}

// VerifySignature validates the provided digest/signature pair for a tenant if signer supports it.
func (m *Manager) VerifySignature(ctx context.Context, tenantID string, digest []byte, signature string) (bool, error) {
	if m == nil || m.signer == nil {
		return false, fmt.Errorf("signature verifier is not configured")
	}
	return m.signer.Verify(ctx, tenantID, digest, signature)
}

type dataAuditPayload struct {
	SubjectType   string          `json:"subjectType"`
	Operation     string          `json:"operation"`
	TenantID      string          `json:"tenantId"`
	SchemaCode    string          `json:"schemaCode"`
	SchemaVersion int             `json:"schemaVersion"`
	RecordID      string          `json:"recordId"`
	Version       int             `json:"version"`
	Actor         string          `json:"actor"`
	Timestamp     time.Time       `json:"timestamp"`
	Data          json.RawMessage `json:"data"`
	PreviousData  json.RawMessage `json:"previousData,omitempty"`
	Metadata      []metadataEntry `json:"metadata,omitempty"`
}

type schemaAuditPayload struct {
	SubjectType   string                   `json:"subjectType"`
	Operation     string                   `json:"operation"`
	TenantID      string                   `json:"tenantId"`
	SchemaCode    string                   `json:"schemaCode"`
	SchemaVersion int                      `json:"schemaVersion"`
	Actor         string                   `json:"actor"`
	Timestamp     time.Time                `json:"timestamp"`
	Definition    json.RawMessage          `json:"definition"`
	Unique        models.UniqueConstraints `json:"x-unique,omitempty"`
	References    []models.RefSchema       `json:"x-ref-schema,omitempty"`
	Webhook       *models.WebhookConfig    `json:"webhook,omitempty"`
}

type metadataEntry struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

func normalizeMetadata(metadata map[string]interface{}) []metadataEntry {
	if len(metadata) == 0 {
		return nil
	}

	keys := make([]string, 0, len(metadata))
	for k := range metadata {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	entries := make([]metadataEntry, 0, len(keys))
	for _, k := range keys {
		entries = append(entries, metadataEntry{Key: k, Value: metadata[k]})
	}
	return entries
}
