# Registry Service

A Go-based registry service using Gin framework and PostgreSQL database, inspired by MDMS v2 but with enhanced features including JSON Schema Draft 2020-12 validation and callback functionality.

## Features

- **Schema Management**: Create, read, update, delete JSON schemas with Draft 2020-12 validation, including custom `x-indexes` that control JSONB indexes per schema
- **Data Management**: CRUD operations for schema-bound data with automatic versioning
- **Tenant Isolation**: Multi-tenant support via X-Tenant-ID header
- **Client Tracking**: Client identification via X-Client-ID header
- **Callback Support**: HTTP callbacks for data operations
- **Search Functionality**: JSON-based filtering and pagination
- **PostgreSQL Storage**: Per-schema tables with JSONB columns for efficient querying and isolation

## API Endpoints

### Schema Management
- `POST /registry/v1/schema` - Create a new schema
- `GET /registry/v1/schema/{schemaCode}` - Get schema by code
- `PUT /registry/v1/schema/{schemaCode}` - Update schema
- `DELETE /registry/v1/schema/{schemaCode}` - Delete schema
- `GET /registry/v1/schema` - List all schemas for tenant

### Data Management
- `POST /registry/v1/data/{schemaCode}` - Create data
- `GET /registry/v1/data/{schemaCode}?id={id}` - Get data by ID or unique identifier
- `PUT /registry/v1/data/{schemaCode}/{id}` - Update data
- `DELETE /registry/v1/data/{schemaCode}/{id}` - Delete data
- `POST /registry/v1/data/{schemaCode}/_search` - Search data with filters
- `GET /registry/v1/data/{schemaCode}/_exists?id={id}` - Check if the latest version exists

> Note: The earlier `/history` and `/rollback` endpoints have been removed in favor of immutable versioned rows per schema.

#### Search Filters

The search payload accepts:

```json
{
  "filters": {
    "ownerName": "Alice",
    "status": "active"
  },
  "contains": {
    "address": {
      "city": "Bangalore"
    }
  },
  "limit": 50,
  "offset": 0
}
```

- `filters` applies equality predicates (`data->>'key' = value`).
- `contains` applies JSON containment (`data @> '{"address":{"city":"Bangalore"}}'`), which lets you leverage GIN indexes on JSONB columns.

### Health Check
- `GET /health` - Service health status

## Required Headers

All API requests must include:
- `X-Tenant-ID`: Tenant identifier
- `X-Client-ID`: Client identifier

## Callback Configuration

For data operations, you can configure callbacks using:
- `X-Callback-URL`: URL to call on data changes
- `X-Callback-Auth`: Authorization header for callback
- `callback_method`: HTTP method for callback (default: POST)

## Setup and Running

### Prerequisites
- Go 1.24.2
- PostgreSQL 15+
- Docker & Docker Compose (optional)

### Using Docker Compose (Recommended)

1. Start the services:
```bash
docker-compose up -d
```

2. The service will be available at `http://localhost:8080`

### Manual Setup

1. Install dependencies:
```bash
go mod download
```

2. Set up PostgreSQL database:
```sql
CREATE DATABASE registry_db;
CREATE USER registry_user WITH PASSWORD 'registry_password';
GRANT ALL PRIVILEGES ON DATABASE registry_db TO registry_user;
```

3. Set environment variables:
```bash
export PORT=8080
export DATABASE_URL=postgres://registry_user:registry_password@localhost:5432/registry_db?sslmode=disable
export VAULT_REQUIRED=false # set to true in prod to enforce Vault signer configuration
```

4. Run the service:
```bash
go run ./internal/server
```

## Testing

Run the API test script:
```bash
./test_apis.sh
```

This script tests all major functionality including:
- Schema creation and validation
- Data CRUD operations
- Schema validation (Draft 2020-12)
- Search functionality
- Error handling

## Schema Example

