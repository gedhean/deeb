-- Deeb - Rollback Allow Self Approval from Environments
-- PostgreSQL 15+
-- This migration removes the allow_self_approval column from environments table

-- Drop allow_self_approval column from environments table
ALTER TABLE environments
    DROP COLUMN IF EXISTS allow_self_approval;
