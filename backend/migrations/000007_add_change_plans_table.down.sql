-- Deeb MVP2 - Add Change Plans Table Migration (Down)
-- Drop change_plans table and supporting infrastructure

-- Drop trigger
DROP TRIGGER IF EXISTS update_change_plans_updated_at ON change_plans;

-- Drop change_plans table
DROP TABLE IF EXISTS change_plans CASCADE;

