# SQL Review V2 Deployment Guide

**Version**: 2.0
**Last Updated**: November 7, 2025
**Audience**: DevOps, System Administrators, Release Managers

## Overview

This guide covers deploying SQL Review V2 (101 rules) to production environments. It includes database migration steps, rollback procedures, monitoring setup, and troubleshooting.

---

## Prerequisites

### System Requirements

**No changes to system requirements**. SQL Review V2 uses existing infrastructure.

### Pre-Deployment Checklist

- [ ] Review [Phase 5 Validation Report](../phase-5-validation-report.md)
- [ ] Backup production database
- [ ] Schedule maintenance window (optional, but recommended)
- [ ] Notify users of potential new SQL review findings
- [ ] Prepare rollback plan
- [ ] Test deployment in staging environment

---

## Deployment Steps

### 1. Pre-Deployment Verification

#### A. Verify Current State

```bash
# Check current database schema version
psql -d deeb_production -c "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1;"

# Should show: 000020 (or earlier)
```

#### B. Backup Database

```bash
# Create backup
pg_dump deeb_production > backup_$(date +%Y%m%d_%H%M%S).sql

# Verify backup
ls -lh backup_*.sql
```

#### C. Test in Staging

```bash
# Deploy to staging first
./deploy.sh staging

# Run smoke tests
./scripts/smoke-test.sh staging

# Verify:
# - Database migration successful
# - API endpoints responding
# - Frontend migration UI visible
# - Test migration of a policy
```

---

### 2. Database Migration

SQL Review V2 requires one database migration:

**Migration**: `000021_add_policy_schema_version`

#### A. Review Migration SQL

**Up Migration** (`migrations/000021_add_policy_schema_version.up.sql`):
```sql
-- Add schema_version column to policies table
ALTER TABLE policies
ADD COLUMN schema_version VARCHAR(10) NOT NULL DEFAULT '2.0';

-- Add index for faster version queries
CREATE INDEX idx_policies_schema_version ON policies(schema_version);

-- Update existing policies to V1
UPDATE policies
SET schema_version = '1.0'
WHERE created_at < NOW();
```

**Down Migration** (`migrations/000021_add_policy_schema_version.down.sql`):
```sql
-- Remove index
DROP INDEX IF EXISTS idx_policies_schema_version;

-- Remove column
ALTER TABLE policies
DROP COLUMN IF EXISTS schema_version;
```

#### B. Run Migration

```bash
# Production deployment
cd backend

# Run migration
make migrate

# Verify migration
psql -d deeb_production -c "\d policies"
# Should show schema_version column

# Check version
psql -d deeb_production -c "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1;"
# Should show: 000021
```

#### C. Verify Data Integrity

```bash
# Check all policies have schema_version
psql -d deeb_production -c "
SELECT
    schema_version,
    COUNT(*) as count
FROM policies
GROUP BY schema_version;
"

# Expected output:
# schema_version | count
# ---------------+-------
# 1.0            |    25
# 2.0            |     0
```

---

### 3. Application Deployment

#### A. Build Application

```bash
# Backend
cd backend
go build -o deeb-server ./cmd

# Frontend
cd frontend
pnpm install
pnpm run build
```

#### B. Deploy Backend

```bash
# Stop current backend
systemctl stop deeb-backend

# Deploy new binary
cp deeb-server /usr/local/bin/deeb-server

# Update configuration (if needed)
# No configuration changes required for V2

# Start backend
systemctl start deeb-backend

# Verify
systemctl status deeb-backend
journalctl -u deeb-backend -f
```

#### C. Deploy Frontend

```bash
# Deploy frontend build
rsync -avz frontend/dist/ /var/www/deeb/

# Verify
curl https://app.deeb.io/health
```

---

### 4. Post-Deployment Verification

#### A. Health Checks

```bash
# Backend health
curl https://api.deeb.io/health

# Expected:
# {"status":"healthy","version":"2.0.0","timestamp":"2025-11-07T12:00:00Z"}

# Check database connection
curl https://api.deeb.io/health | jq '.database'
# Expected: "connected"
```

#### B. API Verification

