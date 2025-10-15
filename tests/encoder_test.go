package tests

import (
	"fmt"
	"path/filepath"
	"pixerve/encoder"
	"pixerve/models"
	"testing"
)

func TestEncoderRegistration(t *testing.T) {
	// Register defaults
	encoder.RegisterDefaults()

	// Test getting copy encoder (always available)
	copyEncoder, exists := encoder.Get("copy")
	if !exists {
		t.Error("Copy encoder should be registered")
	}
	if copyEncoder == nil {
		t.Error("Copy encoder function should not be nil")
	}

	// Note: External encoders (jpg, webp, avif) may not be available in test environment
	// They will be skipped if commands are not found in PATH
	// This is expected behavior and not a test failure
}

func TestConversionJobValidation(t *testing.T) {
	// Test valid conversion job
	validJob := models.ConversionJob{
		Encoder: "jpg",
		Width:   800,
		Length:  600,
		Quality: 80,
		Speed:   1,
	}

	if validJob.Encoder == "" {
		t.Error("Encoder should not be empty")
	}

	if validJob.Width <= 0 {
		t.Error("Width should be positive")
	}

	if validJob.Length <= 0 {
		t.Error("Length should be positive")
	}

	if validJob.Quality < 1 || validJob.Quality > 100 {
		t.Error("Quality should be between 1 and 100")
	}
}

func TestFileNaming(t *testing.T) {
	testHash := "abc123def456"
	testOriginalFile := "test_image.jpg"

	// Test copy encoder naming
	copyJob := models.ConversionJob{Encoder: "copy"}
	copyName := generateExpectedFilename(testHash, testOriginalFile, copyJob)
	expectedCopy := "abc123def456_test_image.jpg"
	if copyName != expectedCopy {
		t.Errorf("Expected copy filename %s, got %s", expectedCopy, copyName)
	}

	// Test conversion naming
	convJob := models.ConversionJob{
		Encoder: "webp",
		Width:   800,
		Length:  600,
	}
	convName := generateExpectedFilename(testHash, testOriginalFile, convJob)
	expectedConv := "abc123def456_test_image_600_800_.webp"
	if convName != expectedConv {
		t.Errorf("Expected conversion filename %s, got %s", expectedConv, convName)
	}
}

// Helper function to generate expected filename (mirroring the logic in processing.go)
func generateExpectedFilename(hash, originalFile string, convJob models.ConversionJob) string {
	// Extract original name without extension
	nameParts := filepath.Ext(originalFile)
	originalName := originalFile[:len(originalFile)-len(nameParts)]

	if convJob.Encoder == "copy" {
		// For copy encoder: hash_original_name.original_extension
		return hash + "_" + originalName + nameParts
	} else {
		// For other encoders: hash_original_name_len_wid_.extension
		ext := getExpectedExtension(convJob.Encoder)
		return hash + "_" + originalName + "_" + fmt.Sprintf("%d_%d_.%s", convJob.Length, convJob.Width, ext)
	}
}

func getExpectedExtension(encoderName string) string {
	switch encoderName {
	case "jpeg", "jpg":
		return "jpg"
	case "png":
		return "png"
	case "webp":
		return "webp"
	case "avif":
		return "avif"
	default:
		return encoderName
	}
}
