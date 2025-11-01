-- Deeb MVP2 - Initial Schema Migration
-- PostgreSQL 15+
-- This migration creates the users table and supporting infrastructure

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
    is_active BOOLEAN NOT NULL DEFAULT true,
    oauth_provider VARCHAR(50), -- NULL, 'google', 'github'
    oauth_id VARCHAR(255), -- Provider's user ID
    two_factor_enabled BOOLEAN DEFAULT false,
    two_factor_secret VARCHAR(255), -- TOTP secret for 2FA
    last_login_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT check_email_format CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    CONSTRAINT check_organization_role CHECK (organization_role IN ('OrganizationAdmin', 'OrganizationDBA', 'OrganizationDeveloper'))
);

-- Indexes for users table
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_oauth ON users(oauth_provider, oauth_id) WHERE oauth_provider IS NOT NULL;
CREATE INDEX idx_users_is_active ON users(is_active);
CREATE INDEX idx_users_organization_role ON users(organization_role);

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

-- Apply trigger to users table
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
