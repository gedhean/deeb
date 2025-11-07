# SQL Review V2 API Documentation

**Version**: 2.0
**Last Updated**: November 7, 2025

## Overview

The SQL Review V2 API provides comprehensive SQL policy management with 101 rules organized by database engine (MySQL/PostgreSQL) and category. This document covers the new V2 endpoints and schema changes introduced in the SQL Reviewer integration.

## Schema Changes

### Policy Schema Version

All policies now include a `schema_version` field to distinguish between V1 (20 rules) and V2 (101 rules) policies:

```json
{
  "id": "uuid",
  "name": "Production SQL Policy",
  "description": "Comprehensive SQL review policy",
  "type": "SQL_REVIEW",
  "schema_version": "2.0",  // New field: "1.0" or "2.0"
  "policy_config": {
    "rules": { ... }
  },
  "created_at": "2025-11-07T12:00:00Z",
  "updated_at": "2025-11-07T12:00:00Z"
}
```

**Schema Versions**:
- `"1.0"`: Original 20-rule system (V1)
- `"2.0"`: Comprehensive 101-rule system (V2)

## New Endpoints

### 1. Get Migration Status

Returns statistics about policy versions across the organization.

**Endpoint**: `GET /api/v1/policies/migration/status`

**Authorization**: Organization Admin

**Response**:
```json
{
  "status": {
    "total": 25,
    "1.0": 10,
    "2.0": 15,
    "unknown": 0,
    "needs_migration": 10
  }
}
```

**Response Fields**:
- `total`: Total number of policies
- `1.0`: Count of V1 policies
- `2.0`: Count of V2 policies
- `unknown`: Count of policies with unknown version
- `needs_migration`: Count of V1 policies that can be migrated

**Status Codes**:
- `200 OK`: Successfully retrieved status
- `401 Unauthorized`: Missing or invalid authentication
- `403 Forbidden`: User is not an Organization Admin
- `500 Internal Server Error`: Server error

**Example**:
```bash
curl -X GET "https://api.deeb.io/api/v1/policies/migration/status" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

### 2. Migrate Single Policy

Migrates a single V1 policy to V2 (101 rules).

**Endpoint**: `POST /api/v1/policies/:id/migrate`

**Authorization**: Organization Admin

**Parameters**:
- `id` (path): Policy UUID to migrate

**Request Body**: Empty `{}`

**Response**:
```json
{
  "result": {
    "policy_id": "uuid",
    "policy_name": "Production SQL Policy",
    "from_version": "1.0",
    "to_version": "2.0",
    "rules_added": 81,
    "rules_updated": 20,
    "rules_removed": 0,
    "success": true,
    "migrated_at": "2025-11-07T12:00:00Z"
  }
}
```

**Response Fields**:
- `policy_id`: UUID of migrated policy
- `policy_name`: Name of the policy
- `from_version`: Original schema version
- `to_version`: New schema version
- `rules_added`: Number of new rules added (typically 81)
- `rules_updated`: Number of existing rules updated (typically 20)
- `rules_removed`: Number of rules removed (typically 0)
- `success`: Whether migration succeeded
- `error`: Error message (only present if success is false)
- `migrated_at`: Timestamp of migration

**Status Codes**:
- `200 OK`: Migration successful
- `400 Bad Request`: Policy already at V2 or invalid policy
- `401 Unauthorized`: Missing or invalid authentication
- `403 Forbidden`: User is not an Organization Admin
- `404 Not Found`: Policy not found
- `500 Internal Server Error`: Migration failed

**Example**:
```bash
curl -X POST "https://api.deeb.io/api/v1/policies/abc-123/migrate" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}'
```

**Error Response** (Policy already V2):
```json
{
  "error": "policy is already at version 2.0"
}
```

---

### 3. Migrate All Policies

Migrates all V1 policies to V2 in a batch operation.

**Endpoint**: `POST /api/v1/policies/migration/migrate-all`

**Authorization**: Organization Admin

**Request Body**: Empty `{}`

**Response**:
```json
{
  "result": {
    "total_policies": 10,
    "success_count": 9,
    "failure_count": 1,
    "results": [
      {
        "policy_id": "uuid-1",
        "policy_name": "Dev Policy",
        "from_version": "1.0",
        "to_version": "2.0",
        "rules_added": 81,
        "rules_updated": 20,
        "rules_removed": 0,
        "success": true,
        "migrated_at": "2025-11-07T12:00:00Z"
      },
      {
        "policy_id": "uuid-2",
        "policy_name": "Prod Policy",
        "from_version": "1.0",
        "to_version": "2.0",
        "rules_added": 81,
        "rules_updated": 20,
        "rules_removed": 0,
        "success": true,
        "migrated_at": "2025-11-07T12:00:01Z"
      }
    ],
    "started_at": "2025-11-07T12:00:00Z",
    "completed_at": "2025-11-07T12:00:10Z"
  }
}
```

**Response Fields**:
- `total_policies`: Total number of V1 policies found
- `success_count`: Number of successful migrations
- `failure_count`: Number of failed migrations
- `results`: Array of individual migration results
- `started_at`: Batch operation start time
- `completed_at`: Batch operation completion time

**Status Codes**:
- `200 OK`: Batch migration completed (check individual results for success/failure)
- `401 Unauthorized`: Missing or invalid authentication
- `403 Forbidden`: User is not an Organization Admin
- `500 Internal Server Error`: Batch operation failed to start

**Example**:
```bash
curl -X POST "https://api.deeb.io/api/v1/policies/migration/migrate-all" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}'
```

**Note**: This endpoint migrates only V1 policies. V2 policies are automatically skipped.

---

### 4. Get SQL Review Rules

Returns available SQL review rules organized by engine and category.

**Endpoint**: `GET /api/v1/policies/sql-review/rules`

**Authorization**: Organization Admin

**Response**:
```json
{
  "mysql": {
    "Engine": [
      {
        "type": "engine.mysql.use-innodb",
        "name": "Use InnoDB Engine",
        "description": "Require InnoDB storage engine for all tables",
        "category": "Engine",
        "engine": "mysql"
      }
    ],
    "Naming": [ ... ],
    "Table": [ ... ],
    "Column": [ ... ],
    "Index": [ ... ],
    "Schema": [ ... ],
    "Statement": [ ... ]
  },
  "postgresql": {
    "Engine": [ ... ],
    "Naming": [ ... ],
    "Table": [ ... ],
    "Column": [ ... ],
    "System": [ ... ],
    "Index": [ ... ],
    "Schema": [ ... ],
    "Statement": [ ... ]
  }
}
```

**Response Structure**:
- Top-level keys: `mysql`, `postgresql`
- Second-level keys: Rule categories
- Rule fields:
  - `type`: Unique rule identifier
  - `name`: Human-readable rule name
  - `description`: Rule description
  - `category`: Rule category
  - `engine`: Database engine (mysql or postgresql)
  - `parameters`: Optional rule-specific parameters

**Categories**:
- **Engine**: Storage engine requirements
- **Naming**: Naming conventions for tables, columns, indexes
- **Table**: Table structure and configuration
- **Column**: Column definitions and constraints
- **System**: System-level configurations (PostgreSQL only)
- **Index**: Index design and usage
- **Schema**: Schema design patterns
- **Statement**: SQL statement restrictions

**Example**:
```bash
curl -X GET "https://api.deeb.io/api/v1/policies/sql-review/rules" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

