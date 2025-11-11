-- Add schema_version column to policies table
ALTER TABLE policies
ADD COLUMN schema_version VARCHAR(10) NOT NULL DEFAULT '1.0';

-- Add comment explaining the version
COMMENT ON COLUMN policies.schema_version IS 'Schema version for policy configuration (1.0 = original 20 rules, 2.0 = comprehensive 101 rules)';

-- Create index for querying by schema version
CREATE INDEX idx_policies_schema_version ON policies(schema_version);
