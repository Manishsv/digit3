-- Flyway Migration V7: Remove is_active column from escalation_configs table
-- Since we use DELETE API for removing configs, isActive field is redundant

-- Remove is_active column from escalation_configs table if it exists
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'escalation_configs' AND column_name = 'is_active'
    ) THEN
        ALTER TABLE escalation_configs DROP COLUMN is_active;
    END IF;
END $$;

-- Remove the index on is_active since the column is being dropped
DROP INDEX IF EXISTS idx_escalation_configs_active; 