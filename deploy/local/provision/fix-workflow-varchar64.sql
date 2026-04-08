-- Workflow Flyway V20250909143000+ uses VARCHAR(64) for tenant_id and audit columns; long JWT sub overflows.
-- Run against the Postgres used by workflow-service, e.g.:
--   docker exec -i postgres psql -U postgres -d postgres -f - < fix-workflow-varchar64.sql

ALTER TABLE IF EXISTS processes
  ALTER COLUMN tenant_id TYPE VARCHAR(255),
  ALTER COLUMN created_by TYPE VARCHAR(255),
  ALTER COLUMN modified_by TYPE VARCHAR(255);

ALTER TABLE IF EXISTS states
  ALTER COLUMN tenant_id TYPE VARCHAR(255),
  ALTER COLUMN created_by TYPE VARCHAR(255),
  ALTER COLUMN modified_by TYPE VARCHAR(255);

ALTER TABLE IF EXISTS attribute_validations
  ALTER COLUMN tenant_id TYPE VARCHAR(255),
  ALTER COLUMN created_by TYPE VARCHAR(255),
  ALTER COLUMN modified_by TYPE VARCHAR(255);

ALTER TABLE IF EXISTS actions
  ALTER COLUMN tenant_id TYPE VARCHAR(255),
  ALTER COLUMN created_by TYPE VARCHAR(255),
  ALTER COLUMN modified_by TYPE VARCHAR(255);

ALTER TABLE IF EXISTS process_instances
  ALTER COLUMN tenant_id TYPE VARCHAR(255),
  ALTER COLUMN status TYPE VARCHAR(255),
  ALTER COLUMN assigner TYPE VARCHAR(255),
  ALTER COLUMN created_by TYPE VARCHAR(255),
  ALTER COLUMN modified_by TYPE VARCHAR(255);

ALTER TABLE IF EXISTS parallel_executions
  ALTER COLUMN tenant_id TYPE VARCHAR(255),
  ALTER COLUMN entity_id TYPE VARCHAR(255),
  ALTER COLUMN created_by TYPE VARCHAR(255),
  ALTER COLUMN modified_by TYPE VARCHAR(255);

ALTER TABLE IF EXISTS escalation_configs
  ALTER COLUMN tenant_id TYPE VARCHAR(255),
  ALTER COLUMN created_by TYPE VARCHAR(255),
  ALTER COLUMN modified_by TYPE VARCHAR(255);

-- Optional: widen branch_id if parallel migration added it (often unused length-wise)
ALTER TABLE IF EXISTS process_instances
  ALTER COLUMN branch_id TYPE VARCHAR(255);
