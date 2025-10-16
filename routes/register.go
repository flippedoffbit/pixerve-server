package routes

import (
	"encoding/json"
	"net/http"
	"pixerve/config"
	"pixerve/credentials"
	"pixerve/logger"
	"pixerve/utils"
)

type S3Credentials struct {
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	Region          string `json:"region"`
	BucketName      string `json:"bucket_name"`
}

func OpenCredentialsDB() error {
	return credentials.OpenDB(config.GetCredentialsDBPath())
}

func DeregisterCredentialsHandler(w http.ResponseWriter, r *http.Request) {
	logger.Debugf("Deregister credentials request: method=%s, remoteAddr=%s", r.Method, r.RemoteAddr)

	if r.Method != http.MethodDelete {
		logger.Warnf("Invalid method for deregister endpoint: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	keyString := r.URL.Query().Get("access_key")

	if keyString == "" {
		logger.Warn("Missing access_key parameter in deregister request")
		http.Error(w, "Missing access_key parameter", http.StatusBadRequest)
		return
	}

	logger.Infof("Attempting to delete credentials for access key: %s", keyString)
	err := credentials.DeleteCredentials(keyString)

	if err != nil {
		logger.Errorf("Failed to delete credentials for key %s: %v", keyString, err)
		http.Error(w, "Failed to delete credentials", http.StatusInternalServerError)
		return
	}

	logger.Infof("Credentials deleted successfully for access key: %s", keyString)
	w.WriteHeader(http.StatusNoContent)
}

func RegisterCredentialsHandler(w http.ResponseWriter, r *http.Request) {
	logger.Debugf("Register credentials request: method=%s, remoteAddr=%s", r.Method, r.RemoteAddr)

	if r.Method != http.MethodPost {
		logger.Warnf("Invalid method for register endpoint: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	logger.Debug("Generating access key for new credentials")
	keyString, err := utils.GenerateRandomHex(16)

	if err != nil {
		logger.Errorf("Failed to generate access key: %v", err)
		http.Error(w, "Failed to generate key", http.StatusInternalServerError)
		return
	}

	logger.Debugf("Generated access key: %s", keyString)

	credsBody := make(map[string]string, 0)

	logger.Debug("Decoding request body for credentials")
	err = json.NewDecoder(r.Body).Decode(&credsBody)

	if err != nil {
		logger.Errorf("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	logger.Infof("Storing credentials for access key: %s", keyString)
	err = credentials.StoreCredentials(keyString, credsBody)

	if err != nil {
		logger.Errorf("Failed to store credentials for key %s: %v", keyString, err)
		http.Error(w, "Failed to store credentials", http.StatusInternalServerError)
		return
	}

	logger.Infof("Credentials stored successfully for access key: %s", keyString)

	response := map[string]string{
		"access_key": keyString,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Errorf("Failed to encode register response: %v", err)
		return
	}
	logger.Debug("Register credentials request completed successfully")
}
