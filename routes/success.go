package routes

import (
	"encoding/json"
	"net/http"
	"pixerve/logger"
	"pixerve/success"
)

// SuccessQueryHandler handles queries for successful processing
func SuccessQueryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	hash := r.URL.Query().Get("hash")
	if hash == "" {
		http.Error(w, "hash parameter required", http.StatusBadRequest)
		return
	}

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
		response := map[string]interface{}{
			"hash":    hash,
			"status":  "not_found",
			"message": "No success record found for this hash",
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Return success details
	response := map[string]interface{}{
		"hash":       record.Hash,
		"status":     "success",
		"timestamp":  record.Timestamp,
		"file_count": record.FileCount,
		"job_data":   record.JobData,
	}
	json.NewEncoder(w).Encode(response)
}

// SuccessListHandler handles listing all success records (admin endpoint)
func SuccessListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Add authentication check for admin access

	records, err := success.ListSuccessRecords()
	if err != nil {
		logger.Errorf("Failed to list success records: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success_records": records,
		"count":           len(records),
	})
}
