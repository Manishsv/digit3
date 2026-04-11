# Database Migrations

This folder hosts the Flyway assets required to manage the Registry service schema.

## Layout

```
db/
├── config/         # Flyway configuration files
├── migrations/     # Versioned SQL migrations
├── migrate.sh      # Convenience entrypoint for the Docker image
├── postgres.go     # Helper for bootstrapping connections in tools/tests
└── Dockerfile      # Builds a Flyway container with the migrations baked in
```

## Running Migrations

### Locally with Flyway CLI

```
export FLYWAY_USER=<db user>
export FLYWAY_PASSWORD=<db password>
export DB_URL=jdbc:postgresql://localhost:5432/registry_db
export SCHEMA_TABLE=registry_schema_history
export FLYWAY_LOCATIONS=filesystem:db/migrations
flyway -configFiles=db/config/flyway.conf migrate
```

### Using the Docker Image

```
docker build -t registry-flyway db
docker run --rm \
  -e DB_URL=jdbc:postgresql://host.docker.internal:5432/registry_db \
  -e FLYWAY_USER=<db user> \
  -e FLYWAY_PASSWORD=<db password> \
  -e SCHEMA_TABLE=registry_schema_history \
  -e FLYWAY_LOCATIONS=filesystem:/flyway/sql \
  registry-flyway
```

## Adding a Migration

1. Create a new SQL file under `db/migrations/` following Flyway's naming convention, for example `V202510300001__add_new_column.sql`.
2. Populate the file with the required DDL.
3. Run the migration using one of the steps above.
4. Commit the new SQL file along with any code changes that depend on it.
