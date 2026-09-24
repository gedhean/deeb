-- Store the latest ghost readiness check report on the change plan
ALTER TABLE change_plans ADD COLUMN ghost_checks JSONB;
