package routes

import (
	"encoding/json"
	"net/http"
	"time"

	"pixerve/logger"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
}

// HealthHandler provides a health check endpoint for load balancers and monitoring
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
		Version:   "1.0.0", // TODO: Make this dynamic with build info
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
