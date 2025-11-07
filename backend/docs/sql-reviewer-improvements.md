# SQL-Reviewer Integration Improvements Plan

**Status**: 📋 Planning
**Created**: 2025-11-07
**Priority**: HIGH
**Owner**: Engineering Team

---

## Executive Summary

### Current State

The Deeb platform has implemented basic sql-reviewer integration, but is significantly underutilizing the package's capabilities:

- **Rule Coverage**: Only ~20 placeholder rules exposed out of 99+ available rules (20% utilization)
- **Rule Type Accuracy**: Rule types in Deeb don't match sql-reviewer's actual schema
- **Feature Utilization**: Not leveraging high-level API, schema-aware validation, or result filtering
- **Configuration**: No payload parameter schemas or validation

### Critical Issues

1. **Incorrect Rule Type Mapping**
   ```
   Deeb:   mysql.engine.no-select-all
   Actual: statement.select.no-select-all
   ```
   Rule types include incorrect engine prefixes and don't align with sql-reviewer's structure.

2. **Missing Rules**: 79+ rules unavailable to users (~80% of sql-reviewer capabilities)

3. **Hardcoded Metadata**: Rule catalog is manually maintained and will drift from sql-reviewer updates

4. **No Payload Configuration**: Users cannot configure rule parameters (e.g., max table name length, required columns)

### Goals

Transform Deeb from a basic SQL validator to a comprehensive SQL quality platform by:

1. ✅ Exposing all 99+ sql-reviewer rules with correct types
2. ✅ Auto-generating rule metadata from sql-reviewer's schema
3. ✅ Implementing payload parameter configuration and validation
4. ✅ Leveraging advanced features (schema-aware validation, result filtering, context support)
5. ✅ Providing complete user documentation and migration path

### Impact

- **Users**: Access to 4x more rules for comprehensive SQL quality enforcement
- **Security**: Advanced rules for detecting vulnerabilities (CASCADE operations, missing WHERE clauses)
- **Performance**: Rules for detecting full table scans, filesort, temporary tables
- **Compliance**: Comment requirements, naming conventions, audit trail rules
- **Developer Experience**: Better error messages, rule organization, and configuration UI

---

## Current Integration Analysis

### What Works ✅

1. **Basic Advisor Wrapper** (`backend/internal/review/sqlreviewer/advisor.go`)
   - Wraps sql-reviewer's `advisor.Check()` function
   - Converts Deeb types to sql-reviewer types
   - Returns `[]*types.Advice` results

2. **Policy CRUD** (`backend/internal/handlers/policy_handler.go`)
   - Full create/read/update/delete for policies
   - Environment-policy relationships
   - JSON/JSONB storage for configurations

3. **Review Service** (`backend/internal/review/service.go`)
   - Loads policies from database
   - Parses policy configs
   - Executes reviews on change plans
   - Stores results in `PolicyFindingsSummary`

4. **Frontend UI** (`frontend/src/components/sql-review-rule-selector.tsx`)
   - Rule selection by engine (MySQL/PostgreSQL)
   - Category organization with accordions
   - Severity level selection (error/warning)
   - Basic parameter editing

### Critical Gaps ⚠️

#### 1. Incomplete Rule Exposure

**Current**: Only ~20 placeholder rules defined in `rules_metadata.go`

**Available in sql-reviewer**: 99+ production-ready rules

**Missing Categories**:
- Schema backward compatibility analysis
- Performance rules (EXPLAIN-based analysis)
- Transaction safety rules
- Advanced naming conventions (FK, UK, IDX patterns)
- PostgreSQL-specific safety rules
- Comment requirements and enforcement
- Online DDL requirement rules
- Charset/collation enforcement

#### 2. Incorrect Rule Type Mapping

Current Deeb rule types don't match sql-reviewer's schema:

| Deeb (Incorrect) | sql-reviewer (Correct) |
|------------------|------------------------|
| `mysql.engine.no-select-all` | `statement.select.no-select-all` |
| `mysql.naming.table` | `naming.table` |
| `postgresql.table.require-pk` | `table.require-pk` |

**Issue**: Rules are NOT engine-specific; they apply to both engines with the same type name.

#### 3. No Payload Parameter Configuration

**Missing**:
- Parameter schemas for rules
- Validation of payload values
- UI for configuring parameters
- Default values from sql-reviewer

**Example**: `naming.table` rule should accept:
```json
{
  "format": "^[a-z][a-z0-9_]*$",
  "maxLength": 64
}
```

But Deeb doesn't expose these parameters or validate them.

#### 4. Low-Level API Usage

**Current**: Using `advisor.Check()` directly (low-level)

**Better**: Use `reviewer.Review()` API (high-level) which provides:
- Automatic rule registration
- Result filtering and aggregation
- Better error handling
- Context cancellation support

#### 5. No Schema-Aware Validation

**Missing**: Database schema context for advanced rules

**Impact**: Cannot detect:
- References to non-existent columns
- Type mismatches in operations
- Index coverage for queries
- Constraint violations

#### 6. Limited Error Handling

**Current**: Failed rules are logged and silently skipped

```go
adviceList, err := advisor.Check(ctx, engine, advType, checkCtx)
if err != nil {
    a.logger.Warn("Failed to run sql-reviewer check for rule",
        zap.String("rule", rule.Type),
        zap.Error(err))
    continue // User never knows this rule failed!
}
```

**Issue**: Users don't know which rules failed or why.

---

## Complete Rule Catalog

### Overview

The sql-reviewer package provides **99+ unique rule types** across 9 categories:

| Category | MySQL Rules | PostgreSQL Rules | Description |
|----------|-------------|------------------|-------------|
| Engine | 1 | 0 | Database engine requirements |
| Naming | 11 | 11 | Identifier naming conventions |
| Table | 15 | 15 | Table-level rules |
| Column | 25 | 25 | Column-level rules |
| Index | 9 | 9 | Index rules and constraints |
| Statement | 45 | 45 | SQL statement rules |
| Schema | 2 | 2 | Schema-level rules |
| System | 9 | 9 | System-level settings |
| Advice | 1 | 1 | Recommendations |
| **Total** | **90** | **56** | **99 unique types** |

### Rule Categories Deep Dive

#### 1. Engine Rules (1 rule)

| Rule Type | Engines | Description | Payload |
|-----------|---------|-------------|---------|
| `engine.mysql.use-innodb` | MySQL | Require InnoDB storage engine | None |

#### 2. Naming Rules (11 rules)

Rules for enforcing naming conventions on database identifiers.

| Rule Type | Engines | Description | Payload Parameters |
|-----------|---------|-------------|--------------------|
| `naming.table` | Both | Table naming format | `format` (regex), `maxLength` (int) |
| `naming.column` | Both | Column naming format | `format` (regex), `maxLength` (int) |
| `naming.index.pk` | Both | Primary key naming format | `format` (regex), `maxLength` (int) |
| `naming.index.uk` | Both | Unique key naming format | `format` (regex), `maxLength` (int) |
| `naming.index.fk` | Both | Foreign key naming format | `format` (regex), `maxLength` (int) |
| `naming.index.idx` | Both | Index naming format | `format` (regex), `maxLength` (int) |
| `naming.column.auto-increment` | MySQL | Auto-increment column naming | `format` (regex), `maxLength` (int) |
| `naming.table.no-keyword` | Both | Disallow SQL keywords in table names | None |
| `naming.identifier.no-keyword` | Both | Disallow SQL keywords in identifiers | None |
| `naming.fully-qualified` | PostgreSQL | Require fully qualified names | None |
| `naming.identifier.case` | PostgreSQL | Enforce identifier case | `upper` (bool) |

**Example Payload**:
```json
{
  "format": "^[a-z][a-z0-9_]*$",
  "maxLength": 63
}
```

#### 3. Table Rules (15 rules)

Rules governing table-level operations and constraints.

| Rule Type | Engines | Description | Payload Parameters |
|-----------|---------|-------------|--------------------|
| `table.require-pk` | Both | Require primary key on all tables | None |
| `table.no-foreign-key` | Both | Disallow foreign key constraints | None |
| `table.drop-naming-convention` | Both | Enforce DROP table naming pattern | `format` (regex) |
| `table.comment` | Both | Comment requirements | `required` (bool), `requiredClassification` (bool), `maxLength` (int) |
| `table.disallow-partition` | Both | Disallow table partitions | None |
| `table.disallow-trigger` | Both | Disallow triggers | None |
| `table.no-duplicate-index` | Both | Prevent duplicate indexes | None |
| `table.text-fields-total-length` | MySQL | Limit total TEXT field length | `number` (int) |
| `table.disallow-set-charset` | Both | Disallow charset changes | None |
| `table.disallow-ddl` | Both | Restrict DDL on specific tables | `list` (string array) |
| `table.disallow-dml` | Both | Restrict DML on specific tables | `list` (string array) |
| `table.limit-size` | Both | Restrict operations on large tables | `number` (int, rows) |
| `table.require-charset` | Both | Enforce specific charset | `charset` (string) |
| `table.require-collation` | Both | Enforce specific collation | `collation` (string) |

