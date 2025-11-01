-- Deeb MVP2 - Add Project Members Table Migration
-- PostgreSQL 15+
-- This migration adds the project_members table

-- ============================================
-- PROJECT_MEMBERS TABLE
-- Stores project-level role assignments for users
-- ============================================
CREATE TABLE project_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL,
    user_id UUID NOT NULL,
    project_role VARCHAR(50) NOT NULL, -- ProjectOwner, ProjectDeveloper, ProjectApprover, ProjectViewer
    added_by_id UUID NOT NULL,
    added_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT check_project_role CHECK (project_role IN ('ProjectOwner', 'ProjectDeveloper', 'ProjectApprover', 'ProjectViewer')),
    CONSTRAINT unique_project_user UNIQUE (project_id, user_id),
    CONSTRAINT fk_project_members_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    CONSTRAINT fk_project_members_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_project_members_added_by FOREIGN KEY (added_by_id) REFERENCES users(id) ON DELETE RESTRICT
);

-- Indexes for project_members table
CREATE INDEX idx_project_members_project ON project_members(project_id);
CREATE INDEX idx_project_members_user ON project_members(user_id);
CREATE INDEX idx_project_members_role ON project_members(project_role);

