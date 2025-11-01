-- Deeb MVP2 - Add Approvals Table Migration
-- PostgreSQL 15+
-- This migration adds the approvals table

-- ============================================
-- APPROVALS TABLE
-- Stores approval/rejection records for change plans
-- ============================================
CREATE TABLE approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    change_plan_id UUID NOT NULL,
    approver_id UUID NOT NULL,
    status VARCHAR(50) NOT NULL, -- Approved, Rejected
    comments TEXT,
    approved_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT check_approval_status CHECK (status IN ('Approved', 'Rejected')),
    CONSTRAINT unique_approval_per_user_change_plan UNIQUE (change_plan_id, approver_id),
    CONSTRAINT fk_approvals_change_plan FOREIGN KEY (change_plan_id) REFERENCES change_plans(id) ON DELETE CASCADE,
    CONSTRAINT fk_approvals_approver FOREIGN KEY (approver_id) REFERENCES users(id) ON DELETE RESTRICT
);

-- Indexes for approvals table
CREATE INDEX idx_approvals_change_plan ON approvals(change_plan_id);
CREATE INDEX idx_approvals_approver ON approvals(approver_id);
CREATE INDEX idx_approvals_status ON approvals(status);
CREATE INDEX idx_approvals_created_at ON approvals(created_at DESC);

