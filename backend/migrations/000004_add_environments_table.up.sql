-- Deeb MVP2 - Add Environments Table Migration
-- PostgreSQL 15+
-- This migration adds the environments table

-- ============================================
-- ENVIRONMENTS TABLE
-- Stores environment stages (Dev, Staging, Production) - organization-level labels
-- ============================================
CREATE TABLE environments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    policy_id UUID, -- NULLable reference to Policy (SQL_REVIEW type)
    created_by_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT unique_environment_name UNIQUE (name),
    CONSTRAINT fk_environments_policy FOREIGN KEY (policy_id) REFERENCES policies(id) ON DELETE SET NULL,
    CONSTRAINT fk_environments_created_by FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE RESTRICT
);

-- Indexes for environments table
CREATE INDEX idx_environments_name ON environments(name);
CREATE INDEX idx_environments_policy ON environments(policy_id) WHERE policy_id IS NOT NULL;
CREATE INDEX idx_environments_created_by ON environments(created_by_id);

-- Apply trigger to environments table for updated_at
CREATE TRIGGER update_environments_updated_at BEFORE UPDATE ON environments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

