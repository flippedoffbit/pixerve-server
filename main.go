package main

import (
	"context"
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
	logger.Info("Starting Pixerve server initialization")

	// Initialize credentials store
	logger.Debug("Initializing credentials database")
	if err := credentials.OpenDB(config.GetCredentialsDBPath()); err != nil {
		logger.Fatalf("Failed to initialize credentials store: %v", err)
	}
	defer credentials.CloseDB()
	logger.Info("Credentials database initialized successfully")

	// Initialize failure store
	logger.Debug("Initializing failures database")
	if err := failures.Init(config.GetFailuresDBPath()); err != nil {
		logger.Fatalf("Failed to initialize failure store: %v", err)
	}
	defer failures.Close()
	logger.Info("Failures database initialized successfully")

	// Initialize success store
	logger.Debug("Initializing success database")
	if err := success.Init(config.GetSuccessDBPath()); err != nil {
		logger.Fatalf("Failed to initialize success store: %v", err)
	}
	defer success.Close()
	logger.Info("Success database initialized successfully")

	// Scan for pending jobs on startup
	logger.Info("Scanning for pending jobs on startup")
	if err := job.ScanForPendingJobs(); err != nil {
		logger.Errorf("Failed to scan for pending jobs: %v", err)
		// Don't exit - continue with server startup
	} else {
		logger.Info("Pending jobs scan completed")
	}

	// Start cleanup routine for old logs (runs every 24 hours)
	logger.Info("Starting cleanup routine (runs every 24 hours)")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // This will stop the cleanup routine when main exits
	go cleanupRoutine(ctx)

	// Start job processing routine
	logger.Info("Starting job processing routine")
	go job.ProcessPendingJobs()

	// Register HTTP routes
	logger.Info("Registering HTTP routes")
	http.HandleFunc("/upload", routes.UploadHandler)
	http.HandleFunc("/health", routes.HealthHandler)
	http.HandleFunc("/version", routes.VersionHandler)
	http.HandleFunc("/status", routes.JobStatusHandler)
	http.HandleFunc("/cancel", routes.CancelJobHandler)
	http.HandleFunc("/failures", routes.FailureQueryHandler)
	http.HandleFunc("/failures/list", routes.FailureListHandler)
	http.HandleFunc("/success", routes.SuccessQueryHandler)
	http.HandleFunc("/success/list", routes.SuccessListHandler)
	logger.Info("HTTP routes registered successfully")

	logger.Infof("Pixerve server starting on port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Fatalf("Server failed to start: %v", err)
	}
}

// cleanupRoutine periodically cleans up old success and failure records
func cleanupRoutine(ctx context.Context) {
	logger.Info("Cleanup routine started - will run every 24 hours")
	ticker := time.NewTicker(24 * time.Hour) // Run every 24 hours
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Cleanup routine stopped due to context cancellation")
			return
		case <-ticker.C:
			logger.Info("Running scheduled cleanup of old records")
			// Clean up records older than 30 days
			maxAge := 30 * 24 * time.Hour

			logger.Debugf("Cleaning up success records older than %v", maxAge)
			if err := success.CleanupOldRecords(maxAge); err != nil {
				logger.Errorf("Failed to cleanup old success records: %v", err)
			} else {
				logger.Info("Successfully cleaned up old success records")
			}

			logger.Debugf("Cleaning up failure records older than %v", maxAge)
			if err := failures.CleanupOldRecords(maxAge); err != nil {
				logger.Errorf("Failed to cleanup old failure records: %v", err)
			} else {
				logger.Info("Successfully cleaned up old failure records")
			}

			logger.Info("Scheduled cleanup completed")
		}
	}
}
