-- Rollback: Remove 'Pending' status from databases table
-- Description: Reverts the check_database_status constraint to only allow 'Active' and 'Inactive'

-- Begin transaction to ensure atomicity
BEGIN;

-- Update all 'Pending' statuses to 'Inactive' before re-adding the constraint
UPDATE databases SET status = 'Inactive' WHERE status = 'Pending';

-- Drop the constraint if it exists
ALTER TABLE databases DROP CONSTRAINT IF EXISTS check_database_status;

-- Restore the original constraint without 'Pending'
ALTER TABLE databases ADD CONSTRAINT check_database_status
    CHECK (status IN ('Active', 'Inactive'));

COMMIT;