**Example Payloads**:

Require table comments with classification:
```json
{
  "required": true,
  "requiredClassification": true,
  "maxLength": 256
}
```

Disallow DML on critical tables:
```json
{
  "list": ["users", "transactions", "audit_log"]
}
```

#### 4. Column Rules (25 rules)

Most comprehensive category with rules for column definitions and constraints.

| Rule Type | Engines | Description | Payload Parameters |
|-----------|---------|-------------|--------------------|
| `column.required` | Both | Require specific columns | `list` (string array) |
| `column.no-null` | Both | Disallow NULL values | None |
| `column.disallow-change-type` | Both | Prevent column type changes | None |
| `column.set-default-for-not-null` | Both | Require default for NOT NULL | None |
| `column.disallow-change` | Both | Disallow CHANGE COLUMN | None |
| `column.disallow-changing-order` | Both | Prevent column reordering | None |
| `column.disallow-drop` | Both | Prevent column drops | None |
| `column.disallow-drop-in-index` | Both | Prevent dropping indexed columns | None |
| `column.comment` | Both | Column comment requirements | `required` (bool), `maxLength` (int) |
| `column.auto-increment-must-integer` | MySQL | Auto-increment must be integer | None |
| `column.type-disallow-list` | Both | Disallowed column types | `list` (string array) |
| `column.disallow-set-charset` | Both | Disallow column charset changes | None |
| `column.maximum-character-length` | Both | Max character length limit | `number` (int) |
| `column.maximum-varchar-length` | Both | Max VARCHAR length limit | `number` (int) |
| `column.auto-increment-initial-value` | MySQL | Initial auto-increment value | `number` (int) |
| `column.auto-increment-must-unsigned` | MySQL | Auto-increment must be UNSIGNED | None |
| `column.current-time-count-limit` | Both | Limit CURRENT_TIMESTAMP columns | `number` (int) |
| `column.require-default` | Both | Require default values | None |
| `column.default-disallow-volatile` | Both | Disallow volatile defaults | None |
| `column.add-not-null-require-default` | Both | Require default when adding NOT NULL | None |
| `column.require-charset` | Both | Enforce column charset | `charset` (string) |
| `column.require-collation` | Both | Enforce column collation | `collation` (string) |

**Example Payloads**:

Require audit columns:
```json
{
  "list": ["created_at", "updated_at", "created_by"]
}
```

Disallow problematic column types:
```json
{
  "list": ["TEXT", "BLOB", "ENUM", "SET"]
}
```

#### 5. Index Rules (9 rules)

Rules for index definitions and constraints.

| Rule Type | Engines | Description | Payload Parameters |
|-----------|---------|-------------|--------------------|
| `index.no-duplicate-column` | Both | No duplicate columns in indexes | None |
| `index.key-number-limit` | Both | Limit index key count | `number` (int) |
| `index.pk-type-limit` | Both | Restrict primary key types | None |
| `index.type-no-blob` | Both | Disallow BLOB in indexes | None |
| `index.total-number-limit` | Both | Limit total index count | `number` (int) |
| `index.primary-key-type-allowlist` | Both | Allowed PK types | `list` (string array) |
| `index.create-concurrently` | PostgreSQL | Require CONCURRENTLY | None |
| `index.type-allow-list` | Both | Allowed index types | `list` (string array) |
| `index.not-redundant` | Both | Prevent redundant indexes | None |

**Example Payloads**:

Allow only specific PK types:
```json
{
  "list": ["INT", "BIGINT", "UUID"]
}
```

Limit index count per table:
```json
{
  "number": 5
}
```

#### 6. Statement Rules (45 rules)

Largest category covering SQL statement validation, performance, and safety.

##### SELECT Statement Rules

| Rule Type | Engines | Description | Payload |
|-----------|---------|-------------|---------|
| `statement.select.no-select-all` | Both | Disallow SELECT * | None |
| `statement.where.require.select` | Both | Require WHERE for SELECT | None |
| `statement.select-full-table-scan` | Both | Detect full table scans | None |
| `statement.query.minimum-plan-level` | Both | Require minimum execution plan level | `level` (string) |

##### WHERE Clause Rules

| Rule Type | Engines | Description | Payload Parameters |
|-----------|---------|-------------|--------------------|
| `statement.where.require.update-delete` | Both | Require WHERE for UPDATE/DELETE | None |
| `statement.where.no-leading-wildcard-like` | Both | Disallow `LIKE '%...'` | None |
| `statement.where.no-equal-null` | Both | Disallow `= NULL` (use IS NULL) | None |
| `statement.where.disallow-functions-and-calculations` | Both | Disallow functions in WHERE | None |
| `statement.where.maximum-logical-operator-count` | Both | Limit AND/OR count | `number` (int) |

##### INSERT Statement Rules

| Rule Type | Engines | Description | Payload Parameters |
|-----------|---------|-------------|--------------------|
| `statement.insert.row-limit` | Both | Limit INSERT rows | `number` (int) |
| `statement.insert.must-specify-column` | Both | Require column specification | None |
| `statement.insert.disallow-order-by-rand` | Both | Disallow ORDER BY RAND() | None |

##### DML Safety Rules

| Rule Type | Engines | Description | Payload Parameters |
|-----------|---------|-------------|--------------------|
| `statement.affected-row-limit` | Both | Limit affected rows | `number` (int) |
| `statement.dml-dry-run` | Both | Execute dry-run validation | None |
| `statement.disallow-limit` | Both | Disallow LIMIT in UPDATE/DELETE | None |
| `statement.disallow-order-by` | Both | Disallow ORDER BY in UPDATE/DELETE | None |

##### DDL Rules

| Rule Type | Engines | Description | Payload |
|-----------|---------|-------------|---------|
| `statement.merge-alter-table` | Both | Suggest merging ALTERs | None |
| `statement.disallow-add-column-with-default` | PostgreSQL | Prevent ADD COLUMN WITH DEFAULT | None |
| `statement.add-check-not-valid` | PostgreSQL | Require NOT VALID for CHECK | None |
| `statement.add-foreign-key-not-valid` | PostgreSQL | Require NOT VALID for FK | None |
| `statement.disallow-add-not-null` | PostgreSQL | Prevent ADD NOT NULL | None |

##### Cascade Rules

| Rule Type | Engines | Description | Payload |
|-----------|---------|-------------|---------|
| `statement.disallow-on-del-cascade` | Both | Disallow ON DELETE CASCADE | None |
| `statement.disallow-rm-tbl-cascade` | Both | Disallow DROP TABLE CASCADE | None |

##### Transaction Rules

| Rule Type | Engines | Description | Payload Parameters |
|-----------|---------|-------------|--------------------|
| `statement.disallow-commit` | Both | Disallow explicit COMMIT | None |
| `statement.maximum-statements-in-transaction` | Both | Limit statements per transaction | `number` (int) |
| `statement.non-transactional` | Both | Detect non-transactional operations | None |
| `statement.prior-backup-check` | Both | Verify prior backup | Callback required |

##### Performance Rules

| Rule Type | Engines | Description | Payload Parameters |
|-----------|---------|-------------|--------------------|
| `statement.disallow-using-filesort` | MySQL | Detect filesort in EXPLAIN | None |
| `statement.disallow-using-temporary` | MySQL | Detect temporary tables in EXPLAIN | None |
| `statement.maximum-limit-value` | Both | Limit LIMIT value | `number` (int) |
| `statement.maximum-join-table-count` | Both | Limit JOIN tables | `number` (int) |
| `statement.join-strict-column-attrs` | Both | Enforce JOIN column attributes match | None |
| `statement.max-execution-time` | Both | Enforce max execution time | `number` (seconds) |

##### DDL-Specific Rules

| Rule Type | Engines | Description | Payload |
|-----------|---------|-------------|---------|
| `statement.require-algorithm-option` | Both | Require ALGORITHM in ALTER TABLE | None |
| `statement.require-lock-option` | Both | Require LOCK in ALTER TABLE | None |
| `statement.add-column-without-position` | Both | Disallow column positioning | None |
| `statement.disallow-offline-ddl` | Both | Require online DDL | None |
| `statement.disallow-cross-db-queries` | Both | Prevent cross-database queries | None |

##### Mixing Rules

| Rule Type | Engines | Description | Payload |
|-----------|---------|-------------|---------|
| `statement.disallow-mix-in-ddl` | Both | Disallow DML in DDL statements | None |
| `statement.disallow-mix-in-dml` | Both | Disallow DDL in DML statements | None |

##### PostgreSQL-Specific Rules

| Rule Type | Engines | Description | Payload |
|-----------|---------|-------------|---------|
| `statement.create-specify-schema` | PostgreSQL | Require schema specification | None |
| `statement.check-set-role-variable` | PostgreSQL | Check SET ROLE variable | None |
| `statement.object-owner-check` | PostgreSQL | Verify object ownership | None |

