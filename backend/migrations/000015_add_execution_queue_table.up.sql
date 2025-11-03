-- Deeb MVP2 - Add Execution Queue Table
-- PostgreSQL 15+
-- This migration creates the execution_queue table for queuing Change Plans when database is locked

CREATE TABLE execution_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    change_plan_id UUID NOT NULL,
    target_database_id UUID NOT NULL,
    executor_id UUID NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'Pending', -- Pending, InProgress, Completed, Failed, Cancelled
    position INTEGER NOT NULL DEFAULT 0, -- Position in queue (0 = next to execute)
    priority INTEGER NOT NULL DEFAULT 0, -- Higher priority = execute first (optional, for future use)
    error_message TEXT, -- Error details if execution failed
    queued_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP WITH TIME ZONE, -- When execution started
    completed_at TIMESTAMP WITH TIME ZONE, -- When execution completed/failed
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT check_queue_status CHECK (status IN ('Pending', 'InProgress', 'Completed', 'Failed', 'Cancelled')),
    CONSTRAINT fk_execution_queue_change_plan FOREIGN KEY (change_plan_id) REFERENCES change_plans(id) ON DELETE CASCADE,
    CONSTRAINT fk_execution_queue_database FOREIGN KEY (target_database_id) REFERENCES databases(id) ON DELETE CASCADE,
    CONSTRAINT fk_execution_queue_executor FOREIGN KEY (executor_id) REFERENCES users(id) ON DELETE RESTRICT
);

CREATE INDEX idx_execution_queue_status ON execution_queue(status);
CREATE INDEX idx_execution_queue_database ON execution_queue(target_database_id);
CREATE INDEX idx_execution_queue_position ON execution_queue(target_database_id, position);
CREATE INDEX idx_execution_queue_priority ON execution_queue(target_database_id, priority DESC, position);
CREATE INDEX idx_execution_queue_created_at ON execution_queue(created_at);

-- Ensure only one InProgress item per database at a time
CREATE UNIQUE INDEX idx_execution_queue_unique_in_progress ON execution_queue(target_database_id) 
WHERE status = 'InProgress';

