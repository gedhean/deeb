-- Remove GIN indexes for full-text search

DROP INDEX IF EXISTS idx_issues_description_trgm;
DROP INDEX IF EXISTS idx_issues_title_trgm;

-- Note: We don't drop the pg_trgm extension as other parts of the system might use it
