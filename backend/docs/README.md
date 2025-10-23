# Deeb Backend Documentation

## Overview

This directory contains the database schema design and documentation for the Deeb metadata database.

## Files

### schema.sql
Complete PostgreSQL schema definition including:
- All table definitions with constraints
- Indexes for performance optimization
- Triggers for automatic timestamp updates
- Seed data for permissions and groups

**Usage**:
```bash
# Create the database
psql -U postgres -c "CREATE DATABASE deeb_metadata;"

# Apply schema
psql -U postgres -d deeb_metadata -f backend/docs/schema.sql
```

### schema-erd.md
Entity-Relationship Diagram and comprehensive documentation including:
- Visual ERD using Mermaid syntax
- Detailed table descriptions
- Indexing strategy
- Data integrity constraints
- Security considerations
- Scalability recommendations

## Database Schema Summary

The Deeb metadata database consists of 18 tables organized into four main categories:

### 1. Core Tables
- **users**: User accounts with authentication details and organization-level roles
- **projects**: Projects for organizing database change requests
- **project_members**: Project-level role assignments for users
- **environments**: Environment stages (Dev/Staging/Prod) - organization-level labels
- **database_instances**: Database server/cluster connection details - organization-level
- **databases**: Individual databases within instances, belonging to projects
- **issues**: Database change requests with full lifecycle tracking (targets specific database)

### 2. Compliance & Audit Tables
- **approvals**: Multi-step approval workflow tracking
- **audit_logs**: Complete audit trail of all user actions
- **policy_violations**: Automated policy validation results

### 3. Schema Drift Detection Tables
- **schema_snapshots**: Periodic snapshots of target database schemas
- **schema_drift_events**: Detected schema drift incidents

### 4. RBAC (Role-Based Access Control) Tables
- **groups**: User groups for additional access control
- **user_groups**: Many-to-many relationship between users and groups
- **permissions**: System-level permissions for resources and actions
- **group_permissions**: Permissions assigned to groups
- **notifications**: User notifications for important events

## Key Features

### Core Concepts: Organization Resources

The schema properly reflects the core concepts from the project documentation:

**Environments** - Simple stages in deployment pipeline:
- Dev, Staging, Production labels
- Organization-level resources
- Used to categorize databases by stage
- Can have environment-specific policies

**Database Instances** - Actual database servers:
- MySQL/PostgreSQL servers or clusters
- Connection details (host, port, credentials)
- Encrypted connection details storage
- RDS IAM authentication support
- SSL enforcement
- Managed by Organization Admins and DBAs
- Can host multiple databases

**Databases** - Individual databases within instances:
- **Each database belongs to exactly one project**
- Links project + instance + environment
- Clear ownership and access boundaries
- What developers primarily interact with

### Complete Audit Trail
All user actions are logged in the `audit_logs` table:
- User who performed the action
- Action type (create, update, delete, approve, execute)
- Resource type and ID
- Timestamp, IP address, user agent
- Additional metadata in JSONB format

### Schema Drift Detection
Periodic snapshots of individual databases enable detection of out-of-band changes:
- SHA256 hash for quick comparison
- Detailed JSONB schema definition
- Per-database snapshots (not per environment)
- Drift event tracking with acknowledgment workflow
- Resolution notes for incident management

### Role-Based Access Control
Two-level RBAC system combining organization and project roles:

**Organization Roles** (user-level):
- OrganizationAdmin: Full system access, manages databases and users
- OrganizationDBA: Administer databases across the organization
- OrganizationDeveloper: Member of the organization (default role)

**Project Roles** (project_members table):
- ProjectOwner: Full control over project resources
- ProjectDeveloper: Create and manage change requests
- ProjectApprover: Approve and reject change requests
- ProjectViewer: View-only access to project resources

Additional features:
- Granular permissions (resource.action format)
- Groups for additional access control
- Many-to-many relationships between users, groups, and permissions
- Complete audit trail of permission grants

### Policy Validation
Automated validation of database changes:
- Syntax validation
- Security risk detection
- Performance issue identification
- Backwards compatibility checking
- Best practice enforcement

## Data Types

- **UUIDs**: Used for all primary keys (distributed system compatibility)
- **TIMESTAMP WITH TIME ZONE**: All timestamps include timezone information
- **JSONB**: Flexible metadata storage with GIN indexing
- **INET**: Native PostgreSQL IP address type
- **VARCHAR/TEXT**: Appropriate sizing for text fields

## Indexing Strategy

### Primary Indexes
- All primary keys (UUID)
- All foreign keys
- Unique constraints (email, permission names, etc.)

### Query Optimization Indexes
- Status fields (for filtering by status)
- Timestamp fields (for time-based queries)
- Composite indexes for common query patterns

### Search Indexes
- GIN indexes on JSONB columns for efficient JSON queries
- Email lookup for authentication
- OAuth provider/ID lookup for OAuth authentication

## Security Features

1. **Password Hashing**: Passwords stored as bcrypt hashes
2. **Encrypted Storage**: Sensitive connection details encrypted
3. **2FA Support**: TOTP secret storage for two-factor authentication
4. **Audit Logging**: Complete audit trail with IP and user agent
5. **Soft Deletes**: Status-based deactivation preserves data integrity

## Performance Considerations

### Scalability
- **Partitioning**: `audit_logs` and `schema_snapshots` tables are candidates for time-based partitioning
- **Archival**: Old completed issues can be archived after retention period
- **Index Optimization**: Balance between query performance and write overhead

### Query Performance
- Strategic indexes on foreign keys and frequently queried columns
- GIN indexes for JSONB columns
- Composite indexes for multi-column queries
- Partial indexes for conditional queries (e.g., unread notifications)

## Migration Strategy

The schema is designed to support migrations:
1. Initial schema creation (this file)
2. Future migrations using golang-migrate or similar tool
3. Version tracking for schema changes
4. Rollback capability for failed migrations

## Next Steps

After creating the database schema:
1. Implement migration system (golang-migrate)
2. Create Go models corresponding to tables
3. Implement repository interfaces for data access
4. Add database connection pooling
5. Configure RDS IAM authentication
6. Set up SSL/TLS for database connections

## Additional Resources

- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [UUID Best Practices](https://www.postgresql.org/docs/current/datatype-uuid.html)
- [JSONB Indexing](https://www.postgresql.org/docs/current/datatype-json.html)
- [Foreign Key Constraints](https://www.postgresql.org/docs/current/ddl-constraints.html)
