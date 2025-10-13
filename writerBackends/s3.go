package writerbackends

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"pixerve/logger"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// uploadToS3WithCreds uploads content from an io.Reader to an S3 object
// and is fully self-contained, initializing its own client.
func UploadToS3WithCreds(ctx context.Context, accessInfo map[string]string, reader io.Reader) error {
	// Create a credentials provider from the provided keys.
	creds := credentials.NewStaticCredentialsProvider(accessInfo["accessKey"], accessInfo["secretKey"], "")
	key := accessInfo["key"]
	bucket := accessInfo["bucket"]
	// Create a new S3 client with the specific credentials and region.
	s3Client := s3.New(s3.Options{
		Region:      accessInfo["region"],
		Credentials: creds,
	})

	// Create an S3 Uploader instance.
	uploader := manager.NewUploader(s3Client)

	// Perform the upload.
	_, err := uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   reader,
	})

	if err != nil {
		return fmt.Errorf("failed to upload object %s to bucket %s: %w", key, bucket, err)
	}

	logger.Infof("Successfully uploaded object '%s' to bucket '%s'", key, bucket)
	return nil
}

func UseUploadToS3WithCredsExample() {
	// Replace with your actual credentials and bucket details.
	// NOTE: Hardcoding credentials is not recommended for production.
	// Use a secure secret management system instead.
	myAccessKey := "YOUR_AWS_ACCESS_KEY_ID"
	mySecretKey := "YOUR_AWS_SECRET_ACCESS_KEY"
	myRegion := "us-east-1"
	myBucket := "your-unique-bucket-name"
	myKey := "example-data.txt"
	content := "This is some data to upload to S3."

	accessInfo := map[string]string{
		"accessKey": myAccessKey,
		"secretKey": mySecretKey,
		"region":    myRegion,
		"bucket":    myBucket,
		"key":       myKey,
	}

	// Create a reader from your content.
	reader := bytes.NewReader([]byte(content))

	// Call the self-contained upload function.
	err := UploadToS3WithCreds(context.TODO(), accessInfo, reader)
	if err != nil {
		logger.Fatal(err)
	}
}
