-- Deeb MVP2 - Add Schema Snapshots Table Migration (Down)
-- Drop schema_snapshots table and supporting infrastructure

-- Drop schema_snapshots table
DROP TABLE IF EXISTS schema_snapshots CASCADE;

