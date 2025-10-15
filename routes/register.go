package routes

import (
	"encoding/json"
	"net/http"
	"pixerve/credentials"
	"pixerve/utils"
)

type S3Credentials struct {
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	Region          string `json:"region"`
	BucketName      string `json:"bucket_name"`
}

func OpenCredentialsDB() error {
	return credentials.OpenDB()
}

func DeregisterCredentialsHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	keyString := r.URL.Query().Get("access_key")

	if keyString == "" {
		http.Error(w, "Missing access_key parameter", http.StatusBadRequest)
		return
	}

	err := credentials.DeleteCredentials(keyString)

	if err != nil {
		http.Error(w, "Failed to delete credentials", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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

	err = credentials.StoreCredentials(keyString, credsBody)

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
