-- Deeb MVP2 - Add Execution Locks Table
-- PostgreSQL 15+
-- This migration creates the execution_locks table for concurrency control

CREATE TABLE execution_locks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_database_id UUID NOT NULL UNIQUE,
    change_plan_id UUID NOT NULL,
    executor_id UUID NOT NULL,
    locked_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    
    CONSTRAINT fk_execution_locks_change_plan FOREIGN KEY (change_plan_id) REFERENCES change_plans(id) ON DELETE CASCADE,
    CONSTRAINT fk_execution_locks_database FOREIGN KEY (target_database_id) REFERENCES databases(id) ON DELETE CASCADE,
    CONSTRAINT fk_execution_locks_executor FOREIGN KEY (executor_id) REFERENCES users(id) ON DELETE RESTRICT
);

CREATE INDEX idx_execution_locks_database ON execution_locks(target_database_id);
CREATE INDEX idx_execution_locks_expires_at ON execution_locks(expires_at);

