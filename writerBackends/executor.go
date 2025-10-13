package writerbackends

import (
	"context"
	"fmt"
	"io"
)

func WriteImage(ctx context.Context, accessInfo map[string]string, reader io.Reader, backendType string) error {
	// Implementation for writing an image
	// we will switch based on the backend type, e.g., directServe, s3, gcs, sftp (files in same dir)
	switch backendType {
	case "directServe":
		err := UploadToDirectServe(ctx, accessInfo, reader)
		if err != nil {
			return fmt.Errorf("failed to upload to direct serve: %w", err)
		}
	case "s3":
		err := UploadToS3WithCreds(ctx, accessInfo, reader)
		if err != nil {
			return fmt.Errorf("failed to upload to S3: %w", err)
		}
	case "gcs":
		err := UploadToGCSWithJSON(ctx, accessInfo, reader)
		if err != nil {
			return fmt.Errorf("failed to upload to GCS: %w", err)
		}
	case "sftp":
		err := UploadToSFTPWithCreds(ctx, accessInfo, reader)
		if err != nil {
			return fmt.Errorf("failed to upload to SFTP: %w", err)
		}
	default:
		return fmt.Errorf("unknown backend type: %s", backendType)
	}
	return nil
}
