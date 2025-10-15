package config

import (
	"os"
	"path/filepath"
)

// DATA_DIR is the directory where Pixerve stores its data (databases, etc.)
// Defaults to "./data" relative to the executable
var DATA_DIR = getDataDir()

func getDataDir() string {
	if dir := os.Getenv("PIXERVE_DATA_DIR"); dir != "" {
		return dir
	}
	// Default to ./data subdirectory
	return "./data"
}

// GetDataDir returns the current data directory, checking environment variable at runtime
func GetDataDir() string {
	return getDataDir()
}

// GetCredentialsDBPath returns the full path to the credentials database
func GetCredentialsDBPath() string {
	return filepath.Join(GetDataDir(), "credentials.db")
}

// GetFailuresDBPath returns the full path to the failures database
func GetFailuresDBPath() string {
	return filepath.Join(GetDataDir(), "failures.db")
}

// GetSuccessDBPath returns the full path to the success database
func GetSuccessDBPath() string {
	return filepath.Join(GetDataDir(), "success.db")
}
