-- Add is_production flag to environments to gate ghost executor usage
ALTER TABLE environments ADD COLUMN is_production BOOLEAN NOT NULL DEFAULT FALSE;

-- Add execution_backend preference to change_plans (direct = raw SQL transaction, ghost = gh-ost/pgosc style runner)
ALTER TABLE change_plans ADD COLUMN execution_backend VARCHAR(16) NOT NULL DEFAULT 'direct';

-- Constrain execution_backend to supported executor tags
ALTER TABLE change_plans ADD CONSTRAINT change_plans_execution_backend_check CHECK (execution_backend IN ('direct', 'ghost'));
