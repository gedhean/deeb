-- Deeb MVP2 - Add Change History Table
-- PostgreSQL 15+
-- This migration creates the change_history table for tracking field-level changes to Change Plans

CREATE TABLE change_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    change_plan_id UUID NOT NULL,
    field_name VARCHAR(100) NOT NULL,
    old_value TEXT,
    new_value TEXT,
    changed_by_id UUID NOT NULL,
    changed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_change_history_change_plan FOREIGN KEY (change_plan_id) REFERENCES change_plans(id) ON DELETE CASCADE,
    CONSTRAINT fk_change_history_changed_by FOREIGN KEY (changed_by_id) REFERENCES users(id) ON DELETE RESTRICT
);

CREATE INDEX idx_change_history_change_plan ON change_history(change_plan_id);
CREATE INDEX idx_change_history_changed_at ON change_history(changed_at DESC);
CREATE INDEX idx_change_history_changed_by ON change_history(changed_by_id);

