package routes

import (
	"encoding/json"
	"net/http"
	"time"
)

// VersionResponse represents the version information response
type VersionResponse struct {
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
	GitCommit string `json:"git_commit,omitempty"`
}

// VersionHandler provides version information about the build
func VersionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := VersionResponse{
		Version:   "1.0.0", // TODO: Make this dynamic with build info
		BuildTime: time.Now().Format(time.RFC3339),
		GoVersion: "1.21", // TODO: Make this dynamic
		GitCommit: "dev",  // TODO: Make this dynamic with git info
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
