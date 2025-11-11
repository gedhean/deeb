# SQL Review System Architecture

**Version**: 2.0
**Last Updated**: November 7, 2025
**Audience**: Backend Developers, Contributors

## Overview

The SQL Review V2 system provides comprehensive SQL validation with 101 rules organized by database engine and category. This guide explains the architecture, components, and how to extend the system.

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                     Change Plan Handler                      │
│  (internal/handlers/change_plan_handler.go)                 │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                   SQL Review Service                         │
│        (internal/review/service_v2.go)                       │
│                                                              │
│  • Determines policy version (V1 vs V2)                     │
│  • Routes to appropriate validator                          │
│  • Aggregates findings from all validators                  │
└──────────────────────────┬──────────────────────────────────┘
                           │
           ┌───────────────┴───────────────┐
           │                               │
           ▼                               ▼
┌──────────────────────┐      ┌──────────────────────┐
│    V1 Validators     │      │    V2 Validators     │
│  (20 rules, legacy)  │      │  (101 rules, new)    │
└──────────────────────┘      └──────────┬───────────┘
                                         │
                   ┌─────────────────────┼─────────────────────┐
                   │                     │                     │
                   ▼                     ▼                     ▼
         ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
         │   sqlreviewer   │  │   Performance   │  │    Security     │
         │   (core rules)  │  │   Validator     │  │   Validator     │
         └─────────────────┘  └─────────────────┘  └─────────────────┘
                   │
                   ▼
         ┌─────────────────┐
         │  Rule Metadata  │
         │  (rules_metadata│
         │        .go)     │
         └─────────────────┘
```

---

## Core Components

### 1. SQL Review Service (`internal/review/service_v2.go`)

**Purpose**: Orchestrates the SQL review process

**Key Functions**:

```go
// Main entry point for SQL review
func (s *Service) ReviewSQL(ctx context.Context, params ReviewParams) (*ReviewResult, error)

// Route to V1 or V2 validators based on policy version
func (s *Service) routeToValidator(policy *models.Policy, sql string) (*ReviewResult, error)
```

**Responsibilities**:
- Load policy configuration
- Determine schema version (V1 or V2)
- Route to appropriate validators
- Aggregate findings from multiple validators
- Return unified review result

---

### 2. SQLReviewer Package (`internal/review/sqlreviewer/`)

**Purpose**: Core validation engine with 101 rules

**Structure**:
```
sqlreviewer/
├── advisor.go              # Main review orchestration
├── engine/
│   ├── mysql/             # MySQL-specific rules
│   └── postgresql/        # PostgreSQL-specific rules
├── validators/
│   ├── naming.go          # Naming convention rules
│   ├── schema.go          # Schema design rules
│   ├── statement.go       # SQL statement rules
│   └── ...
└── types.go               # Shared types and interfaces
```

**Key Interface**:
```go
type Advisor interface {
    // Check SQL against policy rules
    Check(ctx context.Context, sql string, policy Policy) ([]Finding, error)
}

