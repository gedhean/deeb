-- Deeb - Rollback Unique Constraint on Policy Findings
-- PostgreSQL 15+
-- This migration removes the unique constraint from policy_findings

DROP INDEX IF EXISTS idx_policy_findings_unique;
