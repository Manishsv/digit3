-- MDMS v2 stock Flyway DDL uses VARCHAR(64) on audit columns; long JWT `sub` / opaque ids overflow on INSERT.
-- Apply once to your local postgres used by mdms-v2 (default compose: postgres:5432, db postgres, user postgres).
-- Example: docker exec -i postgres psql -U postgres -d postgres -f - < fix-mdms-varchar64.sql

ALTER TABLE IF EXISTS eg_mdms_schema_definition
  ALTER COLUMN createdby TYPE VARCHAR(255),
  ALTER COLUMN lastmodifiedby TYPE VARCHAR(255);

ALTER TABLE IF EXISTS eg_mdms_data
  ALTER COLUMN createdby TYPE VARCHAR(255),
  ALTER COLUMN lastmodifiedby TYPE VARCHAR(255);
