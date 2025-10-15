package tests

import (
	"pixerve/job"
	"pixerve/models"
	"pixerve/utils"
	"testing"
	"time"
)

func TestFullJobProcessingFlow(t *testing.T) {
	// Test the complete flow from JWT parsing to job instruction creation

	// Create a test JWT with complete job specification
	jobSpec := models.JobSpec{
		CompletionCallback: "https://example.com/callback",
		CallbackHeaders: map[string]string{
			"Authorization": "Bearer test-token",
		},
		Priority:     0,
		KeepOriginal: false,
		Formats: map[string]models.FormatSpec{
			"jpg": {
				Settings: models.FormatSettings{
					Quality: 80,
					Speed:   1,
				},
				Sizes: [][]int{
					{800, 600},
					{400, 300},
				},
			},
			"webp": {
				Settings: models.FormatSettings{
					Quality: 85,
					Speed:   2,
				},
				Sizes: [][]int{
					{800},
					{400},
				},
			},
		},
		StorageKeys: map[string]string{
			"s3":  "s3-key-123",
			"gcs": "gcs-key-456",
		},
		DirectHost: true,
		SubDir:     "tenant-123",
	}

	claims := &models.PixerveJWT{
		Issuer:    "test-issuer",
		Subject:   "test-subject",
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
		Job:       jobSpec,
	}

	// Test JWT creation and parsing
	tokenString, err := utils.CreatePixerveJWT(claims)
	if err != nil {
		t.Fatalf("Failed to create JWT: %v", err)
	}

	// Parse the token back
	parsedClaims, err := utils.VerifyPixerveJWT(tokenString, utils.VerifyConfig{
		SecretKey: []byte("test-secret-key-for-jwt-signing-at-least-32-bytes-long"),
	})
	if err != nil {
		t.Fatalf("Failed to verify JWT: %v", err)
	}

	// Convert to combined job
	combinedJob, err := job.ParseTokenIntoJobsFromClaims(parsedClaims)
	if err != nil {
		t.Fatalf("Failed to parse claims into jobs: %v", err)
	}

	// Verify the combined job has the expected properties
	if combinedJob.CallbackURL != jobSpec.CompletionCallback {
		t.Errorf("Expected callback URL %s, got %s", jobSpec.CompletionCallback, combinedJob.CallbackURL)
	}

	if combinedJob.CallbackHeaders["Authorization"] != "Bearer test-token" {
		t.Errorf("Expected Authorization header 'Bearer test-token', got %s", combinedJob.CallbackHeaders["Authorization"])
	}

	if combinedJob.Priority != jobSpec.Priority {
		t.Errorf("Expected priority %d, got %d", jobSpec.Priority, combinedJob.Priority)
	}

	if combinedJob.SubDir != jobSpec.SubDir {
		t.Errorf("Expected subdir %s, got %s", jobSpec.SubDir, combinedJob.SubDir)
	}

	// Verify conversion jobs were created
	if len(combinedJob.ConversionJobs) == 0 {
		t.Error("Expected conversion jobs to be created")
	}

	// Check that we have the expected number of conversion jobs
	// jpg: 2 sizes + webp: 2 sizes = 4 conversion jobs
	expectedConversions := 4
	if len(combinedJob.ConversionJobs) != expectedConversions {
		t.Errorf("Expected %d conversion jobs, got %d", expectedConversions, len(combinedJob.ConversionJobs))
	}

	// Verify writer jobs were created
	if len(combinedJob.WriterJobs) == 0 {
		t.Error("Expected writer jobs to be created")
	}

	// Check that we have the expected storage backends
	expectedWriters := 3 // s3, gcs, and direct
	if len(combinedJob.WriterJobs) != expectedWriters {
		t.Errorf("Expected %d writer jobs, got %d", expectedWriters, len(combinedJob.WriterJobs))
	}
}

