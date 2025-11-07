# Integrate sql-reviewer Package for SQL Review

## Overview

Replace the current custom validator system with the `github.com/nsxbet/sql-reviewer` package. Update the policy configuration structure to support sql-reviewer rules organized by database engine (MySQL/PostgreSQL) and rule category (Engine, Naming, Table, Column, System, etc.), with configurable severity levels (error/warning) and rule parameters.

## Backend Changes

### 1. Add sql-reviewer Dependency

- Update `backend/go.mod` to include `github.com/nsxbet/sql-reviewer`
- Run `go mod tidy` to fetch dependencies

### 2. Create sql-reviewer Integration Layer

- **File**: `backend/internal/review/sqlreviewer/advisor.go`
  - No adapter required for the sql-reviewer package. Implement the `github.com/nsxbet/sql-reviewer/pkg/advisor` Advisor interface and use it instead of the custom validator interface.
  - Implement the `advisor.Check` interface (see `github.com/nsxbet/sql-reviewer/pkg/advisor`)
  - Use sql-reviewer rule results to `advisor.Advice`  (see `github.com/nsxbet/sql-reviewer/pkg/advisor`)
  - Handle rule configuration (severity, parameters)

### 3. Update Policy Configuration Structure

- **File**: `backend/internal/models/policy.go`
  - Define new policy config structure:
    ```go
    type SQLReviewPolicyConfig struct {
        Rules map[string]SQLReviewRule `json:"rules"`
    }
    
    // SQLReviewRule represents a SQL review rule. See: github.com/nsxbet/sql-reviewer/pkg/types
    type SQLReviewRule struct {
        Type    string                 `json:"type"              yaml:"type"`
        Level   SQLReviewRuleLevel     `json:"level"             yaml:"level"` // error, warning
        Payload map[string]interface{} `json:"payload,omitempty" yaml:"payload,omitempty"`
        Engine  Engine                 `json:"engine"            yaml:"engine"`
        Comment string                 `json:"comment,omitempty" yaml:"comment,omitempty"`
    }
    
    // Advice represents a piece of advice from the advisor. See: github.com/nsxbet/sql-reviewer/pkg/types
    type Advice struct {
        Status        Advice_Status `json:"status"`
        Code          int32         `json:"code"`
        Title         string        `json:"title"`
        Content       string        `json:"content"`
        StartPosition *Position     `json:"startPosition"`
    }
    
    // Position represents a position in the source code. See: github.com/nsxbet/sql-reviewer/pkg/types
    type Position struct {
       Line   int32 `json:"line"`
       Column int32 `json:"column"`
    }
    
    // Advice_Status represents the status of an advice
    type Advice_Status int32
    ```

  - Do NOT maintain backward compatibility with old `{"validators": {...}}` format. Can remove it completely.

### 4. Update Review Service

- **File**: `backend/internal/review/service.go`
  - Create method to parse new rule-based config
  - Create sql-reviewer instances based on enabled rules
  - Support only new policy config formats

### 5. Create Rule Metadata Service

- **File**: `backend/internal/review/rules_metadata.go`
  - Define rule metadata structure (name, description, category, engine, parameters)
  - Provide function to get all available rules for MySQL and PostgreSQL
  - Categorize rules: Engine, Naming, Table, Column, System, etc.

### 6. Update Routes

- **File**: `backend/internal/routes/routes.go`
  - Remove old validator registrations
  - Initialize sql-reviewer advisor instead
  - Register sql-reviewer as the primary review engine

### 7. Add API Endpoint for Rule Metadata

- **File**: `backend/internal/handlers/policy_handler.go`
  - Add endpoint: `GET /api/policies/sql-review/rules`
  - Returns available rules organized by engine and category
  - Includes rule descriptions and parameter schemas

## Frontend Changes

### 1. Update Policy Types

- **File**: `frontend/src/lib/api/types.ts`
  - Update `PolicyConfig` interface to support new rule-based structure
  - Add types for rule metadata and rule configuration

### 2. Create Rule Selection Component

- **File**: `frontend/src/components/sql-review-rule-selector.tsx`
  - Component for selecting and configuring rules
  - Organized by engine (MySQL/PostgreSQL) tabs
  - Grouped by category (Engine, Naming, Table, Column, System, etc.)
  - Display rule name, description, and parameter inputs
  - Allow selecting severity level (error/warning) per rule

### 3. Update Policies Page

- **File**: `frontend/src/pages/PoliciesPage.tsx`
  - Replace simple validator checkboxes with new rule selector component
  - Fetch available rules from API endpoint
  - Update form handling to use new policy config structure
  - Maintain JSON editor option for advanced users

### 4. Add Rule Metadata Service

- **File**: `frontend/src/lib/api/services/policies.ts`
  - Add `getSQLReviewRules()` function to fetch rule metadata from API

## Policy Config Migration

### New Policy Config Format

```json
{
  "rules": {
    "mysql.engine.no-select-all": {
      "level": "error",
      "payload": {}
    },
    "postgresql.naming.table-name": {
      "level": "warning",
      "payload": {
        "max_length": 63
      }
    }
  }
}
```

## Testing Considerations

- Do not update existing tests to use sql-reviewer advisor (just remove old tests)
- Create new tests for sql-reviewer advisor
- Test policy config parsing new formats only (can remove old tests)
- Test rule selection and configuration in frontend (can remove old tests)
- Test SQL review execution with sql-reviewer rules (can remove old tests)

## Migration Strategy

1. Support only new policy config format and sql-reviewer rules
2. Remove old policy config format support and tests
