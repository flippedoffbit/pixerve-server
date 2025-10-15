package main

import (
	"net/http"
	"pixerve/config"
	"pixerve/credentials"
	"pixerve/failures"
	"pixerve/job"
	"pixerve/logger"
	"pixerve/routes"
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

	// Scan for pending jobs on startup
	job.ScanForPendingJobs()

	// Start job processing loop
	go job.ProcessPendingJobs()

	http.HandleFunc("/upload", routes.UploadHandler)
	http.HandleFunc("/failures", routes.FailureQueryHandler)
	http.HandleFunc("/failures/list", routes.FailureListHandler)
	http.ListenAndServe(":8080", nil)
}
