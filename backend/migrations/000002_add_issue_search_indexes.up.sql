-- Add GIN indexes for full-text search on title and description
-- Using pg_trgm extension for trigram-based similarity search with ILIKE queries

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS idx_issues_title_trgm ON issues USING gin (title gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_issues_description_trgm ON issues USING gin (description gin_trgm_ops);
