package writerbackends

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"

	"pixerve/logger"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

// uploadToGCSWithJSON uploads content from an io.Reader to a Google Cloud Storage object,
// using a service account key provided as a byte slice.

func UploadToGCSWithJSON(ctx context.Context, accessInfo map[string]string, reader io.Reader) error {
	// Decode base64 credentials
	credentialsJSON, err := base64.RawStdEncoding.DecodeString(accessInfo["credentialsJSON"])
	if err != nil {
		return fmt.Errorf("failed to decode base64 credentials: %w", err)
	}

	bucketName := accessInfo["bucket"]
	objectName := accessInfo["object"]
	client, err := storage.NewClient(ctx, option.WithCredentialsJSON(credentialsJSON))
	if err != nil {
		return fmt.Errorf("storage.NewClient: %w", err)
	}
	defer client.Close()

	// Get a handle to the bucket and object.
	bucket := client.Bucket(bucketName)
	obj := bucket.Object(objectName)

	// Create a writer to stream the data to the object.
	wc := obj.NewWriter(ctx)

	// Copy the content from the reader to the writer.
	if _, err = io.Copy(wc, reader); err != nil {
		return fmt.Errorf("io.Copy: %w", err)
	}

	// Close the writer to complete the upload.
	if err := wc.Close(); err != nil {
		return fmt.Errorf("Writer.Close: %w", err)
	}

	logger.Infof("Successfully uploaded object '%s' to bucket '%s'", objectName, bucketName)
	return nil
}
