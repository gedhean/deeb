-- Deeb MVP2 - Add Change Plans Table Migration
-- PostgreSQL 15+
-- This migration adds the change_plans table

-- ============================================
-- CHANGE_PLANS TABLE
-- Stores database change requests with full lifecycle tracking
-- ============================================
CREATE TABLE change_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL,
    target_database_id UUID NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    change_type VARCHAR(50) NOT NULL, -- DDL, DML
    sql_content TEXT NOT NULL,
    rollback_sql TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'Draft', -- Draft, Open, Approved, Closed, Executed, Failed
    risk_level VARCHAR(50) NOT NULL DEFAULT 'Medium', -- Low, Medium, High
    author_id UUID NOT NULL,
    assigned_reviewer_id UUID,
    scheduled_for TIMESTAMP WITH TIME ZONE,
    executed_at TIMESTAMP WITH TIME ZONE,
    execution_duration_ms INTEGER,
    affected_rows INTEGER,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT check_change_type CHECK (change_type IN ('DDL', 'DML')),
    CONSTRAINT check_change_plan_status CHECK (status IN ('Draft', 'Open', 'Approved', 'Closed', 'Executed', 'Failed')),
    CONSTRAINT check_risk_level CHECK (risk_level IN ('Low', 'Medium', 'High')),
    CONSTRAINT fk_change_plans_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    CONSTRAINT fk_change_plans_database FOREIGN KEY (target_database_id) REFERENCES databases(id) ON DELETE RESTRICT,
    CONSTRAINT fk_change_plans_author FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT fk_change_plans_reviewer FOREIGN KEY (assigned_reviewer_id) REFERENCES users(id) ON DELETE SET NULL
);

-- Indexes for change_plans table
CREATE INDEX idx_change_plans_project ON change_plans(project_id);
CREATE INDEX idx_change_plans_database ON change_plans(target_database_id);
CREATE INDEX idx_change_plans_status ON change_plans(status);
CREATE INDEX idx_change_plans_author ON change_plans(author_id);
CREATE INDEX idx_change_plans_reviewer ON change_plans(assigned_reviewer_id) WHERE assigned_reviewer_id IS NOT NULL;
CREATE INDEX idx_change_plans_scheduled ON change_plans(scheduled_for) WHERE scheduled_for IS NOT NULL;
CREATE INDEX idx_change_plans_created_at ON change_plans(created_at DESC);
CREATE INDEX idx_change_plans_risk_level ON change_plans(risk_level);

-- Apply trigger to change_plans table for updated_at
CREATE TRIGGER update_change_plans_updated_at BEFORE UPDATE ON change_plans
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

