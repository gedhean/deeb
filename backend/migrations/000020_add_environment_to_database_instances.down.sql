-- Rollback migration: Remove environment_id from database_instances table

-- Drop the index
DROP INDEX IF EXISTS idx_database_instances_environment_id;

-- Drop the foreign key constraint
ALTER TABLE database_instances
DROP CONSTRAINT IF EXISTS fk_database_instances_environment;

-- Drop the column
ALTER TABLE database_instances
DROP COLUMN IF EXISTS environment_id;
