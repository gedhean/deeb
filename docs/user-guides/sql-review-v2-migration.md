# SQL Review V2 Migration Guide

**Version**: 2.0
**Last Updated**: November 7, 2025
**Audience**: Organization Administrators, DBAs

## Overview

SQL Review V2 introduces a comprehensive rule system with **101 rules** (up from 20), providing enhanced coverage for:
- ✅ MySQL and PostgreSQL best practices
- ✅ Performance optimization patterns
- ✅ Security requirements
- ✅ Backwards compatibility checks
- ✅ Naming conventions
- ✅ Index design validation

This guide walks you through upgrading your existing SQL review policies from V1 to V2.

---

## What's New in V2

### Comprehensive Rule Coverage

| Category | V1 Rules | V2 Rules | Added |
|----------|----------|----------|-------|
| **Engine** | 2 | 5 | +3 |
| **Naming** | 3 | 15 | +12 |
| **Table** | 4 | 18 | +14 |
| **Column** | 5 | 22 | +17 |
| **Index** | 3 | 20 | +17 |
| **Schema** | 2 | 12 | +10 |
| **Statement** | 1 | 9 | +8 |
| **Total** | **20** | **101** | **+81** |

### Enhanced Validation

V2 provides deeper analysis across multiple dimensions:

1. **Security**: SQL injection prevention, privilege management
2. **Performance**: Query optimization, index usage, table scans
3. **Backwards Compatibility**: Migration safety, breaking changes
4. **Best Practices**: Industry standards, maintainability
5. **Syntax Validation**: Engine-specific syntax checking

---

## Before You Migrate

### Prerequisites

✅ **Role Required**: Organization Admin

✅ **Checklist**:
- [ ] Review current policies and their configurations
- [ ] Understand which environments use which policies
- [ ] Communicate changes to development teams
- [ ] Schedule migration during low-activity period
- [ ] Have rollback plan ready

### Understanding the Impact

#### Non-Breaking Changes
- ✅ Existing V1 rules continue to work
- ✅ No changes to existing rule configurations
- ✅ Backwards compatible with V1 policies
- ✅ Can test with single policy before full migration

#### What Changes
- 📊 81 new rules added to your policies
- 🔍 More comprehensive change plan reviews
- 📈 Potentially more findings in SQL reviews
- 📝 Enhanced review feedback with more categories

---

## Migration Options

### Option 1: Single Policy Migration (Recommended for Testing)

Migrate one policy at a time to test the impact.

**Best For**:
- Testing V2 behavior
- Gradual rollout
- High-risk policies

**Steps**:
1. Navigate to **Policies** page
2. Find a non-production policy
3. Click **"Upgrade to V2"** button
4. Review migration results
5. Test with sample SQL changes

### Option 2: Batch Migration

Migrate all V1 policies at once.

**Best For**:
- Small number of policies
- Consistent policy standards
- Scheduled maintenance windows

**Steps**:
1. Navigate to **Policies** page
2. Review migration status banner
3. Click **"Migrate All to V2"** button
4. Confirm the operation
5. Review batch migration results

---

## Step-by-Step Migration Guide

### Step 1: Check Migration Status

1. Log in to Deeb as an **Organization Admin**
2. Navigate to **Policies** page
3. View the **Migration Status** card at the top

![Migration Status](./images/migration-status.png)

The status card shows:
- **Total Policies**: All SQL review policies
- **V1 Policies**: Policies using 20 rules
- **V2 Policies**: Policies using 101 rules
- **Needs Migration**: V1 policies ready for upgrade

---

### Step 2: Test with a Single Policy

1. **Select a Test Policy**
   - Choose a non-production policy
   - Preferably one used in Dev or Staging

2. **Locate the Policy Card**
   - Find the policy in the policies list
   - Look for the **V1** badge (red)

3. **Click "Upgrade to V2"**
   - Click the upgrade button on the policy card
   - Confirm the migration

4. **Review Migration Results**

   Success message shows:
   ```
   ✅ Policy "Dev SQL Policy" migrated successfully
   Added 81 new rules
   ```

5. **Verify the Changes**
   - Policy badge now shows **V2** (green)
   - Policy edit page shows 101 rules
   - Test with a sample change plan

---

### Step 3: Test with Sample SQL

Create a test change plan to see V2 in action:

```sql
-- Test SQL with various patterns
CREATE TABLE users (
    id INT PRIMARY KEY,
    email VARCHAR(100),
    password VARCHAR(50)  -- V2 will flag this as too short
);

-- Missing index on foreign key
ALTER TABLE orders ADD COLUMN user_id INT;

-- Non-InnoDB engine (MySQL)
CREATE TABLE logs (
    id INT PRIMARY KEY
) ENGINE=MyISAM;  -- V2 will flag this
```

**Expected V2 Findings**:
- ⚠️ Warning: Password column length too short (min 255)
- ❌ Error: Missing index on foreign key column
- ❌ Error: Non-InnoDB storage engine not allowed

