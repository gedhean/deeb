-- Deeb MVP2 - Add Projects Table Migration (Down)
-- Drop projects table and supporting infrastructure

-- Drop trigger
DROP TRIGGER IF EXISTS update_projects_updated_at ON projects;

-- Drop projects table
DROP TABLE IF EXISTS projects CASCADE;