func TestJobInstructionsCreation(t *testing.T) {
	// Test creating job instructions from combined job

	// Create a test JWT to get a combinedJob
	jobSpec := models.JobSpec{
		CompletionCallback: "https://example.com/callback",
		CallbackHeaders: map[string]string{
			"Authorization": "Bearer token",
		},
		Priority:     0,
		KeepOriginal: false,
		Formats: map[string]models.FormatSpec{
			"jpg": {
				Settings: models.FormatSettings{
					Quality: 80,
					Speed:   1,
				},
				Sizes: [][]int{
					{800, 600},
				},
			},
		},
		DirectHost: true,
		SubDir:     "test-tenant",
	}

	claims := &models.PixerveJWT{
		Issuer:    "test-issuer",
		Subject:   "test-subject",
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
		Job:       jobSpec,
	}

	combinedJob, err := job.ParseTokenIntoJobsFromClaims(claims)
	if err != nil {
		t.Fatalf("Failed to parse claims into jobs: %v", err)
	}

	instr := job.JobInstructions{
		FilePath:     "/tmp/test-job",
		OriginalFile: "test-image.jpg",
		Hash:         "test-hash-123",
		Job:          combinedJob,
	}

	// Verify all fields are set correctly
	if instr.FilePath != "/tmp/test-job" {
		t.Errorf("Expected file path '/tmp/test-job', got %s", instr.FilePath)
	}

	if instr.OriginalFile != "test-image.jpg" {
		t.Errorf("Expected original file 'test-image.jpg', got %s", instr.OriginalFile)
	}

	if instr.Hash != "test-hash-123" {
		t.Errorf("Expected hash 'test-hash-123', got %s", instr.Hash)
	}

	// Verify job details
	if instr.Job.CallbackURL != "https://example.com/callback" {
		t.Errorf("Expected callback URL 'https://example.com/callback', got %s", instr.Job.CallbackURL)
	}

	if instr.Job.SubDir != "test-tenant" {
		t.Errorf("Expected subdir 'test-tenant', got %s", instr.Job.SubDir)
	}

	if len(instr.Job.ConversionJobs) != 1 {
		t.Errorf("Expected 1 conversion job, got %d", len(instr.Job.ConversionJobs))
	}

	if len(instr.Job.WriterJobs) != 1 {
		t.Errorf("Expected 1 writer job, got %d", len(instr.Job.WriterJobs))
	}
}

func TestJobProcessingWorkflow(t *testing.T) {
	// Test the overall job processing workflow structure
	// This is a high-level test that verifies the workflow components exist

	// Create a test JWT to get a combinedJob
	jobSpec := models.JobSpec{
		Priority:     0,
		KeepOriginal: false,
		Formats: map[string]models.FormatSpec{
			"jpg": {
				Settings: models.FormatSettings{
					Quality: 80,
					Speed:   1,
				},
				Sizes: [][]int{
					{800, 600},
				},
			},
		},
		DirectHost: true,
	}

	claims := &models.PixerveJWT{
		Subject:   "workflow-test",
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
		Job:       jobSpec,
	}

	combinedJob, err := job.ParseTokenIntoJobsFromClaims(claims)
	if err != nil {
		t.Fatalf("Failed to parse claims into jobs: %v", err)
	}

	// Test that we can create job instructions
	instr := job.JobInstructions{
		FilePath:     "/tmp/test-workflow",
		OriginalFile: "workflow-test.jpg",
		Hash:         "workflow-hash-123",
		Job:          combinedJob,
	}

	// Verify the instruction structure
	if instr.Hash == "" {
		t.Error("Job instructions should have a hash")
	}

	if instr.FilePath == "" {
		t.Error("Job instructions should have a file path")
	}

	if instr.OriginalFile == "" {
		t.Error("Job instructions should have an original filename")
	}

	if len(instr.Job.ConversionJobs) == 0 {
		t.Error("Job should have conversion jobs")
	}

	if len(instr.Job.WriterJobs) == 0 {
		t.Error("Job should have writer jobs")
	}

	// Test that the job has required fields for processing
	if instr.Job.ConversionJobs[0].Encoder == "" {
		t.Error("Conversion job should have an encoder")
	}

	if instr.Job.WriterJobs[0].Type == "" {
		t.Error("Writer job should have a type")
	}
}
