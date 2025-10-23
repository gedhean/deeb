-- Deeb Metadata Database Schema
-- PostgreSQL 15+

-- ============================================
-- USERS TABLE
-- Stores user accounts with authentication details
-- ============================================
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255), -- NULL for OAuth-only users
    display_name VARCHAR(255),
    organization_role VARCHAR(50) NOT NULL DEFAULT 'OrganizationDeveloper', -- OrganizationAdmin, OrganizationDBA, OrganizationDeveloper
    status VARCHAR(50) NOT NULL DEFAULT 'Active', -- Active, Inactive, Suspended
    oauth_provider VARCHAR(50), -- NULL, 'google', 'github'
    oauth_id VARCHAR(255), -- Provider's user ID
    two_factor_enabled BOOLEAN DEFAULT false,
    two_factor_secret VARCHAR(255), -- TOTP secret for 2FA
    last_login_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT check_email_format CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    CONSTRAINT check_organization_role CHECK (organization_role IN ('OrganizationAdmin', 'OrganizationDBA', 'OrganizationDeveloper')),
    CONSTRAINT check_status CHECK (status IN ('Active', 'Inactive', 'Suspended'))
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_oauth ON users(oauth_provider, oauth_id) WHERE oauth_provider IS NOT NULL;
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_organization_role ON users(organization_role);

-- ============================================
-- PROJECTS TABLE
-- Stores projects for organizing database change requests
-- ============================================
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'Active', -- Active, Archived
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by_id UUID NOT NULL,

    CONSTRAINT check_project_status CHECK (status IN ('Active', 'Archived')),
    CONSTRAINT fk_projects_created_by FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE RESTRICT
);

CREATE INDEX idx_projects_status ON projects(status);
CREATE INDEX idx_projects_name ON projects(name);
CREATE INDEX idx_projects_created_by ON projects(created_by_id);

-- ============================================
-- PROJECT_MEMBERS TABLE
-- Stores project-level role assignments for users
-- ============================================
CREATE TABLE project_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL,
    user_id UUID NOT NULL,
    project_role VARCHAR(50) NOT NULL, -- ProjectOwner, ProjectDeveloper, ProjectApprover, ProjectViewer
    added_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    added_by_id UUID NOT NULL,

    CONSTRAINT check_project_role CHECK (project_role IN ('ProjectOwner', 'ProjectDeveloper', 'ProjectApprover', 'ProjectViewer')),
    CONSTRAINT fk_project_members_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    CONSTRAINT fk_project_members_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_project_members_added_by FOREIGN KEY (added_by_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT unique_project_user UNIQUE (project_id, user_id)
);

CREATE INDEX idx_project_members_project ON project_members(project_id);
CREATE INDEX idx_project_members_user ON project_members(user_id);
CREATE INDEX idx_project_members_role ON project_members(project_role);

-- ============================================
-- ENVIRONMENTS TABLE
-- Stores database environment configurations (organization-level)
-- ============================================
CREATE TABLE environments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL, -- Dev, Staging, Production
    description TEXT,
    database_type VARCHAR(50) NOT NULL, -- mysql, postgresql
    host VARCHAR(255) NOT NULL,
    port INTEGER NOT NULL,
    database_name VARCHAR(255) NOT NULL,
    username VARCHAR(255) NOT NULL,
    connection_details_encrypted TEXT, -- Encrypted JSON with connection params
    use_rds_iam_auth BOOLEAN DEFAULT false,
    require_ssl BOOLEAN DEFAULT true,
    status VARCHAR(50) NOT NULL DEFAULT 'Active', -- Active, Inactive, Maintenance
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by_id UUID NOT NULL,

    CONSTRAINT check_database_type CHECK (database_type IN ('mysql', 'postgresql')),
    CONSTRAINT check_environment_status CHECK (status IN ('Active', 'Inactive', 'Maintenance')),
    CONSTRAINT fk_environments_created_by FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE RESTRICT
);

CREATE INDEX idx_environments_status ON environments(status);
CREATE INDEX idx_environments_name ON environments(name);
CREATE INDEX idx_environments_created_by ON environments(created_by_id);

