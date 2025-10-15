package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"pixerve/job"
)

// JobStatusResponse represents the job status response
type JobStatusResponse struct {
	Hash  string `json:"hash"`
	State string `json:"state"`
}

// JobStatusHandler returns the status of a job by hash
func JobStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	hash := r.URL.Query().Get("hash")
	if hash == "" {
		http.Error(w, "Missing hash parameter", http.StatusBadRequest)
		return
	}

	state, exists := job.GetJobState(hash)
	if !exists {
		http.Error(w, fmt.Sprintf("Job with hash %s not found", hash), http.StatusNotFound)
		return
	}

	var stateStr string
	switch state {
	case job.JobStatePending:
		stateStr = "pending"
	case job.JobStateProcessing:
		stateStr = "processing"
	case job.JobStateCompleted:
		stateStr = "completed"
	case job.JobStateFailed:
		stateStr = "failed"
	case job.JobStateCancelled:
		stateStr = "cancelled"
	default:
		stateStr = "unknown"
	}

	response := JobStatusResponse{
		Hash:  hash,
		State: stateStr,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}