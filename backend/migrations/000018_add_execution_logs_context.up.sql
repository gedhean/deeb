-- Deeb MVP2 - Add Error Context to Execution Logs
-- PostgreSQL 15+
-- This migration adds context fields to execution_logs for better error debugging

ALTER TABLE execution_logs
    ADD COLUMN sql_content TEXT,
    ADD COLUMN database_id UUID,
    ADD COLUMN database_name VARCHAR(255),
    ADD COLUMN database_instance_name VARCHAR(255),
    ADD COLUMN database_type VARCHAR(50);

-- Add foreign key for database_id
ALTER TABLE execution_logs
    ADD CONSTRAINT fk_execution_logs_database FOREIGN KEY (database_id) REFERENCES databases(id) ON DELETE SET NULL;

-- Add index for database_id to enable querying by database
CREATE INDEX idx_execution_logs_database ON execution_logs(database_id);

-- Add index for database_type to enable filtering by database type
CREATE INDEX idx_execution_logs_database_type ON execution_logs(database_type);