-- ============================================
-- ISSUES TABLE
-- Stores database change requests
-- ============================================
CREATE TABLE issues (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    change_type VARCHAR(50) NOT NULL, -- DDL, DML
    sql_content TEXT NOT NULL,
    rollback_sql TEXT, -- Optional rollback script
    status VARCHAR(50) NOT NULL DEFAULT 'Draft', -- Draft, PendingReview, PendingApproval, Approved, Scheduled, InProgress, Completed, Failed, Cancelled
    risk_level VARCHAR(50) NOT NULL DEFAULT 'Medium', -- Low, Medium, High, Critical
    target_environment_id UUID NOT NULL,
    author_id UUID NOT NULL,
    assigned_reviewer_id UUID,
    scheduled_for TIMESTAMP WITH TIME ZONE, -- NULL for immediate execution
    executed_at TIMESTAMP WITH TIME ZONE,
    execution_duration_ms INTEGER, -- Execution time in milliseconds
    affected_rows INTEGER, -- Number of rows affected
    error_message TEXT, -- Error details if execution failed
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT check_change_type CHECK (change_type IN ('DDL', 'DML')),
    CONSTRAINT check_issue_status CHECK (status IN ('Draft', 'PendingReview', 'PendingApproval', 'Approved', 'Scheduled', 'InProgress', 'Completed', 'Failed', 'Cancelled')),
    CONSTRAINT check_risk_level CHECK (risk_level IN ('Low', 'Medium', 'High', 'Critical')),
    CONSTRAINT fk_issues_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    CONSTRAINT fk_issues_environment FOREIGN KEY (target_environment_id) REFERENCES environments(id) ON DELETE RESTRICT,
    CONSTRAINT fk_issues_author FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT fk_issues_reviewer FOREIGN KEY (assigned_reviewer_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_issues_project ON issues(project_id);
CREATE INDEX idx_issues_status ON issues(status);
CREATE INDEX idx_issues_author ON issues(author_id);
CREATE INDEX idx_issues_environment ON issues(target_environment_id);
CREATE INDEX idx_issues_reviewer ON issues(assigned_reviewer_id) WHERE assigned_reviewer_id IS NOT NULL;
CREATE INDEX idx_issues_scheduled ON issues(scheduled_for) WHERE scheduled_for IS NOT NULL;
CREATE INDEX idx_issues_created_at ON issues(created_at DESC);
CREATE INDEX idx_issues_risk_level ON issues(risk_level);

-- ============================================
-- POLICY_VIOLATIONS TABLE
-- Stores automated policy validation results
-- ============================================
CREATE TABLE policy_violations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    issue_id UUID NOT NULL,
    violation_type VARCHAR(100) NOT NULL, -- syntax_error, security_risk, performance_issue, backwards_incompatible, best_practice_violation
    severity VARCHAR(50) NOT NULL, -- Info, Warning, Error, Critical
    rule_name VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    line_number INTEGER, -- Line number in SQL content where violation occurs
    detected_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT check_violation_type CHECK (violation_type IN ('syntax_error', 'security_risk', 'performance_issue', 'backwards_incompatible', 'best_practice_violation')),
    CONSTRAINT check_severity CHECK (severity IN ('Info', 'Warning', 'Error', 'Critical')),
    CONSTRAINT fk_violations_issue FOREIGN KEY (issue_id) REFERENCES issues(id) ON DELETE CASCADE
);

CREATE INDEX idx_violations_issue ON policy_violations(issue_id);
CREATE INDEX idx_violations_severity ON policy_violations(severity);

-- ============================================
-- APPROVALS TABLE
-- Stores approval/rejection records for issues
-- ============================================
CREATE TABLE approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    issue_id UUID NOT NULL,
    approver_id UUID NOT NULL,
    status VARCHAR(50) NOT NULL, -- Approved, Rejected, Pending
    comments TEXT,
    approved_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT check_approval_status CHECK (status IN ('Approved', 'Rejected', 'Pending')),
    CONSTRAINT fk_approvals_issue FOREIGN KEY (issue_id) REFERENCES issues(id) ON DELETE CASCADE,
    CONSTRAINT fk_approvals_approver FOREIGN KEY (approver_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT unique_approval_per_user_issue UNIQUE (issue_id, approver_id)
);

CREATE INDEX idx_approvals_issue ON approvals(issue_id);
CREATE INDEX idx_approvals_approver ON approvals(approver_id);
CREATE INDEX idx_approvals_status ON approvals(status);
CREATE INDEX idx_approvals_created_at ON approvals(created_at DESC);

-- ============================================
-- AUDIT_LOGS TABLE
-- Stores all user actions for compliance
-- ============================================
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    action VARCHAR(100) NOT NULL, -- create_issue, approve_issue, reject_issue, execute_issue, update_user, etc.
    resource_type VARCHAR(100) NOT NULL, -- issue, user, environment, approval
    resource_id UUID NOT NULL,
    details JSONB, -- Additional metadata about the action
    ip_address INET,
    user_agent TEXT,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_audit_logs_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT
);

