-- Rollback: Make user_id NOT NULL again
-- First, we need to handle any existing NULL user_ids by setting them to a placeholder
-- We use a special UUID that won't conflict with real users

-- Update any NULL user_ids to use a placeholder UUID
-- This UUID (all zeros except last digit) is reserved for rollback purposes
UPDATE audit_logs
  SET user_id = '00000000-0000-0000-0000-000000000001'
  WHERE user_id IS NULL;

-- Re-add the NOT NULL constraint
ALTER TABLE audit_logs
  ALTER COLUMN user_id SET NOT NULL;

-- Remove the comment
COMMENT ON COLUMN audit_logs.user_id IS NULL;
