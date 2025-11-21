-- Drop metadata field from changelog table
DROP INDEX IF EXISTS idx_changelog_metadata_snapshot_status;

ALTER TABLE changelog
DROP COLUMN IF EXISTS metadata;
