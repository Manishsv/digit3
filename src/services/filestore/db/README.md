# Database Migration Directory

This directory contains all database migration-related files and tools.

## Structure

```
db/
├── migrations/          # SQL migration files (Flyway naming convention)
├── config/             # Flyway configuration files
├── tools/              # Downloaded tools (Flyway installation)
└── README.md           # This file
```

## Files

### Migrations (`db/migrations/`)
- `V20250917153205__egfilestore_tenant_ddl.up.sql` - eg_filestoremap table creation
- `V20250917153730__eg_filestore_alter_ddl.up.sql` - add column filesource
- `V20250917154255__egfilestore_filename_dml.up.sql` - ALTER COLUMN filename TYPE
- `V20250917154820__egfilestore_audit_details.up.sql` - add columns for audit details
- `V20250917155345__egfilestore_seq_id.up.sql` - set default value for id
- `V20250917155910__eg_doc_metadata.up.sql` - eg_doc_metadata table creation

### Configuration (`db/config/`)
- `flyway.conf` - Flyway configuration for local development
- `flyway-docker.conf` - Flyway configuration for Docker container

### Tools (`db/tools/`)
- `flyway/` - Flyway CLI installation (auto-downloaded by script)

## Usage

### Local Development
```bash
# Run migrations
./scripts/migrate.sh

# Check status
./scripts/migrate.sh info
```

### Docker
```bash
cd filestore/db

# Build migration image
docker build -t filestore:dev .

# Run migrations
docker run --rm \
  -e DB_HOST=<your-db-host> \
  -e DB_PORT=5432 \
  -e DB_NAME=<your-db-name> \
  -e DB_USER=<your-db-user> \
  -e DB_PASSWORD=<your-db-password> \
  filestore:dev

```


### Adding New Migrations

1. Create a new SQL file following Flyway naming convention:
   ```
   db/migrations/V4__Your_migration_description.sql
   ```

2. Write your SQL migration:
   ```sql
   -- Add your migration SQL here
   ALTER TABLE eg_filestoremap ADD COLUMN new_field VARCHAR(255);
   ```

3. Run the migration:
   ```bash
   ./scripts/migrate.sh
   ```

## Notes

- All migration files use Flyway naming convention: `V[YEAR][MONTH][DAY][HR][MIN][SEC]__modulecode_…_ddl.sql`
- Migrations are executed in version order
- The system supports out-of-order migrations for compatibility
- Configuration files are optimized for Flyway Community Edition 