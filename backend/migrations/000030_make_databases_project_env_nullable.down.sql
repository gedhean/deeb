-- Rollback: Make project_id and environment_id NOT NULL again
-- WARNING: This will fail if there are databases with NULL project_id or environment_id

-- Drop the unassigned index
DROP INDEX IF EXISTS idx_databases_unassigned;

-- Remove comments
COMMENT ON COLUMN databases.project_id IS NULL;
COMMENT ON COLUMN databases.environment_id IS NULL;

-- Drop existing foreign key constraints
ALTER TABLE databases DROP CONSTRAINT IF EXISTS fk_databases_project;
ALTER TABLE databases DROP CONSTRAINT IF EXISTS fk_databases_environment;

-- Make columns NOT NULL (will fail if NULL values exist)
ALTER TABLE databases ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE databases ALTER COLUMN environment_id SET NOT NULL;

-- Re-add foreign key constraints with CASCADE
ALTER TABLE databases
    ADD CONSTRAINT fk_databases_project
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE;

ALTER TABLE databases
    ADD CONSTRAINT fk_databases_environment
    FOREIGN KEY (environment_id) REFERENCES environments(id) ON DELETE RESTRICT;
