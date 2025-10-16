package routes

import (
	"encoding/json"
	"net/http"
	"pixerve/failures"
	"pixerve/logger"
)

// FailureQueryHandler handles queries for processing failures
func FailureQueryHandler(w http.ResponseWriter, r *http.Request) {
	logger.Debugf("Failure query request: method=%s, remoteAddr=%s", r.Method, r.RemoteAddr)

	if r.Method != http.MethodGet {
		logger.Warnf("Invalid method for failure query endpoint: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify JWT (optional - could be public or require auth)
	hash := r.URL.Query().Get("hash")
	if hash == "" {
		logger.Warn("Missing hash parameter in failure query request")
		http.Error(w, "hash parameter required", http.StatusBadRequest)
		return
	}

	logger.Debugf("Querying failure record for hash: %s", hash)

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
		logger.Debugf("No failure record found for hash: %s (processed successfully)", hash)
		response := map[string]interface{}{
			"hash":    hash,
			"status":  "success",
			"message": "File processed successfully",
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			logger.Errorf("Failed to encode success response: %v", err)
			return
		}
		logger.Debug("Failure query completed - no failure found")
		return
	}

	// Return failure details
	logger.Infof("Failure record found: hash=%s, error=%s", record.Hash, record.Error)
	response := map[string]interface{}{
		"hash":      record.Hash,
		"status":    "failed",
		"timestamp": record.Timestamp,
		"error":     record.Error,
		"job_data":  record.JobData,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Errorf("Failed to encode failure response: %v", err)
		return
	}
	logger.Debug("Failure query completed successfully")
}

// FailureListHandler handles listing all failures (admin endpoint)
func FailureListHandler(w http.ResponseWriter, r *http.Request) {
	logger.Debugf("Failure list request: method=%s, remoteAddr=%s", r.Method, r.RemoteAddr)

	if r.Method != http.MethodGet {
		logger.Warnf("Invalid method for failure list endpoint: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Optional: Add authentication check for admin access
	// For now, allowing public access
	logger.Debug("Listing all failure records")

	failuresList, err := failures.ListFailures()
	if err != nil {
		logger.Errorf("Failed to list failures: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	logger.Infof("Retrieved %d failure records", len(failuresList))

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"failures": failuresList,
		"count":    len(failuresList),
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Errorf("Failed to encode failure list response: %v", err)
		return
	}
	logger.Debug("Failure list request completed successfully")
}
