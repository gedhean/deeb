-- Revert: AwaitingCleanup must be moved out before shrinking the constraint.
UPDATE change_plans SET status = 'Completed', updated_at = NOW() WHERE status = 'AwaitingCleanup';
ALTER TABLE change_plans DROP CONSTRAINT IF EXISTS change_plans_status_check;
ALTER TABLE change_plans DROP CONSTRAINT IF EXISTS check_change_plan_status;
ALTER TABLE change_plans ADD CONSTRAINT change_plans_status_check
    CHECK (status IN ('Draft', 'Open', 'Approved', 'Closed', 'InProgress', 'AwaitingCutover', 'Completed', 'Failed', 'Cancelled'));
