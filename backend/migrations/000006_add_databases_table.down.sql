-- Deeb MVP2 - Add Databases Table Migration (Down)
-- Drop databases table and supporting infrastructure

-- Drop trigger
DROP TRIGGER IF EXISTS update_databases_updated_at ON databases;

-- Drop databases table
DROP TABLE IF EXISTS databases CASCADE;

