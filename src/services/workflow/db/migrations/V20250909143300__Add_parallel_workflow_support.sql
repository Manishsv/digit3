-- Flyway Migration V4: Add parallel workflow support
-- This migration adds support for parallel workflow execution

-- Add parallel workflow columns to process_instances table
DO $$
BEGIN
    -- Add parent_instance_id column if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'process_instances' AND column_name = 'parent_instance_id'
    ) THEN
        ALTER TABLE process_instances ADD COLUMN parent_instance_id UUID REFERENCES process_instances(id);
    END IF;
    
    -- Add branch_id column if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'process_instances' AND column_name = 'branch_id'
    ) THEN
        ALTER TABLE process_instances ADD COLUMN branch_id VARCHAR(64);
    END IF;
    
    -- Add is_parallel_branch column if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'process_instances' AND column_name = 'is_parallel_branch'
    ) THEN
        ALTER TABLE process_instances ADD COLUMN is_parallel_branch BOOLEAN DEFAULT FALSE;
    END IF;
END $$;

-- Create indexes for efficient parallel instance queries
CREATE INDEX IF NOT EXISTS idx_process_instances_parallel 
ON process_instances(tenant_id, entity_id, process_id, is_parallel_branch);

CREATE INDEX IF NOT EXISTS idx_process_instances_branch 
ON process_instances(tenant_id, entity_id, process_id, branch_id);

-- Table for tracking parallel execution coordination
CREATE TABLE IF NOT EXISTS parallel_executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(64) NOT NULL,
    entity_id VARCHAR(64) NOT NULL,
    process_id UUID NOT NULL REFERENCES processes(id) ON DELETE CASCADE,
    parallel_state_id UUID NOT NULL REFERENCES states(id) ON DELETE CASCADE,
    join_state_id UUID NOT NULL REFERENCES states(id) ON DELETE CASCADE,
    active_branches JSONB NOT NULL DEFAULT '[]',
    completed_branches JSONB NOT NULL DEFAULT '[]',
    status VARCHAR(32) DEFAULT 'ACTIVE',
    created_by VARCHAR(64) NOT NULL,
    created_at BIGINT NOT NULL,
    modified_by VARCHAR(64) NOT NULL,
    modified_at BIGINT NOT NULL,
    UNIQUE(tenant_id, entity_id, process_id, parallel_state_id)
);

-- Create indexes for efficient parallel execution queries
CREATE INDEX IF NOT EXISTS idx_parallel_executions_entity 
ON parallel_executions(tenant_id, entity_id, process_id);

CREATE INDEX IF NOT EXISTS idx_parallel_executions_status 
ON parallel_executions(tenant_id, status); 