type Finding struct {
    RuleType     string
    RuleName     string
    Message      string
    Line         int
    Column       int
    Severity     string // "error", "warning", "info"
    ValidatorType string
}
```

---

### 3. Rule Metadata System (`internal/review/rules_metadata.go`)

**Purpose**: Provides rule information for API and UI

**Generated Code**:
```go
// Auto-generated rule metadata
var RuleMetadata = map[string]RuleInfo{
    "engine.mysql.use-innodb": {
        Type:        "engine.mysql.use-innodb",
        Name:        "Use InnoDB Engine",
        Description: "Require InnoDB storage engine for all tables",
        Category:    "Engine",
        Engine:      "mysql",
    },
    // ... 100 more rules
}
```

**Generation**:
- Metadata extracted from sqlreviewer package
- Generated using `internal/review/generator/`
- Auto-updates when rules change

---

### 4. Validator Categories

#### A. Backwards Compatibility Validator
**File**: `internal/review/validators/backwards_compatibility.go`

**Purpose**: Detect breaking schema changes

**Rules**:
- Column drops without safety checks
- Type changes that truncate data
- NOT NULL additions without defaults
- Index drops on foreign keys

**Example**:
```go
func (v *BackwardsCompatibilityValidator) Validate(ctx context.Context, sql string) []Finding {
    findings := []Finding{}

    // Check for unsafe column drops
    if containsDropColumn(sql) {
        findings = append(findings, Finding{
            RuleType: "schema.backwards-compat.column-drop",
            Message:  "Dropping columns may break existing queries",
            Severity: "warning",
        })
    }

    return findings
}
```

---

#### B. Performance Validator
**File**: `internal/review/validators/performance.go`

**Purpose**: Identify performance issues

**Rules**:
- Missing indexes on foreign keys
- Full table scans
- Inefficient data types
- Large IN clauses
- SELECT * usage

**Example**:
```go
func (v *PerformanceValidator) checkForeignKeyIndex(sql string) *Finding {
    // Parse ADD FOREIGN KEY statements
    if hasForeignKeyWithoutIndex(sql) {
        return &Finding{
            RuleType: "index.performance.foreign-key-index",
            Message:  "Foreign key column should have an index",
            Severity: "error",
        }
    }
    return nil
}
```

---

#### C. Security Validator
**File**: `internal/review/validators/security.go`

**Purpose**: Detect security vulnerabilities

**Rules**:
- Password column too short
- Sensitive data without encryption
- SQL injection patterns
- Weak default values
- Missing access controls

**Example**:
```go
func (v *SecurityValidator) checkPasswordColumn(col Column) *Finding {
    if isPasswordColumn(col.Name) && col.Length < 255 {
        return &Finding{
            RuleType: "column.security.password-length",
            Message:  "Password columns should be VARCHAR(255) or longer for hashing",
            Severity: "warning",
        }
    }
    return nil
}
```

---

#### D. Best Practices Validator
**File**: `internal/review/validators/best_practices.go`

**Purpose**: Enforce SQL best practices

**Rules**:
- Require primary keys
- Explicit NULL/NOT NULL
- Appropriate data types
- Consistent naming
- Table comments

---

#### E. Syntax Validators

**MySQL**: `internal/review/validators/mysql_syntax.go`
**PostgreSQL**: `internal/review/validators/postgres_syntax.go`

**Purpose**: Engine-specific syntax validation

**MySQL Rules**:
- InnoDB engine requirement
- UTF-8 character set
- Proper collation

**PostgreSQL Rules**:
- Schema qualification
- Sequence usage
- Extension requirements

---

### 5. Policy Migration Service

**File**: `internal/services/policy_migration_service.go`

**Purpose**: Handle V1 → V2 policy upgrades

**Key Functions**:
```go
// Migrate single policy
func (s *PolicyMigrationService) MigratePolicy(ctx context.Context, policyID string) (*MigrationResult, error)

// Migrate all V1 policies
func (s *PolicyMigrationService) MigrateAllPolicies(ctx context.Context) (*BatchMigrationResult, error)

// Get migration status
func (s *PolicyMigrationService) GetMigrationStatus(ctx context.Context) (*MigrationStatus, error)
```

**Migration Logic**:
1. Load V1 policy configuration
2. Generate V2 rules from metadata
3. Merge with existing V1 rules
4. Update schema_version to "2.0"
5. Save updated policy
6. Return migration statistics

---

## Data Flow

### SQL Review Flow

```
1. User submits change plan with SQL
   ↓
2. Handler validates request
   ↓
3. Load policy for target environment
   ↓
4. Check policy schema_version
   ↓
5a. V1 Policy → Use legacy validators (20 rules)
   OR
5b. V2 Policy → Use sqlreviewer + validators (101 rules)
   ↓
6. Parse SQL into AST
   ↓
7. Apply each applicable rule
   ↓
8. Collect findings (errors, warnings, info)
   ↓
9. Aggregate by severity
   ↓
10. Return PolicyFindingsSummary
```

### Policy Migration Flow

```
1. Admin clicks "Migrate to V2"
   ↓
2. Migration handler receives request
   ↓
3. Load current policy (V1)
   ↓