```json
{
  "tenantId": "test-tenant",
  "schemaCode": "user-profile",
  "version": "1.0.0",
  "definition": {
    "$schema": "https://json-schema.org/draft/2020-12/schema",
    "type": "object",
    "properties": {
      "name": {
        "type": "string",
        "minLength": 1
      },
      "email": {
        "type": "string",
        "format": "email"
      },
      "age": {
        "type": "integer",
        "minimum": 0,
        "maximum": 150
      }
    },
    "required": ["name", "email"]
  }
}
```

## Data Example

```json
{
  "tenantId": "test-tenant",
  "schemaCode": "user-profile",
  "uniqueIdentifier": "user-001",
  "data": {
    "name": "John Doe",
    "email": "john.doe@example.com",
    "age": 30
  }
}
```

## Record Versioning

- Each create call generates a fresh `registryId` via the IDGen service and stores version `1` with `effectiveFrom=now` and `effectiveTo=NULL`.
- Updates require the caller to provide the current `version`. The service clones the payload into a new row with `version+1`, marks the previous version inactive, and preserves its `effectiveTo` timestamp for auditing/comparison.
- Deletes simply flip `isActive` to `false` and set `effectiveTo`, allowing the timeline to remain intact without separate history tables.

## Key Differences from MDMS v2

1. **JSON Schema Draft 2020-12**: Uses latest JSON Schema specification
2. **No RequestInfo**: Simplified request structure without DIGIT RequestInfo wrapper
3. **Per-Schema Tables**: Each schema/tenant pair writes to its own physical table, keeping JSONB data isolated instead of dumping everything into a single table, with immutable record versions per registry entry
4. **Enhanced Callbacks**: HTTP callback support for data operations
5. **Header-based Auth**: Uses X-Tenant-ID and X-Client-ID headers instead of request body
6. **Go Implementation**: Built with Go and Gin framework for better performance

## Database Schema

### Schemas Table
- `id`: UUID primary key
- `tenant_id`: Tenant identifier
- `schema_code`: Unique schema code per tenant
- `version`: Schema version
- `definition`: JSONB schema definition
- `is_active`: Soft delete flag
- `created_at`, `updated_at`: Timestamps
- `created_by`, `updated_by`: Client tracking

### Registry Data Table
- `id`: UUID primary key (unique per version)
- `registry_id`: Business identifier generated via the IDGen service and shared by every version of the same record
- `tenant_id`: Tenant identifier
- `schema_code`: Reference to schema
- `schema_version`: Schema definition version applied to the record
- `version`: Immutable record version number (monotonically increasing)
- `unique_identifier`: Optional user-provided identifier
- `data`: JSONB payload
- `is_active`: Indicates whether the version is the live one
- `effective_from` / `effective_to`: Validity window for each version
- `created_at`, `updated_at`, `created_by`, `updated_by`: Audit metadata
- `table naming`: Runtime tables follow the pattern `registry_<tenant>_<schemaCode>` (sanitized to lowercase snake case). Each table contains only the versions for a single tenant+schema, enabling per-schema tuning and indexes.

### Schema-defined Indexes

- Add an `x-indexes` array either inside your JSON Schema definition or alongside the schema payload body:

```json
{
  "schemaCode": "property-versioned",
  "x-indexes": [
    { "fieldPath": "ownerName", "method": "btree" },
    { "fieldPath": "address.city", "method": "gin" }
  ],
  "definition": { ... }
}
```

- Each entry builds an expression index on the corresponding JSON path when the per-schema table is provisioned. Supported methods today are `btree` (default, indexes `data #>> '{path}'`) and `gin` (indexes `data #> '{path}'`).
- Index definitions persist with the schema version so new versions ensure tables have the same indexes.

### ID Generation

Every create operation now calls the IDGen service to obtain a stable `registryId`. Configure the endpoint via `IDGEN_BASE_URL` (default `http://localhost:9091/idgen`), `IDGEN_TEMPLATE_ID` (default `registryId`), and `IDGEN_ORG_VALUE` (default `REGISTRY`). Ensure the template exists in IDGen before starting the registry service; the generated ID stays constant across all subsequent versions.
