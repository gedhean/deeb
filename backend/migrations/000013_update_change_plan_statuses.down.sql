-- Deeb MVP2 - Rollback Change Plan Statuses Update
-- PostgreSQL 15+

-- Drop the new constraint
ALTER TABLE change_plans DROP CONSTRAINT IF EXISTS check_change_plan_status;

-- Update any 'Completed' status back to 'Executed' (if any exist)
UPDATE change_plans SET status = 'Executed' WHERE status = 'Completed';
UPDATE change_plans SET status = 'Executed' WHERE status = 'InProgress';
UPDATE change_plans SET status = 'Closed' WHERE status = 'Cancelled';

-- Restore the old constraint
ALTER TABLE change_plans 
    ADD CONSTRAINT check_change_plan_status 
    CHECK (status IN ('Draft', 'Open', 'Approved', 'Closed', 'Executed', 'Failed'));