4. Load all V2 rule metadata
   ↓
5. Create V2 rule configurations
   ↓
6. Merge with existing V1 rules
   ↓
7. Set appropriate severity levels
   ↓
8. Update policy.schema_version = "2.0"
   ↓
9. Save to database
   ↓
10. Return migration result
```

---

## Database Schema

### Policies Table

```sql
CREATE TABLE policies (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL,  -- 'SQL_REVIEW'
    policy_config JSONB NOT NULL,
    schema_version VARCHAR(10) NOT NULL DEFAULT '2.0',  -- NEW FIELD
    created_by_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
```

### Policy Config Structure (V2)

```json
{
  "rules": {
    "engine.mysql.use-innodb": {
      "type": "engine.mysql.use-innodb",
      "level": "error",
      "engine": "mysql",
      "comment": "All tables must use InnoDB"
    },
    "naming.table.snake-case": {
      "type": "naming.table.snake-case",
      "level": "warning",
      "engine": "mysql",
      "comment": "Use snake_case for table names"
    }
    // ... 99 more rules
  }
}
```

---

## Adding New Rules

### Step 1: Decide on Rule Placement

**Option A: Add to sqlreviewer package**
- Best for: Core validation rules
- Location: `internal/review/sqlreviewer/`
- Pros: Automatic metadata generation, consistent structure
- Cons: Requires understanding sqlreviewer internals

**Option B: Add to custom validator**
- Best for: Deeb-specific rules, business logic
- Location: `internal/review/validators/`
- Pros: Simpler, independent from sqlreviewer
- Cons: Manual metadata management

---

### Step 2: Implement Rule in sqlreviewer (Option A)

1. **Add rule to appropriate file**:

```go
// internal/review/sqlreviewer/validators/naming.go

func checkTableNaming(table Table) *Finding {
    if !isSnakeCase(table.Name) {
        return &Finding{
            RuleType: "naming.table.snake-case",
            RuleName: "Use Snake Case for Tables",
            Message:  fmt.Sprintf("Table '%s' should use snake_case", table.Name),
            Severity: "warning",
            ValidatorType: "naming",
        }
    }
    return nil
}
```

2. **Regenerate metadata**:

```bash
cd backend
make generate-rules-metadata
```

3. **Add tests**:

```go
func TestTableNaming(t *testing.T) {
    sql := "CREATE TABLE UserAccounts (id INT);"
    findings := checkTableNaming(parseTable(sql))

    assert.NotNil(t, findings)
    assert.Equal(t, "naming.table.snake-case", findings.RuleType)
}
```

---

### Step 3: Implement Custom Validator (Option B)

1. **Create validator file**:

```go
// internal/review/validators/custom_rules.go

package validators

type CustomValidator struct {
    policy *models.Policy
}

func NewCustomValidator(policy *models.Policy) *CustomValidator {
    return &CustomValidator{policy: policy}
}

func (v *CustomValidator) Validate(ctx context.Context, sql string) []review.Finding {
    findings := []review.Finding{}

    // Your custom validation logic
    if violatesCustomRule(sql) {
        findings = append(findings, review.Finding{
            RuleType:      "custom.my-rule",
            RuleName:      "My Custom Rule",
            Message:       "SQL violates custom organizational rule",
            Severity:      "error",
            ValidatorType: "custom",
        })
    }

    return findings
}
```

2. **Register validator**:

```go
// internal/review/service_v2.go

func (s *Service) getValidators(policy *models.Policy) []Validator {
    validators := []Validator{
        validators.NewBackwardsCompatibilityValidator(policy),
        validators.NewPerformanceValidator(policy),
        validators.NewSecurityValidator(policy),
        validators.NewBestPracticesValidator(policy),
        validators.NewCustomValidator(policy),  // Add here
    }
    return validators
}
```

3. **Add to metadata manually**:

```go
// internal/review/rules_metadata.go

var RuleMetadata = map[string]RuleInfo{
    // ... existing rules
    "custom.my-rule": {
        Type:        "custom.my-rule",
        Name:        "My Custom Rule",
        Description: "Enforces custom organizational requirement",
        Category:    "Custom",
        Engine:      "mysql",  // or "postgresql" or "both"
    },
}
```

---

## Testing

### Unit Tests

```go
func TestSQLReview(t *testing.T) {
    // Setup
    policy := &models.Policy{
        SchemaVersion: "2.0",
        PolicyConfig: models.SQLReviewPolicyConfig{
            Rules: map[string]models.SQLReviewRule{
                "engine.mysql.use-innodb": {
                    Type:   "engine.mysql.use-innodb",
                    Level:  "error",
                    Engine: "mysql",
                },
            },
        },
    }

    service := review.NewService(mockRepo)

    // Test
    sql := "CREATE TABLE users (id INT) ENGINE=MyISAM;"
    result, err := service.ReviewSQL(context.Background(), review.ReviewParams{
        SQL:    sql,
        Policy: policy,
        Engine: "mysql",
    })

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, 1, result.Findings.Counts.Errors)
    assert.Contains(t, result.Findings.Errors[0].Message, "InnoDB")
}
```

### Integration Tests

```go
func TestMigrationEndToEnd(t *testing.T) {
    // Create V1 policy
    v1Policy := createV1Policy(t)

    // Migrate to V2
    result, err := migrationService.MigratePolicy(ctx, v1Policy.ID)
    assert.NoError(t, err)
    assert.True(t, result.Success)
    assert.Equal(t, 81, result.RulesAdded)

    // Verify V2 policy works
    v2Policy, err := policyRepo.GetByID(ctx, v1Policy.ID)
    assert.NoError(t, err)
    assert.Equal(t, "2.0", v2Policy.SchemaVersion)

    // Test SQL review with V2
    reviewResult, err := reviewService.ReviewSQL(ctx, review.ReviewParams{
        SQL:    testSQL,
        Policy: v2Policy,
    })
    assert.NoError(t, err)
    assert.Greater(t, len(reviewResult.Findings.Errors), 0)
}
```

---

## Performance Considerations

### 1. Rule Evaluation Order

Rules are evaluated in order of:
1. **Syntax validation** (fast, fails early)
2. **Security rules** (critical, checked early)
3. **Performance rules** (complex, checked mid)
4. **Best practices** (informational, checked last)

### 2. Caching

**Schema Cache**: Parsed ASTs cached per request
```go
type SchemaCache struct {
    mu     sync.RWMutex
    cache  map[string]*ParsedSchema
}
```

**Rule Metadata**: Loaded once at startup
```go
var RuleMetadata = loadRuleMetadata()  // Called once
```

### 3. Parallel Validation

Validators run in parallel when possible:
```go
func (s *Service) runValidators(validators []Validator, sql string) []Finding {
    var wg sync.WaitGroup
    results := make(chan []Finding, len(validators))

    for _, v := range validators {
        wg.Add(1)
        go func(validator Validator) {
            defer wg.Done()
            results <- validator.Validate(context.Background(), sql)
        }(v)
    }

    // Collect results...
}
```

---

## Debugging

### Enable Debug Logging

```go
// internal/review/service_v2.go

logger.Debug("Reviewing SQL",
    "policy_version", policy.SchemaVersion,
    "rule_count", len(policy.PolicyConfig.Rules),
    "sql_length", len(sql))
```

### Common Issues

**Issue**: Rule not triggering
```go
// Check if rule is in policy config
logger.Debug("Active rules", "rules", policy.PolicyConfig.Rules)

// Check rule matching logic
logger.Debug("Checking rule", "type", ruleType, "match", isMatch)
```

**Issue**: Wrong findings returned
```go
// Log each finding
for _, finding := range findings {
    logger.Debug("Finding", "rule", finding.RuleType, "severity", finding.Severity)
}
```

---

## See Also

- [API Documentation](../api/sql-review-v2-api.md)
- [User Migration Guide](../user-guides/sql-review-v2-migration.md)
- [SQL Review Rules Reference](../reference/sql-review-rules-reference.md)
- [Deployment Guide](../deployment/sql-review-v2-deployment.md)
