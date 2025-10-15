package routes

import (
	"fmt"
	"net/http"
	"pixerve/job"
)

// CancelJobHandler cancels a running job by hash
func CancelJobHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	hash := r.URL.Query().Get("hash")
	if hash == "" {
		http.Error(w, "Missing hash parameter", http.StatusBadRequest)
		return
	}

	if err := job.CancelJob(hash); err != nil {
		http.Error(w, fmt.Sprintf("Failed to cancel job: %v", err), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}