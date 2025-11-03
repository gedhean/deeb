-- Deeb MVP2 - Rollback Execution Queue Table
-- PostgreSQL 15+

DROP INDEX IF EXISTS idx_execution_queue_unique_in_progress;
DROP INDEX IF EXISTS idx_execution_queue_created_at;
DROP INDEX IF EXISTS idx_execution_queue_priority;
DROP INDEX IF EXISTS idx_execution_queue_position;
DROP INDEX IF EXISTS idx_execution_queue_database;
DROP INDEX IF EXISTS idx_execution_queue_status;
DROP TABLE IF EXISTS execution_queue;

