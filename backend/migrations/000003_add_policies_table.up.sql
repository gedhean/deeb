-- Deeb MVP2 - Add Policies Table Migration
-- PostgreSQL 15+
-- This migration adds the policies table for storing Policy configurations

-- ============================================
-- POLICIES TABLE
-- Stores Policy configurations (SQL_REVIEW type and future types)
-- ============================================
CREATE TABLE policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL, -- SQL_REVIEW (MVP2), extensible for future types
    policy_config JSONB NOT NULL, -- Policy configuration JSON
    created_by_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT unique_policy_name UNIQUE (name),
    CONSTRAINT check_policy_type CHECK (type IN ('SQL_REVIEW')),
    CONSTRAINT fk_policies_created_by FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE RESTRICT
);

-- Indexes for policies table
CREATE INDEX idx_policies_name ON policies(name);
CREATE INDEX idx_policies_type ON policies(type);
CREATE INDEX idx_policies_created_by ON policies(created_by_id);
CREATE INDEX idx_policies_config ON policies USING GIN (policy_config); -- GIN index for JSONB queries

-- Apply trigger to policies table for updated_at
CREATE TRIGGER update_policies_updated_at BEFORE UPDATE ON policies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

