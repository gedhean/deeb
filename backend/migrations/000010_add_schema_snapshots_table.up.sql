-- Deeb MVP2 - Add Schema Snapshots Table Migration
-- PostgreSQL 15+
-- This migration adds the schema_snapshots table

-- ============================================
-- SCHEMA_SNAPSHOTS TABLE
-- Stores before/after schema snapshots for each change plan execution
-- ============================================
CREATE TABLE schema_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    database_id UUID NOT NULL,
    change_plan_id UUID NOT NULL,
    snapshot_type VARCHAR(50) NOT NULL, -- before, after
    snapshot_data JSONB NOT NULL,
    snapshot_data_raw TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT check_snapshot_type CHECK (snapshot_type IN ('before', 'after')),
    CONSTRAINT fk_snapshots_database FOREIGN KEY (database_id) REFERENCES databases(id) ON DELETE CASCADE,
    CONSTRAINT fk_snapshots_change_plan FOREIGN KEY (change_plan_id) REFERENCES change_plans(id) ON DELETE CASCADE
);

-- Indexes for schema_snapshots table
CREATE INDEX idx_snapshots_database ON schema_snapshots(database_id);
CREATE INDEX idx_snapshots_change_plan ON schema_snapshots(change_plan_id);
CREATE INDEX idx_snapshots_type ON schema_snapshots(snapshot_type);
CREATE INDEX idx_snapshots_created_at ON schema_snapshots(created_at DESC);

