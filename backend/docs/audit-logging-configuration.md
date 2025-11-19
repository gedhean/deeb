# Audit Logging Configuration Guide

This guide explains how to configure the declarative audit logging system in Deeb.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Configuration Structure](#configuration-structure)
- [Default Behavior](#default-behavior)
- [Configuration Examples](#configuration-examples)
- [Best Practices](#best-practices)
- [Available Actions](#available-actions)
- [Available Resource Types](#available-resource-types)
- [Migration Guide](#migration-guide)

---

## Overview

The audit logging system allows you to precisely control **what** gets logged and **what** doesn't through YAML configuration. You can:

- Enable/disable audit logging globally
- Set a default policy (allow-all or deny-all)
- Configure logging per resource type
- Filter by specific actions for each resource
- Define custom paths to skip
- Control sensitive data redaction

---

## Quick Start

### Basic Configuration (Opt-In)

```yaml
audit_logging:
  enabled: true
  default_policy: deny  # Nothing logged unless explicitly enabled
  redaction_enabled: true

  resources:
    user:
      enabled: true
      actions:
        - login
        - logout
        - create_user
```

This configuration:
- Enables audit logging globally
- Uses "deny" policy (opt-in - only logs configured resources/actions)
- Logs user authentication and creation only

### Liberal Configuration (Opt-Out)

```yaml
audit_logging:
  enabled: true
  default_policy: allow  # Log everything by default
  redaction_enabled: true

  resources:
    # Override to disable specific resources
    execution_log:
      enabled: false
      actions: []
```

This configuration:
- Enables audit logging globally
- Uses "allow" policy (opt-out - logs everything except disabled resources)
- Disables logging for execution_log resource

---

## Configuration Structure

### Top-Level Fields

```yaml
audit_logging:
  # Enable/disable audit logging globally
  enabled: true|false

  # Default policy for unconfigured resources
  # "allow" = log everything unless explicitly disabled
  # "deny" = log nothing unless explicitly enabled
  default_policy: allow|deny

  # If true, don't skip any paths (overrides hardcoded health check skips)
  log_all_paths: true|false

  # Additional paths to skip (supports wildcards)
  skip_paths:
    - /metrics
    - /debug/*

  # Per-resource configuration
  resources:
    <resource_type>:
      enabled: true|false
      actions: [<action1>, <action2>, ...]

  # Enable/disable sensitive data redaction
  redaction_enabled: true|false
```

### Resource Rule Structure

Each resource can be configured individually:

```yaml
resources:
  change_plan:
    enabled: true              # Enable logging for this resource
    actions:                   # List of actions to log
      - create_change_plan
      - approve_change_plan
      - execute_change_plan
```

**Empty actions list** means log ALL actions for that resource:

```yaml
resources:
  user:
    enabled: true
    actions: []  # Log all actions (login, logout, create, update, delete, etc.)
```

---

## Default Behavior

### Without Configuration

If no `audit_logging` section is provided in config.yaml:

- **Enabled**: `true`
- **Default Policy**: `deny` (opt-in)
- **Redaction**: `true` (enabled)
- **Resources**: empty map
- **Result**: **Nothing gets logged** (safe default)

### Hardcoded Skip Paths

The following paths are always skipped (unless `log_all_paths: true`):

- Health checks: `/health`, `/healthz`, `/ready`
- Static assets: `/static/*`, `/assets/*`, `*.css`, `*.js`, `*.png`, `*.jpg`, `*.ico`
- Swagger UI: `/api/docs/swagger*`

---

## Configuration Examples

### Example 1: Production Security Focused

Log only sensitive operations (auth, approvals, executions):

```yaml
audit_logging:
  enabled: true
  default_policy: deny
  redaction_enabled: true

  resources:
    user:
      enabled: true
      actions:
        - login
        - logout
        - create_user
        - update_user
        - delete_user

    change_plan:
      enabled: true
      actions:
        - approve_change_plan
        - reject_change_plan
        - execute_change_plan
```

### Example 2: Development Environment

Log everything for debugging:

```yaml
audit_logging:
  enabled: true
  default_policy: allow  # Log everything
  log_all_paths: false
  redaction_enabled: true
  skip_paths:
    - /api/docs/*  # Skip API documentation access
  resources: {}
```

### Example 3: Compliance Focused

Log all database changes and user management:

```yaml
audit_logging:
  enabled: true
  default_policy: deny
  redaction_enabled: true

  resources:
    user:
      enabled: true
      actions: []  # All user actions

    change_plan:
      enabled: true
      actions: []  # All change plan actions

    database:
      enabled: true
      actions:
        - create_database
        - update_database
        - delete_database

    database_instance:
      enabled: true
      actions:
        - create_database_instance
        - update_database_instance
        - delete_database_instance
```

### Example 4: Minimal Logging

Only log authentication and approvals:

```yaml
audit_logging:
  enabled: true
  default_policy: deny
  redaction_enabled: true

  resources:
    user:
      enabled: true
      actions:
        - login
        - logout

    change_plan:
      enabled: true
      actions:
        - approve_change_plan
        - execute_change_plan
```

### Example 5: Custom Skip Paths

Skip metrics and debug endpoints:

```yaml
audit_logging:
  enabled: true
  default_policy: allow
  redaction_enabled: true

  skip_paths:
    - /api/v1/metrics
    - /api/v1/health/*
    - /api/v1/debug/*

  resources: {}
```

---

## Best Practices

### 1. Use Deny Policy for Production

**Recommended for production**:

```yaml
audit_logging:
  default_policy: deny
```

**Why**: Explicit opt-in ensures you only log what you need, reducing storage costs and improving performance.

### 2. Enable Redaction

**Always enable redaction** to protect sensitive data:

```yaml
audit_logging:
  redaction_enabled: true
```

Redaction automatically removes:
- Passwords
- API tokens
- JWT tokens
- Connection strings
- MFA secrets
- Encryption keys

### 3. Be Specific with Actions

Instead of logging ALL actions:

```yaml
# ❌ Not recommended (logs everything)
user:
  enabled: true
  actions: []
```

Be explicit about what you need:

```yaml
# ✅ Recommended (explicit)
user:
  enabled: true
  actions:
    - login
    - create_user
    - delete_user
```

### 4. Review Regularly

Audit logs can grow quickly. Periodically review:
- What resources/actions are being logged
- Storage usage
- Query performance

### 5. Document Your Configuration

Add comments explaining why specific resources/actions are logged:

```yaml
resources:
  change_plan:
    enabled: true
    # Log approvals and executions for compliance (SOX requirement)
    actions:
      - approve_change_plan
      - execute_change_plan
```

---

## Available Actions

### Authentication Actions (3)
- `login`
- `logout`
- `register`

### Generic CRUD Actions (4)
- `create`
- `read`
- `update`
- `delete`

### Change Plan Actions (10)
- `create_change_plan`
- `update_change_plan`
- `delete_change_plan`
- `submit_change_plan`
- `approve_change_plan`
- `reject_change_plan`
- `close_change_plan`
- `execute_change_plan`
- `assign_reviewer`
- `unassign_reviewer`

### Project Actions (6)
- `create_project`
- `update_project`
- `delete_project`
- `add_project_member`
- `remove_project_member`
- `update_project_member_role`

### Database Actions (6)
- `create_database`
- `update_database`
- `delete_database`
- `create_database_instance`
- `update_database_instance`
- `delete_database_instance`

### User Actions (4)
- `create_user`
- `update_user`
- `delete_user`
- `update_user_role`

### Environment Actions (3)
- `create_environment`
- `update_environment`
- `delete_environment`

### Policy Actions (3)
- `create_policy`
- `update_policy`
- `delete_policy`

**Total**: 39 actions

---

## Available Resource Types

1. `user` - User accounts and authentication
2. `project` - Projects and project membership
3. `environment` - Environment configuration (dev, staging, prod)
4. `database` - Database configuration
5. `database_instance` - Database instance connection details
6. `change_plan` - Database change requests
7. `approval` - Change plan approvals
8. `execution_log` - Change execution logs
9. `changelog` - Historical change records
10. `schema_history` - Schema version history
11. `audit_log` - Audit log entries (reading audit logs)
12. `policy` - Policy definitions
13. `policy_finding` - Policy violation findings

**Total**: 13 resource types

---

## Migration Guide

### From Pre-Configuration Behavior

**Old Behavior** (before configuration):
- Everything was logged by default
- No way to filter or exclude specific resources/actions

**New Default Behavior**:
- Nothing logged by default (deny policy)
- Must explicitly configure what to log

### Migration Steps

1. **Start with the migration config** (`config.migration.yaml`)

   Copy this file to `config.yaml` to replicate old behavior:
   ```bash
   cp config.migration.yaml config.yaml
   ```

2. **Review what's being logged**

   Check your audit log database to see volume and types of entries:
   ```sql
   SELECT resource_type, action, COUNT(*)
   FROM audit_logs
   WHERE created_at > NOW() - INTERVAL '7 days'
   GROUP BY resource_type, action
   ORDER BY COUNT(*) DESC;
   ```

3. **Remove unnecessary logging**

   Identify high-volume, low-value entries (like read operations) and remove them:
   ```yaml
   resources:
     change_plan:
       enabled: true
       actions:
         # Remove 'read' to reduce volume
         - create
         - update
         - delete
   ```

4. **Switch to deny policy**

   Once you've identified what you need, switch to deny policy:
   ```yaml
   audit_logging:
     default_policy: deny  # Changed from 'allow'
   ```

5. **Test in staging first**

   Deploy to staging and verify:
   - Expected actions are logged
   - Unexpected actions are NOT logged
   - No performance degradation

### Common Pitfalls

1. **Forgetting to enable resources with deny policy**
   ```yaml
   # ❌ This logs nothing with deny policy
   default_policy: deny
   resources: {}

   # ✅ Must explicitly enable
   default_policy: deny
   resources:
     user:
       enabled: true
       actions: [login, logout]
   ```

2. **Using allow policy without disabling noisy resources**
   ```yaml
   # ❌ This logs EVERYTHING (including reads)
   default_policy: allow
   resources: {}

   # ✅ Disable noisy resources
   default_policy: allow
   resources:
     execution_log:
       enabled: false
   ```

3. **Not testing skip_paths wildcards**
   ```yaml
   skip_paths:
     - /debug/*  # Correct - skips /debug/anything
     - /metrics  # Correct - exact match only
     - /api/*    # Warning - skips ALL API calls!
   ```

---

## Environment Variables

Configuration can be overridden with environment variables (feature not yet implemented but planned):

```bash
# Enable/disable globally
AUDIT_LOGGING_ENABLED=true

# Set default policy
AUDIT_LOGGING_DEFAULT_POLICY=deny

# Enable/disable redaction
AUDIT_LOGGING_REDACTION_ENABLED=true
```

---

## Performance Considerations

### Storage Impact

Audit logs can grow quickly. Example volumes:

- **100 users, 1000 requests/day**: ~365,000 audit logs/year
- **Storage per log**: ~500 bytes (avg)
- **Total storage**: ~180 MB/year

**Recommendation**: Use deny policy and be selective about what you log.

### Query Performance

Audit log queries can be slow on large datasets. Best practices:

1. **Index properly** (already done in migrations):
   - `user_id`
   - `resource_type`, `action`
   - `created_at`

2. **Use time-based queries**:
   ```sql
   WHERE created_at > NOW() - INTERVAL '30 days'
   ```

3. **Archive old logs** (not yet implemented):
   - Move logs older than 1 year to cold storage
   - Keep recent logs in hot database

### Middleware Overhead

Audit logging is asynchronous and has minimal performance impact:

- **Overhead per request**: <1ms (goroutine spawn)
- **Database insert**: Async, doesn't block response
- **Redaction overhead**: ~10µs per log entry

---

## Troubleshooting

### Nothing is being logged

**Check**:
1. Is `enabled: true`?
2. With deny policy, are resources explicitly configured?
3. Are you testing with an authenticated user?
4. Check logs for errors: `grep "Failed to create audit log" /var/log/deeb.log`

### Too much is being logged

**Fix**:
1. Switch to deny policy
2. Remove read actions
3. Add skip_paths for noisy endpoints

### Can't find audit logs for specific action

**Check**:
1. Is the action in the resource's actions list?
2. Is the resource enabled?
3. Check the action name matches exactly (see Available Actions section)

---

## Support

For questions or issues:
- Check the main [audit logging architecture doc](audit-logging-architecture.md)
- Open an issue on GitHub
- Contact the DevOps team