```bash
# Test migration status endpoint
curl -H "Authorization: Bearer $TOKEN" \
  https://api.deeb.io/api/v1/policies/migration/status

# Expected:
# {
#   "status": {
#     "total": 25,
#     "1.0": 25,
#     "2.0": 0,
#     "unknown": 0,
#     "needs_migration": 25
#   }
# }

# Test SQL review rules endpoint
curl -H "Authorization: Bearer $TOKEN" \
  https://api.deeb.io/api/v1/policies/sql-review/rules | jq 'keys'

# Expected: ["mysql", "postgresql"]
```

#### C. Frontend Verification

1. Log in to Deeb UI
2. Navigate to **Policies** page
3. Verify:
   - Migration status card visible
   - V1 badges on existing policies
   - "Upgrade to V2" buttons visible
   - No JavaScript errors in console

#### D. Functional Verification

```bash
# Test single policy migration
curl -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}' \
  https://api.deeb.io/api/v1/policies/$POLICY_ID/migrate

# Verify response
# {"result":{"success":true,"rules_added":81,...}}
```

---

### 5. Enable Features

#### A. No Feature Flags Required

SQL Review V2 is enabled by default after deployment. No feature flags or configuration changes needed.

#### B. Gradual Rollout (Optional)

If you want to roll out gradually:

1. **Communicate to users**:
   - Send announcement about V2 availability
   - Provide link to migration guide
   - Set expectations for new findings

2. **Monitor adoption**:
   ```sql
   -- Track migration progress
   SELECT
       schema_version,
       COUNT(*) as count,
       ROUND(100.0 * COUNT(*) / SUM(COUNT(*)) OVER(), 1) as percentage
   FROM policies
   GROUP BY schema_version;
   ```

---

## Rollback Procedure

### When to Rollback

Consider rollback if:
- Database migration fails
- Critical bugs discovered
- Unexpected performance degradation
- User blocking issues

### Rollback Steps

#### 1. Rollback Application

```bash
# Backend
systemctl stop deeb-backend
cp /backup/deeb-server /usr/local/bin/deeb-server
systemctl start deeb-backend

# Frontend
rsync -avz /backup/frontend/dist/ /var/www/deeb/
```

#### 2. Rollback Database

```bash
# CAUTION: This reverts ALL policies to remove schema_version

# Backup current state first
pg_dump deeb_production > backup_before_rollback_$(date +%Y%m%d_%H%M%S).sql

# Rollback migration
cd backend
make migrate-down

# Verify
psql -d deeb_production -c "\d policies"
# schema_version column should be gone
```

#### 3. Verify Rollback

```bash
# Check health
curl https://api.deeb.io/health

# Test policy listing
curl -H "Authorization: Bearer $TOKEN" \
  https://api.deeb.io/api/v1/policies

# Verify: No schema_version field in response
```

#### 4. Communicate Rollback

- Notify users of rollback
- Explain reason for rollback
- Provide timeline for re-deployment

---

## Monitoring

### Key Metrics

#### 1. Migration Progress

```sql
-- Dashboard query
SELECT
    schema_version,
    COUNT(*) as policy_count,
    ARRAY_AGG(name ORDER BY name) as policy_names
FROM policies
GROUP BY schema_version;
```

#### 2. Review Performance

```sql
-- Average review duration by policy version
SELECT
    p.schema_version,
    AVG(EXTRACT(EPOCH FROM (cp.updated_at - cp.created_at))) as avg_review_seconds
FROM change_plans cp
JOIN databases d ON cp.target_database_id = d.id
JOIN environments e ON d.environment_id = e.id
JOIN policies p ON e.policy_id = p.id
WHERE cp.created_at > NOW() - INTERVAL '7 days'
GROUP BY p.schema_version;
```

#### 3. Finding Statistics

```sql
-- Average findings per change plan by policy version
SELECT
    p.schema_version,
    AVG(COALESCE(
        (cp.policy_findings->'counts'->>'errors')::int, 0
    )) as avg_errors,
    AVG(COALESCE(
        (cp.policy_findings->'counts'->>'warnings')::int, 0
    )) as avg_warnings
FROM change_plans cp
JOIN databases d ON cp.target_database_id = d.id
JOIN environments e ON d.environment_id = e.id
JOIN policies p ON e.policy_id = p.id
WHERE cp.created_at > NOW() - INTERVAL '7 days'
  AND cp.policy_findings IS NOT NULL
GROUP BY p.schema_version;
```

