package config

import (
	"os"
	"path/filepath"
)

// DATA_DIR is the directory where Pixerve stores its data (databases, etc.)
// Defaults to "./data" relative to the executable
var DATA_DIR = getDataDir()

// getDataDir determines the data directory path from environment or default.
// This is called at package initialization to set the DATA_DIR variable.
// Priority: PIXERVE_DATA_DIR environment variable > "./data" default
func getDataDir() string {
	if dir := os.Getenv("PIXERVE_DATA_DIR"); dir != "" {
		return dir
	}
	// Default to ./data subdirectory
	return "./data"
}

// GetDataDir returns the current data directory path.
// This function checks the environment variable at runtime, allowing for
// dynamic configuration changes without restarting the server.
// Used by database path functions to construct full file paths.
func GetDataDir() string {
	return getDataDir()
}

// GetCredentialsDBPath returns the full path to the credentials database.
// The credentials database stores user authentication information.
// Path: {DATA_DIR}/credentials.db
func GetCredentialsDBPath() string {
	return filepath.Join(GetDataDir(), "credentials.db")
}

// GetFailuresDBPath returns the full path to the failures database.
// The failures database tracks jobs that failed processing.
// Path: {DATA_DIR}/failures.db
func GetFailuresDBPath() string {
	return filepath.Join(GetDataDir(), "failures.db")
}

// GetSuccessDBPath returns the full path to the success database.
// The success database tracks successfully completed jobs.
// Path: {DATA_DIR}/success.db
func GetSuccessDBPath() string {
	return filepath.Join(GetDataDir(), "success.db")
}

// GetDirectServeBaseDir returns the base directory for direct file serving.
// This directory contains processed images that are served directly by the HTTP server.
// Configurable via PIXERVE_SERVE_DIR environment variable for server administrators.
// Not configurable by end users for security reasons.
// Defaults to "./serve" relative to the executable.
func GetDirectServeBaseDir() string {
	if dir := os.Getenv("PIXERVE_SERVE_DIR"); dir != "" {
		return dir
	}
	// Default to ./serve subdirectory
	return "./serve"
}
