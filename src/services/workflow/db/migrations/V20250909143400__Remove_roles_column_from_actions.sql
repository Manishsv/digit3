-- Flyway Migration V5: Remove roles column from actions table
-- Roles are now handled through attributeValidation.attributes

-- Remove roles column from actions table if it exists
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'actions' AND column_name = 'roles'
    ) THEN
        ALTER TABLE actions DROP COLUMN roles;
    END IF;
END $$; 