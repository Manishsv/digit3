CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Add a uuid column to the localisation table (if it doesn't exist)
ALTER TABLE localisation ADD COLUMN IF NOT EXISTS uuid UUID NOT NULL DEFAULT gen_random_uuid();

-- Add a unique index on the uuid column
CREATE UNIQUE INDEX IF NOT EXISTS idx_localisation_uuid ON localisation (uuid); 