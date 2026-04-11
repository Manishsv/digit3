CREATE TABLE IF NOT EXISTS registry_data_histories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    record_id UUID NOT NULL,
    tenant_id TEXT NOT NULL,
    schema_code TEXT NOT NULL,
    schema_version INTEGER NOT NULL,
    record_version INTEGER NOT NULL,
    unique_identifier TEXT,
    data JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT
);

CREATE INDEX IF NOT EXISTS idx_registry_data_histories_lookup
    ON registry_data_histories (tenant_id, schema_code, record_id, record_version);
