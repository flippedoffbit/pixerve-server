package tests

import (
	"pixerve/success"
	"testing"
	"time"
)

func TestSuccessStore(t *testing.T) {
	// Initialize success store for testing
	testDBPath := "test_success.db"
	defer func() {
		// Cleanup test database
		success.Close()
		// Note: In real tests, you'd remove the test file
	}()

	err := success.Init(testDBPath)
	if err != nil {
		t.Fatalf("Failed to initialize success store: %v", err)
	}

	// Test storing a success
	testHash := "test-hash-123"
	testJobData := map[string]interface{}{
		"conversionJobs": []map[string]interface{}{
			{"encoder": "jpg", "length": 800, "width": 600, "quality": 80, "speed": 4},
		},
		"writerJobs": []map[string]interface{}{
			{"type": "s3", "credentials": map[string]string{"bucket": "test-bucket"}},
		},
		"callbackURL":     "https://example.com/callback",
		"callbackHeaders": map[string]string{"Authorization": "Bearer token"},
		"priority":        1,
		"keepOriginal":    true,
		"subDir":          "test-dir",
	}
	testFileCount := 3

	err = success.StoreSuccess(testHash, testJobData, testFileCount)
	if err != nil {
		t.Fatalf("Failed to store success: %v", err)
	}

	// Test retrieving the success
	record, err := success.GetSuccess(testHash)
	if err != nil {
		t.Fatalf("Failed to get success: %v", err)
	}

	if record == nil {
		t.Fatal("Expected success record, got nil")
	}

	if record.Hash != testHash {
		t.Errorf("Expected hash %s, got %s", testHash, record.Hash)
	}

	if record.FileCount != testFileCount {
		t.Errorf("Expected file count %d, got %d", testFileCount, record.FileCount)
	}

	// Verify timestamp is recent (within last minute)
	if time.Since(record.Timestamp) > time.Minute {
		t.Errorf("Timestamp seems too old: %v", record.Timestamp)
	}

	// Test getting non-existent success
	nonExistentRecord, err := success.GetSuccess("non-existent-hash")
	if err != nil {
		t.Fatalf("Failed to get non-existent success: %v", err)
	}
	if nonExistentRecord != nil {
		t.Error("Expected nil for non-existent success record")
	}
}

func TestSuccessStoreList(t *testing.T) {
	// Initialize success store for testing
	testDBPath := "test_success_list.db"
	defer func() {
		success.Close()
	}()

	err := success.Init(testDBPath)
	if err != nil {
		t.Fatalf("Failed to initialize success store: %v", err)
	}

	// Store multiple successes
	testData := []struct {
		hash      string
		fileCount int
	}{
		{"hash1", 2},
		{"hash2", 5},
		{"hash3", 1},
	}

	for _, data := range testData {
		jobData := map[string]interface{}{"test": data.hash}
		err := success.StoreSuccess(data.hash, jobData, data.fileCount)
		if err != nil {
			t.Fatalf("Failed to store success %s: %v", data.hash, err)
		}
	}

	// Test listing all successes
	records, err := success.ListSuccessRecords()
	if err != nil {
		t.Fatalf("Failed to list success records: %v", err)
	}

	if len(records) < len(testData) {
		t.Errorf("Expected at least %d records, got %d", len(testData), len(records))
	}

	// Verify we can find our test records
	foundHashes := make(map[string]bool)
	for _, record := range records {
		foundHashes[record.Hash] = true
	}

	for _, data := range testData {
		if !foundHashes[data.hash] {
			t.Errorf("Expected to find hash %s in records", data.hash)
		}
	}
}

func TestSuccessStoreCleanup(t *testing.T) {
	// Initialize success store for testing
	testDBPath := "test_success_cleanup.db"
	defer func() {
		success.Close()
	}()

	err := success.Init(testDBPath)
	if err != nil {
		t.Fatalf("Failed to initialize success store: %v", err)
	}

	// Store a success
	testHash := "cleanup-test-hash"
	jobData := map[string]interface{}{"test": "cleanup"}
	err = success.StoreSuccess(testHash, jobData, 1)
	if err != nil {
		t.Fatalf("Failed to store success: %v", err)
	}

	// Verify it exists
	record, err := success.GetSuccess(testHash)
	if err != nil {
		t.Fatalf("Failed to get success: %v", err)
	}
	if record == nil {
		t.Fatal("Expected success record to exist")
	}

	// Try cleanup with very old cutoff (should not remove recent record)
	oldCutoff := time.Hour * 24 * 365 // 1 year ago
	err = success.CleanupOldRecords(oldCutoff)
	if err != nil {
		t.Fatalf("Failed to cleanup old records: %v", err)
	}

	// Verify record still exists
	record, err = success.GetSuccess(testHash)
	if err != nil {
		t.Fatalf("Failed to get success after cleanup: %v", err)
	}
	if record == nil {
		t.Fatal("Expected success record to still exist after old cleanup")
	}

	// Try cleanup with very recent cutoff (should remove record)
	recentCutoff := time.Nanosecond * 1 // 1 nanosecond ago
	err = success.CleanupOldRecords(recentCutoff)
	if err != nil {
		t.Fatalf("Failed to cleanup recent records: %v", err)
	}

	// Verify record is gone
	record, err = success.GetSuccess(testHash)
	if err != nil {
		t.Fatalf("Failed to get success after recent cleanup: %v", err)
	}
	if record != nil {
		t.Fatal("Expected success record to be removed after recent cleanup")
	}
}

func TestSuccessStoreDelete(t *testing.T) {
	// Initialize success store for testing
	testDBPath := "test_success_delete.db"
	defer func() {
		success.Close()
	}()

	err := success.Init(testDBPath)
	if err != nil {
		t.Fatalf("Failed to initialize success store: %v", err)
	}

	// Store a success
	testHash := "delete-test-hash"
	jobData := map[string]interface{}{"test": "delete"}
	err = success.StoreSuccess(testHash, jobData, 1)
	if err != nil {
		t.Fatalf("Failed to store success: %v", err)
	}

	// Verify it exists
	record, err := success.GetSuccess(testHash)
	if err != nil {
		t.Fatalf("Failed to get success: %v", err)
	}
	if record == nil {
		t.Fatal("Expected success record to exist")
	}

	// Delete the record
	err = success.DeleteSuccess(testHash)
	if err != nil {
		t.Fatalf("Failed to delete success: %v", err)
	}

	// Verify it's gone
	record, err = success.GetSuccess(testHash)
	if err != nil {
		t.Fatalf("Failed to get success after delete: %v", err)
	}
	if record != nil {
		t.Fatal("Expected success record to be deleted")
	}
}
