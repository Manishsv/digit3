-- Create the template_config table
CREATE TABLE IF NOT EXISTS template_config (
    id UUID PRIMARY KEY,
    templateid VARCHAR(256) NOT NULL,
    tenantid VARCHAR(256) NOT NULL,
    version INTEGER NOT NULL CHECK (version > 0),
    fieldmapping JSONB,
    apimapping JSONB,
    createdby VARCHAR(64),
    lastmodifiedby VARCHAR(64),
    createdtime BIGINT,
    lastmodifiedtime BIGINT,
    UNIQUE (tenantid, templateid, version)
);

-- Index to speed up queries for "latest version per template"
CREATE INDEX IF NOT EXISTS idx_template_config_tenant_template_version_desc
ON template_config (tenantid, templateid, version DESC);