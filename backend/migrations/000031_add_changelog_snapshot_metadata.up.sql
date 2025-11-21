-- Add metadata JSONB column to changelog table
-- This will store snapshot status and error information
-- Example structure:
-- {
--   "snapshot_status": "SUCCESS",  // or "FAILED"
--   "snapshot_error": "error message" // only present when failed
-- }
ALTER TABLE changelog
ADD COLUMN metadata JSONB;

-- Add index on metadata->>'snapshot_status' for filtering
CREATE INDEX idx_changelog_metadata_snapshot_status ON changelog((metadata->>'snapshot_status')) WHERE metadata IS NOT NULL;

-- Add comment explaining the metadata field
COMMENT ON COLUMN changelog.metadata IS 'Metadata containing snapshot status ("SUCCESS" or "FAILED") and snapshot_error if capture failed';
