-- Create schema_history table
-- This table stores immutable schema versions at points in time for database instances
CREATE TABLE IF NOT EXISTS schema_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    database_instance_id UUID NOT NULL,
    database_id UUID NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    raw_dump TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Foreign key constraints
    CONSTRAINT fk_schema_history_database_instance
        FOREIGN KEY (database_instance_id)
        REFERENCES database_instances(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_schema_history_database
        FOREIGN KEY (database_id)
        REFERENCES databases(id)
        ON DELETE CASCADE
);

-- Create indexes for schema_history
CREATE INDEX IF NOT EXISTS idx_schema_history_database_instance
    ON schema_history(database_instance_id);
CREATE INDEX IF NOT EXISTS idx_schema_history_database
    ON schema_history(database_id);
CREATE INDEX IF NOT EXISTS idx_schema_history_created_at
    ON schema_history(created_at DESC);

-- Create GIN index on metadata column for JSONB queries
CREATE INDEX IF NOT EXISTS idx_schema_history_metadata
    ON schema_history USING GIN (metadata);

-- Add comments for documentation
COMMENT ON TABLE schema_history IS 'Stores immutable schema versions at points in time for database instances';
COMMENT ON COLUMN schema_history.id IS 'Primary key';
COMMENT ON COLUMN schema_history.database_instance_id IS 'Reference to the database instance';
COMMENT ON COLUMN schema_history.database_id IS 'Reference to the database';
COMMENT ON COLUMN schema_history.metadata IS 'Schema metadata in JSON format (database_name, database_engine, host, port, captured_at)';
COMMENT ON COLUMN schema_history.raw_dump IS 'Raw SQL dump of the schema (from pg_dump or mysqldump)';
COMMENT ON COLUMN schema_history.created_at IS 'Timestamp when this schema version was captured';

-- Create changelog table
-- This table tracks every change plan execution and its status
CREATE TABLE IF NOT EXISTS changelog (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    database_instance_id UUID NOT NULL,
    database_id UUID NOT NULL,
    change_plan_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL,
    prev_schema_history_id UUID,
    schema_history_id UUID,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Check constraint for status enum
    CONSTRAINT chk_changelog_status
        CHECK (status IN ('pending', 'done', 'failed')),

    -- Foreign key constraints
    CONSTRAINT fk_changelog_database_instance
        FOREIGN KEY (database_instance_id)
        REFERENCES database_instances(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_changelog_database
        FOREIGN KEY (database_id)
        REFERENCES databases(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_changelog_change_plan
        FOREIGN KEY (change_plan_id)
        REFERENCES change_plans(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_changelog_prev_schema_history
        FOREIGN KEY (prev_schema_history_id)
        REFERENCES schema_history(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_changelog_schema_history
        FOREIGN KEY (schema_history_id)
        REFERENCES schema_history(id)
        ON DELETE SET NULL
);

-- Create indexes for changelog
CREATE INDEX IF NOT EXISTS idx_changelog_database_instance
    ON changelog(database_instance_id);
CREATE INDEX IF NOT EXISTS idx_changelog_database
    ON changelog(database_id);
CREATE INDEX IF NOT EXISTS idx_changelog_change_plan
    ON changelog(change_plan_id);
CREATE INDEX IF NOT EXISTS idx_changelog_status
    ON changelog(status);
CREATE INDEX IF NOT EXISTS idx_changelog_created_at
    ON changelog(created_at DESC);

-- Add comments for documentation
COMMENT ON TABLE changelog IS 'Tracks every change plan execution and its status';
COMMENT ON COLUMN changelog.id IS 'Primary key';
COMMENT ON COLUMN changelog.database_instance_id IS 'Reference to the database instance';
COMMENT ON COLUMN changelog.database_id IS 'Reference to the database';
COMMENT ON COLUMN changelog.change_plan_id IS 'Reference to the change plan being executed';
COMMENT ON COLUMN changelog.status IS 'Execution status: pending, done, or failed';
COMMENT ON COLUMN changelog.prev_schema_history_id IS 'Reference to the schema version before this change (for future rollback support)';
COMMENT ON COLUMN changelog.schema_history_id IS 'Reference to the schema version created by this change plan execution';
COMMENT ON COLUMN changelog.created_at IS 'Timestamp when this changelog entry was created';
