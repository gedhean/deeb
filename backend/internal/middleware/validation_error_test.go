package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestContext creates a test Echo context
func createTestContext(e *echo.Echo, method, path string, body []byte) (echo.Context, *httptest.ResponseRecorder) {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}

func TestValidationError(t *testing.T) {
	t.Run("empty validation error", func(t *testing.T) {
		ve := &ValidationError{}
		assert.Equal(t, "validation failed", ve.Error())
		assert.Empty(t, ve.Fields)
	})

	t.Run("single field error", func(t *testing.T) {
		ve := &ValidationError{}
		ve.AddField("title", "title is required", "required")
		assert.Equal(t, "title: title is required", ve.Error())
		assert.Len(t, ve.Fields, 1)
		assert.Equal(t, "title", ve.Fields[0].Field)
		assert.Equal(t, "title is required", ve.Fields[0].Message)
		assert.Equal(t, "required", ve.Fields[0].Rule)
	})

	t.Run("multiple field errors", func(t *testing.T) {
		ve := &ValidationError{}
		ve.AddField("title", "title is required", "required")
		ve.AddField("description", "description is required", "required")
		assert.Equal(t, "title: title is required; description: description is required", ve.Error())
		assert.Len(t, ve.Fields, 2)
	})

	t.Run("new validation error", func(t *testing.T) {
		fields := []FieldError{
			{Field: "title", Message: "title is required", Rule: "required"},
			{Field: "description", Message: "description is required", Rule: "required"},
		}
		ve := NewValidationError(fields)
		assert.Len(t, ve.Fields, 2)
		assert.Equal(t, fields, ve.Fields)
	})

	t.Run("new field error", func(t *testing.T) {
		fe := NewFieldError("email", "email is required", "required")
		assert.Equal(t, "email", fe.Field)
		assert.Equal(t, "email is required", fe.Message)
		assert.Equal(t, "required", fe.Rule)
	})
}

func TestExtractValidationError(t *testing.T) {
	t.Run("extract from ValidationError", func(t *testing.T) {
		ve := &ValidationError{}
		ve.AddField("title", "title is required", "required")

		extracted, ok := ExtractValidationError(ve)
		require.True(t, ok)
		assert.Equal(t, ve, extracted)
	})

	t.Run("extract from HTTPErrorWithCode", func(t *testing.T) {
		fields := []FieldError{
			{Field: "title", Message: "title is required", Rule: "required"},
		}

		details := map[string]interface{}{
			"validation_errors": map[string]interface{}{
				"fields": fields,
			},
		}

		hec := NewHTTPErrorWithCode(http.StatusBadRequest, ErrorCodeValidationError, "Validation failed", details)
		extracted, ok := ExtractValidationError(hec)
		require.True(t, ok)
		assert.Len(t, extracted.Fields, 1)
		assert.Equal(t, "title", extracted.Fields[0].Field)
	})

	t.Run("extract from echo.HTTPError with ValidationError internal", func(t *testing.T) {
		ve := &ValidationError{}
		ve.AddField("title", "title is required", "required")

		he := echo.NewHTTPError(http.StatusBadRequest, "Validation failed")
		he.Internal = ve

		extracted, ok := ExtractValidationError(he)
		require.True(t, ok)
		assert.Equal(t, ve, extracted)
	})

	t.Run("extract from echo.HTTPError with HTTPErrorWithCode internal", func(t *testing.T) {
		details := map[string]interface{}{
			"validation_errors": map[string]interface{}{
				"fields": []FieldError{
					{Field: "title", Message: "title is required", Rule: "required"},
				},
			},
		}

		hec := NewHTTPErrorWithCode(http.StatusBadRequest, ErrorCodeValidationError, "Validation failed", details)
		he := echo.NewHTTPError(http.StatusBadRequest, "Validation failed")
		he.Internal = hec

		extracted, ok := ExtractValidationError(he)
		require.True(t, ok)
		assert.Len(t, extracted.Fields, 1)
		assert.Equal(t, "title", extracted.Fields[0].Field)
	})

	t.Run("not a validation error", func(t *testing.T) {
		err := errors.New("some other error")
		extracted, ok := ExtractValidationError(err)
		assert.False(t, ok)
		assert.Nil(t, extracted)
	})
}

