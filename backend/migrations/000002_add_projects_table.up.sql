-- Deeb MVP2 - Add Projects Table Migration
-- PostgreSQL 15+
-- This migration adds the projects table

-- ============================================
-- PROJECTS TABLE
-- Stores projects for organizing database changes and team members
-- ============================================
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    created_by_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT unique_project_name UNIQUE (name),
    CONSTRAINT fk_projects_created_by FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE RESTRICT
);

-- Indexes for projects table
CREATE INDEX idx_projects_name ON projects(name);
CREATE INDEX idx_projects_created_by ON projects(created_by_id);

-- Apply trigger to projects table for updated_at
CREATE TRIGGER update_projects_updated_at BEFORE UPDATE ON projects
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

