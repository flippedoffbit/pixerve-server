package credentials

import (
	"encoding/json"
	"fmt"
	"pixerve/logger"

	"github.com/cockroachdb/pebble"
)

var db *pebble.DB

// OpenDB opens the Pebble DB for credentials at the specified path
func OpenDB(dbPath string) error {
	var err error
	db, err = pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		logger.Errorf("Failed to open Pebble DB: %v", err)
		return err
	}
	return nil
}

// CloseDB closes the DB
func CloseDB() error {
	if db != nil {
		return db.Close()
	}
	return nil

}

func GetCredentials(key string) (map[string]string, error) {
	if db == nil {
		return nil, fmt.Errorf("credentials database not initialized")
	}

	value, closer, err := db.Get([]byte(key))
	if err != nil {
		return nil, err
	}
	defer closer.Close()
	creds := make(map[string]string)
	err = json.Unmarshal(value, &creds)
	if err != nil {
		return nil, err
	}
	return creds, nil
}

// StoreCredentials stores the credentials map under the given key
func StoreCredentials(key string, creds map[string]string) error {
	if db == nil {
		return fmt.Errorf("credentials database not initialized")
	}

	encodedCreds, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	return db.Set([]byte(key), encodedCreds, pebble.Sync)
}

// DeleteCredentials deletes the credentials for the given key
func DeleteCredentials(key string) error {
	if db == nil {
		return fmt.Errorf("credentials database not initialized")
	}

	return db.Delete([]byte(key), pebble.Sync)
}

// CheckHealth performs a basic health check on the credentials database
func CheckHealth() error {
	if db == nil {
		return fmt.Errorf("credentials database not initialized")
	}

	// Try a simple operation to verify database is accessible
	_, closer, err := db.Get([]byte("__health_check__"))
	if err != nil && err != pebble.ErrNotFound {
		return fmt.Errorf("database health check failed: %w", err)
	}
	if closer != nil {
		closer.Close()
	}
	return nil
}
