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
- `V20250917141530__create_boundary_table.sql` - boundary table creation
- `V20250917142045__create_boundary_hierarchy_table.sql` - boundary_hierarchy table creation
- `V20250917143210__create_boundary_relationship_table.sql` - boundary_relationship table creation

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
cd boundary/db

# Build migration image
docker build -t boundary-flyway:dev .

# Run migrations
docker run --rm \
  -e DB_HOST=<your-db-host> \
  -e DB_PORT=5432 \
  -e DB_NAME=<your-db-name> \
  -e DB_USER=<your-db-user> \
  -e DB_PASSWORD=<your-db-password> \
  boundary-flyway:dev

```


### Adding New Migrations

1. Create a new SQL file following Flyway naming convention:
   ```
   db/migrations/V4__Your_migration_description.sql
   ```

2. Write your SQL migration:
   ```sql
   -- Add your migration SQL here
   ALTER TABLE boundary ADD COLUMN new_field VARCHAR(255);
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