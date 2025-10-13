package writerbackends

// suggest an implimention based in filename based on other 2 functions in the package(writerBackends). basically we write the file to fs with folder prefix inside our main serving folder and its served directly by our http server

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"pixerve/logger"
)

// UploadToDirectServe uploads content from an io.Reader to a local file system path,
// which is served directly by the HTTP server.
func UploadToDirectServe(ctx context.Context, accessInfo map[string]string, reader io.Reader) error {
	// Extract the target directory and filename from accessInfo
	baseDir := accessInfo["baseDir"]   // Base directory where files are served from
	folder := accessInfo["folder"]     // Subfolder inside the base directory
	filename := accessInfo["filename"] // Target filename

	// Construct the full file path
	fullDir := filepath.Join(baseDir, folder)
	fullPath := filepath.Join(fullDir, filename)

	// Ensure the target directory exists
	if err := os.MkdirAll(fullDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Create or truncate the target file
	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", fullPath, err)
	}
	defer file.Close()

	// Copy the content from the reader to the file
	if _, err := io.Copy(file, reader); err != nil {
		return fmt.Errorf("failed to write to file %s: %w", fullPath, err)
	}

	logger.Infof("Successfully saved file '%s' to '%s'", filename, fullPath)
	return nil
}

func UseUploadToDirectServeExample() {
	// Example usage of UploadToDirectServe
	baseDir := "./public" // Base directory where files are served from
	folder := "uploads"   // Subfolder inside the base directory
	filename := "example.txt"
	content := "This is some data to save to the local file system."

	accessInfo := map[string]string{
		"baseDir":  baseDir,
		"folder":   folder,
		"filename": filename,
	}

	// Create a reader from your content.
	reader := io.NopCloser(strings.NewReader(content))

	// Call the self-contained upload function.
	err := UploadToDirectServe(context.TODO(), accessInfo, reader)
	if err != nil {
		logger.Fatal(err)
	}
}
