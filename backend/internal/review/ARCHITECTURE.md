# SQL Review Engine Architecture

## Overview

The SQL Review Engine is a pluggable, extensible system for validating SQL content against multiple policy rules. The architecture is designed to support multiple validators running in parallel, with a clear separation of concerns and easy extensibility.

## Architecture Components

### 1. Validator Interface

The core of the architecture is the `Validator` interface defined in `backend/internal/review/validator.go`. All validators must implement this interface:

```go
type Validator interface {
    Validate(ctx context.Context, sqlContent string, dbType models.DatabaseType) Result
    GetName() string
    GetSeverity() Severity
}
```

**Key Design Decisions:**
- **Context-based**: The `Validate` method accepts a `context.Context` for cancellation, timeout, and request-scoped data
- **Database-aware**: Validators receive the database type to enable database-specific validation logic
- **Result-oriented**: Each validator returns a `Result` struct containing violations found
- **Named**: Each validator has a unique name for identification and registration
- **Configurable Severity**: Validators can specify their default severity level

### 2. Result Structures

#### Violation
Represents a single validation finding:
- `Rule`: Name of the violated rule
- `Message`: Human-readable description
- `Severity`: Error, Warning, or Info
- `Line`/`Column`: Location information (optional)
- `SQL`: The SQL statement that caused the violation (optional, for context)

#### Result
Represents the output from a single validator:
- `ValidatorName`: Identifies which validator produced this result
- `Valid`: Boolean indicating if validation passed
- `Violations`: List of violations found
- `Error`: Set if the validator encountered an error

#### ReviewResult
Aggregated result from all validators:
- `Valid`: Overall validation status (true if all validators pass)
- `Results`: List of results from each validator
- `TotalViolations`: Total count of violations
- `Errors`/`Warnings`/`Infos`: Counts by severity level

### 3. Validator Registry

The `ValidatorRegistry` interface provides a way to register and manage validators:

```go
type ValidatorRegistry interface {
    Register(validator Validator) error
    Get(name string) (Validator, error)
    GetAll() []Validator
    Unregister(name string) error
}
```

**Key Features:**
- **Registration**: Validators can be registered dynamically
- **Lookup**: Validators can be retrieved by name
- **Enumeration**: All validators can be listed
- **Removal**: Validators can be unregistered

### 4. Review Engine

The `Engine` interface orchestrates the validation process:

```go
type Engine interface {
    Review(ctx context.Context, sqlContent string, dbType models.DatabaseType) ReviewResult
    RegisterValidator(validator Validator) error
    GetRegistry() ValidatorRegistry
}
```

**Responsibilities:**
- Coordinate validation across all registered validators
- Aggregate results from multiple validators
- Support parallel execution of validators
- Provide a unified API for SQL review

## Parallel Execution

The architecture supports running validators in parallel for performance:

1. **Concurrent Execution**: When `Review()` is called, all registered validators can run concurrently using goroutines
2. **Context Propagation**: The `context.Context` is passed to all validators, allowing cancellation and timeout to work across all validators
3. **Result Aggregation**: Results from all validators are collected and aggregated into a single `ReviewResult`
4. **Error Isolation**: If one validator fails, other validators continue running

## Extensibility

### Adding New Validators

To add a new validator:

1. **Implement the Validator Interface**:
   ```go
   type MyValidator struct {
       name string
   }
   
   func (v *MyValidator) Validate(ctx context.Context, sqlContent string, dbType models.DatabaseType) Result {
       // Validation logic here
   }
   
   func (v *MyValidator) GetName() string {
       return v.name
   }
   
   func (v *MyValidator) GetSeverity() Severity {
       return SeverityError
   }
   ```

2. **Register the Validator**:
   ```go
   validator := &MyValidator{name: "my-validator"}
   engine.RegisterValidator(validator)
   ```

3. **Use the Engine**:
   ```go
   result := engine.Review(ctx, sqlContent, dbType)
   ```

### Validator Types

The architecture supports various types of validators:

- **Syntax Validators**: Check SQL syntax correctness (PostgreSQL, MySQL)
- **Security Validators**: Detect dangerous operations (DROP, TRUNCATE, SQL injection)
- **Performance Validators**: Identify performance issues (missing indexes, full scans)
- **Best Practices Validators**: Enforce naming conventions, data types
- **Compatibility Validators**: Check backwards compatibility

## Design Patterns

### Strategy Pattern
Each validator implements a specific validation strategy. The engine orchestrates multiple strategies.

### Registry Pattern
Validators are registered in a central registry, enabling dynamic discovery and management.

### Result Aggregation
Results from multiple validators are aggregated into a unified result structure.

### Context Pattern
Use of `context.Context` for cancellation, timeout, and request-scoped data propagation.

## Example Usage

```go
// Create engine
engine := review.NewEngine()

// Register validators
engine.RegisterValidator(&syntax.Validator{})
engine.RegisterValidator(&security.Validator{})
engine.RegisterValidator(&performance.Validator{})

// Review SQL
result := engine.Review(ctx, sqlContent, models.DatabaseTypePostgreSQL)

// Check results
if !result.Valid {
    for _, res := range result.Results {
        for _, violation := range res.Violations {
            log.Printf("%s: %s (severity: %s)", 
                violation.Rule, 
                violation.Message, 
                violation.Severity)
        }
    }
}
```

## Future Enhancements

- **Validator Configuration**: Support for configuring validator behavior (e.g., severity levels, rule enable/disable)
- **Validator Dependencies**: Support for validators that depend on other validators
- **Caching**: Cache validation results for unchanged SQL content
- **Async Processing**: Support for async validation processing for long-running validators
- **Metrics**: Built-in metrics for validator performance and usage

