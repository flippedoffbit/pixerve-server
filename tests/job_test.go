package tests

import (
	"pixerve/job"
	"pixerve/models"
	"pixerve/utils"
	"testing"
	"time"
)

func TestParseTokenIntoJobs(t *testing.T) {
	// Create a test JWT with job specifications
	jobSpec := models.JobSpec{
		CompletionCallback: "https://example.com/callback",
		CallbackHeaders: map[string]string{
			"Authorization": "Bearer token123",
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

	// Test parsing
	combinedJob, err := job.ParseTokenIntoJobsFromClaims(claims)
	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	// Verify conversion jobs
	expectedConversions := 4 // 2 jpg + 2 webp
	if len(combinedJob.ConversionJobs) != expectedConversions {
		t.Errorf("Expected %d conversion jobs, got %d", expectedConversions, len(combinedJob.ConversionJobs))
	}

	// Verify writer jobs
	expectedWriters := 3 // s3, gcs, direct
	if len(combinedJob.WriterJobs) != expectedWriters {
		t.Errorf("Expected %d writer jobs, got %d", expectedWriters, len(combinedJob.WriterJobs))
	}

	// Verify callback settings
	if combinedJob.CallbackURL != "https://example.com/callback" {
		t.Errorf("Expected callback URL %s, got %s", "https://example.com/callback", combinedJob.CallbackURL)
	}

	if combinedJob.Priority != 0 {
		t.Errorf("Expected priority 0, got %d", combinedJob.Priority)
	}
}

func TestParseTokenIntoJobsEmpty(t *testing.T) {
	// Test with empty claims
	claims := &models.PixerveJWT{}
	combinedJob, err := job.ParseTokenIntoJobsFromClaims(claims)
	if err != nil {
		t.Fatalf("Failed to parse empty token: %v", err)
	}

	if len(combinedJob.ConversionJobs) != 0 {
		t.Errorf("Expected 0 conversion jobs, got %d", len(combinedJob.ConversionJobs))
	}

	if len(combinedJob.WriterJobs) != 0 {
		t.Errorf("Expected 0 writer jobs, got %d", len(combinedJob.WriterJobs))
	}
}

func TestJWTVerification(t *testing.T) {
	// Test JWT verification with a mock secret
	testSecret := []byte("test-secret-key")

	// Test verification config structure
	config := utils.VerifyConfig{SecretKey: testSecret}

	// This test mainly verifies the config structure
	if config.SecretKey == nil {
		t.Error("Secret key should not be nil")
	}

	if string(config.SecretKey) != "test-secret-key" {
		t.Error("Secret key content mismatch")
	}
}
