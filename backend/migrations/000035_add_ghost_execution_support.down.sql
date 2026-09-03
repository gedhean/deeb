-- Rollback ghost execution support migration
ALTER TABLE change_plans DROP CONSTRAINT IF EXISTS change_plans_execution_backend_check;
ALTER TABLE change_plans DROP COLUMN IF EXISTS execution_backend;
ALTER TABLE environments DROP COLUMN IF EXISTS is_production;
