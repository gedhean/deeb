-- Deeb - Reverse: Rename database_engine back to database_type and revert postgres to postgresql
-- PostgreSQL 15+
-- This migration reverses the rename from database_engine to database_type and reverts 'postgres' to 'postgresql'

-- Update execution_logs table if database_engine column exists
DO $$
BEGIN
    -- Rename column if it exists
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'execution_logs'
        AND column_name = 'database_engine'
    ) THEN
        ALTER TABLE execution_logs RENAME COLUMN database_engine TO database_type;
    END IF;
END $$;

-- Drop the index with new name
DROP INDEX IF EXISTS idx_instances_engine;

-- Drop the new constraint
ALTER TABLE database_instances
DROP CONSTRAINT IF EXISTS check_database_engine;

-- Rename the column back
ALTER TABLE database_instances
RENAME COLUMN database_engine TO database_type;

-- Add back the old constraint
ALTER TABLE database_instances
ADD CONSTRAINT check_database_type CHECK (database_type IN ('mysql', 'postgresql'));

-- Recreate the index with old column name
CREATE INDEX idx_instances_type ON database_instances(database_type);

-- Update data back from 'postgres' to 'postgresql'
UPDATE database_instances
SET database_type = 'postgresql'
WHERE database_type = 'postgres';

UPDATE execution_logs
SET database_type = 'postgresql'
WHERE database_type = 'postgres';
