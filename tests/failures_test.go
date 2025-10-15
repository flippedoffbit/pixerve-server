package tests

import (
	"errors"
	"pixerve/failures"
	"testing"
	"time"
)

func TestFailureStore(t *testing.T) {
	// Initialize failure store for testing
	testDBPath := "test_failures.db"
	defer func() {
		// Cleanup test database
		failures.Close()
		// Note: In real tests, you'd remove the test file
	}()

	err := failures.Init(testDBPath)
	if err != nil {
		t.Fatalf("Failed to initialize failure store: %v", err)
	}

	// Test storing a failure
	testHash := "test-hash-123"
	testError := errors.New("processing failed: invalid image format")
	testJobData := map[string]interface{}{
		"hash": "test-hash-123",
		"job": map[string]interface{}{
			"conversion_jobs": []map[string]interface{}{
				{
					"encoder": "jpg",
					"width":   800,
					"length":  600,
				},
			},
		},
	}

	err = failures.StoreFailure(testHash, testError, testJobData)
	if err != nil {
		t.Fatalf("Failed to store failure: %v", err)
	}

	// Test retrieving the failure
	record, err := failures.GetFailure(testHash)
	if err != nil {
		t.Fatalf("Failed to get failure: %v", err)
	}

	if record == nil {
		t.Fatal("Expected failure record, got nil")
	}

	if record.Hash != testHash {
		t.Errorf("Expected hash %s, got %s", testHash, record.Hash)
	}

	if record.Error != testError.Error() {
		t.Errorf("Expected error %s, got %s", testError.Error(), record.Error)
	}

	// Verify timestamp is recent
	if time.Since(record.Timestamp) > time.Minute {
		t.Error("Timestamp should be recent")
	}

	// Test getting non-existent failure
	nonExistentRecord, err := failures.GetFailure("non-existent-hash")
	if err != nil {
		t.Fatalf("Failed to get non-existent failure: %v", err)
	}

	if nonExistentRecord != nil {
		t.Error("Expected nil for non-existent failure")
	}

	// Test deleting failure
	err = failures.DeleteFailure(testHash)
	if err != nil {
		t.Fatalf("Failed to delete failure: %v", err)
	}

	// Verify it's deleted
	deletedRecord, err := failures.GetFailure(testHash)
	if err != nil {
		t.Fatalf("Failed to check deleted failure: %v", err)
	}

	if deletedRecord != nil {
		t.Error("Expected nil after deletion")
	}
}

func TestFailureList(t *testing.T) {
	// Initialize failure store for testing
	testDBPath := "test_failures_list.db"
	defer failures.Close()

	err := failures.Init(testDBPath)
	if err != nil {
		t.Fatalf("Failed to initialize failure store: %v", err)
	}

	// Store multiple failures
	hashes := []string{"hash1", "hash2", "hash3"}
	for _, hash := range hashes {
		testError := errors.New("test error for " + hash)
		testJobData := map[string]interface{}{"hash": hash}
		err = failures.StoreFailure(hash, testError, testJobData)
		if err != nil {
			t.Fatalf("Failed to store failure %s: %v", hash, err)
		}
	}

	// List all failures
	failuresList, err := failures.ListFailures()
	if err != nil {
		t.Fatalf("Failed to list failures: %v", err)
	}

	if len(failuresList) != len(hashes) {
		t.Errorf("Expected %d failures, got %d", len(hashes), len(failuresList))
	}

	// Verify all hashes are present
	hashMap := make(map[string]bool)
	for _, record := range failuresList {
		hashMap[record.Hash] = true
	}

	for _, hash := range hashes {
		if !hashMap[hash] {
			t.Errorf("Hash %s not found in failure list", hash)
		}
	}
}
