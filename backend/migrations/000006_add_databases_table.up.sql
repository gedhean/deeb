-- Deeb MVP2 - Add Databases Table Migration
-- PostgreSQL 15+
-- This migration adds the databases table

-- ============================================
-- DATABASES TABLE
-- Individual databases within instances, belonging to projects
-- ============================================
CREATE TABLE databases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL,
    database_instance_id UUID NOT NULL,
    environment_id UUID NOT NULL,
    database_name VARCHAR(255) NOT NULL, -- Actual database name on the instance
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'Active', -- Active, Inactive
    created_by_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT check_database_status CHECK (status IN ('Active', 'Inactive')),
    CONSTRAINT unique_database_per_instance UNIQUE (database_instance_id, database_name),
    CONSTRAINT fk_databases_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    CONSTRAINT fk_databases_instance FOREIGN KEY (database_instance_id) REFERENCES database_instances(id) ON DELETE RESTRICT,
    CONSTRAINT fk_databases_environment FOREIGN KEY (environment_id) REFERENCES environments(id) ON DELETE RESTRICT,
    CONSTRAINT fk_databases_created_by FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE RESTRICT
);

-- Indexes for databases table
CREATE INDEX idx_databases_project ON databases(project_id);
CREATE INDEX idx_databases_instance ON databases(database_instance_id);
CREATE INDEX idx_databases_environment ON databases(environment_id);
CREATE INDEX idx_databases_status ON databases(status);
CREATE INDEX idx_databases_created_by ON databases(created_by_id);

-- Apply trigger to databases table for updated_at
CREATE TRIGGER update_databases_updated_at BEFORE UPDATE ON databases
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

