CREATE TABLE IF NOT EXISTS registry_data (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id TEXT NOT NULL,
    schema_code TEXT NOT NULL,
    unique_identifier TEXT,
    schema_version INTEGER NOT NULL,
    record_version INTEGER NOT NULL DEFAULT 1,
    data JSONB NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT,
    updated_by TEXT
);

CREATE INDEX IF NOT EXISTS idx_registry_data_tenant_schema ON registry_data (tenant_id, schema_code);
CREATE INDEX IF NOT EXISTS idx_registry_data_unique_identifier ON registry_data (unique_identifier);
CREATE INDEX IF NOT EXISTS idx_registry_data_json ON registry_data USING GIN (data);
