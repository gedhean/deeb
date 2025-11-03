-- Deeb MVP2 - Rollback Execution Logs Table
-- PostgreSQL 15+

DROP INDEX IF EXISTS idx_execution_logs_status;
DROP INDEX IF EXISTS idx_execution_logs_started_at;
DROP INDEX IF EXISTS idx_execution_logs_executor;
DROP INDEX IF EXISTS idx_execution_logs_change_plan;
DROP TABLE IF EXISTS execution_logs;

