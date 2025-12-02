-- Statement execution results for multi-statement change plans
CREATE TABLE statement_execution_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_log_id UUID NOT NULL REFERENCES execution_logs(id) ON DELETE CASCADE,
    statement_index INTEGER NOT NULL,
    statement_sql TEXT NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('Completed', 'Failed', 'Skipped')),
    rows_affected INTEGER,
    execution_duration_ms INTEGER NOT NULL,
    error_message TEXT,
    start_line INTEGER,
    start_column INTEGER,
    end_line INTEGER,
    end_column INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT unique_execution_statement UNIQUE (execution_log_id, statement_index)
);

CREATE INDEX idx_statement_results_execution_log ON statement_execution_results(execution_log_id);

-- Add statement count to change_plans for quick reference
ALTER TABLE change_plans ADD COLUMN statement_count INTEGER;
