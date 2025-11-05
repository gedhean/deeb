-- Deeb - Update Approval Unique Constraint Migration
-- PostgreSQL 15+
-- This migration updates the unique constraint on approvals table to include status field
-- This allows the same user to have multiple approval records (with different statuses) for the same change plan

-- Drop the old unique constraint
ALTER TABLE approvals DROP CONSTRAINT IF EXISTS unique_approval_per_user_change_plan;

-- Add new unique constraint that includes status
ALTER TABLE approvals ADD CONSTRAINT unique_approval_per_user_change_plan_status UNIQUE (change_plan_id, approver_id, status);
