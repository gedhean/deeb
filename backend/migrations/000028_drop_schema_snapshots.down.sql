-- Recreate schema_snapshots table structure (data not restored)
-- NOTE: This down migration recreates the table structure but does NOT restore data
-- Old snapshot data is permanently lost after the up migration is applied
CREATE TABLE IF NOT EXISTS schema_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    database_id UUID NOT NULL,
    change_plan_id UUID NOT NULL,
    snapshot_type VARCHAR(10) NOT NULL,
    snapshot_data JSONB,
    snapshot_data_raw TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_schema_snapshot_database
        FOREIGN KEY (database_id)
        REFERENCES databases(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_schema_snapshot_change_plan
        FOREIGN KEY (change_plan_id)
        REFERENCES change_plans(id)
        ON DELETE CASCADE,
    CONSTRAINT chk_snapshot_type
        CHECK (snapshot_type IN ('before', 'after'))
);

CREATE INDEX IF NOT EXISTS idx_schema_snapshots_database
    ON schema_snapshots(database_id);
CREATE INDEX IF NOT EXISTS idx_schema_snapshots_change_plan
    ON schema_snapshots(change_plan_id);
CREATE INDEX IF NOT EXISTS idx_schema_snapshots_type
    ON schema_snapshots(snapshot_type);
CREATE INDEX IF NOT EXISTS idx_schema_snapshots_created_at
    ON schema_snapshots(created_at DESC);

COMMENT ON TABLE schema_snapshots IS 'Stores database schema snapshots before and after change plan execution (DEPRECATED - replaced by changelog + schema_history)';
