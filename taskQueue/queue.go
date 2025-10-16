package taskqueue

import (
	"fmt"

	"github.com/cockroachdb/pebble"
)

// DBQueue is a small wrapper around a Pebble DB instance used by the task queues.
type DBQueue struct {
	DB       *pebble.DB
	DataFile string
}

// OpenQueue opens (or creates) a pebble DB at the given dataFile path and
// returns a DBQueue wrapper.
func OpenQueue(dataFile string) (*DBQueue, error) {
	db, err := pebble.Open(dataFile, &pebble.Options{})
	if err != nil {
		return nil, err
	}
	return &DBQueue{DB: db, DataFile: dataFile}, nil
}

// Add stores a value under the given key.
func (q *DBQueue) Add(key string, value []byte) error {
	return q.DB.Set([]byte(key), value, pebble.Sync)
}

// Get returns the value for the given key. The returned bytes are owned by
// Pebble and should not be mutated by callers.
func (q *DBQueue) Get(key string) ([]byte, error) {
	value, closer, err := q.DB.Get([]byte(key))
	if err != nil {
		return nil, err
	}
	defer closer.Close()
	return value, nil
}

// Delete removes the key from the DB.
func (q *DBQueue) Delete(key string) error {
	return q.DB.Delete([]byte(key), pebble.Sync)
}

// Close closes the underlying DB.
func (q *DBQueue) Close() error {
	return q.DB.Close()
}

// CheckHealth performs a basic health check on the queue system
func CheckHealth() error {
	if ConvertQueue == nil {
		return fmt.Errorf("convert queue not initialized")
	}

	// Try a simple operation to verify database is accessible
	_, closer, err := ConvertQueue.DB.Get([]byte("__health_check__"))
	if err != nil && err != pebble.ErrNotFound {
		return fmt.Errorf("queue database health check failed: %w", err)
	}
	if closer != nil {
		closer.Close()
	}
	return nil
}
