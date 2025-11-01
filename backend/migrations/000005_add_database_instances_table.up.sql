-- Deeb MVP2 - Add Database Instances Table Migration
-- PostgreSQL 15+
-- This migration adds the database_instances table

-- ============================================
-- DATABASE_INSTANCES TABLE
-- Stores database server/cluster connection details - organization-level resources
-- ============================================
CREATE TABLE database_instances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    database_type VARCHAR(50) NOT NULL, -- mysql, postgresql
    host VARCHAR(255) NOT NULL,
    port INTEGER NOT NULL,
    username VARCHAR(255) NOT NULL,
    connection_details_encrypted TEXT, -- Encrypted JSON with connection params
    use_rds_iam_auth BOOLEAN DEFAULT false,
    require_ssl BOOLEAN DEFAULT true,
    status VARCHAR(50) NOT NULL DEFAULT 'Active', -- Active, Inactive, Maintenance
    created_by_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT check_database_type CHECK (database_type IN ('mysql', 'postgresql')),
    CONSTRAINT check_instance_status CHECK (status IN ('Active', 'Inactive', 'Maintenance')),
    CONSTRAINT fk_instances_created_by FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE RESTRICT
);

-- Indexes for database_instances table
CREATE INDEX idx_instances_status ON database_instances(status);
CREATE INDEX idx_instances_name ON database_instances(name);
CREATE INDEX idx_instances_type ON database_instances(database_type);
CREATE INDEX idx_instances_created_by ON database_instances(created_by_id);

-- Apply trigger to database_instances table for updated_at
CREATE TRIGGER update_database_instances_updated_at BEFORE UPDATE ON database_instances
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

