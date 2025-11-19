-- Drop schema_snapshots table
-- IMPORTANT: Only apply this migration after thorough testing in production
-- The old schema_snapshots table used a before/after snapshot model which is incompatible
-- with the new changelog + schema_history system. Old data cannot be migrated.
DROP TABLE IF EXISTS schema_snapshots CASCADE;
