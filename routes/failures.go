package routes

import (
	"encoding/json"
	"net/http"
	"pixerve/failures"
	"pixerve/logger"
)

// FailureQueryHandler handles queries for processing failures
func FailureQueryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify JWT (optional - could be public or require auth)
	hash := r.URL.Query().Get("hash")
	if hash == "" {
		http.Error(w, "hash parameter required", http.StatusBadRequest)
		return
	}

	// Get failure record
	record, err := failures.GetFailure(hash)
	if err != nil {
		logger.Errorf("Failed to query failure for hash %s: %v", hash, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if record == nil {
		// No failure found - file processed successfully
		response := map[string]interface{}{
			"hash":    hash,
			"status":  "success",
			"message": "File processed successfully",
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Return failure details
	response := map[string]interface{}{
		"hash":      record.Hash,
		"status":    "failed",
		"timestamp": record.Timestamp,
		"error":     record.Error,
		"job_data":  record.JobData,
	}
	json.NewEncoder(w).Encode(response)
}

// FailureListHandler handles listing all failures (admin endpoint)
func FailureListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Optional: Add authentication check for admin access
	// For now, allowing public access

	failuresList, err := failures.ListFailures()
	if err != nil {
		logger.Errorf("Failed to list failures: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"failures": failuresList,
		"count":    len(failuresList),
	})
}
