-- Add is_production flag to environments to gate ghost executor usage
ALTER TABLE environments ADD COLUMN is_production BOOLEAN NOT NULL DEFAULT FALSE;

-- Add execution_backend preference to change_plans (direct = raw SQL transaction, ghost = gh-ost/pgosc style runner)
ALTER TABLE change_plans ADD COLUMN execution_backend VARCHAR(16) NOT NULL DEFAULT 'direct';

-- Constrain execution_backend to supported executor tags
ALTER TABLE change_plans ADD CONSTRAINT change_plans_execution_backend_check CHECK (execution_backend IN ('direct', 'ghost'));

-- Allow the ghost executor's awaiting-cutover state in the existing status check
ALTER TABLE change_plans DROP CONSTRAINT IF EXISTS change_plans_status_check;
ALTER TABLE change_plans ADD CONSTRAINT change_plans_status_check
    CHECK (status IN ('Draft', 'Open', 'Approved', 'Closed', 'InProgress', 'AwaitingCutover', 'Completed', 'Failed', 'Cancelled'));
