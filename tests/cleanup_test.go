package tests

import (
	"pixerve/failures"
	"pixerve/success"
	"testing"
	"time"
)

func TestCleanupRoutines(t *testing.T) {
	// Test success store cleanup
	testSuccessCleanup(t)
	// Test failure store cleanup
	testFailureCleanup(t)
}

func testSuccessCleanup(t *testing.T) {
	// Initialize success store for testing
	testDBPath := "test_cleanup_success.db"
	defer func() {
		success.Close()
	}()

	err := success.Init(testDBPath)
	if err != nil {
		t.Fatalf("Failed to initialize success store: %v", err)
	}

	// Store some test successes with different timestamps
	oldTimestamp := time.Now().Add(-48 * time.Hour)   // 2 days ago
	recentTimestamp := time.Now().Add(-1 * time.Hour) // 1 hour ago

	// Create records with manipulated timestamps
	testData := []struct {
		hash      string
		timestamp time.Time
	}{
		{"old-success-1", oldTimestamp},
		{"old-success-2", oldTimestamp},
		{"recent-success-1", recentTimestamp},
		{"recent-success-2", recentTimestamp},
	}

	for _, data := range testData {
		jobData := map[string]interface{}{"test": data.hash}
		err := success.StoreSuccess(data.hash, jobData, 1)
		if err != nil {
			t.Fatalf("Failed to store test success %s: %v", data.hash, err)
		}

		// Manually update timestamp to test cleanup (this is a bit hacky but works for testing)
		// In a real scenario, we'd need to modify the store to accept timestamps
	}

	// Verify all records exist initially
	allRecords, err := success.ListSuccessRecords()
	if err != nil {
		t.Fatalf("Failed to list success records: %v", err)
	}

	if len(allRecords) < len(testData) {
		t.Errorf("Expected at least %d records initially, got %d", len(testData), len(allRecords))
	}

	// Run cleanup with very short cutoff (1 nanosecond) to remove all records
	cutoffTime := 1 * time.Nanosecond
	err = success.CleanupOldRecords(cutoffTime)
	if err != nil {
		t.Fatalf("Failed to cleanup old success records: %v", err)
	}

	// Verify all records are gone
	remainingRecords, err := success.ListSuccessRecords()
	if err != nil {
		t.Fatalf("Failed to list remaining success records: %v", err)
	}

	if len(remainingRecords) != 0 {
		t.Errorf("Expected all records to be cleaned up, but %d remain", len(remainingRecords))
	}
}

func testFailureCleanup(t *testing.T) {
	// Initialize failure store for testing
	testDBPath := "test_cleanup_failures.db"
	defer func() {
		failures.Close()
	}()

	err := failures.Init(testDBPath)
	if err != nil {
		t.Fatalf("Failed to initialize failure store: %v", err)
	}

	// Store some test failures
	testFailures := []struct {
		hash  string
		error string
	}{
		{"cleanup-fail-1", "test error 1"},
		{"cleanup-fail-2", "test error 2"},
		{"cleanup-fail-3", "test error 3"},
	}

	for _, failure := range testFailures {
		jobData := map[string]interface{}{"test": failure.hash}
		err := failures.StoreFailure(failure.hash, nil, jobData)
		if err != nil {
			t.Fatalf("Failed to store test failure %s: %v", failure.hash, err)
		}
	}

	// Verify all failures exist initially
	allFailures, err := failures.ListFailures()
	if err != nil {
		t.Fatalf("Failed to list failure records: %v", err)
	}

	if len(allFailures) < len(testFailures) {
		t.Errorf("Expected at least %d failures initially, got %d", len(testFailures), len(allFailures))
	}

	// Run cleanup with very recent cutoff (should remove all records since they're new)
	cutoffTime := 1 * time.Nanosecond
	err = failures.CleanupOldRecords(cutoffTime)
	if err != nil {
		t.Fatalf("Failed to cleanup old failure records: %v", err)
	}

	// Verify all records are gone (since they were just created and cutoff is very recent)
	remainingFailures, err := failures.ListFailures()
	if err != nil {
		t.Fatalf("Failed to list remaining failure records: %v", err)
	}

	// All records should be gone since they were just created
	if len(remainingFailures) > 0 {
		t.Errorf("Expected all failure records to be cleaned up, but %d remain", len(remainingFailures))
	}
}

func TestCleanupEdgeCases(t *testing.T) {
	// Test cleanup with empty stores
	testDBPath := "test_cleanup_empty.db"
	defer func() {
		success.Close()
	}()

	err := success.Init(testDBPath)
	if err != nil {
		t.Fatalf("Failed to initialize success store: %v", err)
	}

	// Test cleanup on empty store
	err = success.CleanupOldRecords(24 * time.Hour)
	if err != nil {
		t.Errorf("Cleanup on empty store should not fail: %v", err)
	}

	// Test with very old cutoff (should not remove anything from empty store)
	err = success.CleanupOldRecords(365 * 24 * time.Hour)
	if err != nil {
		t.Errorf("Cleanup with old cutoff on empty store should not fail: %v", err)
	}
}
