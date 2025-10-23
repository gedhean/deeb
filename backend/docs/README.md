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

The Deeb metadata database consists of 14 tables organized into four main categories:

### 1. Core Tables
- **users**: User accounts with authentication details
- **environments**: Database environment configurations (Dev/Staging/Prod)
- **issues**: Database change requests with full lifecycle tracking

### 2. Compliance & Audit Tables
- **approvals**: Multi-step approval workflow tracking
- **audit_logs**: Complete audit trail of all user actions
- **policy_violations**: Automated policy validation results

### 3. Schema Drift Detection Tables
- **schema_snapshots**: Periodic snapshots of target database schemas
- **schema_drift_events**: Detected schema drift incidents

### 4. RBAC (Role-Based Access Control) Tables
- **groups**: User groups (Developers, Approvers, Administrators)
- **user_groups**: Many-to-many relationship between users and groups
- **permissions**: System-level permissions for resources and actions
- **group_permissions**: Permissions assigned to groups
- **notifications**: User notifications for important events

## Key Features

### Multi-Environment Support
The schema supports multiple database environments (Development, Staging, Production) with separate configurations for each:
- Database type (MySQL/PostgreSQL)
- Connection details (encrypted)
- RDS IAM authentication support
- SSL enforcement

### Complete Audit Trail
All user actions are logged in the `audit_logs` table:
- User who performed the action
- Action type (create, update, delete, approve, execute)
- Resource type and ID
- Timestamp, IP address, user agent
- Additional metadata in JSONB format

### Schema Drift Detection
Periodic snapshots of target database schemas enable detection of out-of-band changes:
- SHA256 hash for quick comparison
- Detailed JSONB schema definition
- Drift event tracking with acknowledgment workflow
- Resolution notes for incident management

### Role-Based Access Control
Flexible RBAC system with groups and permissions:
- Pre-defined groups (Developers, Approvers, Administrators)
- Granular permissions (resource.action format)
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
