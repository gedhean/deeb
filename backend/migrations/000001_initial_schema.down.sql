-- Deeb MVP2 - Initial Schema Migration (Down)
-- Drop users table and supporting infrastructure

-- Drop trigger
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop trigger function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop users table
DROP TABLE IF EXISTS users CASCADE;
