-- Deeb - Add Unique Constraint to Policy Findings
-- PostgreSQL 15+
-- This migration adds a unique constraint to prevent duplicate policy findings

-- First, remove any existing duplicates before adding the constraint
-- Keep only the earliest created_at for each unique finding
DELETE FROM policy_findings a
WHERE a.ctid NOT IN (
    SELECT MIN(b.ctid)
    FROM policy_findings b
    WHERE b.change_plan_id = a.change_plan_id
      AND b.validator_type = a.validator_type
      AND b.severity = a.severity
      AND b.rule_name = a.rule_name
      AND b.message = a.message
      AND COALESCE(b.line_number, -1) = COALESCE(a.line_number, -1)
      AND COALESCE(b.column_number, -1) = COALESCE(a.column_number, -1)
    GROUP BY
        b.change_plan_id,
        b.validator_type,
        b.severity,
        b.rule_name,
        b.message,
        COALESCE(b.line_number, -1),
        COALESCE(b.column_number, -1)
);

-- Add unique constraint to prevent future duplicates
-- Note: We use COALESCE for nullable columns in the constraint
CREATE UNIQUE INDEX IF NOT EXISTS idx_policy_findings_unique ON policy_findings (
    change_plan_id,
    validator_type,
    severity,
    rule_name,
    message,
    COALESCE(line_number, -1),
    COALESCE(column_number, -1)
);
