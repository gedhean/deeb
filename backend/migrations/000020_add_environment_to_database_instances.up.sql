-- Add environment_id column to database_instances table
-- This allows database instances to optionally belong to an environment

ALTER TABLE database_instances
ADD COLUMN environment_id UUID NULL;

-- Add foreign key constraint to ensure environment exists
ALTER TABLE database_instances
ADD CONSTRAINT fk_database_instances_environment
    FOREIGN KEY (environment_id)
    REFERENCES environments(id)
    ON DELETE RESTRICT;

-- Add index for performance when filtering by environment
CREATE INDEX idx_database_instances_environment_id
ON database_instances(environment_id);

-- Add comment for documentation
COMMENT ON COLUMN database_instances.environment_id IS 'Optional environment that this database instance belongs to (e.g., Dev, Staging, Production)';
