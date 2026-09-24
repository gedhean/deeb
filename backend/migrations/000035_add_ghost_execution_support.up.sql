-- Add is_production flag to environments to gate ghost executor usage
ALTER TABLE environments ADD COLUMN is_production BOOLEAN NOT NULL DEFAULT FALSE;

-- Backfill: environments named production/prod are production. Operators
-- should verify no real production environment was left unflagged.
UPDATE environments SET is_production = TRUE WHERE lower(name) IN ('production', 'prod');

-- Add execution_backend preference to change_plans (direct = raw SQL transaction, ghost = gh-ost/pgosc style runner)
ALTER TABLE change_plans ADD COLUMN execution_backend VARCHAR(16) NOT NULL DEFAULT 'direct';

-- Constrain execution_backend to supported executor tags
ALTER TABLE change_plans ADD CONSTRAINT change_plans_execution_backend_check CHECK (execution_backend IN ('direct', 'ghost'));

-- Allow the ghost executor's awaiting-cutover state in the existing status check.
-- Note: the status constraint was created as check_change_plan_status in 000013;
-- drop both possible names so the replacement covers existing databases.
ALTER TABLE change_plans DROP CONSTRAINT IF EXISTS change_plans_status_check;
ALTER TABLE change_plans DROP CONSTRAINT IF EXISTS check_change_plan_status;
ALTER TABLE change_plans ADD CONSTRAINT change_plans_status_check
    CHECK (status IN ('Draft', 'Open', 'Approved', 'Closed', 'InProgress', 'AwaitingCutover', 'Completed', 'Failed', 'Cancelled'));
