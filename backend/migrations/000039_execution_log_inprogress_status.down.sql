-- Revert: remap InProgress rows first, then restore the narrower constraint.
UPDATE execution_logs SET status = 'Failed' WHERE status = 'InProgress';
ALTER TABLE execution_logs DROP CONSTRAINT IF EXISTS check_execution_log_status;
ALTER TABLE execution_logs ADD CONSTRAINT check_execution_log_status
    CHECK (status IN ('Completed', 'Failed'));
