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
- `V20250923144005__create_idgen_templates_table.sql` - templates table creation
- `V20250923144530__create_idgen_sequence_reset_table.sql` - idgen_sequence_resets table creation

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
cd idgen/db

# Build migration image
docker build -t idgen-flyway:dev .

# Run migrations
docker run --rm \
  -e DB_HOST=<your-db-host> \
  -e DB_PORT=5432 \
  -e DB_NAME=<your-db-name> \
  -e DB_USER=<your-db-user> \
  -e DB_PASSWORD=<your-db-password> \
  idgen-flyway:dev



### Adding New Migrations

1. Create a new SQL file following Flyway naming convention:
   ```
   db/migrations/V4__Your_migration_description.sql
   ```

2. Write your SQL migration:
   ```sql
   -- Add your migration SQL here
   ALTER TABLE generated ADD COLUMN new_field VARCHAR(255);
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