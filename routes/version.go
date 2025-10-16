package routes

import (
	"encoding/json"
	"net/http"
	"time"

	"pixerve/logger"
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
	logger.Debugf("Version request: method=%s, remoteAddr=%s", r.Method, r.RemoteAddr)

	if r.Method != http.MethodGet {
		logger.Warnf("Invalid method for version endpoint: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := VersionResponse{
		Version:   "1.0.0", // TODO: Make this dynamic with build info
		BuildTime: time.Now().Format(time.RFC3339),
		GoVersion: "1.21", // TODO: Make this dynamic
		GitCommit: "dev",  // TODO: Make this dynamic with git info
	}

	logger.Debugf("Version response: version=%s, go_version=%s, git_commit=%s",
		response.Version, response.GoVersion, response.GitCommit)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Errorf("Failed to encode version response: %v", err)
		return
	}

	logger.Debug("Version request completed successfully")
}