---

### Step 4: Review Findings

Navigate to the change plan and review the SQL Review Results:

1. **Filter by Severity**
   - Click **"Filter"** button
   - Select severity (Errors, Warnings, Info)

2. **Search by Rule Name**
   - Enter rule name in search box
   - Example: "InnoDB", "index", "password"

3. **Filter by Validator Type**
   - Enter validator category
   - Example: "engine", "security", "performance"

4. **Review Each Finding**
   - Read the detailed message
   - Check line numbers
   - Understand the recommendation

---

### Step 5: Migrate Remaining Policies

Once comfortable with V2 behavior:

#### Option A: Migrate One-by-One

For each policy:
1. Click **"Upgrade to V2"** on policy card
2. Wait for success notification
3. Verify the policy shows V2 badge

#### Option B: Batch Migration

For all policies at once:
1. Click **"Migrate All to V2"** in migration status card
2. Confirm the batch operation
3. Wait for completion (may take 30-60 seconds)
4. Review batch results:
   ```
   ✅ 9 policies migrated successfully
   ❌ 1 policy failed to migrate
   ```
5. Investigate any failures

---

## What to Expect After Migration

### Immediate Changes

1. **More Findings**
   - SQL reviews will be more comprehensive
   - Expect 2-3x more findings initially
   - Many will be warnings, not errors

2. **Enhanced Feedback**
   - More specific error messages
   - Line and column numbers for issues
   - Category-based organization

3. **Stricter Standards**
   - Some previously passing SQL may now fail
   - Warnings about potential issues
   - Best practice recommendations

### Gradual Improvement

Over time, you'll see:
- 📈 Higher code quality
- 🛡️ Fewer security vulnerabilities
- ⚡ Better performance patterns
- 🔄 Safer schema migrations

---

## Understanding V2 Rules

### Rule Categories

#### 1. Engine Rules
Enforce storage engine requirements:
- ✅ **Use InnoDB**: Require InnoDB for all MySQL tables
- ✅ **Character Set**: Enforce UTF-8 character sets
- ✅ **Collation**: Standard collation requirements

#### 2. Naming Conventions
Ensure consistent naming:
- 📝 **Table Names**: snake_case, lowercase
- 📝 **Column Names**: Descriptive, no reserved words
- 📝 **Index Names**: Meaningful prefixes (idx_, uk_, fk_)

#### 3. Table Design
Validate table structure:
- 🔑 **Primary Keys**: Required on all tables
- 📊 **Row Format**: Optimize storage format
- 🔒 **Auto-increment**: Best practices for ID columns

#### 4. Column Design
Check column definitions:
- 🔢 **Data Types**: Appropriate type selection
- 📏 **Column Size**: Reasonable length limits
- 🚫 **Nullability**: Explicit NULL/NOT NULL

#### 5. Index Design
Optimize index usage:
- ⚡ **Foreign Key Indexes**: Required on FK columns
- 🎯 **Composite Indexes**: Optimal column order
- 📈 **Index Cardinality**: High-selectivity columns

#### 6. Schema Changes
Safe migration patterns:
- ⚠️ **Backwards Compatibility**: Non-breaking changes
- 🔄 **Column Drops**: Verify no dependencies
- 📋 **Data Migrations**: Separate DDL from DML

#### 7. Statement Restrictions
SQL statement validation:
- 🛡️ **Injection Prevention**: Parameterized queries
- 🚫 **Dangerous Patterns**: Avoid wildcards, SELECT *
- ⏱️ **Query Limits**: LIMIT clauses on large tables

---

## Common Migration Scenarios

### Scenario 1: New Findings After Migration

**Problem**: SQL that passed V1 now fails V2

**Solution**:
1. Review the specific finding
2. Understand the rule's purpose
3. Options:
   - **Fix the SQL** (recommended)
   - **Downgrade rule to warning** (temporary)
   - **Disable specific rule** (not recommended)

**Example**:
```sql
-- V1: Passed
CREATE TABLE users (password VARCHAR(50));

-- V2: Fails (password column too short)
-- Fix: Increase to VARCHAR(255)
CREATE TABLE users (password VARCHAR(255));
```

---

### Scenario 2: Too Many Warnings

**Problem**: Overwhelmed by warning-level findings

**Solution**:
1. Use filters to focus on errors first
2. Gradually address warnings over time
3. Consider adjusting rule levels:
   - Change warnings to info for non-critical rules
   - Keep errors for critical rules

**Prioritization**:
1. ❌ **Errors first**: Security, performance, breaking changes
2. ⚠️ **Warnings second**: Best practices, naming conventions
3. ℹ️ **Info last**: Optional improvements

---

### Scenario 3: Legacy Schema Issues

**Problem**: Old tables don't meet V2 standards

**Solution**:
1. **Grandfather Existing Tables**:
   - Create cleanup plan
   - Address on new changes only