**Example Payloads**:

Limit affected rows in DML:
```json
{
  "number": 1000
}
```

Require WHERE clause with complexity limit:
```json
{
  "number": 5
}
```

#### 7. Schema Rules (2 rules)

High-level schema validation rules.

| Rule Type | Engines | Description | Payload |
|-----------|---------|-------------|---------|
| `schema.backward-compatibility` | Both | Ensure schema changes are backward compatible | None |
| `database.drop-empty-database` | Both | Only allow dropping empty databases | None |

**Critical**: `schema.backward-compatibility` is a game-changer for production safety!

#### 8. System Rules (9 rules)

System-level configuration and constraint rules.

| Rule Type | Engines | Description | Payload Parameters |
|-----------|---------|-------------|--------------------|
| `system.charset.allowlist` | Both | Allowed character sets | `list` (string array) |
| `system.collation.allowlist` | Both | Allowed collations | `list` (string array) |
| `system.comment.length` | Both | Comment length limit | `number` (int) |
| `system.procedure.disallow-create` | Both | Disallow stored procedures | None |
| `system.event.disallow-create` | Both | Disallow events | None |
| `system.view.disallow-create` | Both | Disallow views | None |
| `system.function.disallow-create` | Both | Disallow functions | None |
| `system.function.disallowed-list` | Both | Disallowed functions | `list` (string array) |

**Example Payloads**:

Enforce UTF-8 only:
```json
{
  "list": ["utf8mb4", "utf8"]
}
```

Disallow dangerous functions:
```json
{
  "list": ["LOAD_FILE", "INTO OUTFILE", "DUMPFILE"]
}
```

#### 9. Advice Rules (1 rule)

Recommendation rules that provide guidance without blocking.

| Rule Type | Engines | Description | Payload Parameters |
|-----------|---------|-------------|--------------------|
| `advice.online-migration` | Both | Suggest online migration for large tables | `number` (row threshold) |

---

## Implementation Phases

### Phase 1: Rule Metadata Generation 🔧

**Priority**: CRITICAL
**Duration**: 1 week
**Dependencies**: None

#### Goals
- Auto-generate rule metadata from sql-reviewer's `config/schema.yaml`
- Ensure 100% accuracy with sql-reviewer package
- Support automatic updates when sql-reviewer evolves

#### Tasks

**1.1 Create Rule Generator Tool**

Create `backend/internal/review/generator/` with:

```
generator/
├── main.go              # CLI tool entry point
├── schema_parser.go     # Parse config/schema.yaml
├── metadata_generator.go # Generate rules_metadata.go
└── types_generator.go   # Generate TypeScript types
```

**Implementation**:
- Parse `~/w/sql-reviewer/config/schema.yaml`
- Extract rule definitions with categories, descriptions, payloads
- Generate Go code for `rules_metadata.go`
- Generate TypeScript types for frontend
- Add validation against sql-reviewer package

**1.2 Update rules_metadata.go**

Replace hardcoded rules with auto-generated content:

```go
// Code generated by generator/main.go. DO NOT EDIT.
// Source: github.com/nsxbet/sql-reviewer config/schema.yaml

package review

// RuleMetadata represents metadata for a SQL review rule
type RuleMetadata struct {
    Type        string                 `json:"type"`
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Category    string                 `json:"category"`
    Engine      Engine                 `json:"engine"`
    Parameters  []RuleParameter        `json:"parameters"`
}

// RuleParameter represents a configurable rule parameter
type RuleParameter struct {
    Name        string   `json:"name"`
    Type        string   `json:"type"` // STRING, NUMBER, BOOLEAN, STRING_ARRAY, TEMPLATE
    Description string   `json:"description"`
    Required    bool     `json:"required"`
    Default     any      `json:"default,omitempty"`
    Options     []string `json:"options,omitempty"` // For enum types
}

// GetAvailableRules returns all available SQL review rules
// organized by engine and category
func GetAvailableRules() map[string]map[string][]RuleMetadata {
    return rulesCatalog
}

// ValidateRuleType checks if a rule type exists in sql-reviewer
func ValidateRuleType(ruleType string, engine Engine) error {
    // Implementation
}

// ValidatePayload validates rule payload against parameter schema
func ValidatePayload(ruleType string, payload map[string]interface{}) error {
    // Implementation
}
```

**1.3 Generate TypeScript Types**

Create `frontend/src/lib/api/generated-types.ts`:

```typescript
// Code generated by generator/main.go. DO NOT EDIT.

export type RuleParameterType =
  | 'STRING'
  | 'NUMBER'
  | 'BOOLEAN'
  | 'STRING_ARRAY'
  | 'TEMPLATE';

export interface RuleParameter {
  name: string;
  type: RuleParameterType;
  description: string;
  required: boolean;
  default?: any;
  options?: string[];
}

export interface RuleMetadata {
  type: string;
  name: string;
  description: string;
  category: string;
  engine: 'mysql' | 'postgresql' | 'both';
  parameters: RuleParameter[];
}

export interface RulesCatalog {
  [engine: string]: {
    [category: string]: RuleMetadata[];
  };
}
```

**1.4 CI/CD Integration**

Add to CI pipeline:
```bash
# Check if generated code is up-to-date
make generate-rules
git diff --exit-code backend/internal/review/rules_metadata.go
```

#### Success Criteria
- ✅ All 99+ rules correctly defined
- ✅ Rule types match sql-reviewer exactly
- ✅ Parameter schemas complete and accurate
- ✅ TypeScript types generated
- ✅ Validation functions working
- ✅ CI/CD check in place

---

### Phase 2: API Architecture Upgrade 🏗️

**Priority**: HIGH
**Duration**: 1-2 weeks
**Dependencies**: Phase 1

#### Goals
- Switch to high-level `reviewer` API
- Add schema-aware validation
- Implement result filtering and aggregation
- Better error handling and reporting

#### Tasks

**2.1 Refactor Review Service**

Update `backend/internal/review/service.go`:

```go
package review

import (
    "context"
    "time"

    "github.com/nsxbet/sql-reviewer/pkg/reviewer"
    "github.com/nsxbet/sql-reviewer/pkg/types"
)

type Service struct {
    policyRepo    PolicyRepository
    schemaCache   SchemaCache // NEW: For schema-aware validation
    logger        *zap.Logger
}

// ReviewOptions for SQL review
type ReviewOptions struct {
    ChangePlanID    uuid.UUID
    EnvironmentID   uuid.UUID
    DatabaseType    DatabaseType
    SQLContent      string
    Schema          *types.DatabaseSchemaMetadata // NEW
    Timeout         time.Duration // NEW
}

// ReviewResult with enhanced metadata
type ReviewResult struct {
    Violations      []Violation
    Warnings        []Violation
    EnabledRules    []string
    SkippedRules    []SkippedRule // NEW
    ExecutionTime   time.Duration // NEW
    ReviewedAt      time.Time
}

// SkippedRule represents a rule that failed to execute
type SkippedRule struct {
    RuleType string
    Reason   string
    Error    string
}

// Review performs SQL review using sql-reviewer
func (s *Service) Review(ctx context.Context, opts ReviewOptions) (*ReviewResult, error) {
    // Set timeout
    timeout := opts.Timeout
    if timeout == 0 {
        timeout = 30 * time.Second
    }
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    // Load policy for environment
    policy, err := s.policyRepo.GetPolicyForEnvironment(ctx, opts.EnvironmentID)
    if err != nil {
        return nil, fmt.Errorf("failed to load policy: %w", err)
    }

    // Convert engine
    engine := convertDatabaseTypeToEngine(opts.DatabaseType)

    // Create reviewer
    r := reviewer.New(engine)

    // Configure with policy rules
    config := convertPolicyToConfig(policy)
    if err := r.WithConfigObject(config); err != nil {
        return nil, fmt.Errorf("failed to configure reviewer: %w", err)
    }

    // Execute review
    start := time.Now()
    var result *reviewer.Result
    if opts.Schema != nil {
        result, err = r.ReviewWithSchema(ctx, opts.SQLContent, opts.Schema)
    } else {
        result, err = r.Review(ctx, opts.SQLContent)
    }
    executionTime := time.Since(start)

    if err != nil {
        return nil, fmt.Errorf("review failed: %w", err)
    }

    // Convert result
    return &ReviewResult{
        Violations:    convertAdvicesToViolations(result.FilterByStatus(types.Advice_ERROR)),
        Warnings:      convertAdvicesToViolations(result.FilterByStatus(types.Advice_WARN)),
        EnabledRules:  getEnabledRules(policy),
        SkippedRules:  getSkippedRules(result),
        ExecutionTime: executionTime,
        ReviewedAt:    time.Now(),
    }, nil
}
```

**2.2 Add Schema Cache**

Create `backend/internal/review/schema_cache.go`:

