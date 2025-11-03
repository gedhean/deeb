-- Deeb MVP2 - Add Execution Logs Table
-- PostgreSQL 15+
-- This migration creates the execution_logs table for tracking execution attempts

CREATE TABLE execution_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    change_plan_id UUID NOT NULL,
    executor_id UUID NOT NULL,
    status VARCHAR(50) NOT NULL, -- Completed, Failed
    execution_duration_ms INTEGER NOT NULL,
    rows_affected INTEGER,
    error_message TEXT,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT check_execution_log_status CHECK (status IN ('Completed', 'Failed')),
    CONSTRAINT fk_execution_logs_change_plan FOREIGN KEY (change_plan_id) REFERENCES change_plans(id) ON DELETE CASCADE,
    CONSTRAINT fk_execution_logs_executor FOREIGN KEY (executor_id) REFERENCES users(id) ON DELETE RESTRICT
);

CREATE INDEX idx_execution_logs_change_plan ON execution_logs(change_plan_id);
CREATE INDEX idx_execution_logs_executor ON execution_logs(executor_id);
CREATE INDEX idx_execution_logs_started_at ON execution_logs(started_at DESC);
CREATE INDEX idx_execution_logs_status ON execution_logs(status);