CREATE INDEX idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp DESC);
CREATE INDEX idx_audit_logs_details ON audit_logs USING GIN (details); -- For JSON queries

-- ============================================
-- SCHEMA_SNAPSHOTS TABLE
-- Stores periodic snapshots of target database schemas for drift detection
-- ============================================
CREATE TABLE schema_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    environment_id UUID NOT NULL,
    schema_hash VARCHAR(64) NOT NULL, -- SHA256 hash of schema definition
    schema_definition JSONB NOT NULL, -- JSON representation of tables, columns, indexes
    captured_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_snapshots_environment FOREIGN KEY (environment_id) REFERENCES environments(id) ON DELETE CASCADE
);

CREATE INDEX idx_snapshots_environment ON schema_snapshots(environment_id);
CREATE INDEX idx_snapshots_captured_at ON schema_snapshots(captured_at DESC);
CREATE INDEX idx_snapshots_hash ON schema_snapshots(schema_hash);

-- ============================================
-- SCHEMA_DRIFT_EVENTS TABLE
-- Records detected schema drift incidents
-- ============================================
CREATE TABLE schema_drift_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    environment_id UUID NOT NULL,
    snapshot_id UUID NOT NULL,
    drift_type VARCHAR(100) NOT NULL, -- table_added, table_removed, column_modified, index_changed
    affected_object VARCHAR(255) NOT NULL, -- Table or column name
    drift_details JSONB NOT NULL, -- Detailed diff information
    detected_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    acknowledged_by_id UUID,
    acknowledged_at TIMESTAMP WITH TIME ZONE,
    resolution_notes TEXT,

    CONSTRAINT check_drift_type CHECK (drift_type IN ('table_added', 'table_removed', 'column_added', 'column_removed', 'column_modified', 'index_added', 'index_removed', 'constraint_changed')),
    CONSTRAINT fk_drift_environment FOREIGN KEY (environment_id) REFERENCES environments(id) ON DELETE CASCADE,
    CONSTRAINT fk_drift_snapshot FOREIGN KEY (snapshot_id) REFERENCES schema_snapshots(id) ON DELETE CASCADE,
    CONSTRAINT fk_drift_acknowledged_by FOREIGN KEY (acknowledged_by_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_drift_environment ON schema_drift_events(environment_id);
CREATE INDEX idx_drift_snapshot ON schema_drift_events(snapshot_id);
CREATE INDEX idx_drift_detected_at ON schema_drift_events(detected_at DESC);
CREATE INDEX idx_drift_acknowledged ON schema_drift_events(acknowledged_at) WHERE acknowledged_at IS NOT NULL;

-- ============================================
-- GROUPS TABLE
-- User groups for RBAC
-- ============================================
CREATE TABLE groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_groups_name ON groups(name);

-- ============================================
-- USER_GROUPS TABLE
-- Many-to-many relationship between users and groups
-- ============================================
CREATE TABLE user_groups (
    user_id UUID NOT NULL,
    group_id UUID NOT NULL,
    added_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    added_by_id UUID NOT NULL,

    PRIMARY KEY (user_id, group_id),
    CONSTRAINT fk_user_groups_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_user_groups_group FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    CONSTRAINT fk_user_groups_added_by FOREIGN KEY (added_by_id) REFERENCES users(id) ON DELETE RESTRICT
);

CREATE INDEX idx_user_groups_user ON user_groups(user_id);
CREATE INDEX idx_user_groups_group ON user_groups(group_id);

-- ============================================
-- PERMISSIONS TABLE
-- System permissions
-- ============================================
CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    resource_type VARCHAR(100) NOT NULL, -- issue, user, environment, approval
    action VARCHAR(100) NOT NULL, -- create, read, update, delete, approve, execute

    CONSTRAINT unique_permission UNIQUE (resource_type, action)
);