```go
package review

import (
    "context"
    "sync"
    "time"

    "github.com/nsxbet/sql-reviewer/pkg/types"
)

// SchemaCache caches database schema metadata
type SchemaCache struct {
    cache   map[uuid.UUID]*cacheEntry
    mu      sync.RWMutex
    ttl     time.Duration
}

type cacheEntry struct {
    schema    *types.DatabaseSchemaMetadata
    timestamp time.Time
}

func NewSchemaCache(ttl time.Duration) *SchemaCache {
    return &SchemaCache{
        cache: make(map[uuid.UUID]*cacheEntry),
        ttl:   ttl,
    }
}

// Get retrieves schema from cache or database
func (c *SchemaCache) Get(ctx context.Context, databaseID uuid.UUID) (*types.DatabaseSchemaMetadata, error) {
    // Check cache first
    c.mu.RLock()
    entry, exists := c.cache[databaseID]
    c.mu.RUnlock()

    if exists && time.Since(entry.timestamp) < c.ttl {
        return entry.schema, nil
    }

    // Fetch from database
    schema, err := c.fetchSchema(ctx, databaseID)
    if err != nil {
        return nil, err
    }

    // Update cache
    c.mu.Lock()
    c.cache[databaseID] = &cacheEntry{
        schema:    schema,
        timestamp: time.Now(),
    }
    c.mu.Unlock()

    return schema, nil
}

// Invalidate removes schema from cache
func (c *SchemaCache) Invalidate(databaseID uuid.UUID) {
    c.mu.Lock()
    delete(c.cache, databaseID)
    c.mu.Unlock()
}
```

**2.3 Update Change Plan Handler**

Modify `backend/internal/handlers/change_plan_handler.go`:

```go
// In CreateChangePlan handler
reviewResult, err := h.reviewService.Review(ctx, review.ReviewOptions{
    ChangePlanID:  changePlan.ID,
    EnvironmentID: changePlan.EnvironmentID,
    DatabaseType:  changePlan.DatabaseType,
    SQLContent:    changePlan.Statement,
    // Enable schema-aware validation if available
    Schema:        h.getSchemaForEnvironment(ctx, changePlan.EnvironmentID),
    Timeout:       30 * time.Second,
})
```

**2.4 Add Result Filtering API**

Create `backend/internal/handlers/review_handler.go`:

```go
// GET /api/v1/change-plans/:id/review
func (h *Handler) GetReviewResults(c echo.Context) error {
    changePlanID := c.Param("id")

    // Query parameters for filtering
    severityFilter := c.QueryParam("severity") // error, warning
    ruleTypeFilter := c.QueryParam("rule_type")
    categoryFilter := c.QueryParam("category")

    results, err := h.reviewService.GetReviewResults(ctx, changePlanID)
    if err != nil {
        return err
    }

    // Apply filters
    filtered := results
    if severityFilter != "" {
        filtered = filtered.FilterBySeverity(severityFilter)
    }
    if ruleTypeFilter != "" {
        filtered = filtered.FilterByRuleType(ruleTypeFilter)
    }
    if categoryFilter != "" {
        filtered = filtered.FilterByCategory(categoryFilter)
    }

    return c.JSON(http.StatusOK, filtered)
}
```

#### Success Criteria
- ✅ Using high-level `reviewer` API
- ✅ Schema caching implemented
- ✅ Context cancellation working
- ✅ Result filtering available
- ✅ Skipped rules reported to users
- ✅ Performance metrics tracked

---

### Phase 3: Policy Migration 📋

**Priority**: HIGH
**Duration**: 1 week
**Dependencies**: Phases 1-2

#### Goals
- Update policy structure to use correct rule types
- Add payload parameter validation
- Create migration script for existing policies
- Maintain backward compatibility during transition

#### Tasks

**3.1 Update Policy Models**

Modify `backend/internal/models/policy.go`:

```go
// PolicyConfigVersion for versioning policy configs
type PolicyConfigVersion string

const (
    PolicyConfigV1 PolicyConfigVersion = "v1" // Legacy (deprecated)
    PolicyConfigV2 PolicyConfigVersion = "v2" // New format
)

// SQLReviewPolicyConfig with versioning
type SQLReviewPolicyConfig struct {
    Version PolicyConfigVersion        `json:"version"`
    Rules   map[string]SQLReviewRule   `json:"rules"`
}

// ParseSQLReviewConfig with version handling
func ParseSQLReviewConfig(data map[string]interface{}) (*SQLReviewPolicyConfig, error) {
    version, _ := data["version"].(string)

    if version == "" || version == string(PolicyConfigV1) {
        // Migrate from v1 to v2
        return migrateV1ToV2(data)
    }

    // Parse v2 format
    return parseV2Config(data)
}
```

**3.2 Create Migration Script**

Create `backend/cmd/migrate-policies/main.go`:

```go
package main

import (
    "context"
    "log"

    "github.com/nsxbet/deeb/internal/models"
    "github.com/nsxbet/deeb/internal/review"
)

func main() {
    ctx := context.Background()

    // Load all policies
    policies, err := policyRepo.ListAll(ctx)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Found %d policies to migrate", len(policies))

    for _, policy := range policies {
        if policy.Type != models.PolicyTypeSQLReview {
            continue
        }

        // Check version
        config, err := models.ParseSQLReviewConfig(policy.PolicyConfig)
        if err != nil {
            log.Printf("ERROR: Failed to parse policy %s: %v", policy.ID, err)
            continue
        }

        if config.Version == models.PolicyConfigV2 {
            log.Printf("SKIP: Policy %s already v2", policy.ID)
            continue
        }

        // Migrate
        migratedConfig, err := migratePolicy(config)
        if err != nil {
            log.Printf("ERROR: Failed to migrate policy %s: %v", policy.ID, err)
            continue
        }

        // Validate
        if err := review.ValidatePolicyConfig(migratedConfig); err != nil {
            log.Printf("ERROR: Migrated policy %s failed validation: %v", policy.ID, err)
            continue
        }

        // Update
        policy.PolicyConfig = migratedConfig.ToPolicyConfig()
        if err := policyRepo.Update(ctx, policy); err != nil {
            log.Printf("ERROR: Failed to update policy %s: %v", policy.ID, err)
            continue
        }

        log.Printf("SUCCESS: Migrated policy %s", policy.ID)
    }
}

// migratePolicy converts v1 rule types to v2
func migratePolicy(config *models.SQLReviewPolicyConfig) (*models.SQLReviewPolicyConfig, error) {
    migratedRules := make(map[string]models.SQLReviewRule)

    for oldType, rule := range config.Rules {
        // Map old type to new type
        newType, err := mapRuleType(oldType)
        if err != nil {
            return nil, fmt.Errorf("unknown rule type: %s", oldType)
        }

        // Remove engine prefix
        rule.Engine = models.EngineAll // Rules apply to both engines
        migratedRules[newType] = rule
    }

    return &models.SQLReviewPolicyConfig{
        Version: models.PolicyConfigV2,
        Rules:   migratedRules,
    }, nil
}

// mapRuleType converts old rule types to new format
func mapRuleType(oldType string) (string, error) {
    mapping := map[string]string{
        "mysql.engine.no-select-all":       "statement.select.no-select-all",
        "mysql.naming.table":               "naming.table",
        "mysql.table.require-pk":           "table.require-pk",
        "postgresql.engine.no-select-all":  "statement.select.no-select-all",
        "postgresql.naming.table":          "naming.table",
        // ... complete mapping
    }

    newType, exists := mapping[oldType]
    if !exists {
        return "", fmt.Errorf("no mapping for: %s", oldType)
    }
    return newType, nil
}
```

**3.3 Add Policy Validation**

Create `backend/internal/review/validation.go`:

```go
package review

// ValidatePolicyConfig validates policy configuration
func ValidatePolicyConfig(config *models.SQLReviewPolicyConfig) error {
    if config.Version != models.PolicyConfigV2 {
        return fmt.Errorf("unsupported policy version: %s", config.Version)
    }

    for ruleType, rule := range config.Rules {
        // Validate rule type exists
        if err := ValidateRuleType(ruleType, rule.Engine); err != nil {
            return fmt.Errorf("invalid rule type %s: %w", ruleType, err)
        }

        // Validate payload
        if err := ValidatePayload(ruleType, rule.Payload); err != nil {
            return fmt.Errorf("invalid payload for rule %s: %w", ruleType, err)
        }

        // Validate level
        if rule.Level != models.ERROR && rule.Level != models.WARNING {
            return fmt.Errorf("invalid level for rule %s: %s", ruleType, rule.Level)
        }
    }

    return nil
}
```

**3.4 Update Policy Handler**

Modify `backend/internal/handlers/policy_handler.go`:

