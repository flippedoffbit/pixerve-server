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
	"pixerve/models"
	"pixerve/utils"
)

// verifyJWT verifies the JWT from the request and returns the claims
func verifyJWT(r *http.Request) (*models.PixerveJWT, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("authorization header required")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		return nil, fmt.Errorf("invalid authorization header format")
	}

	claims, err := utils.VerifyPixerveJWT(token, utils.VerifyConfig{
		SecretKey: []byte(config.SHARED_JWT_SECRET),
	})
	if err != nil {
		return nil, err
	}
	return claims, nil
}

// computeHash computes SHA256 hash from io.Reader
func computeHash(reader io.Reader) (string, error) {
	hash := sha256.New()
	_, err := io.Copy(hash, reader)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// createTempDir creates temp directory with hash name
func createTempDir(hash string) (string, error) {
	tempDir := filepath.Join(os.TempDir(), hash)
	return tempDir, os.MkdirAll(tempDir, 0755)
}

// saveFile saves data to file in dir
func saveFile(dir, filename string, data []byte) error {
	destPath := filepath.Join(dir, filename)
	return os.WriteFile(destPath, data, 0644)
}

// respondSuccess sends success response
func respondSuccess(w http.ResponseWriter, hash string, expectedFiles []string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"hash":"%s","expected_files":%q}`, hash, expectedFiles)
}

// calculateExpectedFiles calculates the expected output filenames
func calculateExpectedFiles(hash, originalFile string, conversionJobs []models.ConversionJob) []string {
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
			files = append(files, fmt.Sprintf("%s_%s.%s", hash, originalName, originalExt))
		} else {
			// For other encoders: hash_original_name_len_wid_.extension
			ext := getExtensionForEncoder(convJob.Encoder)
			files = append(files, fmt.Sprintf("%s_%s_%d_%d_.%s", hash, originalName, convJob.Length, convJob.Width, ext))
		}
	}

	return files
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

func UploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify JWT and get claims
	claims, err := verifyJWT(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid token: %v", err), http.StatusUnauthorized)
		return
	}

	// Parse multipart form
	err = r.ParseMultipartForm(32 << 20) // 32 MB max
	if err != nil {
		http.Error(w, "Failed to parse multipart form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file from form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Compute SHA256 hash
	hashSum, err := computeHash(file)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	// Reset file pointer to beginning
	file.Seek(0, 0)

	// Create temp directory with hash
	tempDir, err := createTempDir(hashSum)
	if err != nil {
		http.Error(w, "Failed to create temp directory", http.StatusInternalServerError)
		return
	}

	// Read file data
	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file data", http.StatusInternalServerError)
		return
	}

	// Save file with original name
	err = saveFile(tempDir, header.Filename, data)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	// Parse job from claims
	combinedJob, err := job.ParseTokenIntoJobsFromClaims(claims)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse job: %v", err), http.StatusBadRequest)
		return
	}

	// Calculate expected output filenames
	expectedFiles := calculateExpectedFiles(hashSum, header.Filename, combinedJob.ConversionJobs)

	// Create instructions
	instr := job.JobInstructions{
		FilePath:     tempDir,
		OriginalFile: header.Filename,
		Hash:         hashSum,
		Job:          combinedJob,
	}

	// Write instructions.json
	err = job.WriteInstructions(tempDir, instr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to write instructions: %v", err), http.StatusInternalServerError)
		return
	}

	// Add to pending jobs
	job.AddPendingJob(tempDir)

	respondSuccess(w, hashSum, expectedFiles)
}
