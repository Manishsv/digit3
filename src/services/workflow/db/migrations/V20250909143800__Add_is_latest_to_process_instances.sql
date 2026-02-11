-- Add is_latest flag to speed up latest instance lookups and backfill existing data

ALTER TABLE process_instances
    ADD COLUMN IF NOT EXISTS is_latest BOOLEAN DEFAULT FALSE;

-- Mark the latest linear (non-parallel) instance per entity/process as latest
WITH ranked_linear AS (
    SELECT id
    FROM (
        SELECT id,
               ROW_NUMBER() OVER (
                   PARTITION BY tenant_id, entity_id, process_id
                   ORDER BY created_at DESC
               ) AS rn
        FROM process_instances
        WHERE is_parallel_branch = FALSE OR is_parallel_branch IS NULL
    ) sub
    WHERE rn = 1
)
UPDATE process_instances
SET is_latest = TRUE
WHERE id IN (SELECT id FROM ranked_linear);

-- Mark the latest parallel branch instance per branch as latest to keep branch tracking consistent
WITH ranked_branch AS (
    SELECT id
    FROM (
        SELECT id,
               ROW_NUMBER() OVER (
                   PARTITION BY tenant_id, entity_id, process_id, branch_id
                   ORDER BY created_at DESC
               ) AS rn
        FROM process_instances
        WHERE is_parallel_branch = TRUE AND branch_id IS NOT NULL
    ) sub
    WHERE rn = 1
)
UPDATE process_instances
SET is_latest = TRUE
WHERE id IN (SELECT id FROM ranked_branch);

-- Ensure only one latest linear instance exists per entity/process
CREATE UNIQUE INDEX IF NOT EXISTS idx_process_instances_latest_linear
    ON process_instances (tenant_id, entity_id, process_id)
    WHERE is_latest = TRUE AND (is_parallel_branch = FALSE OR is_parallel_branch IS NULL);

-- Accelerate lookups for latest branch instances when parallel workflows are used
CREATE INDEX IF NOT EXISTS idx_process_instances_latest_branch
    ON process_instances (tenant_id, entity_id, process_id, branch_id)
    WHERE is_latest = TRUE AND is_parallel_branch = TRUE;