```go
// CreatePolicy with validation
func (h *Handler) CreatePolicy(c echo.Context) error {
    var req CreatePolicyRequest
    if err := c.Bind(&req); err != nil {
        return err
    }

    // For SQL_REVIEW policies, validate config
    if req.Type == models.PolicyTypeSQLReview {
        config, err := models.ParseSQLReviewConfig(req.PolicyConfig)
        if err != nil {
            return echo.NewHTTPError(http.StatusBadRequest, "Invalid policy config: "+err.Error())
        }

        // Validate
        if err := review.ValidatePolicyConfig(config); err != nil {
            return echo.NewHTTPError(http.StatusBadRequest, "Policy validation failed: "+err.Error())
        }
    }

    // Create policy...
}
```

#### Success Criteria
- ✅ Policy models support versioning
- ✅ Migration script tested on production data
- ✅ Validation catches invalid configurations
- ✅ API rejects invalid policies
- ✅ Backward compatibility maintained
- ✅ Migration runbook documented

---

### Phase 4: Frontend Enhancements 🎨

**Priority**: MEDIUM
**Duration**: 1-2 weeks
**Dependencies**: Phases 1-3

#### Goals
- Display all 99+ rules in UI
- Add parameter configuration with validation
- Implement rule search and filtering
- Add rule profiles and templates

#### Tasks

**4.1 Update Rule Selector Component**

Enhance `frontend/src/components/sql-review-rule-selector.tsx`:

```typescript
import { useState, useMemo } from 'react';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Accordion } from '@/components/ui/accordion';
import { RulesCatalog, RuleMetadata } from '@/lib/api/generated-types';

interface SQLReviewRuleSelectorProps {
  value: SQLReviewPolicyConfig;
  onChange: (config: SQLReviewPolicyConfig) => void;
}

export function SQLReviewRuleSelector({ value, onChange }: SQLReviewRuleSelectorProps) {
  const [rules, setRules] = useState<RulesCatalog>({});
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedEngine, setSelectedEngine] = useState<'mysql' | 'postgresql'>('mysql');

  // Load rules
  useEffect(() => {
    getSQLReviewRules().then(setRules);
  }, []);

  // Filter rules by search
  const filteredRules = useMemo(() => {
    if (!searchQuery) return rules;

    const filtered: RulesCatalog = {};
    Object.entries(rules).forEach(([engine, categories]) => {
      filtered[engine] = {};
      Object.entries(categories).forEach(([category, ruleList]) => {
        const matchingRules = ruleList.filter(rule =>
          rule.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
          rule.description.toLowerCase().includes(searchQuery.toLowerCase()) ||
          rule.type.toLowerCase().includes(searchQuery.toLowerCase())
        );
        if (matchingRules.length > 0) {
          filtered[engine][category] = matchingRules;
        }
      });
    });
    return filtered;
  }, [rules, searchQuery]);

  // Count enabled rules
  const enabledCount = Object.keys(value.rules || {}).length;
  const totalCount = Object.values(rules[selectedEngine] || {})
    .flat()
    .length;

  return (
    <div className="space-y-4">
      {/* Header with stats */}
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-semibold">SQL Review Rules</h3>
          <p className="text-sm text-muted-foreground">
            {enabledCount} of {totalCount} rules enabled
          </p>
        </div>

        {/* Quick actions */}
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={handleEnableAll}>
            Enable All
          </Button>
          <Button variant="outline" size="sm" onClick={handleDisableAll}>
            Disable All
          </Button>
          <Button variant="outline" size="sm" onClick={handleLoadTemplate}>
            Load Template
          </Button>
        </div>
      </div>

      {/* Search */}
      <Input
        placeholder="Search rules by name, description, or type..."
        value={searchQuery}
        onChange={(e) => setSearchQuery(e.target.value)}
      />

      {/* Engine tabs */}
      <Tabs value={selectedEngine} onValueChange={setSelectedEngine}>
        <TabsList>
          <TabsTrigger value="mysql">
            MySQL
            <Badge className="ml-2">{countRules(rules.mysql)}</Badge>
          </TabsTrigger>
          <TabsTrigger value="postgresql">
            PostgreSQL
            <Badge className="ml-2">{countRules(rules.postgresql)}</Badge>
          </TabsTrigger>
        </TabsList>

        <TabsContent value={selectedEngine}>
          {/* Category accordion */}
          <Accordion type="multiple" className="w-full">
            {Object.entries(filteredRules[selectedEngine] || {}).map(
              ([category, categoryRules]) => (
                <AccordionItem key={category} value={category}>
                  <AccordionTrigger>
                    <div className="flex items-center gap-2">
                      <span className="font-medium capitalize">{category}</span>
                      <Badge variant="secondary">
                        {countEnabledInCategory(categoryRules, value)} / {categoryRules.length}
                      </Badge>
                    </div>
                  </AccordionTrigger>
                  <AccordionContent>
                    <div className="space-y-4">
                      {categoryRules.map((rule) => (
                        <RuleConfigCard
                          key={rule.type}
                          rule={rule}
                          config={value.rules[rule.type]}
                          onChange={(config) => handleRuleChange(rule.type, config)}
                        />
                      ))}
                    </div>
                  </AccordionContent>
                </AccordionItem>
              )
            )}
          </Accordion>
        </TabsContent>
      </Tabs>
    </div>
  );
}
```

**4.2 Create Rule Config Card**

Create `frontend/src/components/rule-config-card.tsx`:

```typescript
import { Switch } from '@/components/ui/switch';
import { Select } from '@/components/ui/select';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { RuleMetadata, SQLReviewRule } from '@/lib/api/types';

interface RuleConfigCardProps {
  rule: RuleMetadata;
  config?: SQLReviewRule;
  onChange: (config: SQLReviewRule | undefined) => void;
}

export function RuleConfigCard({ rule, config, onChange }: RuleConfigCardProps) {
  const isEnabled = !!config;

  const handleToggle = (enabled: boolean) => {
    if (enabled) {
      onChange({
        type: rule.type,
        level: 'error',
        payload: {},
        engine: rule.engine,
      });
    } else {
      onChange(undefined);
    }
  };

  const handleLevelChange = (level: 'error' | 'warning') => {
    if (config) {
      onChange({ ...config, level });
    }
  };

  const handlePayloadChange = (paramName: string, value: any) => {
    if (config) {
      onChange({
        ...config,
        payload: { ...config.payload, [paramName]: value },
      });
    }
  };

  return (
    <div className="border rounded-lg p-4 space-y-3">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <Switch checked={isEnabled} onCheckedChange={handleToggle} />
            <Label className="font-medium">{rule.name}</Label>
          </div>
          <p className="text-sm text-muted-foreground mt-1">
            {rule.description}
          </p>
          <code className="text-xs text-muted-foreground">{rule.type}</code>
        </div>

        {/* Severity selector */}
        {isEnabled && (
          <Select value={config?.level} onValueChange={handleLevelChange}>
            <SelectTrigger className="w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="error">Error</SelectItem>
              <SelectItem value="warning">Warning</SelectItem>
            </SelectContent>
          </Select>
        )}
      </div>

      {/* Parameters */}
      {isEnabled && rule.parameters.length > 0 && (
        <div className="space-y-2 pt-2 border-t">
          <Label className="text-sm font-medium">Parameters</Label>
          {rule.parameters.map((param) => (
            <div key={param.name} className="grid grid-cols-3 gap-2 items-center">
              <Label className="text-sm">{param.name}</Label>
              <div className="col-span-2">
                {renderParameterInput(param, config?.payload?.[param.name], (value) =>
                  handlePayloadChange(param.name, value)
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function renderParameterInput(
  param: RuleParameter,
  value: any,
  onChange: (value: any) => void
) {
  switch (param.type) {
    case 'STRING':
    case 'TEMPLATE':
      return (
        <Input
          value={value || param.default || ''}
          onChange={(e) => onChange(e.target.value)}
          placeholder={param.description}
        />
      );

    case 'NUMBER':
      return (
        <Input
          type="number"
          value={value || param.default || 0}
          onChange={(e) => onChange(parseInt(e.target.value))}
        />
      );

    case 'BOOLEAN':
      return (
        <Switch
          checked={value ?? param.default ?? false}
          onCheckedChange={onChange}
        />
      );

    case 'STRING_ARRAY':
      return (
        <Input
          value={(value || []).join(', ')}
          onChange={(e) => onChange(e.target.value.split(',').map((s) => s.trim()))}
          placeholder="Comma-separated values"
        />
      );

    default:
      return <span className="text-sm text-muted-foreground">Unknown type</span>;
  }
}
```

**4.3 Add Rule Templates**

Create `frontend/src/lib/policy-templates.ts`:

