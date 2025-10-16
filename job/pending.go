package job

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"pixerve/logger"
)

// JobState represents the current state of a job
type JobState int

const (
	JobStatePending JobState = iota
	JobStateProcessing
	JobStateCompleted
	JobStateFailed
	JobStateCancelled
)

var (
	pendingJobs []string                              // slice of directory paths with pending jobs
	activeJobs  = make(map[string]context.CancelFunc) // hash -> cancel function
	jobStates   = make(map[string]JobState)           // hash -> job state
	mu          sync.RWMutex
)

// getMaxWorkers returns the maximum number of concurrent workers for job processing.
// Configurable via PIXERVE_MAX_WORKERS environment variable.
// Defaults to runtime.NumCPU() - 1 (minimum 1) to utilize available cores while leaving one for system processes.
// Values are clamped between 1 and 10 to prevent resource exhaustion.
func getMaxWorkers() int {
	const maxWorkersLimit = 10
	const minWorkers = 1

	// Default to NumCPU - 1, minimum 1
	defaultWorkers := runtime.NumCPU() - 1
	if defaultWorkers < minWorkers {
		defaultWorkers = minWorkers
	}

	if env := os.Getenv("PIXERVE_MAX_WORKERS"); env != "" {
		if workers, err := strconv.Atoi(env); err == nil {
			if workers < minWorkers {
				return minWorkers
			}
			if workers > maxWorkersLimit {
				return maxWorkersLimit
			}
			return workers
		}
	}
	return defaultWorkers
}

// AddPendingJob adds a job directory to the pending list
func AddPendingJob(dir string) {
	hash := filepath.Base(dir)
	mu.Lock()
	defer mu.Unlock()
	pendingJobs = append(pendingJobs, dir)
	jobStates[hash] = JobStatePending
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

// CancelJob cancels a running job by hash
func CancelJob(hash string) error {
	mu.Lock()
	defer mu.Unlock()

	state, exists := jobStates[hash]
	if !exists {
		return fmt.Errorf("job with hash %s not found", hash)
	}

	switch state {
	case JobStateCompleted:
		return fmt.Errorf("job with hash %s is already completed", hash)
	case JobStateFailed:
		return fmt.Errorf("job with hash %s has already failed", hash)
	case JobStateCancelled:
		return fmt.Errorf("job with hash %s is already cancelled", hash)
	case JobStateProcessing:
		return fmt.Errorf("job with hash %s is currently processing and cannot be cancelled", hash)
	case JobStatePending:
		// Allow cancellation of pending jobs
		cancel, exists := activeJobs[hash]
		if !exists {
			return fmt.Errorf("job with hash %s is pending but not active", hash)
		}
		cancel()
		delete(activeJobs, hash)
		jobStates[hash] = JobStateCancelled
		return nil
	default:
		return fmt.Errorf("job with hash %s is in unknown state", hash)
	}
}

// GetJobState returns the current state of a job
func GetJobState(hash string) (JobState, bool) {
	mu.RLock()
	defer mu.RUnlock()
	state, exists := jobStates[hash]
	return state, exists
}

// IsJobCancellable checks if a job can be cancelled
func IsJobCancellable(hash string) bool {
	mu.RLock()
	defer mu.RUnlock()
	state, exists := jobStates[hash]
	return exists && state == JobStatePending
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
	// Extract hash from job directory path
	hash := filepath.Base(jobDir)

	// Mark job as processing
	mu.Lock()
	jobStates[hash] = JobStateProcessing
	mu.Unlock()

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Register the cancel function
	mu.Lock()
	activeJobs[hash] = cancel
	mu.Unlock()

	// Ensure cleanup
	defer func() {
		mu.Lock()
		delete(activeJobs, hash)
		mu.Unlock()
	}()

	err := ProcessJob(ctx, jobDir)

	// Mark job as completed or failed
	mu.Lock()
	if err != nil {
		if ctx.Err() == context.Canceled {
			jobStates[hash] = JobStateCancelled
		} else {
			jobStates[hash] = JobStateFailed
		}
	} else {
		jobStates[hash] = JobStateCompleted
	}
	mu.Unlock()

	return err
}

// ProcessPendingJobs runs in a continuous loop processing pending image conversion jobs.
// This function is designed to run as a background goroutine and handles the job queue.
//
// Processing logic:
// 1. Continuously checks for pending jobs every 1 second when queue is empty
// 2. Processes jobs concurrently using a worker pool (configurable max workers, default 2)
// 3. Uses a semaphore to limit concurrent workers and prevent resource exhaustion
// 4. For each job:
//   - Calls processJob() to handle the conversion
//   - Removes job from pending queue on success
//   - Removes failed jobs from queue to prevent infinite retry loops
//   - Logs processing status and errors
//
// Concurrency benefits:
// - Multiple jobs can be processed simultaneously (configurable via PIXERVE_MAX_WORKERS)
// - I/O-bound operations (file writing) happen concurrently within each job
// - CPU-bound operations (image encoding) are naturally parallelized
//
// Configuration:
// - PIXERVE_MAX_WORKERS: Number of concurrent workers (default: NumCPU-1, minimum 1, range: 1-10)
//
// This function runs indefinitely and should be started as a goroutine in main().
// It provides the async processing capability that allows the HTTP server to remain responsive.
func ProcessPendingJobs() {
	maxWorkers := getMaxWorkers()
	semaphore := make(chan struct{}, maxWorkers)

	for {
		jobs := GetPendingJobs()
		if len(jobs) == 0 {
			time.Sleep(1 * time.Second) // Wait before checking again
			continue
		}
		logger.Infof("Processing %d pending jobs", len(jobs))

		// Process jobs concurrently with worker limit
		var wg sync.WaitGroup
		for _, jobDir := range jobs {
			wg.Add(1)
			go func(jobDir string) {
				defer wg.Done()

				// Acquire worker slot
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				// Process the job
				if err := processJob(jobDir); err != nil {
					logger.Errorf("Failed to process job in %s: %v", jobDir, err)
					// Remove failed jobs from pending queue to prevent infinite retries
					RemovePendingJob(jobDir)
				} else {
					// Remove from pending on success
					RemovePendingJob(jobDir)
					logger.Infof("Processed job in %s", jobDir)
				}
			}(jobDir)
		}

		// Wait for all jobs in this batch to complete
		wg.Wait()
	}
}
