-- Deeb MVP2 - Rollback Execution Logs Context Fields
-- PostgreSQL 15+

DROP INDEX IF EXISTS idx_execution_logs_database_type;
DROP INDEX IF EXISTS idx_execution_logs_database;
ALTER TABLE execution_logs DROP CONSTRAINT IF EXISTS fk_execution_logs_database;
ALTER TABLE execution_logs
    DROP COLUMN IF EXISTS sql_content,
    DROP COLUMN IF EXISTS database_id,
    DROP COLUMN IF EXISTS database_name,
    DROP COLUMN IF EXISTS database_instance_name,
    DROP COLUMN IF EXISTS database_type;