## Modified Endpoints

### List Policies

**Changes**: Response now includes `schema_version` field

**Endpoint**: `GET /api/v1/policies`

**Response Example**:
```json
{
  "policies": [
    {
      "id": "uuid",
      "name": "Production Policy",
      "schema_version": "2.0",  // New field
      "type": "SQL_REVIEW",
      ...
    }
  ],
  "count": 10,
  "limit": 100,
  "offset": 0
}
```

---

### Get Policy by ID

**Changes**: Response now includes `schema_version` field

**Endpoint**: `GET /api/v1/policies/:id`

**Response Example**:
```json
{
  "id": "uuid",
  "name": "Production Policy",
  "schema_version": "2.0",  // New field
  "type": "SQL_REVIEW",
  "policy_config": {
    "rules": { ... }
  },
  ...
}
```

---

### Create Policy

**Changes**: Automatically sets `schema_version` to "2.0" for new policies

**Endpoint**: `POST /api/v1/policies`

**Request Body**:
```json
{
  "name": "New Policy",
  "description": "My new SQL review policy",
  "type": "SQL_REVIEW",
  "policy_config": {
    "rules": {
      "engine.mysql.use-innodb": {
        "type": "engine.mysql.use-innodb",
        "level": "error",
        "engine": "mysql"
      }
    }
  }
}
```

**Response**:
```json
{
  "id": "uuid",
  "name": "New Policy",
  "schema_version": "2.0",  // Automatically set
  "type": "SQL_REVIEW",
  ...
}
```

---

### Update Policy

**Changes**: `schema_version` is read-only and cannot be changed via update

**Endpoint**: `PUT /api/v1/policies/:id`

**Note**: To change schema version, use the migration endpoints.

---

## Rule Configuration Format

### V2 Rule Structure

```json
{
  "type": "engine.mysql.use-innodb",
  "level": "error",  // "error" or "warning"
  "engine": "mysql",
  "comment": "All tables must use InnoDB",
  "payload": {
    // Optional rule-specific parameters
  }
}
```

