-- Deeb MVP2 - Add Policies Table Migration (Down)
-- Drop policies table and supporting infrastructure

-- Drop trigger
DROP TRIGGER IF EXISTS update_policies_updated_at ON policies;

-- Drop policies table
DROP TABLE IF EXISTS policies CASCADE;

