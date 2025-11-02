-- Deeb MVP2 - Update Change Plan Statuses
-- PostgreSQL 15+
-- This migration updates the change_plans status constraint to match Phase 3 requirements

-- Drop the old constraint
ALTER TABLE change_plans DROP CONSTRAINT IF EXISTS check_change_plan_status;

-- Update any existing 'Executed' status to 'Completed' (if any exist)
UPDATE change_plans SET status = 'Completed' WHERE status = 'Executed';

-- Add the new constraint with correct statuses
ALTER TABLE change_plans 
    ADD CONSTRAINT check_change_plan_status 
    CHECK (status IN ('Draft', 'Open', 'Approved', 'Closed', 'InProgress', 'Completed', 'Failed', 'Cancelled'));