```typescript
import { SQLReviewPolicyConfig } from '@/lib/api/types';

export const policyTemplates = {
  production_strict: {
    name: 'Production (Strict)',
    description: 'Strict rules for production environments',
    config: {
      version: 'v2',
      rules: {
        'table.require-pk': { level: 'error', payload: {} },
        'column.no-null': { level: 'error', payload: {} },
        'statement.where.require.update-delete': { level: 'error', payload: {} },
        'statement.affected-row-limit': { level: 'error', payload: { number: 1000 } },
        'schema.backward-compatibility': { level: 'error', payload: {} },
        'statement.select.no-select-all': { level: 'warning', payload: {} },
        // ... more rules
      },
    } as SQLReviewPolicyConfig,
  },

  production_moderate: {
    name: 'Production (Moderate)',
    description: 'Balanced rules for production',
    config: {
      version: 'v2',
      rules: {
        'table.require-pk': { level: 'error', payload: {} },
        'statement.where.require.update-delete': { level: 'error', payload: {} },
        'statement.select.no-select-all': { level: 'warning', payload: {} },
        // ... more rules
      },
    } as SQLReviewPolicyConfig,
  },

  development: {
    name: 'Development',
    description: 'Permissive rules for development',
    config: {
      version: 'v2',
      rules: {
        'naming.table': { level: 'warning', payload: { format: '^[a-z][a-z0-9_]*$' } },
        'naming.column': { level: 'warning', payload: { format: '^[a-z][a-z0-9_]*$' } },
        // ... more rules
      },
    } as SQLReviewPolicyConfig,
  },

  security_focused: {
    name: 'Security Focused',
    description: 'Emphasis on security rules',
    config: {
      version: 'v2',
      rules: {
        'statement.where.require.update-delete': { level: 'error', payload: {} },
        'statement.disallow-on-del-cascade': { level: 'error', payload: {} },
        'statement.affected-row-limit': { level: 'error', payload: { number: 100 } },
        'column.disallow-drop': { level: 'error', payload: {} },
        'table.disallow-dml': { level: 'error', payload: { list: ['users', 'audit_log'] } },
        // ... more rules
      },
    } as SQLReviewPolicyConfig,
  },
};
```

**4.4 Update Policies Page**

Modify `frontend/src/pages/PoliciesPage.tsx`:

```typescript
// Add template loading
const handleLoadTemplate = (templateKey: string) => {
  const template = policyTemplates[templateKey];
  if (template) {
    setConfig(template.config);
  }
};

// Add in the form
<div className="space-y-2">
  <Label>Load Template</Label>
  <Select onValueChange={handleLoadTemplate}>
    <SelectTrigger>
      <SelectValue placeholder="Choose a template..." />
    </SelectTrigger>
    <SelectContent>
      {Object.entries(policyTemplates).map(([key, template]) => (
        <SelectItem key={key} value={key}>
          <div>
            <div className="font-medium">{template.name}</div>
            <div className="text-sm text-muted-foreground">
              {template.description}
            </div>
          </div>
        </SelectItem>
      ))}
    </SelectContent>
  </Select>
</div>
```

#### Success Criteria
- ✅ All 99+ rules visible in UI
- ✅ Search and filtering working
- ✅ Parameter configuration UI complete
- ✅ Validation prevents invalid configs
- ✅ Templates available
- ✅ Bulk operations working

---

### Phase 5: Advanced Features 🚀

**Priority**: MEDIUM
**Duration**: 1-2 weeks
**Dependencies**: Phases 1-4

#### Goals
- Schema-aware validation with database connection
- Rule statistics and reporting
- Custom rule support
- Performance optimization

#### Tasks

**5.1 Implement Schema Fetching**

Create `backend/internal/review/schema_fetcher.go`:

```go
package review

import (
    "context"
    "database/sql"

    "github.com/nsxbet/sql-reviewer/pkg/types"
)

// SchemaFetcher fetches database schema metadata
type SchemaFetcher struct {
    db *sql.DB
}

// FetchSchema retrieves complete schema metadata
func (f *SchemaFetcher) FetchSchema(ctx context.Context, dbType DatabaseType) (*types.DatabaseSchemaMetadata, error) {
    switch dbType {
    case DatabaseTypeMySQL:
        return f.fetchMySQLSchema(ctx)
    case DatabaseTypePostgreSQL:
        return f.fetchPostgreSQLSchema(ctx)
    default:
        return nil, fmt.Errorf("unsupported database type: %s", dbType)
    }
}

// fetchMySQLSchema queries MySQL information_schema
func (f *SchemaFetcher) fetchMySQLSchema(ctx context.Context) (*types.DatabaseSchemaMetadata, error) {
    schema := &types.DatabaseSchemaMetadata{
        Name:    f.getDatabaseName(),
        Schemas: []*types.SchemaMetadata{},
    }

    // Fetch tables
    tables, err := f.fetchTables(ctx)
    if err != nil {
        return nil, err
    }

    // For each table, fetch columns and indexes
    for _, table := range tables {
        columns, err := f.fetchColumns(ctx, table.Name)
        if err != nil {
            return nil, err
        }
        table.Columns = columns

        indexes, err := f.fetchIndexes(ctx, table.Name)
        if err != nil {
            return nil, err
        }
        table.Indexes = indexes
    }

    schema.Schemas = append(schema.Schemas, &types.SchemaMetadata{
        Name:   f.getDatabaseName(),
        Tables: tables,
    })

    return schema, nil
}

// fetchTables queries for tables
func (f *SchemaFetcher) fetchTables(ctx context.Context) ([]*types.TableMetadata, error) {
    query := `
        SELECT table_name, engine, table_rows, table_comment
        FROM information_schema.tables
        WHERE table_schema = DATABASE()
        AND table_type = 'BASE TABLE'
    `

    rows, err := f.db.QueryContext(ctx, query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var tables []*types.TableMetadata
    for rows.Next() {
        var table types.TableMetadata
        var comment sql.NullString
        if err := rows.Scan(&table.Name, &table.Engine, &table.RowCount, &comment); err != nil {
            return nil, err
        }
        if comment.Valid {
            table.Comment = comment.String
        }
        tables = append(tables, &table)
    }

    return tables, nil
}

// Similar implementations for fetchColumns, fetchIndexes, fetchPostgreSQLSchema...
```

**5.2 Add Rule Statistics**

Create `backend/internal/handlers/statistics_handler.go`:

```go
package handlers

// GET /api/v1/policies/:id/statistics
func (h *Handler) GetPolicyStatistics(c echo.Context) error {
    policyID := c.Param("id")

    stats, err := h.reviewService.GetPolicyStatistics(ctx, policyID)
    if err != nil {
        return err
    }

    return c.JSON(http.StatusOK, stats)
}

// PolicyStatistics represents rule usage stats
type PolicyStatistics struct {
    PolicyID      string                `json:"policy_id"`
    TotalReviews  int                   `json:"total_reviews"`
    RuleStats     []RuleStatistic       `json:"rule_stats"`
    TopViolations []ViolationStatistic  `json:"top_violations"`
}

type RuleStatistic struct {
    RuleType        string  `json:"rule_type"`
    ExecutionCount  int     `json:"execution_count"`
    ViolationCount  int     `json:"violation_count"`
    ViolationRate   float64 `json:"violation_rate"`
}
```

**5.3 Add Custom Rule Support**

Create `backend/internal/review/custom_rules.go`:

```go
package review

// CustomRule allows organizations to define custom rules
type CustomRule struct {
    ID          uuid.UUID
    Name        string
    Description string
    Category    string
    Handler     CustomRuleHandler
}

// CustomRuleHandler is the function that implements the rule logic
type CustomRuleHandler func(ctx context.Context, sql string, schema *types.DatabaseSchemaMetadata) ([]types.Advice, error)

// RegisterCustomRule adds a custom rule to the registry
func (s *Service) RegisterCustomRule(rule CustomRule) error {
    // Validate and register
    return s.customRuleRegistry.Register(rule)
}
```

**5.4 Performance Optimization**

Add caching and optimization:

```go
// Cache review results for unchanged SQL
type ReviewCache struct {
    cache *lru.Cache
    ttl   time.Duration
}

func (c *ReviewCache) Get(key string) (*ReviewResult, bool) {
    // Implementation
}

func (c *ReviewCache) Set(key string, result *ReviewResult) {
    // Implementation
}

// Key generation from SQL + policy
func generateCacheKey(sql string, policyID uuid.UUID) string {
    h := sha256.New()
    h.Write([]byte(sql))
    h.Write([]byte(policyID.String()))
    return hex.EncodeToString(h.Sum(nil))
}
```

#### Success Criteria
- ✅ Schema fetching working for MySQL and PostgreSQL
- ✅ Schema caching reduces database load
- ✅ Rule statistics tracking violations
- ✅ Custom rule registration API available
- ✅ Performance improved with caching

---

### Phase 6: Testing & Documentation 📚

**Priority**: HIGH
**Duration**: Ongoing
**Dependencies**: All phases

#### Goals
- Comprehensive test coverage for all components
- Complete user-facing documentation
- Developer documentation for extending the system
- Performance benchmarks

#### Tasks

**6.1 Backend Tests**

Create comprehensive test suite:

```
backend/internal/review/
├── service_test.go           # Review service tests
├── rules_metadata_test.go    # Rule metadata tests
├── validation_test.go        # Policy validation tests
├── schema_cache_test.go      # Schema caching tests
└── sqlreviewer/
    └── advisor_test.go       # Advisor integration tests
```

