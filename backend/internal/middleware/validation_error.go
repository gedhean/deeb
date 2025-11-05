package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// FieldError represents a validation error for a specific field
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Rule    string `json:"rule,omitempty"` // e.g., "required", "max", "min", "format"
}

// ValidationError represents multiple field validation errors
type ValidationError struct {
	Fields []FieldError `json:"fields"`
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	if len(e.Fields) == 0 {
		return "validation failed"
	}

	var messages []string
	for _, field := range e.Fields {
		messages = append(messages, fmt.Sprintf("%s: %s", field.Field, field.Message))
	}

	return strings.Join(messages, "; ")
}

// NewValidationError creates a new ValidationError with field errors
func NewValidationError(fields []FieldError) *ValidationError {
	return &ValidationError{
		Fields: fields,
	}
}

// AddField adds a field error to the validation error
func (e *ValidationError) AddField(field, message, rule string) {
	e.Fields = append(e.Fields, FieldError{
		Field:   field,
		Message: message,
		Rule:    rule,
	})
}

// NewFieldError creates a new FieldError
func NewFieldError(field, message, rule string) FieldError {
	return FieldError{
		Field:   field,
		Message: message,
		Rule:    rule,
	}
}

// ExtractValidationError extracts validation errors from various error types
// Supports:
// - ValidationError (structured)
// - echo.HTTPError with validation errors in Internal field
// - Bind errors from Echo
// - String errors that can be parsed
func ExtractValidationError(err error) (*ValidationError, bool) {
	// Check if it's already a ValidationError
	if ve, ok := err.(*ValidationError); ok {
		return ve, true
	}

	// Check if it's wrapped in HTTPError
	if he, ok := err.(*echo.HTTPError); ok {
		if ve, ok := he.Internal.(*ValidationError); ok {
			return ve, true
		}
		// Check if HTTPErrorWithCode wraps a ValidationError
		if hec, ok := he.Internal.(*HTTPErrorWithCode); ok {
			if details, ok := hec.Details["validation_errors"]; ok {
				if veMap, ok := details.(map[string]interface{}); ok {
					fields := extractFieldsFromMap(veMap)
					return NewValidationError(fields), true
				}
			}
		}
		// Check if HTTPErrorWithCode directly has validation errors
		if hec, ok := err.(*HTTPErrorWithCode); ok {
			if details, ok := hec.Details["validation_errors"]; ok {
				if veMap, ok := details.(map[string]interface{}); ok {
					fields := extractFieldsFromMap(veMap)
					return NewValidationError(fields), true
				}
			}
		}
	}

	// Check if it's an HTTPErrorWithCode
	if hec, ok := err.(*HTTPErrorWithCode); ok {
		if details, ok := hec.Details["validation_errors"]; ok {
			if veMap, ok := details.(map[string]interface{}); ok {
				fields := extractFieldsFromMap(veMap)
				return NewValidationError(fields), true
			}
		}
	}

	return nil, false
}

// extractFieldsFromMap extracts FieldError slice from a map
func extractFieldsFromMap(veMap map[string]interface{}) []FieldError {
	var fields []FieldError

	// Try to extract fields array
	if fieldsArr, ok := veMap["fields"].([]interface{}); ok {
		for _, f := range fieldsArr {
			if fieldMap, ok := f.(map[string]interface{}); ok {
				field := FieldError{}
				if fieldName, ok := fieldMap["field"].(string); ok {
					field.Field = fieldName
				}
				if message, ok := fieldMap["message"].(string); ok {
					field.Message = message
				}
				if rule, ok := fieldMap["rule"].(string); ok {
					field.Rule = rule
				}
				fields = append(fields, field)
			} else if fe, ok := f.(FieldError); ok {
				// Direct FieldError struct
				fields = append(fields, fe)
			}
		}
	} else if fieldsArr, ok := veMap["fields"].([]FieldError); ok {
		// Direct slice of FieldError
		fields = fieldsArr
	}

	return fields
}

