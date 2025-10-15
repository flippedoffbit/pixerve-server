package main

import (
	"fmt"
	"net/http"
	"pixerve/failures"
	"pixerve/job"
	"pixerve/logger"
	"pixerve/routes"

	pebble "github.com/cockroachdb/pebble"
)

func main() {
	// Initialize failure store
	if err := failures.Init("failures.db"); err != nil {
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

	db, err := pebble.Open("demo", &pebble.Options{})
	if err != nil {
		logger.Fatal(err)
	}
	key := []byte("hello")
	if err := db.Set(key, []byte("world"), pebble.Sync); err != nil {
		logger.Fatal(err)
	}
	value, closer, err := db.Get(key)
	if err != nil {
		logger.Fatal(err)
	}
	fmt.Printf("%s %s\n", key, value)
	if err := closer.Close(); err != nil {
		logger.Fatal(err)
	}
	if err := db.Close(); err != nil {
		logger.Fatal(err)
	}

}
