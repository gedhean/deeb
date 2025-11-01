-- Deeb MVP2 - Add Policy Findings Table Migration
-- PostgreSQL 15+
-- This migration adds the policy_findings table

-- ============================================
-- POLICY_FINDINGS TABLE
-- Stores SQL Review findings (results) from policy evaluation
-- ============================================
CREATE TABLE policy_findings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    change_plan_id UUID NOT NULL,
    validator_type VARCHAR(100) NOT NULL, -- syntax_error, security_risk, performance_issue, backwards_incompatible, best_practice_violation
    severity VARCHAR(50) NOT NULL, -- Error, Warning, Info
    rule_name VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    line_number INTEGER,
    column_number INTEGER,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT check_validator_type CHECK (validator_type IN ('syntax_error', 'security_risk', 'performance_issue', 'backwards_incompatible', 'best_practice_violation')),
    CONSTRAINT check_severity CHECK (severity IN ('Error', 'Warning', 'Info')),
    CONSTRAINT fk_policy_findings_change_plan FOREIGN KEY (change_plan_id) REFERENCES change_plans(id) ON DELETE CASCADE
);

-- Indexes for policy_findings table
CREATE INDEX idx_policy_findings_change_plan ON policy_findings(change_plan_id);
CREATE INDEX idx_policy_findings_severity ON policy_findings(severity);
CREATE INDEX idx_policy_findings_validator_type ON policy_findings(validator_type);

