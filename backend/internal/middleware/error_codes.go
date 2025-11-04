package middleware

// Error codes for common error scenarios
const (
	// Validation Errors
	ErrorCodeValidationError = "VALIDATION_ERROR"
	ErrorCodeInvalidInput    = "INVALID_INPUT"
	ErrorCodeInvalidFormat   = "INVALID_FORMAT"
	ErrorCodeMissingRequired = "MISSING_REQUIRED_FIELD"

	// Authentication/Authorization Errors
	ErrorCodeUnauthorized            = "UNAUTHORIZED"
	ErrorCodeForbidden               = "FORBIDDEN"
	ErrorCodeInvalidToken            = "INVALID_TOKEN"
	ErrorCodeExpiredToken            = "EXPIRED_TOKEN"
	ErrorCodeInsufficientPermissions = "INSUFFICIENT_PERMISSIONS"

	// Resource Errors
	ErrorCodeNotFound      = "RESOURCE_NOT_FOUND"
	ErrorCodeAlreadyExists = "RESOURCE_ALREADY_EXISTS"
	ErrorCodeConflict      = "RESOURCE_CONFLICT"
	ErrorCodeInvalidState  = "INVALID_STATE"

	// Server Errors
	ErrorCodeInternalError        = "INTERNAL_ERROR"
	ErrorCodeDatabaseError        = "DATABASE_ERROR"
	ErrorCodeExternalServiceError = "EXTERNAL_SERVICE_ERROR"
	ErrorCodeTimeout              = "TIMEOUT"
)
