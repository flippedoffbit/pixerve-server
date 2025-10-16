package tests

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMain is the main test function that runs before and after all tests
// It automatically cleans up test database directories after all tests complete
func TestMain(m *testing.M) {
	// Run all tests
	code := m.Run()

	// Cleanup test database files after all tests complete
	cleanupTestDatabases()

	// Exit with the test result code
	os.Exit(code)
}

// cleanupTestDatabases removes all test database files
func cleanupTestDatabases() {
	// Get current directory (tests directory)
	testDir := "."

	// Find all files matching test_*.db pattern
	pattern := filepath.Join(testDir, "test_*.db")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		// If glob fails, just return (don't fail the tests)
		return
	}

	// Remove each test database directory (they contain multiple files)
	for _, dir := range matches {
		if err := os.RemoveAll(dir); err != nil {
			// Log error but don't fail - cleanup is best effort
			// We can't use t.Logf here since we're not in a test context
			continue
		}
	}
}
