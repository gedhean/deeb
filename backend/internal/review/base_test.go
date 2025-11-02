package review

import (
	"context"
	"fmt"
	"testing"

	"github.com/nsxbet/deeb/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockValidator is a test implementation of the Validator interface
type mockValidator struct {
	BaseValidator
	validateFunc func(ctx context.Context, sqlContent string, dbType models.DatabaseType) Result
}

func (m *mockValidator) Validate(ctx context.Context, sqlContent string, dbType models.DatabaseType) Result {
	if m.validateFunc != nil {
		return m.validateFunc(ctx, sqlContent, dbType)
	}
	return NewResult(m.GetName(), true, []Violation{})
}

func TestBaseValidator(t *testing.T) {
	t.Run("GetName returns validator name", func(t *testing.T) {
		validator := &BaseValidator{
			Name:            "test-validator",
			DefaultSeverity: SeverityError,
		}

		assert.Equal(t, "test-validator", validator.GetName())
	})

	t.Run("GetSeverity returns default severity", func(t *testing.T) {
		validator := &BaseValidator{
			Name:            "test-validator",
			DefaultSeverity: SeverityWarning,
		}

		assert.Equal(t, SeverityWarning, validator.GetSeverity())
	})
}

func TestNewViolation(t *testing.T) {
	t.Run("creates violation with basic fields", func(t *testing.T) {
		violation := NewViolation("rule1", "test message", SeverityError)

		assert.Equal(t, "rule1", violation.Rule)
		assert.Equal(t, "test message", violation.Message)
		assert.Equal(t, SeverityError, violation.Severity)
		assert.Equal(t, 0, violation.Line)
		assert.Equal(t, 0, violation.Column)
		assert.Empty(t, violation.SQL)
	})

	t.Run("creates violation with location", func(t *testing.T) {
		violation := NewViolationWithLocation("rule1", "test message", SeverityWarning, 10, 5)

		assert.Equal(t, "rule1", violation.Rule)
		assert.Equal(t, "test message", violation.Message)
		assert.Equal(t, SeverityWarning, violation.Severity)
		assert.Equal(t, 10, violation.Line)
		assert.Equal(t, 5, violation.Column)
	})

	t.Run("creates violation with SQL context", func(t *testing.T) {
		violation := NewViolationWithSQL("rule1", "test message", SeverityInfo, "SELECT * FROM users")

		assert.Equal(t, "rule1", violation.Rule)
		assert.Equal(t, "test message", violation.Message)
		assert.Equal(t, SeverityInfo, violation.Severity)
		assert.Equal(t, "SELECT * FROM users", violation.SQL)
	})
}

func TestNewResult(t *testing.T) {
	t.Run("creates valid result", func(t *testing.T) {
		result := NewResult("validator1", true, []Violation{})

		assert.Equal(t, "validator1", result.ValidatorName)
		assert.True(t, result.Valid)
		assert.Empty(t, result.Violations)
		assert.Empty(t, result.Error)
	})

	t.Run("creates result with violations", func(t *testing.T) {
		violations := []Violation{
			NewViolation("rule1", "error message", SeverityError),
			NewViolation("rule2", "warning message", SeverityWarning),
		}

		result := NewResult("validator1", false, violations)

		assert.Equal(t, "validator1", result.ValidatorName)
		assert.False(t, result.Valid)
		assert.Len(t, result.Violations, 2)
		assert.Equal(t, violations, result.Violations)
	})

	t.Run("creates error result", func(t *testing.T) {
		err := assert.AnError
		result := NewErrorResult("validator1", err)

		assert.Equal(t, "validator1", result.ValidatorName)
		assert.False(t, result.Valid)
		assert.Empty(t, result.Violations)
		assert.Equal(t, err.Error(), result.Error)
	})
}

func TestValidatorRegistry(t *testing.T) {
	t.Run("NewValidatorRegistry creates empty registry", func(t *testing.T) {
		registry := NewValidatorRegistry()

		validators := registry.GetAll()
		assert.Empty(t, validators)
	})

	t.Run("Register adds validator", func(t *testing.T) {
		registry := NewValidatorRegistry()
		validator := &mockValidator{
			BaseValidator: BaseValidator{
				Name:            "test-validator",
				DefaultSeverity: SeverityError,
			},
		}

		err := registry.Register(validator)
		require.NoError(t, err)

		registered, err := registry.Get("test-validator")
		require.NoError(t, err)
		assert.Equal(t, validator, registered)
	})

	t.Run("Register rejects nil validator", func(t *testing.T) {
		registry := NewValidatorRegistry()

		err := registry.Register(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("Register rejects validator with empty name", func(t *testing.T) {
		registry := NewValidatorRegistry()
		validator := &mockValidator{
			BaseValidator: BaseValidator{
				Name:            "",
				DefaultSeverity: SeverityError,
			},
		}

		err := registry.Register(validator)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name cannot be empty")
	})

	t.Run("Register rejects duplicate name", func(t *testing.T) {
		registry := NewValidatorRegistry()
		validator1 := &mockValidator{
			BaseValidator: BaseValidator{
				Name:            "test-validator",
				DefaultSeverity: SeverityError,
			},
		}
		validator2 := &mockValidator{
			BaseValidator: BaseValidator{
				Name:            "test-validator",
				DefaultSeverity: SeverityWarning,
			},
		}

		err := registry.Register(validator1)
		require.NoError(t, err)

		err = registry.Register(validator2)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrValidatorAlreadyRegistered)
	})

	t.Run("Get returns registered validator", func(t *testing.T) {
		registry := NewValidatorRegistry()
		validator := &mockValidator{
			BaseValidator: BaseValidator{
				Name:            "test-validator",
				DefaultSeverity: SeverityError,
			},
		}

		registry.Register(validator)

		retrieved, err := registry.Get("test-validator")
		require.NoError(t, err)
		assert.Equal(t, validator, retrieved)
	})

	t.Run("Get returns error for non-existent validator", func(t *testing.T) {
		registry := NewValidatorRegistry()

		_, err := registry.Get("non-existent")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrValidatorNotFound)
	})

	t.Run("GetAll returns all validators", func(t *testing.T) {
		registry := NewValidatorRegistry()
		validator1 := &mockValidator{
			BaseValidator: BaseValidator{
				Name:            "validator1",
				DefaultSeverity: SeverityError,
			},
		}
		validator2 := &mockValidator{
			BaseValidator: BaseValidator{
				Name:            "validator2",
				DefaultSeverity: SeverityWarning,
			},
		}

		registry.Register(validator1)
		registry.Register(validator2)

		validators := registry.GetAll()
		assert.Len(t, validators, 2)
	})

	t.Run("Unregister removes validator", func(t *testing.T) {
		registry := NewValidatorRegistry()
		validator := &mockValidator{
			BaseValidator: BaseValidator{
				Name:            "test-validator",
				DefaultSeverity: SeverityError,
			},
		}

		registry.Register(validator)

		err := registry.Unregister("test-validator")
		require.NoError(t, err)

		_, err = registry.Get("test-validator")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrValidatorNotFound)
	})

	t.Run("Unregister returns error for non-existent validator", func(t *testing.T) {
		registry := NewValidatorRegistry()

		err := registry.Unregister("non-existent")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrValidatorNotFound)
	})

	t.Run("thread-safe concurrent access", func(t *testing.T) {
		registry := NewValidatorRegistry()

		// Register validators concurrently
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func(idx int) {
				defer func() { done <- true }()
				validator := &mockValidator{
					BaseValidator: BaseValidator{
						Name:            fmt.Sprintf("validator-%d", idx),
						DefaultSeverity: SeverityError,
					},
				}
				registry.Register(validator)
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		validators := registry.GetAll()
		assert.Len(t, validators, 10)
	})
}

func TestEngine(t *testing.T) {
	t.Run("NewEngine creates engine with new registry", func(t *testing.T) {
		engine := NewEngine()

		registry := engine.GetRegistry()
		assert.NotNil(t, registry)
		assert.Empty(t, registry.GetAll())
	})

	t.Run("RegisterValidator adds validator", func(t *testing.T) {
		engine := NewEngine()
		validator := &mockValidator{
			BaseValidator: BaseValidator{
				Name:            "test-validator",
				DefaultSeverity: SeverityError,
			},
		}

		err := engine.RegisterValidator(validator)
		require.NoError(t, err)

		retrieved, err := engine.GetRegistry().Get("test-validator")
		require.NoError(t, err)
		assert.Equal(t, validator, retrieved)
	})

	t.Run("Review with no validators returns valid result", func(t *testing.T) {
		engine := NewEngine()
		ctx := context.Background()

		result := engine.Review(ctx, "SELECT * FROM users", models.DatabaseTypePostgreSQL)

		assert.True(t, result.Valid)
		assert.Empty(t, result.Results)
		assert.Equal(t, 0, result.TotalViolations)
		assert.Equal(t, 0, result.Errors)
		assert.Equal(t, 0, result.Warnings)
		assert.Equal(t, 0, result.Infos)
	})

	t.Run("Review runs all validators", func(t *testing.T) {
		engine := NewEngine()
		ctx := context.Background()

		validator1 := &mockValidator{
			BaseValidator: BaseValidator{
				Name:            "validator1",
				DefaultSeverity: SeverityError,
			},
			validateFunc: func(ctx context.Context, sqlContent string, dbType models.DatabaseType) Result {
				return NewResult("validator1", false, []Violation{
					NewViolation("rule1", "error", SeverityError),
				})
			},
		}

		validator2 := &mockValidator{
			BaseValidator: BaseValidator{
				Name:            "validator2",
				DefaultSeverity: SeverityWarning,
			},
			validateFunc: func(ctx context.Context, sqlContent string, dbType models.DatabaseType) Result {
				return NewResult("validator2", false, []Violation{
					NewViolation("rule2", "warning", SeverityWarning),
				})
			},
		}

		engine.RegisterValidator(validator1)
		engine.RegisterValidator(validator2)

		result := engine.Review(ctx, "SELECT * FROM users", models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		assert.Len(t, result.Results, 2)
		assert.Equal(t, 2, result.TotalViolations)
		assert.Equal(t, 1, result.Errors)
		assert.Equal(t, 1, result.Warnings)
		assert.Equal(t, 0, result.Infos)
	})

	t.Run("Review aggregates results correctly", func(t *testing.T) {
		engine := NewEngine()
		ctx := context.Background()

		validator := &mockValidator{
			BaseValidator: BaseValidator{
				Name:            "test-validator",
				DefaultSeverity: SeverityError,
			},
			validateFunc: func(ctx context.Context, sqlContent string, dbType models.DatabaseType) Result {
				return NewResult("test-validator", false, []Violation{
					NewViolation("rule1", "error", SeverityError),
					NewViolation("rule2", "warning", SeverityWarning),
					NewViolation("rule3", "info", SeverityInfo),
				})
			},
		}

		engine.RegisterValidator(validator)

		result := engine.Review(ctx, "SELECT * FROM users", models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		assert.Equal(t, 3, result.TotalViolations)
		assert.Equal(t, 1, result.Errors)
		assert.Equal(t, 1, result.Warnings)
		assert.Equal(t, 1, result.Infos)
	})

	t.Run("Review with all valid results returns valid", func(t *testing.T) {
		engine := NewEngine()
		ctx := context.Background()

		validator := &mockValidator{
			BaseValidator: BaseValidator{
				Name:            "test-validator",
				DefaultSeverity: SeverityError,
			},
			validateFunc: func(ctx context.Context, sqlContent string, dbType models.DatabaseType) Result {
				return NewResult("test-validator", true, []Violation{})
			},
		}

		engine.RegisterValidator(validator)

		result := engine.Review(ctx, "SELECT * FROM users", models.DatabaseTypePostgreSQL)

		assert.True(t, result.Valid)
		assert.Equal(t, 0, result.TotalViolations)
	})
}

func TestAggregateResults(t *testing.T) {
	t.Run("aggregates multiple results", func(t *testing.T) {
		results := []Result{
			NewResult("validator1", false, []Violation{
				NewViolation("rule1", "error", SeverityError),
				NewViolation("rule2", "warning", SeverityWarning),
			}),
			NewResult("validator2", false, []Violation{
				NewViolation("rule3", "info", SeverityInfo),
			}),
		}

		aggregated := aggregateResults(results)

		assert.False(t, aggregated.Valid)
		assert.Equal(t, 3, aggregated.TotalViolations)
		assert.Equal(t, 1, aggregated.Errors)
		assert.Equal(t, 1, aggregated.Warnings)
		assert.Equal(t, 1, aggregated.Infos)
	})

	t.Run("empty results return valid", func(t *testing.T) {
		results := []Result{}

		aggregated := aggregateResults(results)

		assert.True(t, aggregated.Valid)
		assert.Equal(t, 0, aggregated.TotalViolations)
	})

	t.Run("results with only warnings return invalid", func(t *testing.T) {
		results := []Result{
			NewResult("validator1", false, []Violation{
				NewViolation("rule1", "warning", SeverityWarning),
			}),
		}

		aggregated := aggregateResults(results)

		// Currently, valid is false if there are any violations
		// This might need adjustment based on business logic
		assert.False(t, aggregated.Valid)
		assert.Equal(t, 1, aggregated.Warnings)
	})
}