### Alerts

Set up alerts for:

1. **Migration Failures**:
   ```
   Alert: Policy migration failure rate > 5%
   Query: Count(failed_migrations) / Count(total_migrations)
   ```

2. **Performance Degradation**:
   ```
   Alert: SQL review duration > 5 seconds
   Query: P95(review_duration_ms) > 5000
   ```

3. **Error Rate Increase**:
   ```
   Alert: API error rate > 1%
   Query: Count(5xx_responses) / Count(total_responses)
   ```

### Logging

Enable debug logging for troubleshooting:

```yaml
# config.yaml
logging:
  level: info  # or debug for troubleshooting
  components:
    - review_service: debug
    - migration_service: debug
```

---

## Performance Tuning

### Database Indexes

Migration creates index automatically:
```sql
CREATE INDEX idx_policies_schema_version ON policies(schema_version);
```

### Query Optimization

V2 uses same query patterns as V1. No additional optimization needed.

### Caching

Consider adding cache layer for:
- Rule metadata (cached in-memory by default)
- Policy configurations (cache-aside pattern)

Example:
```go
// In-memory cache for policies
var policyCache = cache.New(5*time.Minute, 10*time.Minute)

func (r *PolicyRepository) GetByID(id string) (*Policy, error) {
    // Check cache
    if cached, found := policyCache.Get(id); found {
        return cached.(*Policy), nil
    }

    // Load from DB
    policy, err := r.loadFromDB(id)
    if err != nil {
        return nil, err
    }

    // Cache for 5 minutes
    policyCache.Set(id, policy, cache.DefaultExpiration)
    return policy, nil
}
```

---

## Troubleshooting

### Issue: Migration Hangs

**Symptoms**: Migration command doesn't complete

**Causes**:
- Long-running transactions blocking
- Lock contention on policies table

**Solution**:
```sql
-- Check for blocking queries
SELECT
    pg_blocking_pids(pid) as blocked_by,
    query,
    state
FROM pg_stat_activity
WHERE datname = 'deeb_production'
  AND query LIKE '%policies%';

-- Terminate blocking query (if safe)
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE pid IN (SELECT unnest(pg_blocking_pids(YOUR_MIGRATION_PID)));
```

---

### Issue: Migration Fails

**Symptoms**: Migration exits with error

**Error**: "column schema_version already exists"

**Solution**:
```sql
-- Check if migration already ran
SELECT version FROM schema_migrations WHERE version = '000021';

-- If exists, skip migration
-- If not exists, check table structure:
\d policies

-- If column exists but migration not recorded:
INSERT INTO schema_migrations (version) VALUES ('000021');
```

---

### Issue: API Returns 500 Errors

**Symptoms**: Migration endpoints return Internal Server Error

**Debugging**:
```bash
# Check logs
journalctl -u deeb-backend -n 100

# Common issues:
# 1. Database connection issues
# 2. Missing schema_version column
# 3. Invalid policy configurations
```

**Solution**:
```bash
# Verify database connection
psql -d deeb_production -c "SELECT 1;"

# Verify column exists
psql -d deeb_production -c "\d policies"

# Check application logs for specific error
```

---

### Issue: Frontend Shows Errors

**Symptoms**: JavaScript console errors or blank pages

**Debugging**:
1. Open browser DevTools (F12)
2. Check Console tab for errors
3. Check Network tab for failed API calls

**Common Issues**:
- API endpoint URL mismatch
- CORS configuration issues
- Authentication token expired

**Solution**:
```bash
# Verify API is accessible
curl https://api.deeb.io/api/v1/policies/migration/status

# Check CORS headers
curl -I -X OPTIONS https://api.deeb.io/api/v1/policies/migration/status

# Verify frontend build
ls -la /var/www/deeb/
```

---

### Issue: High Memory Usage

**Symptoms**: Backend memory usage increases significantly

**Causes**:
- Large policy configurations
- Too many concurrent reviews
- Memory leak in validator

**Monitoring**:
```bash
# Check memory usage
ps aux | grep deeb-server

# Check Go memory stats
curl localhost:6060/debug/pprof/heap
```

**Solution**:
1. Increase memory limits if needed
2. Review policy sizes (should be < 1MB)
3. Check for memory leaks in logs
4. Contact support if issue persists

