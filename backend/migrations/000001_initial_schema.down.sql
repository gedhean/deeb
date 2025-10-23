-- Drop tables in reverse order (respecting foreign key dependencies)

DROP TABLE IF EXISTS notifications CASCADE;
DROP TABLE IF EXISTS group_permissions CASCADE;
DROP TABLE IF EXISTS permissions CASCADE;
DROP TABLE IF EXISTS user_groups CASCADE;
DROP TABLE IF EXISTS groups CASCADE;
DROP TABLE IF EXISTS schema_drift_events CASCADE;
DROP TABLE IF EXISTS schema_snapshots CASCADE;
DROP TABLE IF EXISTS audit_logs CASCADE;
DROP TABLE IF EXISTS approvals CASCADE;
DROP TABLE IF EXISTS policy_violations CASCADE;
DROP TABLE IF EXISTS issues CASCADE;
DROP TABLE IF EXISTS databases CASCADE;
DROP TABLE IF EXISTS database_instances CASCADE;
DROP TABLE IF EXISTS environments CASCADE;
DROP TABLE IF EXISTS project_members CASCADE;
DROP TABLE IF EXISTS projects CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- Drop trigger function
DROP FUNCTION IF EXISTS update_updated_at_column();
