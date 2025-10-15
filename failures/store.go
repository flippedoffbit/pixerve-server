package failures

import (
	"encoding/json"
	"fmt"
	"time"

	pebble "github.com/cockroachdb/pebble"
)

// FailureRecord represents a processing failure
type FailureRecord struct {
	Hash      string    `json:"hash"`
	Timestamp time.Time `json:"timestamp"`
	Error     string    `json:"error"`
	JobData   string    `json:"job_data"` // JSON string of the job instructions
}

var db *pebble.DB

// Init initializes the failure store
func Init(dbPath string) error {
	var err error
	db, err = pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		return fmt.Errorf("failed to open failure store: %w", err)
	}
	return nil
}

// Close closes the failure store
func Close() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

// StoreFailure stores a processing failure
func StoreFailure(hash string, err error, jobData interface{}) error {
	if db == nil {
		return fmt.Errorf("failure store not initialized")
	}

	// Convert job data to JSON
	jobJSON, jsonErr := json.Marshal(jobData)
	if jsonErr != nil {
		jobJSON = []byte(fmt.Sprintf("failed to marshal job data: %v", jsonErr))
	}

	record := FailureRecord{
		Hash:      hash,
		Timestamp: time.Now(),
		Error:     err.Error(),
		JobData:   string(jobJSON),
	}

	// Convert record to JSON
	data, jsonErr := json.Marshal(record)
	if jsonErr != nil {
		return fmt.Errorf("failed to marshal failure record: %w", jsonErr)
	}

	// Store with hash as key
	key := []byte(hash)
	return db.Set(key, data, pebble.Sync)
}

// GetFailure retrieves a failure record by hash
func GetFailure(hash string) (*FailureRecord, error) {
	if db == nil {
		return nil, fmt.Errorf("failure store not initialized")
	}

	key := []byte(hash)
	data, closer, err := db.Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil // No failure found
		}
		return nil, fmt.Errorf("failed to get failure: %w", err)
	}
	defer closer.Close()

	var record FailureRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal failure record: %w", err)
	}

	return &record, nil
}

// DeleteFailure removes a failure record
func DeleteFailure(hash string) error {
	if db == nil {
		return fmt.Errorf("failure store not initialized")
	}

	key := []byte(hash)
	return db.Delete(key, pebble.Sync)
}

// ListFailures returns all failure records (for admin purposes)
func ListFailures() ([]FailureRecord, error) {
	if db == nil {
		return nil, fmt.Errorf("failure store not initialized")
	}

	var failures []FailureRecord
	iter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %w", err)
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		var record FailureRecord
		if err := json.Unmarshal(iter.Value(), &record); err != nil {
			continue // Skip invalid records
		}
		failures = append(failures, record)
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("iteration error: %w", err)
	}

	return failures, nil
}
