package tests

import (
	"os"
	"path/filepath"
	"pixerve/config"
	"testing"
)

func TestConfigDataDir(t *testing.T) {
	// Test default data directory
	defaultDir := config.DATA_DIR
	if defaultDir == "" {
		t.Error("Expected non-empty default data directory")
	}

	// Test that it defaults to "./data"
	expectedDefault := "./data"
	if defaultDir != expectedDefault {
		t.Errorf("Expected default data dir %s, got %s", expectedDefault, defaultDir)
	}
}

func TestConfigDataDirEnv(t *testing.T) {
	// Save original env var
	originalEnv := os.Getenv("PIXERVE_DATA_DIR")
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("PIXERVE_DATA_DIR")
		} else {
			os.Setenv("PIXERVE_DATA_DIR", originalEnv)
		}
	}()

	// Test custom data directory via environment variable
	customDir := "/tmp/pixerve-test-data"
	os.Setenv("PIXERVE_DATA_DIR", customDir)

	// We need to re-initialize the config package to pick up the env var
	// Since the variable is initialized at package load time, we'll test the functions directly
	// by setting the env and checking the functions

	// Test GetCredentialsDBPath with custom dir
	credentialsPath := config.GetCredentialsDBPath()
	expectedCredentialsPath := filepath.Join(customDir, "credentials.db")
	if credentialsPath != expectedCredentialsPath {
		t.Errorf("Expected credentials path %s, got %s", expectedCredentialsPath, credentialsPath)
	}

	// Test GetFailuresDBPath with custom dir
	failuresPath := config.GetFailuresDBPath()
	expectedFailuresPath := filepath.Join(customDir, "failures.db")
	if failuresPath != expectedFailuresPath {
		t.Errorf("Expected failures path %s, got %s", expectedFailuresPath, failuresPath)
	}

	// Test GetSuccessDBPath with custom dir
	successPath := config.GetSuccessDBPath()
	expectedSuccessPath := filepath.Join(customDir, "success.db")
	if successPath != expectedSuccessPath {
		t.Errorf("Expected success path %s, got %s", expectedSuccessPath, successPath)
	}
}

func TestConfigDBPaths(t *testing.T) {
	// Test with default data directory
	credentialsPath := config.GetCredentialsDBPath()
	failuresPath := config.GetFailuresDBPath()
	successPath := config.GetSuccessDBPath()

	// All should be within the data directory
	expectedBaseDir := "data"

	if filepath.Dir(credentialsPath) != expectedBaseDir {
		t.Errorf("Expected credentials path to be in %s, got %s", expectedBaseDir, filepath.Dir(credentialsPath))
	}

	if filepath.Dir(failuresPath) != expectedBaseDir {
		t.Errorf("Expected failures path to be in %s, got %s", expectedBaseDir, filepath.Dir(failuresPath))
	}

	if filepath.Dir(successPath) != expectedBaseDir {
		t.Errorf("Expected success path to be in %s, got %s", expectedBaseDir, filepath.Dir(successPath))
	}

	// Test filenames
	if filepath.Base(credentialsPath) != "credentials.db" {
		t.Errorf("Expected credentials filename to be credentials.db, got %s", filepath.Base(credentialsPath))
	}

	if filepath.Base(failuresPath) != "failures.db" {
		t.Errorf("Expected failures filename to be failures.db, got %s", filepath.Base(failuresPath))
	}

	if filepath.Base(successPath) != "success.db" {
		t.Errorf("Expected success filename to be success.db, got %s", filepath.Base(successPath))
	}
}