---

## Security Considerations

### 1. API Endpoints

All migration endpoints require **Organization Admin** role:
- `GET /api/v1/policies/migration/status`
- `POST /api/v1/policies/:id/migrate`
- `POST /api/v1/policies/migration/migrate-all`

Verify RBAC enforcement:
```bash
# Test without admin role (should fail)
curl -X POST \
  -H "Authorization: Bearer $NON_ADMIN_TOKEN" \
  https://api.deeb.io/api/v1/policies/migration/migrate-all

# Expected: 403 Forbidden
```

### 2. Database Access

Migration requires database write access. Ensure:
- Migration runs with appropriate privileges
- Production database credentials secured
- Audit logging enabled

### 3. Audit Trail

All migrations are logged:
```sql
-- Check audit logs
SELECT *
FROM audit_logs
WHERE action = 'policy.migrate'
  AND timestamp > NOW() - INTERVAL '1 day'
ORDER BY timestamp DESC;
```

---

## Maintenance

### Regular Tasks

#### Daily
- Monitor migration adoption rate
- Check for API errors
- Review performance metrics

#### Weekly
- Analyze SQL review findings trends
- Review policy configurations
- Check database performance

#### Monthly
- Review and optimize slow queries
- Update documentation
- Plan rule additions/modifications

---

## Support

### Getting Help

**Documentation**:
- [API Documentation](../api/sql-review-v2-api.md)
- [User Migration Guide](../user-guides/sql-review-v2-migration.md)
- [Developer Guide](../developer-guides/sql-review-architecture.md)

**Support Channels**:
- **Email**: support@deeb.io
- **Slack**: #deeb-support
- **Docs**: https://docs.deeb.io

### Reporting Issues

When reporting issues, include:
1. Error messages from logs
2. Steps to reproduce
3. Environment details (staging/production)
4. Database migration version
5. Application version

---

## Appendix

### A. Complete Deployment Checklist

```
Pre-Deployment:
☐ Review validation report
☐ Backup database
☐ Schedule maintenance window
☐ Notify users
☐ Test in staging

Deployment:
☐ Run database migration
☐ Verify migration success
☐ Deploy backend
☐ Deploy frontend
☐ Run health checks
☐ Verify API endpoints
☐ Test frontend UI
☐ Test migration functionality

Post-Deployment:
☐ Monitor metrics
☐ Check error logs
☐ Communicate success
☐ Document any issues
☐ Update runbooks

Rollback Ready:
☐ Database backup verified
☐ Rollback scripts tested
☐ Communication plan ready
```

### B. Emergency Contacts

```
Role                    | Contact
------------------------|------------------
On-Call Engineer        | oncall@deeb.io
Database Admin          | dba@deeb.io
Release Manager         | releases@deeb.io
Product Owner           | product@deeb.io
```

### C. Database Migration SQL

**Complete Up Migration**:
```sql
BEGIN;

-- Add column with default
ALTER TABLE policies
ADD COLUMN schema_version VARCHAR(10) NOT NULL DEFAULT '2.0';

-- Set existing policies to V1
UPDATE policies
SET schema_version = '1.0'
WHERE created_at < CURRENT_TIMESTAMP;

-- Add index
CREATE INDEX idx_policies_schema_version
ON policies(schema_version);

-- Verify
DO $$
DECLARE
    v_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_count
    FROM policies
    WHERE schema_version IS NULL;

    IF v_count > 0 THEN
        RAISE EXCEPTION 'Found % policies with NULL schema_version', v_count;
    END IF;
END $$;

COMMIT;
```

**Complete Down Migration**:
```sql
BEGIN;

-- Remove index
DROP INDEX IF EXISTS idx_policies_schema_version;

-- Remove column
ALTER TABLE policies
DROP COLUMN IF EXISTS schema_version;

-- Verify
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'policies'
        AND column_name = 'schema_version'
    ) THEN
        RAISE EXCEPTION 'Column schema_version still exists';
    END IF;
END $$;

COMMIT;
```

---

## Changelog

| Date | Version | Changes |
|------|---------|---------|
| 2025-11-07 | 1.0 | Initial deployment guide for SQL Review V2 |

---

**End of Deployment Guide**
