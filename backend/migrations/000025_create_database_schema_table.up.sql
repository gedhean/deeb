-- Deeb - Create Database Schema Table
-- PostgreSQL 15+
-- This migration creates the database_schema table to cache database schema metadata

-- Create database_schema table to store cached schema metadata
CREATE TABLE IF NOT EXISTS database_schema (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    database_id UUID NOT NULL REFERENCES databases(id) ON DELETE CASCADE,
    schema_data JSONB NOT NULL,
    synced_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_database_schema_database FOREIGN KEY (database_id) REFERENCES databases(id) ON DELETE CASCADE
);

-- Create unique index on database_id (one schema cache per database)
CREATE UNIQUE INDEX idx_database_schema_database_id ON database_schema(database_id);

-- Create index on synced_at for finding stale schemas
CREATE INDEX idx_database_schema_synced_at ON database_schema(synced_at);

-- Add comments to document the table and columns
COMMENT ON TABLE database_schema IS 'Caches database schema metadata (tables, columns, indexes, etc.) to avoid repeated extraction from target databases';
COMMENT ON COLUMN database_schema.id IS 'Primary key';
COMMENT ON COLUMN database_schema.database_id IS 'Reference to the database this schema belongs to';
COMMENT ON COLUMN database_schema.schema_data IS 'JSON representation of the schema metadata (tables, columns, indexes, foreign keys, etc.)';
COMMENT ON COLUMN database_schema.synced_at IS 'Timestamp when the schema was last synced from the target database';
COMMENT ON COLUMN database_schema.created_at IS 'Timestamp when this cache entry was created';
COMMENT ON COLUMN database_schema.updated_at IS 'Timestamp when this cache entry was last updated';
