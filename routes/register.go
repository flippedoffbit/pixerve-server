package routes

import (
	"net/http"
	"pixerve/logger"

	"github.com/cockroachdb/pebble"
)

type S3Credentials struct {
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	Region          string `json:"region"`
	BucketName      string `json:"bucket_name"`
}

func RegisterS3CredentialsHandler(w http.ResponseWriter, r *http.Request) {
	db, err := pebble.Open("s3credentials.store", &pebble.Options{})

	if err != nil {
		logger.Error("Failed to open database: " + err.Error())
		http.Error(w, "Failed to open database", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	//db.Set()
}
