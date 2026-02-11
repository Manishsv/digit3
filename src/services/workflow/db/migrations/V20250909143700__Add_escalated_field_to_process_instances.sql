-- Flyway Migration V8: Add escalated field to process_instances table
-- Following the Java service pattern for tracking escalated instances

-- Add escalated column to process_instances table if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'process_instances' AND column_name = 'escalated'
    ) THEN
        ALTER TABLE process_instances ADD COLUMN escalated BOOLEAN DEFAULT false;
    END IF;
END $$;

-- Create indexes for better search performance on escalated instances
CREATE INDEX IF NOT EXISTS idx_process_instances_escalated_tenant 
ON process_instances(tenant_id, escalated);

CREATE INDEX IF NOT EXISTS idx_process_instances_escalated_business 
ON process_instances(tenant_id, escalated, process_id); 