-- Add AwaitingCleanup to change plan statuses: after ghost cutover, the
-- plan waits for the _<table>_del backup drop (the completion gate).
ALTER TABLE change_plans DROP CONSTRAINT IF EXISTS change_plans_status_check;
ALTER TABLE change_plans DROP CONSTRAINT IF EXISTS check_change_plan_status;
ALTER TABLE change_plans ADD CONSTRAINT change_plans_status_check
    CHECK (status IN ('Draft', 'Open', 'Approved', 'Closed', 'InProgress', 'AwaitingCutover', 'AwaitingCleanup', 'Completed', 'Failed', 'Cancelled'));
