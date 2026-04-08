-- Optional: if notification template create still fails with VARCHAR(64) on createdby/lastmodifiedby
-- (e.g. stack insists on JWT sub for audit), widen columns after Flyway baseline.
-- Apply: docker exec -i postgres psql -U postgres -d postgres < extend_notification_template_audit_varchar.sql

ALTER TABLE IF EXISTS notification_template
  ALTER COLUMN createdby TYPE VARCHAR(255),
  ALTER COLUMN lastmodifiedby TYPE VARCHAR(255);