2. **Incremental Improvements**:
   - Fix issues as you touch tables
   - Don't require full refactor upfront

3. **Custom Policy for Legacy**:
   - Create separate policy with relaxed rules
   - Apply to legacy databases
   - Use strict V2 policy for new projects

---

## Customizing V2 Policies

### Adjusting Rule Levels

You can customize any rule's severity:

1. Navigate to **Policies** → **Edit Policy**
2. Find the specific rule
3. Change level:
   - `error`: Blocks change plan approval
   - `warning`: Shows warning but allows approval

**Example**:
```json
{
  "naming.table.no-leading-underscore": {
    "type": "naming.table.no-leading-underscore",
    "level": "warning",  // Changed from "error"
    "engine": "mysql",
    "comment": "Prefer tables without leading underscores"
  }
}
```

---

### Disabling Specific Rules

To disable a rule entirely, remove it from the policy configuration.

**⚠️ Warning**: Disabling rules reduces code quality protection.

**When to Disable**:
- Rule doesn't apply to your use case
- Conflicts with organizational standards
- Legacy compatibility requirements

**How to Disable**:
1. Edit policy
2. Find rule in configuration
3. Delete the rule entry
4. Save policy

---

## Troubleshooting

### Migration Failed

**Error**: "Migration failed: invalid policy configuration"

**Causes**:
- Corrupted policy configuration
- Missing required fields
- Invalid rule format

**Solution**:
1. View policy details
2. Check for configuration errors
3. Contact support if issue persists

---

### Policy Shows Unknown Version

**Error**: Policy has no version badge

**Causes**:
- Database migration not run
- Manual policy creation issue

**Solution**:
1. Check if database migration 000021 was applied
2. Run `make migrate` if missing
3. Policy should default to V2 for new policies

---

### Can't Find Migrate Button

**Issue**: No "Upgrade to V2" button visible

**Causes**:
- Not logged in as Organization Admin
- Policy already at V2
- Policy type is not SQL_REVIEW

**Solution**:
1. Verify Organization Admin role
2. Check policy badge (V1 vs V2)
3. Verify policy type is SQL_REVIEW

---

## Rollback Procedure

If you need to rollback V2 migration:

### Rollback Single Policy

Currently, single policy rollback is not supported through UI. Contact support for assistance.

### Rollback All Policies (Database Level)

**⚠️ Warning**: This rolls back ALL policies to V1.

```bash
# Rollback migration
make migrate-down

# Confirm rollback
psql -d deeb -c "SELECT schema_version, COUNT(*) FROM policies GROUP BY schema_version;"
```

---

## Best Practices

### 1. Communicate Early

Inform development teams about:
- Migration timeline
- Expected new findings
- How to address failures
- Where to get help

### 2. Start Small

- Test with one policy first
- Validate with sample SQL
- Get team feedback
- Roll out gradually

### 3. Monitor Impact

Track metrics:
- Number of findings per change plan
- Rule categories triggering most
- Time to address findings
- Developer satisfaction

### 4. Provide Training

Help teams understand:
- Why rules exist
- How to fix common issues
- Best practices for SQL
- When to ask for exceptions

### 5. Iterate on Configuration

- Adjust rule levels based on feedback
- Disable rules that don't fit
- Add custom comments to explain rules
- Review policy quarterly

---

## Getting Help

### Documentation
- [API Documentation](../api/sql-review-v2-api.md)
- [SQL Review Rules Reference](../reference/sql-review-rules-reference.md)
- [Developer Guide](../developer-guides/sql-review-rules.md)

### Support Channels
- **Email**: support@deeb.io
- **Slack**: #deeb-support
- **Documentation**: https://docs.deeb.io

### Common Questions

**Q: Will existing change plans be re-reviewed?**
A: No, only new change plans use V2 rules.

**Q: Can I rollback a single policy?**
A: Currently not supported through UI. Contact support.

**Q: Do I have to migrate all policies?**
A: No, V1 policies continue to work. Migration is optional.

**Q: What if a rule conflicts with our standards?**
A: Customize the policy by adjusting rule levels or disabling specific rules.

**Q: How long does batch migration take?**
A: Typically 30-60 seconds for 10-20 policies.

---

## Next Steps

After successful migration:

1. ✅ **Monitor Initial Impact**
   - Review first week of findings
   - Gather developer feedback
   - Adjust rules as needed

2. ✅ **Create Documentation**
   - Document custom rule configurations
   - Share SQL best practices
   - Provide fix examples

3. ✅ **Continuous Improvement**
   - Quarterly policy reviews
   - Track quality metrics
   - Refine rules based on usage

---

## Summary

SQL Review V2 provides:
- ✅ 101 comprehensive rules (up from 20)
- ✅ Enhanced security and performance validation
- ✅ Backwards compatible migration path
- ✅ Flexible customization options
- ✅ Better developer experience

Take your time with the migration, test thoroughly, and don't hesitate to reach out for support!
