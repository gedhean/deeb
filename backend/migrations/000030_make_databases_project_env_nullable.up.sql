-- Migration: Make project_id and environment_id nullable in databases table
-- This allows databases to exist without project/environment assignment for sync functionality

-- Drop existing foreign key constraints
ALTER TABLE databases DROP CONSTRAINT IF EXISTS fk_databases_project;
ALTER TABLE databases DROP CONSTRAINT IF EXISTS fk_databases_environment;

-- Make columns nullable
ALTER TABLE databases ALTER COLUMN project_id DROP NOT NULL;
ALTER TABLE databases ALTER COLUMN environment_id DROP NOT NULL;

-- Re-add foreign key constraints (will allow NULL values)
ALTER TABLE databases
    ADD CONSTRAINT fk_databases_project
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE;

ALTER TABLE databases
    ADD CONSTRAINT fk_databases_environment
    FOREIGN KEY (environment_id) REFERENCES environments(id) ON DELETE RESTRICT;

-- Add index for finding unassigned databases
CREATE INDEX idx_databases_unassigned ON databases(database_instance_id)
    WHERE project_id IS NULL OR environment_id IS NULL;

-- Add comment explaining the nullable columns
COMMENT ON COLUMN databases.project_id IS 'Project assignment - can be NULL for unassigned databases created via sync';
COMMENT ON COLUMN databases.environment_id IS 'Environment assignment - can be NULL for unassigned databases created via sync';
