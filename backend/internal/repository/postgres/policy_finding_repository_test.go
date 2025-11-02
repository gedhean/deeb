package postgres

import (
	"testing"
)

// TestPolicyFindingRepository is an integration test that requires a real database
// This test should be run with integration tags
func TestPolicyFindingRepository(t *testing.T) {
	t.Skip("Integration test - requires actual database connection. Run with -tags=integration")
}
