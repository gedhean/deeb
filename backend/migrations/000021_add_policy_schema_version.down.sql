-- Drop schema_version column from policies table
DROP INDEX IF EXISTS idx_policies_schema_version;

ALTER TABLE policies
DROP COLUMN IF EXISTS schema_version;
