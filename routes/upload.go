package routes

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"pixerve/config"
	"pixerve/job"
	"pixerve/logger"
	"pixerve/models"
	"pixerve/utils"
)

// verifyJWT verifies the JWT from the request and returns the claims
func verifyJWT(r *http.Request) (*models.PixerveJWT, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		logger.Debug("Missing authorization header")
		return nil, fmt.Errorf("authorization header required")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		logger.Debug("Invalid authorization header format")
		return nil, fmt.Errorf("invalid authorization header format")
	}

	logger.Debug("Verifying JWT token")
	claims, err := utils.VerifyPixerveJWT(token, utils.VerifyConfig{
		SecretKey: []byte(config.SHARED_JWT_SECRET),
	})
	if err != nil {
		logger.Errorf("JWT verification failed: %v", err)
		return nil, err
	}
	logger.Debug("JWT verification successful")
	return claims, nil
}

// computeHash computes SHA256 hash from io.Reader
func computeHash(reader io.Reader) (string, error) {
	logger.Debug("Computing SHA256 hash")
	hash := sha256.New()
	_, err := io.Copy(hash, reader)
	if err != nil {
		logger.Errorf("Failed to compute hash: %v", err)
		return "", err
	}
	hashStr := hex.EncodeToString(hash.Sum(nil))
	logger.Debugf("Hash computed: %s", hashStr)
	return hashStr, nil
}

// createTempDir creates temp directory with hash name
func createTempDir(hash string) (string, error) {
	tempDir := filepath.Join(os.TempDir(), hash)
	logger.Debugf("Creating temp directory: %s", tempDir)
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		logger.Errorf("Failed to create temp directory %s: %v", tempDir, err)
		return "", err
	}
	logger.Debugf("Temp directory created successfully: %s", tempDir)
	return tempDir, nil
}

// saveFile saves data to file in dir
func saveFile(dir, filename string, data []byte) error {
	destPath := filepath.Join(dir, filename)
	logger.Debugf("Saving file: %s", destPath)
	err := os.WriteFile(destPath, data, 0644)
	if err != nil {
		logger.Errorf("Failed to save file %s: %v", destPath, err)
		return err
	}
	logger.Debugf("File saved successfully: %s (%d bytes)", destPath, len(data))
	return nil
}

