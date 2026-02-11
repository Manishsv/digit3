-- Create the localisation table
CREATE TABLE IF NOT EXISTS localisation (
    id BIGSERIAL PRIMARY KEY,
    tenant_id VARCHAR(256) NOT NULL,
    module VARCHAR(256) NOT NULL,
    locale VARCHAR(256) NOT NULL,
    code VARCHAR(256) NOT NULL,
    message TEXT NOT NULL,
    created_by BIGINT,
    created_date TIMESTAMP WITH TIME ZONE NOT NULL,
    last_modified_by BIGINT,
    last_modified_date TIMESTAMP WITH TIME ZONE NOT NULL,
    UNIQUE(tenant_id, locale, module, code)
);

-- Add indexes for frequent queries
CREATE INDEX IF NOT EXISTS idx_localisation_tenant_module_locale ON localisation(tenant_id, module, locale);
CREATE INDEX IF NOT EXISTS idx_localisation_tenant_locale ON localisation(tenant_id, locale); 