// HandleValidationError creates a standardized validation error response
func HandleValidationError(c echo.Context, err error) error {
	// Try to extract structured validation error
	ve, ok := ExtractValidationError(err)
	if !ok {
		// If not a structured validation error, try to parse it
		// Check if it's a bind error from Echo
		if bindErr, ok := err.(*echo.HTTPError); ok && bindErr.Code == http.StatusBadRequest {
			// Try to extract field-level errors from bind error message
			return NewHTTPErrorWithCode(
				http.StatusBadRequest,
				ErrorCodeValidationError,
				"Validation failed",
				map[string]interface{}{
					"validation_errors": map[string]interface{}{
						"fields": []FieldError{
							{
								Field:   "request_body",
								Message: fmt.Sprintf("%v", bindErr.Message),
								Rule:    "invalid_format",
							},
						},
					},
				},
			)
		}

		// Fallback: treat as single validation error
		return NewHTTPErrorWithCode(
			http.StatusBadRequest,
			ErrorCodeValidationError,
			err.Error(),
			map[string]interface{}{
				"validation_errors": map[string]interface{}{
					"fields": []FieldError{
						{
							Field:   "general",
							Message: err.Error(),
							Rule:    "validation_failed",
						},
					},
				},
			},
		)
	}

	// Return structured validation error
	return NewHTTPErrorWithCode(
		http.StatusBadRequest,
		ErrorCodeValidationError,
		"Validation failed",
		map[string]interface{}{
			"validation_errors": map[string]interface{}{
				"fields": ve.Fields,
			},
		},
	)
}

// ParseStringValidationError attempts to parse a string error message into field errors
// This handles errors like "title is required", "title must not exceed 255 characters"
func ParseStringValidationError(err error) *ValidationError {
	errMsg := err.Error()

	// Common patterns:
	// "field_name is required"
	// "field_name must not exceed X characters"
	// "field_name must be X or Y"
	// "field_name: message"

	ve := &ValidationError{}

	// Try to extract field name from common patterns
	parts := strings.Split(errMsg, ":")
	if len(parts) >= 2 {
		// Format: "field: message"
		field := strings.TrimSpace(parts[0])
		message := strings.TrimSpace(strings.Join(parts[1:], ":"))
		rule := extractRuleFromMessage(message)

		ve.AddField(field, message, rule)
		return ve
	}

	// Try patterns like "field is required"
	if strings.Contains(errMsg, " is required") {
		field := strings.TrimSuffix(errMsg, " is required")
		field = strings.TrimSpace(field)
		ve.AddField(field, errMsg, "required")
		return ve
	}

	// Try patterns like "field must not exceed"
	if strings.Contains(errMsg, " must not exceed") {
		parts := strings.Split(errMsg, " must not exceed")
		if len(parts) >= 2 {
			field := strings.TrimSpace(parts[0])
			message := errMsg
			ve.AddField(field, message, "max")
			return ve
		}
	}

	// Try patterns like "field must be"
	if strings.Contains(errMsg, " must be ") {
		parts := strings.Split(errMsg, " must be ")
		if len(parts) >= 2 {
			field := strings.TrimSpace(parts[0])
			message := errMsg
			ve.AddField(field, message, "enum")
			return ve
		}
	}

	// Fallback: treat as general error
	ve.AddField("general", errMsg, "validation_failed")
	return ve
}

// extractRuleFromMessage extracts validation rule from error message
func extractRuleFromMessage(message string) string {
	message = strings.ToLower(message)

	if strings.Contains(message, "required") {
		return "required"
	}
	if strings.Contains(message, "must not exceed") || strings.Contains(message, "exceeds maximum") {
		return "max"
	}
	if strings.Contains(message, "must be at least") || strings.Contains(message, "minimum") {
		return "min"
	}
	// Check for format before invalid (so "invalid format" matches format)
	if strings.Contains(message, "format") || strings.Contains(message, "invalid format") {
		return "format"
	}
	if strings.Contains(message, "must be") || strings.Contains(message, "invalid") {
		return "enum"
	}

	return "validation_failed"
}
