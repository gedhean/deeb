-- Deeb MVP2 - Add Audit Logs Table Migration (Down)
-- Drop audit_logs table and supporting infrastructure

-- Drop audit_logs table
DROP TABLE IF EXISTS audit_logs CASCADE;