**Example test**:
```go
func TestReviewService_Review(t *testing.T) {
    tests := []struct {
        name          string
        sql           string
        policy        *models.Policy
        wantViolations int
        wantWarnings  int
    }{
        {
            name: "SELECT * violation",
            sql:  "SELECT * FROM users",
            policy: &models.Policy{
                PolicyConfig: map[string]interface{}{
                    "version": "v2",
                    "rules": map[string]interface{}{
                        "statement.select.no-select-all": map[string]interface{}{
                            "level": "error",
                        },
                    },
                },
            },
            wantViolations: 1,
            wantWarnings:   0,
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

**6.2 Frontend Tests**

Add tests for components:

```typescript
// frontend/src/components/__tests__/sql-review-rule-selector.test.tsx
describe('SQLReviewRuleSelector', () => {
  it('renders all rules', async () => {
    render(<SQLReviewRuleSelector value={{}} onChange={() => {}} />);

    // Wait for rules to load
    await waitFor(() => {
      expect(screen.getByText('Engine')).toBeInTheDocument();
      expect(screen.getByText('Naming')).toBeInTheDocument();
    });
  });

  it('enables rule on toggle', async () => {
    const onChange = jest.fn();
    render(<SQLReviewRuleSelector value={{}} onChange={onChange} />);

    // Find and toggle a rule
    const toggle = await screen.findByLabelText('Require primary key');
    fireEvent.click(toggle);

    expect(onChange).toHaveBeenCalledWith({
      version: 'v2',
      rules: {
        'table.require-pk': { level: 'error', payload: {} },
      },
    });
  });

  // More tests...
});
```

**6.3 Integration Tests**

Create end-to-end integration tests:

```go
// backend/internal/integration/review_integration_test.go
func TestReviewIntegration(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    defer db.Close()

    // Create policy
    policy := createTestPolicy(t, db)

    // Create change plan
    changePlan := createTestChangePlan(t, db)

    // Trigger review
    reviewService := review.NewService(db)
    result, err := reviewService.Review(ctx, review.ReviewOptions{
        ChangePlanID:  changePlan.ID,
        EnvironmentID: changePlan.EnvironmentID,
        DatabaseType:  changePlan.DatabaseType,
        SQLContent:    changePlan.Statement,
    })

    // Verify results
    assert.NoError(t, err)
    assert.NotNil(t, result)
    // More assertions...
}
```

**6.4 User Documentation**

Create documentation at `docs/sql-review-rules.md`:

```markdown
# SQL Review Rules Reference

## Overview

Deeb supports 99+ SQL review rules powered by the sql-reviewer package. This guide provides a complete reference of all available rules.

## Rule Categories

### Engine Rules

Rules for database engine requirements.

#### engine.mysql.use-innodb

**Description**: Require InnoDB storage engine for all tables.

**Level**: Error or Warning

**Parameters**: None

**Example**:
```sql
-- ❌ Violation
CREATE TABLE users (id INT) ENGINE=MyISAM;

-- ✅ Pass
CREATE TABLE users (id INT) ENGINE=InnoDB;
```

### Naming Rules

Rules for enforcing naming conventions...

(Continue for all 99+ rules)
```

**6.5 Developer Documentation**

Create `docs/sql-review-development.md`:

```markdown
# SQL Review Development Guide

## Architecture

Overview of the review system architecture...

## Adding New Rules

To add a new rule:

1. Define rule in sql-reviewer package
2. Run `make generate-rules` to update metadata
3. Add tests
4. Update documentation

## Custom Rules

Organizations can define custom rules...

## Performance Optimization

Best practices for performance...
```

**6.6 Migration Guide**

Create `docs/sql-review-migration.md`:

```markdown
# SQL Review Migration Guide

## Migrating from v1 to v2

This guide helps you migrate existing policies to the new rule format.

### What Changed

- Rule types no longer include engine prefixes
- New payload parameter system
- Enhanced validation

### Migration Steps

1. Backup existing policies
2. Run migration script
3. Validate migrated policies
4. Test in non-production environment
5. Deploy to production

### Rule Type Mapping

| Old (v1) | New (v2) |
|----------|----------|
| `mysql.engine.no-select-all` | `statement.select.no-select-all` |
| ... | ... |
```

#### Success Criteria
- ✅ Test coverage >80%
- ✅ All components have unit tests
- ✅ Integration tests passing
- ✅ Complete rule reference documentation
- ✅ Developer guide available
- ✅ Migration guide tested
- ✅ Performance benchmarks documented

---

## Technical Specifications

### Backend Architecture

#### High-Level API Usage

**Before** (Low-level):
```go
import "github.com/nsxbet/sql-reviewer/pkg/advisor"

advices, err := advisor.Check(ctx, engine, advType, checkCtx)
```

**After** (High-level):
```go
import "github.com/nsxbet/sql-reviewer/pkg/reviewer"

r := reviewer.New(types.Engine_MYSQL)
result, err := r.Review(ctx, sql)
```

#### Type Conversions

```go
// Convert Deeb DatabaseType to sql-reviewer Engine
func convertDatabaseTypeToEngine(dbType models.DatabaseType) types.Engine {
    switch dbType {
    case models.DatabaseTypeMySQL:
        return types.Engine_MYSQL
    case models.DatabaseTypePostgreSQL:
        return types.Engine_POSTGRES
    default:
        return types.Engine_UNKNOWN
    }
}

