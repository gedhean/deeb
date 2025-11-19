-- Drop changelog table (must drop first due to foreign key constraints)
DROP TABLE IF EXISTS changelog CASCADE;

-- Drop schema_history table
DROP TABLE IF EXISTS schema_history CASCADE;
