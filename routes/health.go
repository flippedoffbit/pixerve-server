package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"pixerve/logger"
)

// Build-time variables (injected by ldflags)
var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	GoVersion string    `json:"go_version"`
	Uptime    string    `json:"uptime"`
	StartTime string    `json:"start_time"`
}

// Global start time for uptime calculation
var startTime = time.Now()

// formatUptime formats a duration into days, hours, minutes, seconds
func formatUptime(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
}

// HealthHandler provides a basic health check endpoint for load balancers and monitoring
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	logger.Debugf("Health check request: method=%s, remoteAddr=%s", r.Method, r.RemoteAddr)

	if r.Method != http.MethodGet {
		logger.Warnf("Invalid method for health endpoint: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   getVersion(),
		GoVersion: runtime.Version(),
		Uptime:    formatUptime(time.Since(startTime)),
		StartTime: startTime.Format("2006-01-02 15:04:05 MST"),
	}

	logger.Debugf("Health check response: status=%s, version=%s", response.Status, response.Version)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Errorf("Failed to encode health response: %v", err)
		return
	}

	logger.Debug("Health check completed successfully")
}

// getVersion returns the application version (injected at build time)
func getVersion() string {
	if version != "dev" {
		return version
	}
	return "dev" // Default for development
}