func TestParseStringValidationError(t *testing.T) {
	t.Run("field: message format", func(t *testing.T) {
		err := errors.New("title: title is required")
		ve := ParseStringValidationError(err)
		require.Len(t, ve.Fields, 1)
		assert.Equal(t, "title", ve.Fields[0].Field)
		assert.Equal(t, "title is required", ve.Fields[0].Message)
		assert.Equal(t, "required", ve.Fields[0].Rule)
	})

	t.Run("field is required format", func(t *testing.T) {
		err := errors.New("title is required")
		ve := ParseStringValidationError(err)
		require.Len(t, ve.Fields, 1)
		assert.Equal(t, "title", ve.Fields[0].Field)
		assert.Equal(t, "title is required", ve.Fields[0].Message)
		assert.Equal(t, "required", ve.Fields[0].Rule)
	})

	t.Run("field must not exceed format", func(t *testing.T) {
		err := errors.New("title must not exceed 255 characters")
		ve := ParseStringValidationError(err)
		require.Len(t, ve.Fields, 1)
		assert.Equal(t, "title", ve.Fields[0].Field)
		assert.Equal(t, "title must not exceed 255 characters", ve.Fields[0].Message)
		assert.Equal(t, "max", ve.Fields[0].Rule)
	})

	t.Run("field must be format", func(t *testing.T) {
		err := errors.New("change_type must be DDL or DML")
		ve := ParseStringValidationError(err)
		require.Len(t, ve.Fields, 1)
		assert.Equal(t, "change_type", ve.Fields[0].Field)
		assert.Equal(t, "change_type must be DDL or DML", ve.Fields[0].Message)
		assert.Equal(t, "enum", ve.Fields[0].Rule)
	})

	t.Run("unknown format", func(t *testing.T) {
		err := errors.New("some random error message")
		ve := ParseStringValidationError(err)
		require.Len(t, ve.Fields, 1)
		assert.Equal(t, "general", ve.Fields[0].Field)
		assert.Equal(t, "some random error message", ve.Fields[0].Message)
		assert.Equal(t, "validation_failed", ve.Fields[0].Rule)
	})
}

func TestExtractRuleFromMessage(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{"required", "field is required", "required"},
		{"must not exceed", "field must not exceed 255", "max"},
		{"exceeds maximum", "field exceeds maximum size", "max"},
		{"must be at least", "field must be at least 8", "min"},
		{"minimum", "field minimum length", "min"},
		{"must be", "field must be X or Y", "enum"},
		{"invalid", "field is invalid", "enum"},
		{"format", "field has invalid format", "format"},
		{"invalid format", "field invalid format", "format"},
		{"unknown", "some random message", "validation_failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRuleFromMessage(tt.message)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHandleValidationError(t *testing.T) {
	e := echo.New()

	t.Run("handle ValidationError", func(t *testing.T) {
		c, rec := createTestContext(e, "POST", "/test", nil)
		ve := &ValidationError{}
		ve.AddField("title", "title is required", "required")
		ve.AddField("description", "description is required", "required")

		err := HandleValidationError(c, ve)
		// HandleValidationError returns an error which Echo will handle
		require.NotNil(t, err)
		// Use Echo's error handler to process it
		CustomErrorHandler(err, c)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var response ErrorResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
		assert.Equal(t, "Bad Request", response.Error)
		// Message should be the ValidationError's Error() output
		assert.Contains(t, response.Message, "title: title is required")
		assert.Contains(t, response.Message, "description: description is required")
		assert.Equal(t, ErrorCodeValidationError, response.Code)
		require.NotNil(t, response.Details)
		validationErrors, ok := response.Details["validation_errors"].(map[string]interface{})
		require.True(t, ok)
		fields, ok := validationErrors["fields"].([]interface{})
		require.True(t, ok)
		assert.Len(t, fields, 2)
	})

	t.Run("handle bind error", func(t *testing.T) {
		c, rec := createTestContext(e, "POST", "/test", nil)
		bindErr := echo.NewHTTPError(http.StatusBadRequest, "invalid request body")

		err := HandleValidationError(c, bindErr)
		require.NotNil(t, err)
		CustomErrorHandler(err, c)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var response ErrorResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
		assert.Equal(t, ErrorCodeValidationError, response.Code)
	})

	t.Run("handle string error", func(t *testing.T) {
		c, rec := createTestContext(e, "POST", "/test", nil)
		err := errors.New("title is required")

		result := HandleValidationError(c, err)
		require.NotNil(t, result)
		CustomErrorHandler(result, c)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var response ErrorResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
		assert.Equal(t, ErrorCodeValidationError, response.Code)
	})
}
