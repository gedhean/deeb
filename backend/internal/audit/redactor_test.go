package audit

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRedactor(t *testing.T) {
	r := NewRedactor()
	assert.NotNil(t, r)
	assert.NotEmpty(t, r.patterns)
}

func TestRedactor_RedactMap_Passwords(t *testing.T) {
	r := NewRedactor()

	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "password field",
			input: map[string]interface{}{
				"email":    "user@example.com",
				"password": "secret123",
			},
			expected: map[string]interface{}{
				"email":    "user@example.com",
				"password": RedactedValue,
			},
		},
		{
			name: "case insensitive password",
			input: map[string]interface{}{
				"Password": "secret123",
				"PASSWORD": "secret456",
			},
			expected: map[string]interface{}{
				"Password": RedactedValue,
				"PASSWORD": RedactedValue,
			},
		},
		{
			name: "password variations",
			input: map[string]interface{}{
				"old_password":     "old123",
				"new_password":     "new123",
				"current_password": "current123",
				"passwd":           "pass123",
				"pwd":              "pwd123",
			},
			expected: map[string]interface{}{
				"old_password":     RedactedValue,
				"new_password":     RedactedValue,
				"current_password": RedactedValue,
				"passwd":           RedactedValue,
				"pwd":              RedactedValue,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.RedactMap(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRedactor_RedactMap_DatabaseCredentials(t *testing.T) {
	r := NewRedactor()

	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "database password fields",
			input: map[string]interface{}{
				"db_host":     "localhost",
				"db_password": "dbpass123",
				"db_user":     "admin",
			},
			expected: map[string]interface{}{
				"db_host":     "localhost",
				"db_password": RedactedValue,
				"db_user":     "admin",
			},
		},
		{
			name: "datasource password",
			input: map[string]interface{}{
				"datasource_password": "dspass123",
				"database_password":   "dbpass456",
			},
			expected: map[string]interface{}{
				"datasource_password": RedactedValue,
				"database_password":   RedactedValue,
			},
		},
		{
			name: "connection string",
			input: map[string]interface{}{
				"connection_string": "postgresql://user:pass@localhost:5432/db",
			},
			expected: map[string]interface{}{
				"connection_string": RedactedValue,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.RedactMap(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRedactor_RedactMap_MFASecrets(t *testing.T) {
	r := NewRedactor()

	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "MFA secrets",
			input: map[string]interface{}{
				"mfa_secret":  "JBSWY3DPEHPK3PXP",
				"totp_secret": "JBSWY3DPEHPK3PXP",
				"otp_secret":  "123456",
			},
			expected: map[string]interface{}{
				"mfa_secret":  RedactedValue,
				"totp_secret": RedactedValue,
				"otp_secret":  RedactedValue,
			},
		},
		{
			name: "backup codes",
			input: map[string]interface{}{
				"backup_codes": []string{"code1", "code2", "code3"},
			},
			expected: map[string]interface{}{
				"backup_codes": RedactedValue,
			},
		},
		{
			name: "recovery keys",
			input: map[string]interface{}{
				"recovery_key":  "key123",
				"recovery_keys": []string{"key1", "key2"},
			},
			expected: map[string]interface{}{
				"recovery_key":  RedactedValue,
				"recovery_keys": RedactedValue,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.RedactMap(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRedactor_RedactMap_Tokens(t *testing.T) {
	r := NewRedactor()

	input := map[string]interface{}{
		"api_key":        "sk_live_123456",
		"api_secret":     "secret_abc",
		"access_token":   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		"refresh_token":  "refresh_xyz",
		"bearer_token":   "bearer_token_123",
		"secret_key":     "sk_123",
		"encryption_key": "aes256_key",
	}

	result := r.RedactMap(input)

	// All token fields should be redacted
	for key := range input {
		assert.Equal(t, RedactedValue, result[key], "Field %s should be redacted", key)
	}
}

func TestRedactor_RedactMap_Nested(t *testing.T) {
	r := NewRedactor()

	input := map[string]interface{}{
		"user": map[string]interface{}{
			"email":    "user@example.com",
			"password": "secret123",
			"profile": map[string]interface{}{
				"name":        "John Doe",
				"api_key":     "key123",
				"public_data": "visible",
			},
		},
		"database": map[string]interface{}{
			"host":     "localhost",
			"password": "dbpass",
		},
	}

	result := r.RedactMap(input)

	// Check nested redaction
	user := result["user"].(map[string]interface{})
	assert.Equal(t, "user@example.com", user["email"])
	assert.Equal(t, RedactedValue, user["password"])

	profile := user["profile"].(map[string]interface{})
	assert.Equal(t, "John Doe", profile["name"])
	assert.Equal(t, RedactedValue, profile["api_key"])
	assert.Equal(t, "visible", profile["public_data"])

	database := result["database"].(map[string]interface{})
	assert.Equal(t, "localhost", database["host"])
	assert.Equal(t, RedactedValue, database["password"])
}

func TestRedactor_RedactMap_Arrays(t *testing.T) {
	r := NewRedactor()

	input := map[string]interface{}{
		"users": []interface{}{
			map[string]interface{}{
				"email":    "user1@example.com",
				"password": "pass1",
			},
			map[string]interface{}{
				"email":    "user2@example.com",
				"password": "pass2",
			},
		},
	}

	result := r.RedactMap(input)

	users := result["users"].([]interface{})
	assert.Len(t, users, 2)

	user1 := users[0].(map[string]interface{})
	assert.Equal(t, "user1@example.com", user1["email"])
	assert.Equal(t, RedactedValue, user1["password"])

	user2 := users[1].(map[string]interface{})
	assert.Equal(t, "user2@example.com", user2["email"])
	assert.Equal(t, RedactedValue, user2["password"])
}

func TestRedactor_RedactJSON(t *testing.T) {
	r := NewRedactor()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple object",
			input:    `{"email":"user@example.com","password":"secret123"}`,
			expected: `{"email":"user@example.com","password":"[REDACTED]"}`,
		},
		{
			name:     "nested object",
			input:    `{"user":{"email":"test@example.com","password":"pass"}}`,
			expected: `{"user":{"email":"test@example.com","password":"[REDACTED]"}}`,
		},
		{
			name:     "array of objects",
			input:    `[{"password":"pass1"},{"password":"pass2"}]`,
			expected: `[{"password":"[REDACTED]"},{"password":"[REDACTED]"}]`,
		},
		{
			name:     "empty json",
			input:    `{}`,
			expected: `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.RedactJSON(json.RawMessage(tt.input))
			require.NoError(t, err)

			// Parse both to compare (order might differ)
			var expectedMap interface{}
			var resultMap interface{}
			require.NoError(t, json.Unmarshal([]byte(tt.expected), &expectedMap))
			require.NoError(t, json.Unmarshal(result, &resultMap))

			assert.Equal(t, expectedMap, resultMap)
		})
	}
}

func TestRedactor_RedactJSON_InvalidJSON(t *testing.T) {
	r := NewRedactor()

	invalidJSON := json.RawMessage(`{invalid json}`)
	result, err := r.RedactJSON(invalidJSON)

	assert.Error(t, err)
	assert.Equal(t, invalidJSON, result)
}

func TestRedactor_RedactConnectionString(t *testing.T) {
	r := NewRedactor()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "postgresql connection string",
			input:    "postgresql://user:password123@localhost:5432/mydb",
			expected: "postgresql://user:[REDACTED]@localhost:5432/mydb",
		},
		{
			name:     "mysql connection string",
			input:    "mysql://admin:secret@db.example.com:3306/database",
			expected: "mysql://admin:[REDACTED]@db.example.com:3306/database",
		},
		{
			name:     "connection string with special chars",
			input:    "postgresql://user:p@ssw0rd!@host:5432/db",
			expected: "postgresql://user:[REDACTED]@host:5432/db",
		},
		{
			name:     "connection string without password",
			input:    "postgresql://localhost:5432/mydb",
			expected: "postgresql://localhost:5432/mydb",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.RedactConnectionString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRedactor_AddPattern(t *testing.T) {
	r := NewRedactor()

	// Add custom pattern
	err := r.AddPattern(`(?i)custom_secret`, "Custom secret field")
	require.NoError(t, err)

	input := map[string]interface{}{
		"custom_secret": "my_secret",
		"other_field":   "visible",
	}

	result := r.RedactMap(input)

	assert.Equal(t, RedactedValue, result["custom_secret"])
	assert.Equal(t, "visible", result["other_field"])
}

func TestRedactor_AddPattern_InvalidRegex(t *testing.T) {
	r := NewRedactor()

	err := r.AddPattern(`[invalid(regex`, "Invalid pattern")
	assert.Error(t, err)
}

func TestRedactor_RedactMap_PreservesStructure(t *testing.T) {
	r := NewRedactor()

	input := map[string]interface{}{
		"string_field": "value",
		"int_field":    42,
		"float_field":  3.14,
		"bool_field":   true,
		"null_field":   nil,
		"password":     "secret",
	}

	result := r.RedactMap(input)

	assert.Equal(t, "value", result["string_field"])
	assert.Equal(t, 42, result["int_field"])
	assert.Equal(t, 3.14, result["float_field"])
	assert.Equal(t, true, result["bool_field"])
	assert.Nil(t, result["null_field"])
	assert.Equal(t, RedactedValue, result["password"])
}

func TestRedactor_RedactMap_NilInput(t *testing.T) {
	r := NewRedactor()

	result := r.RedactMap(nil)
	assert.Nil(t, result)
}

func TestRedactor_RedactJSON_EmptyInput(t *testing.T) {
	r := NewRedactor()

	result, err := r.RedactJSON(json.RawMessage{})
	require.NoError(t, err)
	assert.Empty(t, result)
}
