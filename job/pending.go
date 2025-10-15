package job

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"pixerve/logger"
)

var (
	pendingJobs []string // slice of directory paths with pending jobs
	mu          sync.RWMutex
)

// AddPendingJob adds a job directory to the pending list
func AddPendingJob(dir string) {
	mu.Lock()
	defer mu.Unlock()
	pendingJobs = append(pendingJobs, dir)
}

// RemovePendingJob removes a job directory from the pending list
func RemovePendingJob(dir string) {
	mu.Lock()
	defer mu.Unlock()
	for i, p := range pendingJobs {
		if p == dir {
			pendingJobs = append(pendingJobs[:i], pendingJobs[i+1:]...)
			break
		}
	}
}

// GetPendingJobs returns a copy of the pending jobs list
func GetPendingJobs() []string {
	mu.RLock()
	defer mu.RUnlock()
	// Return a copy to avoid race conditions
	jobs := make([]string, len(pendingJobs))
	copy(jobs, pendingJobs)
	return jobs
}

// ScanForPendingJobs scans the temp directory for job folders with instructions.json
func ScanForPendingJobs() error {
	tempDir := os.TempDir()
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dirPath := filepath.Join(tempDir, entry.Name())
		instrPath := filepath.Join(dirPath, "instructions.json")
		if _, err := os.Stat(instrPath); err == nil {
			// instructions.json exists, add to pending
			AddPendingJob(dirPath)
		}
	}
	return nil
}

// processJob processes a single job directory
func processJob(jobDir string) error {
	return ProcessJob(jobDir)
}

// ProcessPendingJobs runs in a loop processing pending jobs
func ProcessPendingJobs() {
	for {
		jobs := GetPendingJobs()
		if len(jobs) == 0 {
			time.Sleep(1 * time.Second) // Wait before checking again
			continue
		}
		logger.Infof("Processing %d pending jobs", len(jobs))

		for _, jobDir := range jobs {
			// Process the job
			if err := processJob(jobDir); err != nil {
				logger.Errorf("Failed to process job in %s: %v", jobDir, err)
			} else {
				// Remove from pending on success
				RemovePendingJob(jobDir)
				logger.Infof("Processed job in %s", jobDir)
			}
		}
	}
}
