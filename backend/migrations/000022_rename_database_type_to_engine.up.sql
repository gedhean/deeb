-- Deeb - Rename database_type to database_engine and update postgres value
-- PostgreSQL 15+
-- This migration renames database_type column to database_engine and updates 'postgresql' to 'postgres'

-- Step 1: Drop the old constraint FIRST (before updating data)
ALTER TABLE database_instances
DROP CONSTRAINT IF EXISTS check_database_type;

-- Step 2: Update existing data from 'postgresql' to 'postgres'
UPDATE database_instances
SET database_type = 'postgres'
WHERE database_type = 'postgresql';

UPDATE execution_logs
SET database_type = 'postgres'
WHERE database_type = 'postgresql';

-- Step 3: Rename the column
ALTER TABLE database_instances
RENAME COLUMN database_type TO database_engine;

-- Step 4: Add new constraint with updated values
ALTER TABLE database_instances
ADD CONSTRAINT check_database_engine CHECK (database_engine IN ('mysql', 'postgres'));

-- Step 5: Drop and recreate the index with new column name
DROP INDEX IF EXISTS idx_instances_type;
CREATE INDEX idx_instances_engine ON database_instances(database_engine);

-- Step 6: Update execution_logs table if database_type column exists
-- Note: This column may be nullable, so we handle it carefully
DO $$
BEGIN
    -- Rename column if it exists
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'execution_logs'
        AND column_name = 'database_type'
    ) THEN
        ALTER TABLE execution_logs RENAME COLUMN database_type TO database_engine;
    END IF;
END $$;
