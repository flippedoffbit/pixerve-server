package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"pixerve/routes"
	"pixerve/success"
	"testing"
	"time"
)

func TestSuccessQueryHandler(t *testing.T) {
	// Initialize success store for testing
	testDBPath := "test_success_routes.db"
	defer func() {
		success.Close()
	}()

	err := success.Init(testDBPath)
	if err != nil {
		t.Fatalf("Failed to initialize success store: %v", err)
	}

	// Store a test success record
	testHash := "test-success-hash"
	testJobData := map[string]interface{}{
		"hash": testHash,
		"job": map[string]interface{}{
			"formats": map[string]interface{}{
				"jpg": map[string]interface{}{
					"settings": map[string]interface{}{"quality": 80.0},
				},
			},
		},
	}
	testFileCount := 2

	err = success.StoreSuccess(testHash, testJobData, testFileCount)
	if err != nil {
		t.Fatalf("Failed to store test success: %v", err)
	}

	// Test successful query
	req := httptest.NewRequest("GET", "/success?hash="+testHash, nil)
	w := httptest.NewRecorder()

	routes.SuccessQueryHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	if response["hash"] != testHash {
		t.Errorf("Expected hash %s, got %v", testHash, response["hash"])
	}

	if response["status"] != "success" {
		t.Errorf("Expected status 'success', got %v", response["status"])
	}

	if int(response["file_count"].(float64)) != testFileCount {
		t.Errorf("Expected file_count %d, got %v", testFileCount, response["file_count"])
	}

	// Verify timestamp exists and is reasonable
	timestampStr, ok := response["timestamp"].(string)
	if !ok {
		t.Error("Expected timestamp field in response")
	} else {
		// Parse timestamp to verify it's valid
		_, err := time.Parse(time.RFC3339, timestampStr)
		if err != nil {
			t.Errorf("Invalid timestamp format: %v", err)
		}
	}

	// Test query for non-existent hash
	req2 := httptest.NewRequest("GET", "/success?hash=non-existent-hash", nil)
	w2 := httptest.NewRecorder()

	routes.SuccessQueryHandler(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200 for non-existent hash, got %d", w2.Code)
	}

	var response2 map[string]interface{}
	err = json.Unmarshal(w2.Body.Bytes(), &response2)
	if err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	if response2["status"] != "not_found" {
		t.Errorf("Expected status 'not_found', got %v", response2["status"])
	}

	// Test missing hash parameter
	req3 := httptest.NewRequest("GET", "/success", nil)
	w3 := httptest.NewRecorder()

	routes.SuccessQueryHandler(w3, req3)

	if w3.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing hash, got %d", w3.Code)
	}

	// Test wrong HTTP method
	req4 := httptest.NewRequest("POST", "/success?hash="+testHash, nil)
	w4 := httptest.NewRecorder()

	routes.SuccessQueryHandler(w4, req4)

	if w4.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 for wrong method, got %d", w4.Code)
	}
}

func TestSuccessListHandler(t *testing.T) {
	// Initialize success store for testing
	testDBPath := "test_success_list_routes.db"
	defer func() {
		success.Close()
	}()

	err := success.Init(testDBPath)
	if err != nil {
		t.Fatalf("Failed to initialize success store: %v", err)
	}

	// Store multiple test success records
	testData := []struct {
		hash      string
		fileCount int
	}{
		{"list-test-hash-1", 3},
		{"list-test-hash-2", 1},
		{"list-test-hash-3", 5},
	}

	for _, data := range testData {
		jobData := map[string]interface{}{"test": data.hash}
		err := success.StoreSuccess(data.hash, jobData, data.fileCount)
		if err != nil {
			t.Fatalf("Failed to store test success %s: %v", data.hash, err)
		}
	}

	// Test list endpoint
	req := httptest.NewRequest("GET", "/success/list", nil)
	w := httptest.NewRecorder()

	routes.SuccessListHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	records, ok := response["success_records"].([]interface{})
	if !ok {
		t.Error("Expected success_records array in response")
	}

	if len(records) < len(testData) {
		t.Errorf("Expected at least %d records, got %d", len(testData), len(records))
	}

	count, ok := response["count"].(float64)
	if !ok {
		t.Error("Expected count field in response")
	}

	if int(count) != len(records) {
		t.Errorf("Count %d doesn't match records length %d", int(count), len(records))
	}

	// Test wrong HTTP method
	req2 := httptest.NewRequest("POST", "/success/list", nil)
	w2 := httptest.NewRecorder()

	routes.SuccessListHandler(w2, req2)

	if w2.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 for wrong method, got %d", w2.Code)
	}
}