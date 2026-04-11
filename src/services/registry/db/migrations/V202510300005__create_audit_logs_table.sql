CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id TEXT,
    subject_type TEXT,
    schema_code TEXT,
    schema_version INTEGER,
    record_id UUID,
    record_version INTEGER,
    unique_identifier TEXT,
    operation TEXT,
    actor TEXT,
    event_timestamp TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL,
    payload_hash TEXT NOT NULL,
    previous_hash TEXT,
    signature TEXT NOT NULL,
    signature_algo TEXT NOT NULL,
    key_version INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_id ON audit_logs (tenant_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_subject_type ON audit_logs (subject_type);
CREATE INDEX IF NOT EXISTS idx_audit_logs_schema_code ON audit_logs (schema_code);
CREATE INDEX IF NOT EXISTS idx_audit_logs_operation ON audit_logs (operation);
