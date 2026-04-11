CREATE TABLE IF NOT EXISTS schemas (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id TEXT NOT NULL,
    schema_code TEXT NOT NULL,
    version INTEGER NOT NULL,
    definition JSONB NOT NULL,
    x_unique JSONB,
    x_ref_schema JSONB,
    webhook JSONB,
    is_latest BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT,
    updated_by TEXT
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tenant_schema_version ON schemas (tenant_id, schema_code, version);
CREATE INDEX IF NOT EXISTS idx_schemas_x_unique ON schemas USING GIN (x_unique);
CREATE INDEX IF NOT EXISTS idx_schemas_x_ref_schema ON schemas USING GIN (x_ref_schema);
