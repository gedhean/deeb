-- Latest ghost progress snapshot (Copy: X/Y %, ETA, lag) for UI progress rendering
ALTER TABLE change_plans ADD COLUMN ghost_progress TEXT;