CREATE INDEX idx_permissions_resource ON permissions(resource_type, action);

-- ============================================
-- GROUP_PERMISSIONS TABLE
-- Permissions assigned to groups
-- ============================================
CREATE TABLE group_permissions (
    group_id UUID NOT NULL,
    permission_id UUID NOT NULL,
    granted_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    granted_by_id UUID NOT NULL,

    PRIMARY KEY (group_id, permission_id),
    CONSTRAINT fk_group_permissions_group FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    CONSTRAINT fk_group_permissions_permission FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE,
    CONSTRAINT fk_group_permissions_granted_by FOREIGN KEY (granted_by_id) REFERENCES users(id) ON DELETE RESTRICT
);

CREATE INDEX idx_group_permissions_group ON group_permissions(group_id);
CREATE INDEX idx_group_permissions_permission ON group_permissions(permission_id);

-- ============================================
-- NOTIFICATIONS TABLE
-- User notifications for important events
-- ============================================
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    notification_type VARCHAR(100) NOT NULL, -- issue_approved, issue_rejected, approval_requested, schema_drift_detected
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    related_resource_type VARCHAR(100),
    related_resource_id UUID,
    is_read BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    read_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT fk_notifications_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_notifications_user ON notifications(user_id);
CREATE INDEX idx_notifications_unread ON notifications(user_id, is_read, created_at) WHERE is_read = false;
CREATE INDEX idx_notifications_created_at ON notifications(created_at DESC);

-- ============================================
-- UPDATED_AT TRIGGER FUNCTION
-- Automatically updates updated_at timestamp
-- ============================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply trigger to tables with updated_at column
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_projects_updated_at BEFORE UPDATE ON projects
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_environments_updated_at BEFORE UPDATE ON environments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_issues_updated_at BEFORE UPDATE ON issues
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_groups_updated_at BEFORE UPDATE ON groups
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- SEED DATA FOR PERMISSIONS
-- ============================================
INSERT INTO permissions (name, description, resource_type, action) VALUES
    -- Project permissions
    ('project.create', 'Create new projects', 'project', 'create'),
    ('project.read', 'View project information', 'project', 'read'),
    ('project.update', 'Modify project information', 'project', 'update'),
    ('project.delete', 'Delete projects', 'project', 'delete'),
    ('project.manage_members', 'Add/remove project members', 'project', 'manage_members'),
    -- Issue permissions
    ('issue.create', 'Create new database change issues', 'issue', 'create'),
    ('issue.read', 'View database change issues', 'issue', 'read'),
    ('issue.update', 'Modify database change issues', 'issue', 'update'),
    ('issue.delete', 'Delete database change issues', 'issue', 'delete'),
    ('issue.approve', 'Approve database change issues', 'issue', 'approve'),
    ('issue.execute', 'Execute approved database changes', 'issue', 'execute'),
    -- User permissions
    ('user.create', 'Create new users', 'user', 'create'),
    ('user.read', 'View user information', 'user', 'read'),
    ('user.update', 'Modify user information', 'user', 'update'),
    ('user.delete', 'Deactivate users', 'user', 'delete'),
    -- Environment permissions (organization-level)
    ('environment.create', 'Create new environments', 'environment', 'create'),
    ('environment.read', 'View environment configurations', 'environment', 'read'),
    ('environment.update', 'Modify environment configurations', 'environment', 'update'),
    ('environment.delete', 'Delete environment configurations', 'environment', 'delete'),
    -- Audit log permissions
    ('audit_log.read', 'View audit logs', 'audit_log', 'read');
