package success

import (
	"encoding/json"
	"fmt"
	"time"

	pebble "github.com/cockroachdb/pebble"
)

// SuccessRecord represents a successful job completion
type SuccessRecord struct {
	Hash      string    `json:"hash"`
	Timestamp time.Time `json:"timestamp"`
	JobData   string    `json:"job_data"`   // JSON string of the job instructions
	FileCount int       `json:"file_count"` // Number of files generated
}

var db *pebble.DB

// Init initializes the success store
func Init(dbPath string) error {
	var err error
	db, err = pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		return fmt.Errorf("failed to open success store: %w", err)
	}
	return nil
}

// Close closes the success store
func Close() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

// StoreSuccess stores a successful job completion
func StoreSuccess(hash string, jobData interface{}, fileCount int) error {
	if db == nil {
		return fmt.Errorf("success store not initialized")
	}

	// Convert job data to JSON
	jobJSON, jsonErr := json.Marshal(jobData)
	if jsonErr != nil {
		jobJSON = []byte(fmt.Sprintf("failed to marshal job data: %v", jsonErr))
	}

	record := SuccessRecord{
		Hash:      hash,
		Timestamp: time.Now(),
		JobData:   string(jobJSON),
		FileCount: fileCount,
	}

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal success record: %w", err)
	}

	key := []byte(hash)
	return db.Set(key, data, pebble.Sync)
}

// GetSuccess retrieves a success record by hash
func GetSuccess(hash string) (*SuccessRecord, error) {
	if db == nil {
		return nil, fmt.Errorf("success store not initialized")
	}

	key := []byte(hash)
	data, closer, err := db.Get(key)
	if err != nil {
		if err.Error() == "pebble: not found" {
			return nil, nil // Not found is not an error
		}
		return nil, err
	}
	defer closer.Close()

	var record SuccessRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal success record: %w", err)
	}

	return &record, nil
}

// DeleteSuccess removes a success record
func DeleteSuccess(hash string) error {
	if db == nil {
		return fmt.Errorf("success store not initialized")
	}

	key := []byte(hash)
	return db.Delete(key, pebble.Sync)
}

// ListSuccessRecords returns all success records (for admin/debugging)
func ListSuccessRecords() ([]SuccessRecord, error) {
	if db == nil {
		return nil, fmt.Errorf("success store not initialized")
	}

	var records []SuccessRecord
	iter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		var record SuccessRecord
		if err := json.Unmarshal(iter.Value(), &record); err != nil {
			continue // Skip invalid records
		}
		records = append(records, record)
	}

	return records, nil
}

// CleanupOldRecords removes success records older than the specified duration
func CleanupOldRecords(maxAge time.Duration) error {
	if db == nil {
		return fmt.Errorf("success store not initialized")
	}

	cutoff := time.Now().Add(-maxAge)
	iter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		return err
	}
	defer iter.Close()

	var keysToDelete [][]byte
	for iter.First(); iter.Valid(); iter.Next() {
		var record SuccessRecord
		if err := json.Unmarshal(iter.Value(), &record); err != nil {
			continue
		}
		if record.Timestamp.Before(cutoff) {
			key := make([]byte, len(iter.Key()))
			copy(key, iter.Key())
			keysToDelete = append(keysToDelete, key)
		}
	}

	// Delete old records
	for _, key := range keysToDelete {
		if err := db.Delete(key, pebble.Sync); err != nil {
			return fmt.Errorf("failed to delete old success record: %w", err)
		}
	}

	return nil
}

// CheckHealth performs a basic health check on the success database
func CheckHealth() error {
	if db == nil {
		return fmt.Errorf("success database not initialized")
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
