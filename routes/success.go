package routes

import (
	"encoding/json"
	"net/http"
	"pixerve/logger"
	"pixerve/success"
)

// SuccessQueryHandler handles queries for successful processing
func SuccessQueryHandler(w http.ResponseWriter, r *http.Request) {
	logger.Debugf("Success query request: method=%s, remoteAddr=%s", r.Method, r.RemoteAddr)

	if r.Method != http.MethodGet {
		logger.Warnf("Invalid method for success query endpoint: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	hash := r.URL.Query().Get("hash")
	if hash == "" {
		logger.Warn("Missing hash parameter in success query request")
		http.Error(w, "hash parameter required", http.StatusBadRequest)
		return
	}

	logger.Debugf("Querying success record for hash: %s", hash)

	// Get success record
	record, err := success.GetSuccess(hash)
	if err != nil {
		logger.Errorf("Failed to query success for hash %s: %v", hash, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if record == nil {
		// No success record found
		logger.Debugf("No success record found for hash: %s", hash)
		response := map[string]interface{}{
			"hash":    hash,
			"status":  "not_found",
			"message": "No success record found for this hash",
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			logger.Errorf("Failed to encode not_found response: %v", err)
			return
		}
		logger.Debug("Success query completed - record not found")
		return
	}

	// Return success details
	logger.Infof("Success record found: hash=%s, file_count=%d", record.Hash, record.FileCount)
	response := map[string]interface{}{
		"hash":       record.Hash,
		"status":     "success",
		"timestamp":  record.Timestamp,
		"file_count": record.FileCount,
		"job_data":   record.JobData,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Errorf("Failed to encode success response: %v", err)
		return
	}
	logger.Debug("Success query completed successfully")
}

// SuccessListHandler handles listing all success records (admin endpoint)
func SuccessListHandler(w http.ResponseWriter, r *http.Request) {
	logger.Debugf("Success list request: method=%s, remoteAddr=%s", r.Method, r.RemoteAddr)

	if r.Method != http.MethodGet {
		logger.Warnf("Invalid method for success list endpoint: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Add authentication check for admin access
	logger.Debug("Listing all success records")

	records, err := success.ListSuccessRecords()
	if err != nil {
		logger.Errorf("Failed to list success records: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	logger.Infof("Retrieved %d success records", len(records))

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"success_records": records,
		"count":           len(records),
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Errorf("Failed to encode success list response: %v", err)
		return
	}
	logger.Debug("Success list request completed successfully")
}
