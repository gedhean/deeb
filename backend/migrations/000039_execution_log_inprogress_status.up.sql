-- Allow InProgress on execution_logs.status (ghost runs are long-lived; the
-- log row is created when the run starts and flipped at completion).
ALTER TABLE execution_logs DROP CONSTRAINT IF EXISTS check_execution_log_status;
ALTER TABLE execution_logs ADD CONSTRAINT check_execution_log_status
    CHECK (status IN ('Completed', 'Failed', 'InProgress'));
