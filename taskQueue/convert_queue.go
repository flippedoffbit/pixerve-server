package taskqueue

import (
	"os"
	"path/filepath"
)

// Backwards-compatible wrapper around the generic DBQueue for the convert queue.

var ConvertQueue *DBQueue

// getConvertQueueDataFile returns the path to the convert queue database file
func getConvertQueueDataFile() string {
	dataDir := os.Getenv("PIXERVE_DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}
	return filepath.Join(dataDir, "ConvertQueue.db")
}

func OpenConvertQueueDB() error {
	q, err := OpenQueue(getConvertQueueDataFile())
	if err != nil {
		return err
	}
	ConvertQueue = q
	return nil
}

func AddToConvertQueue(key string, value []byte) error {
	return ConvertQueue.Add(key, value)
}

func GetFromConvertQueue(key string) ([]byte, error) {
	return ConvertQueue.Get(key)
}

func DeleteFromConvertQueue(key string) error {
	return ConvertQueue.Delete(key)
}
