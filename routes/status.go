package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"pixerve/job"
	"pixerve/logger"
)

// JobStatusResponse represents the job status response
type JobStatusResponse struct {
	Hash  string `json:"hash"`
	State string `json:"state"`
}

// JobStatusHandler returns the status of a job by hash
func JobStatusHandler(w http.ResponseWriter, r *http.Request) {
	logger.Debugf("Job status request: method=%s, remoteAddr=%s", r.Method, r.RemoteAddr)

	if r.Method != http.MethodGet {
		logger.Warnf("Invalid method for status endpoint: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	hash := r.URL.Query().Get("hash")
	if hash == "" {
		logger.Warn("Missing hash parameter in status request")
		http.Error(w, "Missing hash parameter", http.StatusBadRequest)
		return
	}

	logger.Debugf("Checking status for job: %s", hash)
	state, exists := job.GetJobState(hash)
	if !exists {
		logger.Warnf("Job not found: %s", hash)
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

	logger.Debugf("Job status: hash=%s, state=%s", hash, stateStr)

	response := JobStatusResponse{
		Hash:  hash,
		State: stateStr,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Errorf("Failed to encode status response: %v", err)
		return
	}

	logger.Debug("Job status request completed successfully")
}
