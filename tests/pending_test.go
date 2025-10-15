package tests

import (
	"fmt"
	"pixerve/job"
	"testing"
)

func TestPendingJobs(t *testing.T) {
	// Clear any existing pending jobs for clean test
	// Note: In a real scenario, you'd use a separate test database

	// Test adding pending jobs
	job1 := "/tmp/job1"
	job2 := "/tmp/job2"
	job3 := "/tmp/job3"

	job.AddPendingJob(job1)
	job.AddPendingJob(job2)
	job.AddPendingJob(job3)

	// Test getting pending jobs
	pending := job.GetPendingJobs()
	if len(pending) < 3 {
		t.Errorf("Expected at least 3 pending jobs, got %d", len(pending))
	}

	// Verify jobs are in the list
	jobMap := make(map[string]bool)
	for _, j := range pending {
		jobMap[j] = true
	}

	if !jobMap[job1] {
		t.Errorf("Job %s not found in pending list", job1)
	}
	if !jobMap[job2] {
		t.Errorf("Job %s not found in pending list", job2)
	}
	if !jobMap[job3] {
		t.Errorf("Job %s not found in pending list", job3)
	}

	// Test removing jobs
	job.RemovePendingJob(job2)
	pendingAfterRemove := job.GetPendingJobs()

	jobMapAfterRemove := make(map[string]bool)
	for _, j := range pendingAfterRemove {
		jobMapAfterRemove[j] = true
	}

	if jobMapAfterRemove[job2] {
		t.Errorf("Job %s should have been removed", job2)
	}

	if !jobMapAfterRemove[job1] {
		t.Errorf("Job %s should still be present", job1)
	}

	if !jobMapAfterRemove[job3] {
		t.Errorf("Job %s should still be present", job3)
	}

	// Clean up
	job.RemovePendingJob(job1)
	job.RemovePendingJob(job3)

	finalPending := job.GetPendingJobs()
	if len(finalPending) > 0 {
		t.Logf("Warning: %d jobs still pending after cleanup", len(finalPending))
	}
}

func TestPendingJobsConcurrency(t *testing.T) {
	// Test concurrent access to pending jobs
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			jobDir := fmt.Sprintf("/tmp/concurrent-job-%d", id)
			job.AddPendingJob(jobDir)

			// Simulate some work
			pending := job.GetPendingJobs()
			if len(pending) == 0 {
				t.Errorf("No pending jobs found in goroutine %d", id)
			}

			job.RemovePendingJob(jobDir)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Final check
	finalPending := job.GetPendingJobs()
	if len(finalPending) > 0 {
		t.Logf("Warning: %d jobs still pending after concurrent test", len(finalPending))
		// Clean up any remaining jobs
		for _, j := range finalPending {
			job.RemovePendingJob(j)
		}
	}
}
