package routes

import (
	"encoding/json"
	"net/http"
	"os"
	"runtime"

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
		Version:   getVersionFromEnv(),
		BuildTime: getBuildTimeFromEnv(),
		GoVersion: runtime.Version(),
		GitCommit: getGitCommitFromEnv(),
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

// getVersionFromEnv returns the application version from package variables or environment
func getVersionFromEnv() string {
	if version != "dev" {
		return version
	}
	if envVersion := os.Getenv("PIXERVE_VERSION"); envVersion != "" {
		return envVersion
	}
	return "dev" // Default for development
}

// getBuildTimeFromEnv returns the build time from package variables or environment
func getBuildTimeFromEnv() string {
	if buildTime != "unknown" {
		return buildTime
	}
	if envBuildTime := os.Getenv("PIXERVE_BUILD_TIME"); envBuildTime != "" {
		return envBuildTime
	}
	return "unknown" // Default for development
}

// getGitCommitFromEnv returns the git commit hash from package variables or environment
func getGitCommitFromEnv() string {
	if gitCommit != "unknown" {
		return gitCommit
	}
	if envGitCommit := os.Getenv("PIXERVE_GIT_COMMIT"); envGitCommit != "" {
		return envGitCommit
	}
	return "unknown" // Default for development
}
