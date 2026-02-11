-- Flyway Migration V3: Add sample test column
-- Test migration to verify the migration system is working
-- Note: This migration is kept for compatibility with existing databases
-- that may have applied this migration using the old system

-- Since Flyway creates its own migration tracking table (flyway_schema_history),
-- we don't need to modify the workflow_migrations table
-- This migration is kept as a no-op for compatibility

-- No-op migration for compatibility with existing databases
-- that may have applied this test migration
SELECT 1; 