package repository

import (
	"fmt"
	"regexp"
	"strings"

	"registry-service/internal/models"
)

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

// registryTableName returns the physical table name used for a schema/tenant pair.
func registryTableName(tenantID, schemaCode string) string {
	return fmt.Sprintf("registry_%s_%s", sanitizeIdentifier(tenantID), sanitizeIdentifier(schemaCode))
}

func sanitizeIdentifier(value string) string {
	value = strings.ToLower(value)
	value = nonAlphanumeric.ReplaceAllString(value, "_")
	value = strings.Trim(value, "_")
	if value == "" {
		return "default"
	}
	return value
}

func quoteIdentifier(identifier string) string {
	return fmt.Sprintf("\"%s\"", strings.ReplaceAll(identifier, "\"", "\"\""))
}

func (r *registryRepository) EnsureDataTable(tenantID, schemaCode string, indexes []models.SchemaIndex) (string, error) {
	tableName := registryTableName(tenantID, schemaCode)
	quotedTable := quoteIdentifier(tableName)
	ddl := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
	id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
	registry_id TEXT NOT NULL,
	tenant_id TEXT NOT NULL,
	schema_code TEXT NOT NULL,
	schema_version INTEGER NOT NULL,
	version INTEGER NOT NULL,
	data JSONB NOT NULL,
	is_active BOOLEAN NOT NULL DEFAULT TRUE,
	effective_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	effective_to TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	created_by TEXT,
	updated_by TEXT
);

CREATE INDEX IF NOT EXISTS %s ON %s (tenant_id, schema_code);
CREATE INDEX IF NOT EXISTS %s ON %s (registry_id);
CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s (registry_id, version);
`,
		quotedTable,
		quoteIdentifier(tableName+"_tenant_schema_idx"), quotedTable,
		quoteIdentifier(tableName+"_registry_idx"), quotedTable,
		quoteIdentifier(tableName+"_registry_version_uidx"), quotedTable,
	)

	if err := r.db.Exec(ddl).Error; err != nil {
		return "", err
	}

	if err := r.ensureCustomIndexes(tableName, indexes); err != nil {
		return "", err
	}
	return tableName, nil
}

func (r *registryRepository) tableNameForSchema(tenantID, schemaCode string) (string, error) {
	return r.EnsureDataTable(tenantID, schemaCode, nil)
}

func (r *registryRepository) ensureCustomIndexes(tableName string, indexes []models.SchemaIndex) error {
	if len(indexes) == 0 {
		return nil
	}
	for _, idx := range indexes {
		path := strings.TrimSpace(idx.FieldPath)
		if path == "" {
			continue
		}
		method := strings.ToLower(strings.TrimSpace(idx.Method))
		if method == "" {
			method = "btree"
		}
		expr, err := buildJSONIndexExpression(path, method)
		if err != nil {
			return err
		}
		name := idx.Name
		if strings.TrimSpace(name) == "" {
			sanitizedPath := sanitizeIdentifier(strings.ReplaceAll(path, ".", "_"))
			name = fmt.Sprintf("%s_%s_%s_idx", tableName, sanitizedPath, method)
		}
		statement := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s USING %s %s;",
			quoteIdentifier(name), quoteIdentifier(tableName), strings.ToUpper(method), expr,
		)
		if err := r.db.Exec(statement).Error; err != nil {
			return fmt.Errorf("failed to create index %s: %w", name, err)
		}
	}
	return nil
}

func buildJSONIndexExpression(path string, method string) (string, error) {
	pgPath := postgresPathLiteral(path)
	switch method {
	case "gin":
		return fmt.Sprintf("((data #> '%s'))", pgPath), nil
	case "btree":
		return fmt.Sprintf("((data #>> '%s'))", pgPath), nil
	default:
		return "", fmt.Errorf("unsupported index method '%s'", method)
	}
}

func postgresPathLiteral(path string) string {
	parts := strings.Split(path, ".")
	for i, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.ReplaceAll(part, "\"", "\"\"")
		parts[i] = fmt.Sprintf("\"%s\"", part)
	}
	return fmt.Sprintf("{%s}", strings.Join(parts, ","))
}
