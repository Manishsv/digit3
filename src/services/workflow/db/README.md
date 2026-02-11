# Database Migration Directory

This directory contains all database migration-related files and tools for the Workflow Service.

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
- `V1__Create_initial_schema.sql` - Initial workflow database schema
- `V2__Remove_unique_constraint_process_instances.sql` - Enable audit trail
- `V3__Add_sample_test_column.sql` - Compatibility migration (no-op)
- `V4__Add_parallel_workflow_support.sql` - Parallel workflow execution support
- `V5__Remove_roles_column_from_actions.sql` - Remove deprecated roles column
- `V6__Create_escalation_configs.sql` - Auto-escalation configuration table
- `V7__Remove_is_active_from_escalation_configs.sql` - Remove redundant column
- `V8__Add_escalated_field_to_process_instances.sql` - Add escalation tracking

### Configuration (`db/config/`)
- `flyway.conf` - Flyway configuration for local development
- `flyway-docker.conf` - Flyway configuration for Docker container

### Tools (`db/tools/`)
- `flyway/` - Flyway CLI installation (auto-downloaded by script)

## Usage

### Local Development

#### Prerequisites
- PostgreSQL database running locally or accessible
- `curl` and `tar` commands available
- Optional: `psql` for connection testing

#### Running Migrations
```bash
# Run migrations with default settings (localhost:5432, postgres/postgres)
./scripts/migrate.sh

# Run migrations with custom database settings
DB_HOST=your-db-host DB_PASSWORD=your-password ./scripts/migrate.sh

# Check migration status
./scripts/migrate.sh info

# Validate migrations
./scripts/migrate.sh validate

# Show help
./scripts/migrate.sh help
```

#### Environment Variables
```bash
export DB_HOST=localhost      # Database host
export DB_PORT=5432          # Database port
export DB_NAME=postgres      # Database name
export DB_USER=postgres      # Database user
export DB_PASSWORD=postgres  # Database password
```

### Docker Container

#### Build Migration Image
```bash
# Build single architecture
docker build -f Dockerfile.migrator -t workflow-migrator .

# Build multi-architecture image
docker buildx build --platform linux/amd64,linux/arm64 \
  -f Dockerfile.migrator -t workflow-migrator .
```

#### Run Migrations
```bash
# Basic usage
docker run --rm \
  -e DB_HOST=your-db-host \
  -e DB_PASSWORD=your-password \
  workflow-migrator

# With custom command
docker run --rm \
  -e DB_HOST=your-db-host \
  -e DB_PASSWORD=your-password \
  -e FLYWAY_COMMAND=info \
  workflow-migrator

# With network (if database is in Docker)
docker run --rm --network your-network \
  -e DB_HOST=postgres \
  -e DB_PASSWORD=your-password \
  workflow-migrator
```

### Docker Compose

Add to your `docker-compose.yml`:

```yaml
version: '3.8'
services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: postgres
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports:
      - "5432:5432"

  migrate:
    build:
      context: .
      dockerfile: Dockerfile.migrator
    environment:
      DB_HOST: postgres
      DB_NAME: postgres
      DB_USER: postgres
      DB_PASSWORD: postgres
    depends_on:
      - postgres
    # Remove this container after migration completes
    restart: "no"

  workflow-service:
    build: .
    environment:
      DB_HOST: postgres
      DB_NAME: postgres
      DB_USER: postgres
      DB_PASSWORD: postgres
    depends_on:
      - migrate
    ports:
      - "8080:8080"
```

### Kubernetes Init Container

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: workflow-service
spec:
  replicas: 1
  selector:
    matchLabels:
      app: workflow-service
  template:
    metadata:
      labels:
        app: workflow-service
    spec:
      initContainers:
      - name: migrate
        image: workflow-migrator:latest
        env:
        - name: DB_HOST
          value: "postgres-service"
        - name: DB_NAME
          value: "postgres"
        - name: DB_USER
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: username
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: password
      containers:
      - name: workflow-service
        image: workflow-service:latest
        ports:
        - containerPort: 8080
        env:
        - name: DB_HOST
          value: "postgres-service"
        - name: DB_NAME
          value: "postgres"
        - name: DB_USER
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: username
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: password
```

### Adding New Migrations

1. Create a new SQL file following Flyway naming convention:
   ```
   db/migrations/V9__Your_migration_description.sql
   ```

2. Write your SQL migration with idempotent operations:
   ```sql
   -- Use IF NOT EXISTS for table creation
   CREATE TABLE IF NOT EXISTS new_table (
       id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
       name VARCHAR(255) NOT NULL
   );
   
   -- Use conditional logic for column additions
   DO $$
   BEGIN
       IF NOT EXISTS (
           SELECT 1 FROM information_schema.columns 
           WHERE table_name = 'existing_table' AND column_name = 'new_column'
       ) THEN
           ALTER TABLE existing_table ADD COLUMN new_column VARCHAR(255);
       END IF;
   END $$;
   
   -- Use IF NOT EXISTS for index creation
   CREATE INDEX IF NOT EXISTS idx_new_table_name ON new_table(name);
   ```

3. Run the migration:
   ```bash
   ./scripts/migrate.sh
   ```

## Multi-Architecture Docker Build

To build and push multi-architecture images:

```bash
# Create and use a new builder
docker buildx create --name multiarch --use

# Build and push multi-architecture image
docker buildx build --platform linux/amd64,linux/arm64 \
  -f Dockerfile.migrator \
  -t your-registry/workflow-migrator:latest \
  --push .
```

## Migration Best Practices

1. **Idempotent Migrations**: All migrations should be idempotent and safe to run multiple times
2. **Backward Compatibility**: Avoid breaking changes that would affect running services
3. **Testing**: Test migrations on a copy of production data before deploying
4. **Rollback Strategy**: Plan for rollback scenarios, especially for data migrations
5. **Performance**: Consider the impact of large migrations on production systems

## Troubleshooting

### Common Issues

1. **Connection Timeout**: Increase `flyway.connectRetries` in config
2. **Permission Denied**: Ensure database user has necessary privileges
3. **Migration Conflicts**: Use `repair` command to fix migration history issues
4. **Out of Order**: The configuration allows out-of-order migrations for compatibility

### Useful Commands

```bash
# Show migration status
./scripts/migrate.sh info

# Validate migrations without running them
./scripts/migrate.sh validate

# Repair migration history (if needed)
./scripts/migrate.sh repair

# Baseline existing database
./scripts/migrate.sh baseline
```

## Security Considerations

1. **Database Credentials**: Never hardcode credentials in configuration files
2. **Container Security**: The migration container runs as non-root user
3. **Network Security**: Use appropriate network policies in Kubernetes
4. **Secret Management**: Use Kubernetes secrets or similar for credential management

## Notes

- All migration files use Flyway naming convention: `V{version}__{description}.sql`
- Migrations are executed in version order
- The system supports out-of-order migrations for compatibility with existing databases
- Configuration files are optimized for Flyway Community Edition
- The migration container includes database readiness checks
- Multi-architecture support for both AMD64 and ARM64 platforms 