-- Widen tenant + audit columns to avoid demo/provision overflows.
-- Some integrations pass user identifiers that can exceed VARCHAR(64).

ALTER TABLE processes
  ALTER COLUMN tenant_id TYPE VARCHAR(255),
  ALTER COLUMN created_by TYPE VARCHAR(255),
  ALTER COLUMN modified_by TYPE VARCHAR(255);

ALTER TABLE states
  ALTER COLUMN tenant_id TYPE VARCHAR(255),
  ALTER COLUMN created_by TYPE VARCHAR(255),
  ALTER COLUMN modified_by TYPE VARCHAR(255);

ALTER TABLE actions
  ALTER COLUMN tenant_id TYPE VARCHAR(255),
  ALTER COLUMN created_by TYPE VARCHAR(255),
  ALTER COLUMN modified_by TYPE VARCHAR(255);

ALTER TABLE attribute_validations
  ALTER COLUMN tenant_id TYPE VARCHAR(255),
  ALTER COLUMN created_by TYPE VARCHAR(255),
  ALTER COLUMN modified_by TYPE VARCHAR(255);

ALTER TABLE process_instances
  ALTER COLUMN tenant_id TYPE VARCHAR(255),
  ALTER COLUMN created_by TYPE VARCHAR(255),
  ALTER COLUMN modified_by TYPE VARCHAR(255),
  ALTER COLUMN assigner TYPE VARCHAR(255);

