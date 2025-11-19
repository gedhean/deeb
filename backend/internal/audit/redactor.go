package audit

import (
	"encoding/json"
	"regexp"
	"strings"
)

const (
	// RedactedValue is the placeholder for redacted sensitive data
	RedactedValue = "[REDACTED]"
)

// SensitiveFieldPattern defines patterns for sensitive fields that should be redacted
type SensitiveFieldPattern struct {
	Pattern     *regexp.Regexp
	Description string
}

// Redactor handles redaction of sensitive data in audit logs
type Redactor struct {
	patterns []SensitiveFieldPattern
}

// NewRedactor creates a new Redactor with default sensitive field patterns
func NewRedactor() *Redactor {
	r := &Redactor{
		patterns: make([]SensitiveFieldPattern, 0),
	}

	// Add default patterns
	r.addDefaultPatterns()

	return r
}

// addDefaultPatterns adds the default set of sensitive field patterns
func (r *Redactor) addDefaultPatterns() {
	// Password fields (case-insensitive)
	mustAddPattern(r, `(?i)password`, "Password fields")
	mustAddPattern(r, `(?i)passwd`, "Password abbreviation")
	mustAddPattern(r, `(?i)pwd`, "Password short form")

	// Database credentials
	mustAddPattern(r, `(?i)db_password`, "Database password")
	mustAddPattern(r, `(?i)datasource_password`, "Datasource password")
	mustAddPattern(r, `(?i)connection_string`, "Connection string (may contain credentials)")
	mustAddPattern(r, `(?i)database_password`, "Database password variant")
	mustAddPattern(r, `(?i)db_pass`, "Database password abbreviation")

	// MFA/2FA secrets
	mustAddPattern(r, `(?i)mfa_secret`, "MFA secret")
	mustAddPattern(r, `(?i)totp_secret`, "TOTP secret")
	mustAddPattern(r, `(?i)backup_codes?`, "Backup codes")
	mustAddPattern(r, `(?i)recovery_keys?`, "Recovery keys")
	mustAddPattern(r, `(?i)otp_secret`, "OTP secret")
	mustAddPattern(r, `(?i)two_factor_secret`, "Two-factor secret")

	// Authentication tokens and keys
	mustAddPattern(r, `(?i)api_key`, "API key")
	mustAddPattern(r, `(?i)api_secret`, "API secret")
	mustAddPattern(r, `(?i)access_token`, "Access token")
	mustAddPattern(r, `(?i)refresh_token`, "Refresh token")
	mustAddPattern(r, `(?i)auth_token`, "Auth token")
	mustAddPattern(r, `(?i)bearer_token`, "Bearer token")
	mustAddPattern(r, `(?i)secret_key`, "Secret key")
	mustAddPattern(r, `(?i)private_key`, "Private key")
	mustAddPattern(r, `(?i)encryption_key`, "Encryption key")

	// Credit card and financial data
	mustAddPattern(r, `(?i)credit_card`, "Credit card number")
	mustAddPattern(r, `(?i)card_number`, "Card number")
	mustAddPattern(r, `(?i)cvv`, "CVV")
	mustAddPattern(r, `(?i)ssn`, "SSN")
	mustAddPattern(r, `(?i)social_security`, "Social security number")

	// Personal identifiable information
	mustAddPattern(r, `(?i)ssn`, "Social Security Number")
	mustAddPattern(r, `(?i)tax_id`, "Tax ID")
	mustAddPattern(r, `(?i)passport`, "Passport number")
	mustAddPattern(r, `(?i)license_number`, "License number")
}

// AddPattern adds a new sensitive field pattern to the redactor
func (r *Redactor) AddPattern(pattern, description string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}

	r.patterns = append(r.patterns, SensitiveFieldPattern{
		Pattern:     re,
		Description: description,
	})

	return nil
}

// RedactMap redacts sensitive fields in a map recursively
func (r *Redactor) RedactMap(data map[string]interface{}) map[string]interface{} {
	if data == nil {
		return nil
	}

	result := make(map[string]interface{})

	for key, value := range data {
		// Check if the key matches any sensitive pattern
		if r.isSensitiveKey(key) {
			result[key] = RedactedValue
			continue
		}

		// Recursively redact nested structures
		switch v := value.(type) {
		case map[string]interface{}:
			result[key] = r.RedactMap(v)
		case []interface{}:
			result[key] = r.redactSlice(v)
		default:
			result[key] = value
		}
	}

	return result
}

// redactSlice redacts sensitive fields in a slice recursively
func (r *Redactor) redactSlice(data []interface{}) []interface{} {
	if data == nil {
		return nil
	}

	result := make([]interface{}, len(data))

	for i, item := range data {
		switch v := item.(type) {
		case map[string]interface{}:
			result[i] = r.RedactMap(v)
		case []interface{}:
			result[i] = r.redactSlice(v)
		default:
			result[i] = item
		}
	}

	return result
}

// isSensitiveKey checks if a key matches any sensitive field pattern
func (r *Redactor) isSensitiveKey(key string) bool {
	for _, pattern := range r.patterns {
		if pattern.Pattern.MatchString(key) {
			return true
		}
	}
	return false
}

// RedactJSON redacts sensitive fields in JSON data
func (r *Redactor) RedactJSON(data json.RawMessage) (json.RawMessage, error) {
	if len(data) == 0 {
		return data, nil
	}

	var parsed interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return data, err
	}

	var redacted interface{}
	switch v := parsed.(type) {
	case map[string]interface{}:
		redacted = r.RedactMap(v)
	case []interface{}:
		redacted = r.redactSlice(v)
	default:
		// Primitive types don't need redaction
		redacted = parsed
	}

	result, err := json.Marshal(redacted)
	if err != nil {
		return data, err
	}

	return result, nil
}

// RedactConnectionString redacts credentials from connection strings
// Examples:
//   - postgresql://user:password@host:5432/db -> postgresql://user:[REDACTED]@host:5432/db
//   - mysql://user:password@host:3306/db -> mysql://user:[REDACTED]@host:3306/db
func (r *Redactor) RedactConnectionString(connStr string) string {
	if connStr == "" {
		return connStr
	}

	// Pattern to match user:password in connection strings
	// Uses greedy match .+ to capture everything between : and the LAST @
	re := regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9+.-]*://[^:/@]+:)(.+)(@[^@]+)`)

	return re.ReplaceAllString(connStr, "${1}"+RedactedValue+"${3}")
}

// RedactStringValue redacts a string if it appears to be sensitive
// This is useful for values that might be connection strings or other sensitive data
func (r *Redactor) RedactStringValue(value string) string {
	// Check if it's a connection string
	if strings.Contains(value, "://") && strings.Contains(value, "@") {
		return r.RedactConnectionString(value)
	}

	return value
}

// mustAddPattern is a helper that panics if pattern compilation fails
// This should only be used for hardcoded patterns that are known to be valid
func mustAddPattern(r *Redactor, pattern, description string) {
	if err := r.AddPattern(pattern, description); err != nil {
		panic("invalid redaction pattern: " + pattern + ": " + err.Error())
	}
}
