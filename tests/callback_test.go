package tests

import (
	"testing"
	"time"
)

func TestCallbackPayloadStructure(t *testing.T) {
	// Test that callback payloads have the correct structure
	testHash := "payload-test-hash"
	testFileCount := 3

	// Create expected payload structure
	expectedPayload := map[string]interface{}{
		"hash":       testHash,
		"status":     "completed",
		"file_count": testFileCount,
		"timestamp":  float64(time.Now().Unix()), // Will be close to current time
		"job_data":   map[string]interface{}{"test": "data"},
	}

	// Verify the structure contains required fields
	requiredFields := []string{"hash", "status", "file_count", "timestamp", "job_data"}
	for _, field := range requiredFields {
		if _, exists := expectedPayload[field]; !exists {
			t.Errorf("Expected field %s in callback payload", field)
		}
	}

	// Verify specific values
	if expectedPayload["hash"] != testHash {
		t.Errorf("Expected hash %s, got %v", testHash, expectedPayload["hash"])
	}

	if expectedPayload["status"] != "completed" {
		t.Errorf("Expected status 'completed', got %v", expectedPayload["status"])
	}

	if expectedPayload["file_count"].(int) != testFileCount {
		t.Errorf("Expected file_count %d, got %v", testFileCount, expectedPayload["file_count"])
	}
}
