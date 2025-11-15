-- Deeb - Drop Database Schema Table
-- PostgreSQL 15+
-- This migration drops the database_schema table

-- Drop indexes
DROP INDEX IF EXISTS idx_database_schema_synced_at;
DROP INDEX IF EXISTS idx_database_schema_database_id;

-- Drop table
DROP TABLE IF EXISTS database_schema;
