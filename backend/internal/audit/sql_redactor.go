package audit

import (
	"regexp"
	"strings"
)

// SQLStatementType represents the type of SQL statement
type SQLStatementType string

const (
	// DDL (Data Definition Language) statements
	SQLTypeDDL SQLStatementType = "DDL"
	// DML (Data Manipulation Language) statements
	SQLTypeDML SQLStatementType = "DML"
	// DCL (Data Control Language) statements
	SQLTypeDCL SQLStatementType = "DCL"
	// TCL (Transaction Control Language) statements
	SQLTypeTCL SQLStatementType = "TCL"
	// Query statements
	SQLTypeQuery SQLStatementType = "QUERY"
	// Unknown statement type
	SQLTypeUnknown SQLStatementType = "UNKNOWN"
)

// SQLRiskLevel represents the risk level of a SQL statement
type SQLRiskLevel string

const (
	// High risk operations (DROP, TRUNCATE, DELETE without WHERE, etc.)
	SQLRiskHigh SQLRiskLevel = "high"
	// Medium risk operations (UPDATE, DELETE with WHERE, ALTER, etc.)
	SQLRiskMedium SQLRiskLevel = "medium"
	// Low risk operations (SELECT, INSERT, etc.)
	SQLRiskLow SQLRiskLevel = "low"
)

// SQLExecutionDetails contains metadata about SQL execution
// This struct is safe to log as it explicitly excludes query results
type SQLExecutionDetails struct {
	// SQL statement (full text)
	Statement string `json:"statement"`
	// Type of SQL statement (DDL, DML, etc.)
	StatementType SQLStatementType `json:"statement_type"`
	// Risk level of the operation
	RiskLevel SQLRiskLevel `json:"risk_level"`
	// Database name where query was executed
	DatabaseName string `json:"database_name,omitempty"`
	// Environment (dev, staging, prod)
	Environment string `json:"environment,omitempty"`
	// Number of rows affected/returned
	RowsAffected int64 `json:"rows_affected,omitempty"`
	// Execution time in milliseconds
	ExecutionTimeMS int64 `json:"execution_time_ms,omitempty"`
	// Whether the execution was successful
	Success bool `json:"success"`
	// Error message if execution failed
	ErrorMessage string `json:"error_message,omitempty"`
	// NOTE: Query results are NEVER included in this struct
}

// ClassifySQL analyzes a SQL statement and returns its type and risk level
func ClassifySQL(sql string) (SQLStatementType, SQLRiskLevel) {
	// Normalize SQL for analysis
	normalized := strings.TrimSpace(strings.ToUpper(sql))

	if normalized == "" {
		return SQLTypeUnknown, SQLRiskLow
	}

	// Get the first word (statement type)
	firstWord := strings.Split(normalized, " ")[0]

	var stmtType SQLStatementType
	var riskLevel SQLRiskLevel

	switch firstWord {
	// High risk DDL operations
	case "DROP":
		stmtType = SQLTypeDDL
		riskLevel = SQLRiskHigh
	case "TRUNCATE":
		stmtType = SQLTypeDDL
		riskLevel = SQLRiskHigh

	// Medium risk DDL operations
	case "CREATE":
		stmtType = SQLTypeDDL
		riskLevel = SQLRiskMedium
	case "ALTER":
		stmtType = SQLTypeDDL
		riskLevel = SQLRiskMedium
	case "RENAME":
		stmtType = SQLTypeDDL
		riskLevel = SQLRiskMedium

	// DML operations
	case "INSERT":
		stmtType = SQLTypeDML
		riskLevel = SQLRiskLow
	case "UPDATE":
		stmtType = SQLTypeDML
		// Check if UPDATE has WHERE clause
		if !strings.Contains(normalized, "WHERE") {
			riskLevel = SQLRiskHigh // UPDATE without WHERE is dangerous
		} else {
			riskLevel = SQLRiskMedium
		}
	case "DELETE":
		stmtType = SQLTypeDML
		// Check if DELETE has WHERE clause
		if !strings.Contains(normalized, "WHERE") {
			riskLevel = SQLRiskHigh // DELETE without WHERE is dangerous
		} else {
			riskLevel = SQLRiskMedium
		}

	// Query operations
	case "SELECT", "SHOW", "DESCRIBE", "DESC", "EXPLAIN":
		stmtType = SQLTypeQuery
		riskLevel = SQLRiskLow

	// DCL operations
	case "GRANT", "REVOKE":
		stmtType = SQLTypeDCL
		riskLevel = SQLRiskMedium

	// TCL operations
	case "COMMIT", "ROLLBACK", "SAVEPOINT":
		stmtType = SQLTypeTCL
		riskLevel = SQLRiskLow

	// Other operations
	case "BEGIN", "START":
		stmtType = SQLTypeTCL
		riskLevel = SQLRiskLow

	default:
		stmtType = SQLTypeUnknown
		riskLevel = SQLRiskLow
	}

	return stmtType, riskLevel
}

// SanitizeSQL performs basic sanitization on SQL for logging
// Returns the SQL with potential sensitive literals removed or truncated
func SanitizeSQL(sql string, maxLength int) string {
	if maxLength > 0 && len(sql) > maxLength {
		return sql[:maxLength] + "... [truncated]"
	}
	return sql
}

// ValidateSQLExecutionDetails ensures that query results are not included
// This is a safety check to prevent accidental data leakage
func ValidateSQLExecutionDetails(details map[string]interface{}) map[string]interface{} {
	// List of keys that should never be present (query results)
	forbiddenKeys := []string{
		"results", "rows", "data", "result_set", "query_result",
		"columns", "values", "records", "dataset",
	}

	// Remove any forbidden keys
	for _, key := range forbiddenKeys {
		delete(details, key)
	}

	// Ensure only safe fields are present
	safeDetails := make(map[string]interface{})
	allowedKeys := map[string]bool{
		"statement":         true,
		"statement_type":    true,
		"risk_level":        true,
		"database_name":     true,
		"environment":       true,
		"rows_affected":     true,
		"execution_time_ms": true,
		"success":           true,
		"error_message":     true,
		// Additional safe metadata
		"schema":    true,
		"table":     true,
		"operation": true,
		"user":      true,
	}

	for key, value := range details {
		if allowedKeys[key] {
			safeDetails[key] = value
		}
	}

	return safeDetails
}

// TruncateSQLForDisplay returns a truncated version of SQL suitable for display
// This is useful for UI where you want to show a preview without the full statement
func TruncateSQLForDisplay(sql string, maxLength int) string {
	if maxLength <= 0 {
		maxLength = 100
	}

	// Remove extra whitespace and newlines
	re := regexp.MustCompile(`\s+`)
	cleaned := re.ReplaceAllString(strings.TrimSpace(sql), " ")

	if len(cleaned) <= maxLength {
		return cleaned
	}

	return cleaned[:maxLength] + "..."
}
