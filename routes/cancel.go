package routes

import (
	"fmt"
	"net/http"
	"pixerve/job"
	"pixerve/logger"
)

// CancelJobHandler cancels a running job by hash
func CancelJobHandler(w http.ResponseWriter, r *http.Request) {
	logger.Debugf("Cancel job request: method=%s, remoteAddr=%s", r.Method, r.RemoteAddr)

	if r.Method != http.MethodDelete {
		logger.Warnf("Invalid method for cancel endpoint: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	hash := r.URL.Query().Get("hash")
	if hash == "" {
		logger.Warn("Missing hash parameter in cancel request")
		http.Error(w, "Missing hash parameter", http.StatusBadRequest)
		return
	}

	logger.Infof("Attempting to cancel job: %s", hash)
	if err := job.CancelJob(hash); err != nil {
		logger.Errorf("Failed to cancel job %s: %v", hash, err)
		// Return appropriate status based on error
		if err.Error() == "job with hash "+hash+" not found" {
			http.Error(w, fmt.Sprintf("Job not found: %v", err), http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Cannot cancel job: %v", err), http.StatusConflict)
		}
		return
	}

	logger.Infof("Job cancelled successfully: %s", hash)
	w.WriteHeader(http.StatusNoContent)
}
