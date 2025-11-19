-- Make user_id nullable in audit_logs table to support logging unauthenticated events
-- such as failed login attempts where the user cannot be identified
-- NULL user_id explicitly represents "no authenticated user"

ALTER TABLE audit_logs
  ALTER COLUMN user_id DROP NOT NULL;

-- Add comment to document the NULL semantic
COMMENT ON COLUMN audit_logs.user_id IS 'User ID for authenticated operations. NULL for unauthenticated events like failed login attempts.';
