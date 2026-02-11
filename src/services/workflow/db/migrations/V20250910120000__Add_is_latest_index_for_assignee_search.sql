-- Adds a partial GIN index to support fast assignee-based lookups on latest instances
CREATE INDEX IF NOT EXISTS idx_process_instances_latest_assignees
    ON process_instances USING GIN (assignees)
    WHERE is_latest = TRUE AND (is_parallel_branch = FALSE OR is_parallel_branch IS NULL);