// Convert sql-reviewer Advice to Deeb Violation
func convertAdviceToViolation(advice *types.Advice) Violation {
    return Violation{
        RuleType:    extractRuleType(advice),
        Severity:    convertStatus(advice.Status),
        Message:     advice.Content,
        Line:        advice.StartPosition.Line,
        Column:      advice.StartPosition.Column,
    }
}
```

### Database Schema

#### Policy Table (No changes needed)

```sql
CREATE TABLE policies (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL,
    policy_config JSONB NOT NULL,  -- Stores SQLReviewPolicyConfig
    created_by_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    FOREIGN KEY (created_by_id) REFERENCES users(id)
);
```

#### Policy Config Format

```json
{
  "version": "v2",
  "rules": {
    "table.require-pk": {
      "type": "table.require-pk",
      "level": "error",
      "payload": {},
      "engine": "all",
      "comment": "All tables must have a primary key"
    },
    "naming.table": {
      "type": "naming.table",
      "level": "warning",
      "payload": {
        "format": "^[a-z][a-z0-9_]*$",
        "maxLength": 63
      },
      "engine": "all",
      "comment": "Table names must be lowercase with underscores"
    }
  }
}
```

### API Endpoints

#### New Endpoints

```
GET    /api/v1/policies/sql-review/rules              # Get all available rules
GET    /api/v1/policies/:id/validate                   # Validate policy config
POST   /api/v1/policies/:id/test                       # Test policy against sample SQL
GET    /api/v1/policies/:id/statistics                 # Get policy statistics
GET    /api/v1/change-plans/:id/review                 # Get review results
GET    /api/v1/change-plans/:id/review?severity=error  # Filter by severity
```

#### Enhanced Endpoints

```
POST   /api/v1/policies                   # Now validates v2 config
PUT    /api/v1/policies/:id               # Now validates v2 config
GET    /api/v1/policies/:id               # Returns version info
```

### Frontend Components

#### New Components

```
src/components/
├── sql-review-rule-selector.tsx      # Main rule selector (enhanced)
├── rule-config-card.tsx               # Individual rule configuration
├── rule-parameter-input.tsx           # Parameter input component
├── policy-template-selector.tsx       # Template selection
└── rule-statistics-chart.tsx          # Rule statistics visualization
```

#### Updated Components

```
src/pages/
└── PoliciesPage.tsx                   # Enhanced with templates and validation
```

---

## Breaking Changes & Migration

### Breaking Changes

#### 1. Rule Type Format

**Before**: `mysql.naming.table`, `postgresql.naming.table`
**After**: `naming.table` (applies to both engines)

**Impact**: All existing policies need rule types updated

#### 2. Policy Config Version

**Before**: No version field
**After**: Required `version: "v2"` field

**Impact**: Policies without version will be treated as v1 and auto-migrated

#### 3. Engine Field

**Before**: Rules had engine-specific types
**After**: Rules have `engine: "all"` or specific engine

**Impact**: Engine field now stored in rule config

### Migration Strategy

#### Timeline

- **Week 1**: Deploy migration script to staging
- **Week 2**: Test migration on production snapshot
- **Week 3**: Deploy to production with monitoring
- **Week 4**: Clean up v1 support code

#### Rollback Plan

1. Keep v1 parsing code for 2 release cycles
2. Maintain ability to downgrade policy config
3. Database backups before migration
4. Feature flag to disable v2 validation

#### Communication Plan

1. Announce migration 2 weeks before deployment
2. Provide migration guide and examples
3. Offer office hours for questions
4. Monitor for issues post-deployment

---

## Performance Considerations

### Bottlenecks

1. **Rule Execution**: Some rules require database queries (EXPLAIN analysis)
2. **Schema Fetching**: Loading complete schema for large databases
3. **Frontend Rendering**: Displaying 99+ rules with parameters

### Optimizations

#### Backend

1. **Schema Caching**: Cache schema metadata with TTL
   - Cache hit reduces review time by 50-70%
   - TTL: 5 minutes for active development, 1 hour for production

2. **Result Caching**: Cache review results for identical SQL + policy
   - Key: SHA256(sql + policyID)
   - TTL: 10 minutes
   - Invalidate on policy update

3. **Parallel Rule Execution**: Execute independent rules in parallel
   - Current: Sequential execution
   - Goal: 3-5x speedup with worker pool

4. **Database Connection Pooling**: Reuse connections for schema queries
   - Pool size: 10 connections
   - Max lifetime: 1 hour

#### Frontend

1. **Virtual Scrolling**: Render only visible rules
   - Renders ~20 rules instead of 99+
   - Reduces initial render time by 80%

2. **Lazy Loading**: Load rule metadata on demand
   - Load categories as accordion expands
   - Reduces initial bundle size

3. **Debounced Search**: Debounce search input
   - Wait 300ms after last keystroke
   - Reduces unnecessary re-renders

### Performance Targets

| Metric | Current | Target |
|--------|---------|--------|
| Review execution time | ~500ms | <200ms |
| Schema fetch time | ~1s | <300ms (with cache) |
| Frontend initial load | ~2s | <1s |
| Rule selector render | ~500ms | <100ms (with virtualization) |

---

## Success Criteria

### Functional Requirements

- ✅ All 99+ sql-reviewer rules exposed
- ✅ Rule types match sql-reviewer exactly
- ✅ Payload parameters configurable and validated
- ✅ Schema-aware validation working
- ✅ Result filtering by severity, rule type, category
- ✅ Policy templates available
- ✅ Migration script tested and documented

### Non-Functional Requirements

- ✅ Test coverage >80%
- ✅ Review execution <200ms (cached)
- ✅ Zero breaking changes for existing users
- ✅ Complete documentation
- ✅ Performance benchmarks established
- ✅ Monitoring and alerting configured

### User Experience

- ✅ Search and filter rules easily
- ✅ Understand rule parameters
- ✅ Load templates quickly
- ✅ See clear error messages for violations
- ✅ Track which rules catch most issues

---

## Timeline & Resources

### Phase-by-Phase Timeline

| Phase | Duration | Start | End | Dependencies |
|-------|----------|-------|-----|--------------|
| Phase 1: Rule Generation | 1 week | Week 1 | Week 1 | None |
| Phase 2: API Upgrade | 1-2 weeks | Week 2 | Week 3 | Phase 1 |
| Phase 3: Migration | 1 week | Week 3 | Week 4 | Phases 1-2 |
| Phase 4: Frontend | 1-2 weeks | Week 4 | Week 5 | Phases 1-3 |
| Phase 5: Advanced Features | 1-2 weeks | Week 5 | Week 7 | Phases 1-4 |
| Phase 6: Testing & Docs | Ongoing | Week 1 | Week 8 | All phases |

**Total Duration**: 6-8 weeks

### Resource Requirements

#### Engineering

- **Backend Engineer**: 2 FTE for 6-8 weeks
- **Frontend Engineer**: 1 FTE for 4-6 weeks
- **QA Engineer**: 0.5 FTE for 6-8 weeks

#### Infrastructure

- **Staging Environment**: For migration testing
- **Database Snapshots**: For migration rehearsal
- **Monitoring**: Enhanced metrics for review performance

---

## Appendices

### Appendix A: Complete Rule Type Mapping

| Old (v1) | New (v2) | Category |
|----------|----------|----------|
| `mysql.engine.use-innodb` | `engine.mysql.use-innodb` | Engine |
| `mysql.naming.table` | `naming.table` | Naming |
| `mysql.naming.column` | `naming.column` | Naming |
| `mysql.table.require-pk` | `table.require-pk` | Table |
| `mysql.column.no-null` | `column.no-null` | Column |
| `mysql.statement.select.no-select-all` | `statement.select.no-select-all` | Statement |
| `postgresql.naming.table` | `naming.table` | Naming |
| `postgresql.table.require-pk` | `table.require-pk` | Table |
| ... | ... | ... |

### Appendix B: Payload Parameter Reference

#### STRING Parameters

Used for regex patterns, templates, single string values.

**Example**:
```json
{
  "format": "^[a-z][a-z0-9_]*$"
}
```

#### NUMBER Parameters

Used for limits, thresholds, counts.

**Example**:
```json
{
  "number": 1000
}
```

#### BOOLEAN Parameters

Used for flags, toggle options.

**Example**:
```json
{
  "required": true
}
```

#### STRING_ARRAY Parameters

Used for lists, allowlists, disallowlists.

**Example**:
```json
{
  "list": ["users", "transactions", "audit_log"]
}
```

#### TEMPLATE Parameters

Used for pattern matching with tokens.

**Example**:
```json
{
  "format": "{{table}}_idx_{{column_list}}",
  "maxLength": 64
}
```

### Appendix C: Code Examples

#### Example 1: Using High-Level API

```go
package main

import (
    "context"
    "log"

    "github.com/nsxbet/sql-reviewer/pkg/reviewer"
    "github.com/nsxbet/sql-reviewer/pkg/types"
)

func main() {
    // Create reviewer for MySQL
    r := reviewer.New(types.Engine_MYSQL)

    // SQL to review
    sql := `
        CREATE TABLE users (
            id INT PRIMARY KEY,
            email VARCHAR(255)
        );
    `

    // Execute review
    ctx := context.Background()
    result, err := r.Review(ctx, sql)
    if err != nil {
        log.Fatal(err)
    }

    // Check results
    if result.HasErrors() {
        log.Println("Found errors:")
        for _, advice := range result.FilterByStatus(types.Advice_ERROR) {
            log.Printf("  - %s: %s\n", advice.Title, advice.Content)
        }
    }
}
```

#### Example 2: Schema-Aware Validation

```go
// Fetch schema
schema, err := schemaCache.Get(ctx, databaseID)
if err != nil {
    return nil, err
}

// Review with schema
result, err := r.ReviewWithSchema(ctx, sql, schema)
if err != nil {
    return nil, err
}

// Now rules can validate:
// - Column existence
// - Type compatibility
// - Index coverage
// - Constraint violations
```

#### Example 3: Custom Policy Configuration

```typescript
const policy: SQLReviewPolicyConfig = {
  version: 'v2',
  rules: {
    // Naming conventions
    'naming.table': {
      level: 'error',
      payload: {
        format: '^[a-z][a-z0-9_]*$',
        maxLength: 63,
      },
    },

    // Required columns
    'column.required': {
      level: 'error',
      payload: {
        list: ['id', 'created_at', 'updated_at'],
      },
    },

    // Safety limits
    'statement.affected-row-limit': {
      level: 'error',
      payload: {
        number: 1000,
      },
    },

    // Performance
    'statement.select.no-select-all': {
      level: 'warning',
      payload: {},
    },
  },
};
```

### Appendix D: API Specification

#### GET /api/v1/policies/sql-review/rules

**Response**:
```json
{
  "mysql": {
    "engine": [
      {
        "type": "engine.mysql.use-innodb",
        "name": "Require InnoDB Engine",
        "description": "All tables must use InnoDB storage engine",
        "category": "engine",
        "engine": "mysql",
        "parameters": []
      }
    ],
    "naming": [
      {
        "type": "naming.table",
        "name": "Table Naming Convention",
        "description": "Enforce table naming format and length",
        "category": "naming",
        "engine": "both",
        "parameters": [
          {
            "name": "format",
            "type": "STRING",
            "description": "Regex pattern for table names",
            "required": false,
            "default": "^[a-z][a-z0-9_]*$"
          },
          {
            "name": "maxLength",
            "type": "NUMBER",
            "description": "Maximum table name length",
            "required": false,
            "default": 64
          }
        ]
      }
    ]
  },
  "postgresql": {
    // Similar structure
  }
}
```

---

## Conclusion

This improvement plan addresses the critical gaps in Deeb's sql-reviewer integration, transforming it from a basic SQL validator into a comprehensive SQL quality platform. By implementing all phases, Deeb will:

- **Expose all 99+ sql-reviewer rules** with correct types and parameters
- **Auto-generate rule metadata** ensuring accuracy and future compatibility
- **Leverage advanced features** like schema-aware validation and result filtering
- **Provide excellent UX** with search, templates, and validation
- **Maintain production quality** with comprehensive testing and documentation

The phased approach allows for incremental delivery while managing risk through proper testing and migration strategies.

---

**Document Version**: 1.0
**Last Updated**: 2025-11-07
**Next Review**: After Phase 1 completion
