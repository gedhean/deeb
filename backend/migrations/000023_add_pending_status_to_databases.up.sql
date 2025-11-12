-- Migration: Add 'Pending' status to databases table
-- Description: Updates the check_database_status constraint to allow 'Pending' status
--              for databases that are awaiting creation via Change Plans

-- Drop the existing constraint
ALTER TABLE databases DROP CONSTRAINT IF EXISTS check_database_status;

-- Add the updated constraint with 'Pending' status
ALTER TABLE databases ADD CONSTRAINT check_database_status
    CHECK (status IN ('Active', 'Inactive', 'Pending'));
