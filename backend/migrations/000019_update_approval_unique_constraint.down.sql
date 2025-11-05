-- Deeb - Update Approval Unique Constraint Migration (Down)
-- Revert the unique constraint change on approvals table

-- Drop the new unique constraint
ALTER TABLE approvals DROP CONSTRAINT IF EXISTS unique_approval_per_user_change_plan_status;

-- Restore the old unique constraint (without status)
ALTER TABLE approvals ADD CONSTRAINT unique_approval_per_user_change_plan UNIQUE (change_plan_id, approver_id);