**Required Fields**:
- `type`: Rule type identifier
- `level`: Severity level ("error" or "warning")
- `engine`: Database engine ("mysql" or "postgresql")

**Optional Fields**:
- `comment`: Custom comment explaining the rule
- `payload`: Rule-specific configuration parameters

---

## Review Findings Format

When a change plan is reviewed, findings are returned in the following format:

```json
{
  "policy_findings": {
    "errors": [
      {
        "validator_type": "engine",
        "rule_name": "Use InnoDB Engine",
        "message": "Table 'users' uses MyISAM engine instead of InnoDB",
        "line_number": 5,
        "column_number": 1
      }
    ],
    "warnings": [ ... ],
    "infos": [ ... ],
    "counts": {
      "errors": 1,
      "warnings": 2,
      "infos": 3
    }
  }
}
```

**Finding Fields**:
- `validator_type`: Category of validator (engine, naming, table, etc.)
- `rule_name`: Human-readable rule name
- `message`: Detailed finding message
- `line_number`: SQL line number (optional)
- `column_number`: SQL column number (optional)

---

## Backwards Compatibility

### V1 Policies

Existing V1 policies continue to work without modification:
- V1 policies have `schema_version: "1.0"`
- V1 rules are evaluated using the original 20-rule validator
- No breaking changes to existing V1 policy configurations

### Migration Path

1. **Non-destructive**: Migration creates new V2 rule configuration while preserving V1 rules
2. **Additive**: Adds 81 new rules to existing 20 rules
3. **Reversible**: Can roll back by reverting database migration
4. **Opt-in**: Policies remain at V1 until explicitly migrated

---

## Error Handling

### Standard Error Response

```json
{
  "error": "Detailed error message"
}
```

### Common Error Codes

- `400 Bad Request`: Invalid request parameters or policy already at target version
- `401 Unauthorized`: Missing or invalid authentication token
- `403 Forbidden`: Insufficient permissions (requires Organization Admin)
- `404 Not Found`: Policy not found
- `500 Internal Server Error`: Server-side error during operation

---

## Rate Limiting

All migration endpoints follow standard API rate limits:
- **Single migration**: Standard rate limit applies
- **Batch migration**: No special rate limiting (processes all V1 policies)

---

## Best Practices

### 1. Check Migration Status First

Before migrating, check the status to understand the impact:

```bash
# Get migration status
curl -X GET "https://api.deeb.io/api/v1/policies/migration/status" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 2. Test with Single Policy

Migrate a single test policy first before batch migration:

```bash
# Migrate single policy
curl -X POST "https://api.deeb.io/api/v1/policies/TEST_POLICY_ID/migrate" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{}'
```

### 3. Review Migration Results

Always check the migration results for errors:

```json
{
  "success": false,
  "error": "Invalid policy configuration"
}
```

### 4. Schedule Batch Migrations

For large organizations, schedule batch migrations during maintenance windows.

---

## Integration Examples

### JavaScript/TypeScript

```typescript
import { apiClient } from './api-client';

// Get migration status
async function getMigrationStatus() {
  const response = await apiClient.get('/api/v1/policies/migration/status');
  return response.data.status;
}

// Migrate single policy
async function migratePolicy(policyId: string) {
  const response = await apiClient.post(
    `/api/v1/policies/${policyId}/migrate`,
    {}
  );
  return response.data.result;
}

// Migrate all policies
async function migrateAllPolicies() {
  const response = await apiClient.post(
    '/api/v1/policies/migration/migrate-all',
    {}
  );
  return response.data.result;
}
```

### Python

```python
import requests

class DeebAPI:
    def __init__(self, token: str, base_url: str):
        self.token = token
        self.base_url = base_url
        self.headers = {"Authorization": f"Bearer {token}"}

    def get_migration_status(self):
        response = requests.get(
            f"{self.base_url}/api/v1/policies/migration/status",
            headers=self.headers
        )
        return response.json()["status"]

    def migrate_policy(self, policy_id: str):
        response = requests.post(
            f"{self.base_url}/api/v1/policies/{policy_id}/migrate",
            json={},
            headers=self.headers
        )
        return response.json()["result"]

    def migrate_all_policies(self):
        response = requests.post(
            f"{self.base_url}/api/v1/policies/migration/migrate-all",
            json={},
            headers=self.headers
        )
        return response.json()["result"]
```

---

## See Also

- [User Migration Guide](../user-guides/sql-review-v2-migration.md)
- [Developer Guide](../developer-guides/sql-review-rules.md)
- [SQL Review Rules Reference](../reference/sql-review-rules-reference.md)
- [Deployment Guide](../deployment/sql-review-v2-deployment.md)
