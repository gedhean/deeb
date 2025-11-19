package audit

import (
	"encoding/json"
	"testing"
)

// BenchmarkRedactor_RedactMap benchmarks the redaction of a typical audit log payload
func BenchmarkRedactor_RedactMap(b *testing.B) {
	redactor := NewRedactor()

	// Typical audit log payload with mixed sensitive and non-sensitive fields
	data := map[string]interface{}{
		"method":      "POST",
		"path":        "/api/v1/auth/login",
		"status":      200,
		"status_code": 200,
		"result":      "success",
		"email":       "user@example.com",
		"password":    "SuperSecretPassword123!",
		"user_agent":  "Mozilla/5.0...",
		"ip_address":  "192.168.1.1",
		"timestamp":   "2025-01-18T10:30:00Z",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = redactor.RedactMap(data)
	}
}

// BenchmarkRedactor_RedactMap_Nested benchmarks redaction with nested structures
func BenchmarkRedactor_RedactMap_Nested(b *testing.B) {
	redactor := NewRedactor()

	// Nested structure with multiple levels
	data := map[string]interface{}{
		"method": "POST",
		"path":   "/api/v1/database-instances",
		"status": 201,
		"config": map[string]interface{}{
			"host": "db.example.com",
			"port": 5432,
			"credentials": map[string]interface{}{
				"username":          "dbuser",
				"password":          "SecretDBPassword",
				"connection_string": "postgresql://user:pass@host:5432/db",
			},
			"options": map[string]interface{}{
				"ssl":     true,
				"timeout": 30,
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = redactor.RedactMap(data)
	}
}

// BenchmarkRedactor_RedactMap_LargePayload benchmarks redaction with a large payload
func BenchmarkRedactor_RedactMap_LargePayload(b *testing.B) {
	redactor := NewRedactor()

	// Large payload with many fields
	data := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		data["field_"+string(rune(i))] = "value"
	}
	// Add some sensitive fields
	data["password"] = "SecretPassword"
	data["api_key"] = "sk-1234567890"
	data["db_password"] = "DBSecret"
	data["connection_string"] = "postgresql://user:pass@host/db"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = redactor.RedactMap(data)
	}
}

// BenchmarkRedactor_RedactJSON benchmarks JSON redaction
func BenchmarkRedactor_RedactJSON(b *testing.B) {
	redactor := NewRedactor()

	jsonData := []byte(`{
		"method": "POST",
		"path": "/api/v1/auth/login",
		"status": 200,
		"email": "user@example.com",
		"password": "SuperSecretPassword123!",
		"api_key": "sk-1234567890",
		"user_agent": "Mozilla/5.0..."
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = redactor.RedactJSON(jsonData)
	}
}

// BenchmarkRedactor_RedactConnectionString benchmarks connection string redaction
func BenchmarkRedactor_RedactConnectionString(b *testing.B) {
	redactor := NewRedactor()
	connStr := "postgresql://username:SuperSecretPassword@db.example.com:5432/production_db"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = redactor.RedactConnectionString(connStr)
	}
}

// BenchmarkRedactor_isSensitiveKey benchmarks the sensitive key check
func BenchmarkRedactor_isSensitiveKey(b *testing.B) {
	redactor := NewRedactor()

	keys := []string{
		"password",
		"email",
		"api_key",
		"username",
		"db_password",
		"normal_field",
		"connection_string",
		"public_data",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, key := range keys {
			_ = redactor.isSensitiveKey(key)
		}
	}
}

// BenchmarkRedactor_AddPattern benchmarks adding new patterns
func BenchmarkRedactor_AddPattern(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		redactor := NewRedactor()
		_ = redactor.AddPattern(`(?i)custom_secret`, "Custom secret")
	}
}

// BenchmarkRedactor_RedactMap_NoSensitiveData benchmarks redaction when no sensitive data is present
func BenchmarkRedactor_RedactMap_NoSensitiveData(b *testing.B) {
	redactor := NewRedactor()

	// Data with no sensitive fields
	data := map[string]interface{}{
		"method":      "GET",
		"path":        "/api/v1/projects",
		"status":      200,
		"status_code": 200,
		"result":      "success",
		"user_agent":  "Mozilla/5.0...",
		"ip_address":  "192.168.1.1",
		"timestamp":   "2025-01-18T10:30:00Z",
		"project_id":  "123e4567-e89b-12d3-a456-426614174000",
		"count":       42,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = redactor.RedactMap(data)
	}
}

// BenchmarkRedactor_RedactMap_WithArrays benchmarks redaction with arrays
func BenchmarkRedactor_RedactMap_WithArrays(b *testing.B) {
	redactor := NewRedactor()

	data := map[string]interface{}{
		"method": "POST",
		"users": []interface{}{
			map[string]interface{}{
				"name":     "User 1",
				"email":    "user1@example.com",
				"password": "Secret1",
			},
			map[string]interface{}{
				"name":     "User 2",
				"email":    "user2@example.com",
				"password": "Secret2",
			},
			map[string]interface{}{
				"name":     "User 3",
				"email":    "user3@example.com",
				"password": "Secret3",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = redactor.RedactMap(data)
	}
}

// Benchmark comparison: with vs without redaction
func BenchmarkMiddleware_BuildDetails_WithRedaction(b *testing.B) {
	redactor := NewRedactor()

	details := map[string]interface{}{
		"method":      "POST",
		"path":        "/api/v1/auth/login",
		"status":      200,
		"email":       "user@example.com",
		"password":    "SecretPassword",
		"api_key":     "sk-1234567890",
		"user_agent":  "Mozilla/5.0...",
		"normal_data": "some public information",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		redacted := redactor.RedactMap(details)
		_, _ = json.Marshal(redacted)
	}
}

func BenchmarkMiddleware_BuildDetails_WithoutRedaction(b *testing.B) {
	details := map[string]interface{}{
		"method":      "POST",
		"path":        "/api/v1/auth/login",
		"status":      200,
		"email":       "user@example.com",
		"password":    "SecretPassword",
		"api_key":     "sk-1234567890",
		"user_agent":  "Mozilla/5.0...",
		"normal_data": "some public information",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(details)
	}
}
