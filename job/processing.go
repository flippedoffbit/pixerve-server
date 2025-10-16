package job

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pixerve/config"
	"pixerve/encoder"
	"pixerve/failures"
	"pixerve/logger"
	"pixerve/models"
	"pixerve/success"
	writerbackends "pixerve/writerBackends"
)

// ProcessJob processes a single job from the pending queue
func ProcessJob(ctx context.Context, jobDir string) error {
	// Ensure encoders are registered
	encoder.RegisterDefaults()

	// Create a channel to signal cleanup completion
	cleanupDone := make(chan struct{})

	// Start cleanup goroutine for cancellation
	go func() {
		defer close(cleanupDone)
		<-ctx.Done()
		logger.Infof("Job cancelled, cleaning up %s", jobDir)
		// Only cleanup if context was cancelled (not if job completed successfully)
		if ctx.Err() == context.Canceled {
			if err := os.RemoveAll(jobDir); err != nil {
				logger.Errorf("Failed to cleanup cancelled job directory %s: %v", jobDir, err)
			}
		}
	}()

	// Ensure cleanup goroutine completes
	defer func() {
		<-cleanupDone
	}()

	// Read instructions
	instr, err := ReadInstructions(jobDir)
	if err != nil {
		logger.Errorf("Failed to read instructions for %s: %v", jobDir, err)
		// Create a minimal instr for failure storage
		hash := filepath.Base(jobDir)
		return storeFailure(JobInstructions{Hash: hash}, err)
	}

	logger.Infof("Processing job in %s: %s", jobDir, instr.OriginalFile)

	// Create output subdirectory
	outputDir := filepath.Join(jobDir, "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		logger.Errorf("Failed to create output directory for %s: %v", jobDir, err)
		return storeFailure(instr, err)
	}

	// Process conversions
	convertedFiles, err := processConversions(ctx, instr, outputDir)
	if err != nil {
		logger.Errorf("Failed to process conversions for %s: %v", jobDir, err)
		return storeFailure(instr, err)
	}

	// Write to storage backends
	if err := processWriters(ctx, instr, convertedFiles); err != nil {
		logger.Errorf("Failed to write to storage backends for %s: %v", jobDir, err)
		return storeFailure(instr, err)
	}

	// Store success record
	if err := success.StoreSuccess(instr.Hash, instr.Job, len(convertedFiles)); err != nil {
		logger.Errorf("Failed to store success record for %s: %v", jobDir, err)
		// Don't fail the job for success storage errors
	}

	// Send callback if configured
	if err := sendCallback(instr); err != nil {
		logger.Errorf("Failed to send callback for %s: %v", jobDir, err)
		// Don't fail the job for callback errors
	}

	// Cleanup temp directory
	if err := os.RemoveAll(jobDir); err != nil {
		logger.Errorf("Failed to cleanup temp directory %s: %v", jobDir, err)
		// Don't fail for cleanup errors
	}

	logger.Infof("Successfully processed job in %s", jobDir)
	return nil
}

// processConversions runs all conversion jobs and returns list of output files
func processConversions(ctx context.Context, instr JobInstructions, outputDir string) ([]string, error) {
	var convertedFiles []string

	inputPath := filepath.Join(instr.FilePath, instr.OriginalFile)

	for _, convJob := range instr.Job.ConversionJobs {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("job cancelled: %w", ctx.Err())
		default:
		}

		outputFile, err := runConversion(ctx, inputPath, convJob, outputDir, instr.Hash, instr.OriginalFile)
		if err != nil {
			return nil, fmt.Errorf("conversion failed for %s: %w", convJob.Encoder, err)
		}
		convertedFiles = append(convertedFiles, outputFile)
	}

	return convertedFiles, nil
}

// runConversion executes a single conversion job
func runConversion(ctx context.Context, inputPath string, convJob models.ConversionJob, outputDir, hash, originalFile string) (string, error) {
	// Generate output filename
	outputFile := generateOutputFilename(hash, originalFile, convJob)

	outputPath := filepath.Join(outputDir, outputFile)

	// Get encoder function
	enc, ok := encoder.Get(convJob.Encoder)
	if !ok {
		return "", fmt.Errorf("encoder %s not found", convJob.Encoder)
	}

	// Run conversion
	opts := encoder.EncodeOptions{
		Width:   convJob.Width,
		Height:  convJob.Length, // Note: Length is height in the model
		Quality: convJob.Quality,
		Speed:   convJob.Speed,
	}

	if err := enc(ctx, inputPath, outputPath, opts); err != nil {
		return "", fmt.Errorf("encoding failed: %w", err)
	}

	return outputFile, nil
}

