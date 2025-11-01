-- Deeb MVP2 - Add Environments Table Migration (Down)
-- Drop environments table and supporting infrastructure

-- Drop trigger
DROP TRIGGER IF EXISTS update_environments_updated_at ON environments;

-- Drop environments table
DROP TABLE IF EXISTS environments CASCADE;

