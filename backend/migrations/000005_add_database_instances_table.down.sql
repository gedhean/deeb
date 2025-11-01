-- Deeb MVP2 - Add Database Instances Table Migration (Down)
-- Drop database_instances table and supporting infrastructure

-- Drop trigger
DROP TRIGGER IF EXISTS update_database_instances_updated_at ON database_instances;

-- Drop database_instances table
DROP TABLE IF EXISTS database_instances CASCADE;

