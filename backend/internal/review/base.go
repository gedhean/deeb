package review

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/nsxbet/deeb/internal/models"
)

// ErrValidatorNotFound is returned when a validator is not found in the registry
var ErrValidatorNotFound = errors.New("validator not found")

// ErrValidatorAlreadyRegistered is returned when trying to register a validator with a duplicate name
var ErrValidatorAlreadyRegistered = errors.New("validator already registered")

// BaseValidator provides a base implementation of the Validator interface
// Validators can embed this struct to get common functionality
type BaseValidator struct {
	// Name is the unique identifier for this validator
	Name string
	// DefaultSeverity is the default severity level for violations from this validator
	DefaultSeverity Severity
}

// GetName returns the name of the validator
func (b *BaseValidator) GetName() string {
	return b.Name
}

// GetSeverity returns the default severity level for violations from this validator
func (b *BaseValidator) GetSeverity() Severity {
	return b.DefaultSeverity
}

// Validate must be implemented by concrete validators
// BaseValidator does not provide a default implementation
func (b *BaseValidator) Validate(ctx context.Context, sqlContent string, dbType models.DatabaseType) Result {
	// This should never be called directly - concrete validators must implement Validate
	panic("BaseValidator.Validate() should not be called directly - implement in concrete validator")
}

// NewViolation creates a new Violation with the given parameters
// This is a helper function for validators to create violations consistently
func NewViolation(rule string, message string, severity Severity) Violation {
	return Violation{
		Rule:     rule,
		Message:  message,
		Severity: severity,
	}
}

// NewViolationWithLocation creates a new Violation with location information
func NewViolationWithLocation(rule string, message string, severity Severity, line int, column int) Violation {
	return Violation{
		Rule:     rule,
		Message:  message,
		Severity: severity,
		Line:     line,
		Column:   column,
	}
}

// NewViolationWithSQL creates a new Violation with SQL context
func NewViolationWithSQL(rule string, message string, severity Severity, sql string) Violation {
	return Violation{
		Rule:     rule,
		Message:  message,
		Severity: severity,
		SQL:      sql,
	}
}

// NewResult creates a new Result for a validator
func NewResult(validatorName string, valid bool, violations []Violation) Result {
	return Result{
		ValidatorName: validatorName,
		Valid:         valid,
		Violations:    violations,
	}
}

// NewErrorResult creates a Result indicating an error occurred during validation
func NewErrorResult(validatorName string, err error) Result {
	return Result{
		ValidatorName: validatorName,
		Valid:         false,
		Violations:    []Violation{},
		Error:         err.Error(),
	}
}

// validatorRegistry is a thread-safe implementation of ValidatorRegistry
type validatorRegistry struct {
	mu         sync.RWMutex
	validators map[string]Validator
}

// NewValidatorRegistry creates a new validator registry
func NewValidatorRegistry() ValidatorRegistry {
	return &validatorRegistry{
		validators: make(map[string]Validator),
	}
}

// Register adds a validator to the registry
func (r *validatorRegistry) Register(validator Validator) error {
	if validator == nil {
		return errors.New("validator cannot be nil")
	}

	name := validator.GetName()
	if name == "" {
		return errors.New("validator name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.validators[name]; exists {
		return fmt.Errorf("%w: %s", ErrValidatorAlreadyRegistered, name)
	}

	r.validators[name] = validator
	return nil
}

// Get retrieves a validator by name
func (r *validatorRegistry) Get(name string) (Validator, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	validator, exists := r.validators[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrValidatorNotFound, name)
	}

	return validator, nil
}

// GetAll returns all registered validators
func (r *validatorRegistry) GetAll() []Validator {
	r.mu.RLock()
	defer r.mu.RUnlock()

	validators := make([]Validator, 0, len(r.validators))
	for _, validator := range r.validators {
		validators = append(validators, validator)
	}

	return validators
}

// Unregister removes a validator from the registry
func (r *validatorRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.validators[name]; !exists {
		return fmt.Errorf("%w: %s", ErrValidatorNotFound, name)
	}

	delete(r.validators, name)
	return nil
}

// engine is a concrete implementation of the Engine interface
type engine struct {
	registry ValidatorRegistry
}

// NewEngine creates a new SQL Review engine
func NewEngine() Engine {
	return &engine{
		registry: NewValidatorRegistry(),
	}
}

// NewEngineWithRegistry creates a new engine with a custom registry
func NewEngineWithRegistry(registry ValidatorRegistry) Engine {
	return &engine{
		registry: registry,
	}
}

// RegisterValidator adds a validator to the engine
func (e *engine) RegisterValidator(validator Validator) error {
	return e.registry.Register(validator)
}

// GetRegistry returns the validator registry
func (e *engine) GetRegistry() ValidatorRegistry {
	return e.registry
}

// Review validates SQL content using all registered validators
// Validators run concurrently for performance
func (e *engine) Review(ctx context.Context, sqlContent string, dbType models.DatabaseType) ReviewResult {
	validators := e.registry.GetAll()

	if len(validators) == 0 {
		return ReviewResult{
			Valid:           true,
			Results:         []Result{},
			TotalViolations: 0,
			Errors:          0,
			Warnings:        0,
			Infos:           0,
		}
	}

	// Run validators concurrently
	type validatorResult struct {
		result Result
	}

	resultChan := make(chan validatorResult, len(validators))
	var wg sync.WaitGroup

	for _, validator := range validators {
		wg.Add(1)
		go func(v Validator) {
			defer wg.Done()

			// Run validation
			result := v.Validate(ctx, sqlContent, dbType)
			resultChan <- validatorResult{result: result}
		}(validator)
	}

	// Wait for all validators to complete
	wg.Wait()
	close(resultChan)

	// Collect results
	results := make([]Result, 0, len(validators))
	for vr := range resultChan {
		results = append(results, vr.result)
	}

	// Aggregate results
	return aggregateResults(results)
}

// aggregateResults combines results from multiple validators into a single ReviewResult
func aggregateResults(results []Result) ReviewResult {
	totalViolations := 0
	errors := 0
	warnings := 0
	infos := 0

	for _, result := range results {
		for _, violation := range result.Violations {
			totalViolations++
			switch violation.Severity {
			case SeverityError:
				errors++
			case SeverityWarning:
				warnings++
			case SeverityInfo:
				infos++
			}
		}
	}

	// Valid if no errors occurred and no violations found
	valid := errors == 0 && totalViolations == 0

	return ReviewResult{
		Valid:           valid,
		Results:         results,
		TotalViolations: totalViolations,
		Errors:          errors,
		Warnings:        warnings,
		Infos:           infos,
	}
}