// respondSuccess sends success response
func respondSuccess(w http.ResponseWriter, hash string, expectedFiles []string) {
	logger.Debugf("Sending success response: hash=%s, expectedFiles=%v", hash, expectedFiles)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"hash":"%s","expected_files":%q}`, hash, expectedFiles)
	logger.Debug("Success response sent")
}

// calculateExpectedFiles calculates the expected output filenames
func calculateExpectedFiles(hash, originalFile string, conversionJobs []models.ConversionJob) []string {
	logger.Debugf("Calculating expected files for hash=%s, originalFile=%s, jobs=%d",
		hash, originalFile, len(conversionJobs))
	var files []string

	for _, convJob := range conversionJobs {
		// Extract original name without extension
		nameParts := strings.Split(originalFile, ".")
		originalName := strings.Join(nameParts[:len(nameParts)-1], ".")
		originalExt := ""
		if len(nameParts) > 1 {
			originalExt = nameParts[len(nameParts)-1]
		}

		if convJob.Encoder == "copy" {
			// For copy encoder: hash_original_name.original_extension
			filename := fmt.Sprintf("%s_%s.%s", hash, originalName, originalExt)
			files = append(files, filename)
			logger.Debugf("Added copy file: %s", filename)
		} else {
			// For other encoders: hash_original_name_len_wid_.extension
			ext := getExtensionForEncoder(convJob.Encoder)
			filename := fmt.Sprintf("%s_%s_%d_%d_.%s", hash, originalName, convJob.Length, convJob.Width, ext)
			files = append(files, filename)
			logger.Debugf("Added conversion file: %s", filename)
		}
	}

	logger.Debugf("Calculated %d expected files", len(files))
	return files
}

// getExtensionForEncoder returns the file extension for a given encoder
func getExtensionForEncoder(encoderName string) string {
	logger.Debugf("Getting extension for encoder: %s", encoderName)
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
		logger.Debugf("Using encoder name as extension: %s", encoderName)
		return encoderName // fallback
	}
}

func UploadHandler(w http.ResponseWriter, r *http.Request) {
	logger.Debugf("Upload request received: method=%s, content-type=%s, content-length=%d",
		r.Method, r.Header.Get("Content-Type"), r.ContentLength)

	if r.Method != http.MethodPost {
		logger.Warnf("Invalid method for upload endpoint: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify JWT and get claims
	logger.Debug("Verifying JWT token")
	claims, err := verifyJWT(r)
	if err != nil {
		logger.Errorf("JWT verification failed: %v", err)
		http.Error(w, fmt.Sprintf("Invalid token: %v", err), http.StatusUnauthorized)
		return
	}
	logger.Infof("JWT verified successfully for subject: %s", claims.Subject)

	// Parse multipart form
	logger.Debug("Parsing multipart form data")
	err = r.ParseMultipartForm(32 << 20) // 32 MB max
	if err != nil {
		logger.Errorf("Failed to parse multipart form: %v", err)
		http.Error(w, "Failed to parse multipart form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		logger.Errorf("Failed to get file from form: %v", err)
		http.Error(w, "Failed to get file from form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	logger.Infof("File received: %s, size: %d bytes", header.Filename, header.Size)

	// Compute SHA256 hash
	logger.Debug("Computing SHA256 hash of file")
	hashSum, err := computeHash(file)
	if err != nil {
		logger.Errorf("Failed to compute file hash: %v", err)
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}
	logger.Debugf("File hash computed: %s", hashSum)

	// Reset file pointer to beginning
	file.Seek(0, 0)

	// Create temp directory with hash
	logger.Debugf("Creating temporary directory: %s", hashSum)
	tempDir, err := createTempDir(hashSum)
	if err != nil {
		logger.Errorf("Failed to create temp directory: %v", err)
		http.Error(w, "Failed to create temp directory", http.StatusInternalServerError)
		return
	}
	logger.Debugf("Temporary directory created: %s", tempDir)

	// Read file data
	logger.Debug("Reading file data into memory")
	data, err := io.ReadAll(file)
	if err != nil {
		logger.Errorf("Failed to read file data: %v", err)
		http.Error(w, "Failed to read file data", http.StatusInternalServerError)
		return
	}
	logger.Debugf("File data read successfully: %d bytes", len(data))

	// Save file with original name
	logger.Debugf("Saving file to temp directory: %s", header.Filename)
	err = saveFile(tempDir, header.Filename, data)
	if err != nil {
		logger.Errorf("Failed to save file: %v", err)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	logger.Debugf("File saved successfully: %s", filepath.Join(tempDir, header.Filename))

	// Parse job from claims
	logger.Debug("Parsing job specifications from JWT claims")
	combinedJob, err := job.ParseTokenIntoJobsFromClaims(claims)
	if err != nil {
		logger.Errorf("Failed to parse job from claims: %v", err)
		http.Error(w, fmt.Sprintf("Failed to parse job: %v", err), http.StatusBadRequest)
		return
	}
	logger.Infof("Job parsed successfully: %d conversion jobs", len(combinedJob.ConversionJobs))

	// Calculate expected output filenames
	logger.Debug("Calculating expected output filenames")
	expectedFiles := calculateExpectedFiles(hashSum, header.Filename, combinedJob.ConversionJobs)
	logger.Debugf("Expected output files: %v", expectedFiles)

	// Create instructions
	instr := job.JobInstructions{
		FilePath:     tempDir,
		OriginalFile: header.Filename,
		Hash:         hashSum,
		Job:          combinedJob,
	}

	// Write instructions.json
	logger.Debug("Writing job instructions to instructions.json")
	err = job.WriteInstructions(tempDir, instr)
	if err != nil {
		logger.Errorf("Failed to write instructions: %v", err)
		http.Error(w, fmt.Sprintf("Failed to write instructions: %v", err), http.StatusInternalServerError)
		return
	}
	logger.Debug("Job instructions written successfully")

	// Add to pending jobs
	logger.Info("Adding job to pending queue")
	job.AddPendingJob(tempDir)

	logger.Infof("Upload completed successfully: hash=%s, files=%v", hashSum, expectedFiles)
	respondSuccess(w, hashSum, expectedFiles)
}