// generateOutputFilename creates the output filename based on conversion job
func generateOutputFilename(hash, originalFile string, convJob models.ConversionJob) string {
	// Extract original name without extension
	nameParts := strings.Split(originalFile, ".")
	originalName := strings.Join(nameParts[:len(nameParts)-1], ".")
	originalExt := ""
	if len(nameParts) > 1 {
		originalExt = nameParts[len(nameParts)-1]
	}

	if convJob.Encoder == "copy" {
		// For copy encoder: hash_original_name.original_extension
		return fmt.Sprintf("%s_%s.%s", hash, originalName, originalExt)
	} else {
		// For other encoders: hash_original_name_len_wid_.extension
		ext := getExtensionForEncoder(convJob.Encoder)
		return fmt.Sprintf("%s_%s_%d_%d_.%s", hash, originalName, convJob.Length, convJob.Width, ext)
	}
}

// getExtensionForEncoder returns the file extension for a given encoder
func getExtensionForEncoder(encoderName string) string {
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
		return encoderName // fallback
	}
}

// processWriters writes converted files to all configured storage backends
func processWriters(ctx context.Context, instr JobInstructions, convertedFiles []string) error {
	for _, writerJob := range instr.Job.WriterJobs {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("job cancelled during writing: %w", ctx.Err())
		default:
		}

		for _, file := range convertedFiles {
			// Check for cancellation
			select {
			case <-ctx.Done():
				return fmt.Errorf("job cancelled during writing: %w", ctx.Err())
			default:
			}

			filePath := filepath.Join(instr.FilePath, "output", file)

			// Open the file for reading
			reader, err := os.Open(filePath)
			if err != nil {
				return fmt.Errorf("failed to open file %s: %w", filePath, err)
			}

			// Prepare access info
			accessInfo := prepareAccessInfo(writerJob, file, instr.Job.SubDir)

			// Write to backend (closes reader when done)
			if err := writerbackends.WriteImage(ctx, accessInfo, reader, writerJob.Type); err != nil {
				reader.Close() // Close on error
				return fmt.Errorf("failed to write %s to %s: %w", file, writerJob.Type, err)
			}

			// Close reader after successful write
			reader.Close()
		}
	}

	return nil
}

// prepareAccessInfo prepares the access info map for the writer backend
func prepareAccessInfo(writerJob models.WriterJob, filename, subDir string) map[string]string {
	accessInfo := make(map[string]string)

	// Copy credentials
	for k, v := range writerJob.Credentials {
		accessInfo[k] = v
	}

	// Add filename and subdir
	accessInfo["filename"] = filename
	accessInfo["folder"] = subDir

	// Set backend-specific configuration
	switch writerJob.Type {
	case "directServe":
		accessInfo["baseDir"] = config.GetDirectServeBaseDir()
	}

	return accessInfo
}

// storeFailure stores a processing failure in the failure store
func storeFailure(instr JobInstructions, err error) error {
	if instr.Hash == "" {
		logger.Errorf("Cannot store failure: missing hash")
		return err
	}

	if storeErr := failures.StoreFailure(instr.Hash, err, instr); storeErr != nil {
		logger.Errorf("Failed to store failure for hash %s: %v", instr.Hash, storeErr)
	}

	return err
}

// sendCallback sends completion callback if configured
func sendCallback(instr JobInstructions) error {
	if instr.Job.CallbackURL == "" {
		return nil // No callback configured
	}

	// Prepare callback payload
	payload := map[string]interface{}{
		"hash":       instr.Hash,
		"status":     "completed",
		"file_count": len(instr.Job.ConversionJobs) + 1, // +1 for original if kept
		"timestamp":  time.Now().Unix(),
		"job_data":   instr.Job,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal callback payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", instr.Job.CallbackURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create callback request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Pixerve/1.0")

	// Add custom callback headers if provided
	for key, value := range instr.Job.CallbackHeaders {
		req.Header.Set(key, value)
	}

	// Send request with timeout
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("callback request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("callback returned non-2xx status: %d", resp.StatusCode)
	}

	logger.Infof("Successfully sent callback to %s", instr.Job.CallbackURL)
	return nil
}
