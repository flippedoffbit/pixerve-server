package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"pixerve/config"
	"pixerve/credentials"
	"pixerve/failures"
	"pixerve/job"
	"pixerve/logger"
	"pixerve/routes"
	"pixerve/success"
	"pixerve/taskqueue"
	"syscall"
	"time"
)

// main is the entry point for the Pixerve image processing server.
// It performs the following initialization steps:
// 1. Initializes all database stores (credentials, failures, success)
// 2. Opens the task queue for async job processing
// 3. Scans for any pending jobs from previous runs
// 4. Starts background cleanup and job processing routines
// 5. Registers HTTP routes for the REST API
// 6. Sets up file serving for processed images
// 7. Starts the HTTP server with graceful shutdown handling
//
// The server provides endpoints for:
// - Image upload and processing (/upload)
// - Health checks (/health)
// - Job status monitoring (/status, /cancel)
// - Success/failure tracking (/success, /failures)
// - Direct file serving (/files/)
//
// Environment variables:
// - PIXERVE_DATA_DIR: Custom data directory (default: ./data)
// - PIXERVE_SERVE_DIR: Custom serve directory (default: ./serve)
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

	// Initialize task queue
	logger.Debug("Initializing task queue")
	if err := taskqueue.OpenConvertQueueDB(); err != nil {
		logger.Fatalf("Failed to initialize task queue: %v", err)
	}
	defer taskqueue.ConvertQueue.Close()
	logger.Info("Task queue initialized successfully")

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

	// Serve static files from direct serve directory
	serveDir := config.GetDirectServeBaseDir()
	logger.Infof("Setting up file server for direct serve directory: %s", serveDir)
	http.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir(serveDir))))

	logger.Info("HTTP routes registered successfully")

	logger.Infof("Pixerve server starting on port 8080")

	// Create HTTP server with timeouts for graceful shutdown
	server := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Channel to listen for interrupt signal
	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)

	// Register for interrupt signals
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		logger.Info("HTTP server started, listening on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-quit
	logger.Info("Received shutdown signal, initiating graceful shutdown...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Gracefully shutdown the server
	logger.Info("Stopping HTTP server...")
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Errorf("Server forced to shutdown: %v", err)
	} else {
		logger.Info("HTTP server stopped gracefully")
	}

	// Stop cleanup routine
	logger.Info("Stopping cleanup routine...")
	cancel() // This will stop the cleanup routine

	// Wait for cleanup to complete
	time.Sleep(2 * time.Second)

	// Close databases (defer statements will handle this)
	logger.Info("Closing database connections...")

	close(done)
	logger.Info("Pixerve server shutdown complete")
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
