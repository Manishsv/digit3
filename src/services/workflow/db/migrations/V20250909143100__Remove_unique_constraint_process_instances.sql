-- Flyway Migration V2: Remove unique constraint from process_instances
-- This enables audit trail by creating new records for each transition

-- Remove unique constraint to allow multiple process instance records per entity
-- This enables audit trail by creating new records for each transition
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.table_constraints 
        WHERE constraint_name = 'process_instances_tenant_id_entity_id_process_id_key'
        AND table_name = 'process_instances'
    ) THEN
        ALTER TABLE process_instances DROP CONSTRAINT process_instances_tenant_id_entity_id_process_id_key;
    END IF;
END $$;

-- Add an index to maintain query performance for latest instance lookups
CREATE INDEX IF NOT EXISTS idx_process_instances_entity_latest 
ON process_instances (tenant_id, entity_id, process_id, created_at DESC); 