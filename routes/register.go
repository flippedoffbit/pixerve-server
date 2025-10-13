package routes

import (
	"encoding/json"
	"net/http"
	"pixerve/logger"
	"pixerve/utils"

	"github.com/cockroachdb/pebble"
)

var db *pebble.DB

const credentialsDataFile = "CredentialsData.db"

type S3Credentials struct {
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	Region          string `json:"region"`
	BucketName      string `json:"bucket_name"`
}

func OpenDB() error {

	var err error
	db, err = pebble.Open(credentialsDataFile, &pebble.Options{})
	if err != nil {
		logger.Fatalf("Failed to open Pebble DB: %v", err)
		return err
	}
	return nil
}

func RegisterCredentialsHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	keyString, err := utils.GenerateRandomHex(16)

	if err != nil {
		http.Error(w, "Failed to generate key", http.StatusInternalServerError)
		return
	}

	credsBody := make(map[string]string, 0)

	err = json.NewDecoder(r.Body).Decode(&credsBody)

	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	encodedCreds, err := json.Marshal(credsBody)

	if err != nil {
		http.Error(w, "Failed to encode credentials", http.StatusInternalServerError)
		return
	}

	err = db.Set([]byte(keyString), encodedCreds, pebble.Sync)

	if err != nil {
		http.Error(w, "Failed to store credentials", http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"access_key": keyString,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
