-- Deeb MVP2 - Rollback Execution Locks Table
-- PostgreSQL 15+

DROP INDEX IF EXISTS idx_execution_locks_expires_at;
DROP INDEX IF EXISTS idx_execution_locks_database;
DROP TABLE IF EXISTS execution_locks;

