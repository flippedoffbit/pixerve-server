package taskqueue

// Backwards-compatible wrapper around the generic DBQueue for the write queue.

var WriteQueue *DBQueue

const WriteQueueDataFile = "WriteQueue.db"

func OpenWriteQueueDB() error {
	q, err := OpenQueue(WriteQueueDataFile)
	if err != nil {
		return err
	}
	WriteQueue = q
	return nil
}

func AddToWriteQueue(key string, value []byte) error {
	return WriteQueue.Add(key, value)
}

func GetFromWriteQueue(key string) ([]byte, error) {
	return WriteQueue.Get(key)
}

func DeleteFromWriteQueue(key string) error {
	return WriteQueue.Delete(key)
}
