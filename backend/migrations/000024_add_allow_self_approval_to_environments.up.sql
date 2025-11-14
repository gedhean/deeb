-- Deeb - Add Allow Self Approval to Environments
-- PostgreSQL 15+
-- This migration adds the allow_self_approval column to environments table

-- Add allow_self_approval column to environments table
-- Default to false for security (require separate reviewer by default)
ALTER TABLE environments
    ADD COLUMN allow_self_approval BOOLEAN NOT NULL DEFAULT false;

-- Add comment to document the column
COMMENT ON COLUMN environments.allow_self_approval IS 'When true, allows developers to approve their own change plans in this environment without requiring a separate reviewer. Useful for dev/staging environments.';
