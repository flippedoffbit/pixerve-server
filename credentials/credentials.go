package credentials

import (
	"encoding/json"
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
	encodedCreds, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	return db.Set([]byte(key), encodedCreds, pebble.Sync)
}

// DeleteCredentials deletes the credentials for the given key
func DeleteCredentials(key string) error {
	return db.Delete([]byte(key), pebble.Sync)
}
