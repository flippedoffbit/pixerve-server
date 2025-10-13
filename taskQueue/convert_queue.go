package taskqueue

// Backwards-compatible wrapper around the generic DBQueue for the convert queue.

var ConvertQueue *DBQueue

const ConvertQueueDataFile = "ConvertQueue.db"

func OpenConvertQueueDB() error {
	q, err := OpenQueue(ConvertQueueDataFile)
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
