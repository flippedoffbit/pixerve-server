package main

import (
	"net/http"
	"pixerve/config"
	"pixerve/credentials"
	"pixerve/failures"
	"pixerve/job"
	"pixerve/logger"
	"pixerve/routes"
	"pixerve/success"
	"time"
)

func main() {
	// Initialize credentials store
	if err := credentials.OpenDB(config.GetCredentialsDBPath()); err != nil {
		logger.Fatalf("Failed to initialize credentials store: %v", err)
	}
	defer credentials.CloseDB()

	// Initialize failure store
	if err := failures.Init(config.GetFailuresDBPath()); err != nil {
		logger.Fatalf("Failed to initialize failure store: %v", err)
	}
	defer failures.Close()

	// Initialize success store
	if err := success.Init(config.GetSuccessDBPath()); err != nil {
		logger.Fatalf("Failed to initialize success store: %v", err)
	}
	defer success.Close()

	// Scan for pending jobs on startup
	job.ScanForPendingJobs()

	// Start cleanup routine for old logs (runs every 24 hours)
	go cleanupRoutine()

	http.HandleFunc("/upload", routes.UploadHandler)
	http.HandleFunc("/health", routes.HealthHandler)
	http.HandleFunc("/version", routes.VersionHandler)
	http.HandleFunc("/failures", routes.FailureQueryHandler)
	http.HandleFunc("/failures/list", routes.FailureListHandler)
	http.HandleFunc("/success", routes.SuccessQueryHandler)
	http.HandleFunc("/success/list", routes.SuccessListHandler)
	http.ListenAndServe(":8080", nil)
}

// cleanupRoutine periodically cleans up old success and failure records
func cleanupRoutine() {
	ticker := time.NewTicker(24 * time.Hour) // Run every 24 hours
	defer ticker.Stop()

	for range ticker.C {
		// Clean up records older than 30 days
		maxAge := 30 * 24 * time.Hour

		if err := success.CleanupOldRecords(maxAge); err != nil {
			logger.Errorf("Failed to cleanup old success records: %v", err)
		} else {
			logger.Infof("Cleaned up old success records older than %v", maxAge)
		}

		if err := failures.CleanupOldRecords(maxAge); err != nil {
			logger.Errorf("Failed to cleanup old failure records: %v", err)
		} else {
			logger.Infof("Cleaned up old failure records older than %v", maxAge)
		}
	}
}
