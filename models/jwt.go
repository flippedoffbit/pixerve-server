package models

// --- Top-level JWT structure ---

type PixerveJWT struct {
	Issuer    string  `json:"iss"` // e.g. "pixerve.issuer.service"
	Subject   string  `json:"sub"` // "upload-job"
	IssuedAt  int64   `json:"iat"`
	ExpiresAt int64   `json:"exp"`
	Job       JobSpec `json:"job"`
}

// --- Job definition (core of Pixerve spec) ---

type JobSpec struct {
	CompletionCallback string                `json:"completionCallback"`
	CallbackHeaders    map[string]string     `json:"callbackHeaders,omitempty"`
	Priority           int                   `json:"priority"`     // 0 = realtime, 1 = queued
	KeepOriginal       bool                  `json:"keepOriginal"` // keep the uploaded original
	Formats            map[string]FormatSpec `json:"formats"`      // e.g. "jpg", "webp"
	Storage            StorageSpec           `json:"storage"`
}

// --- Image format specifications ---

type FormatSpec struct {
	Settings FormatSettings `json:"settings"`
	Sizes    [][]int        `json:"sizes"` // [[128,256], [256]] — interpreted as [256,256]
}

type FormatSettings struct {
	Quality int `json:"quality"` // 1–100 for lossy formats
	Speed   int `json:"speed"`   // encoder speed/efficiency tradeoff
}

// --- Storage configuration ---

type StorageSpec struct {
	DirectHost    bool          `json:"directHost"`              // serve via Pixerve HTTP
	DirectHostDir string        `json:"directHostDir,omitempty"` // root for direct hosting
	Local         *LocalStorage `json:"local,omitempty"`         // optional local storage
	S3            *S3Storage    `json:"s3,omitempty"`            // optional S3 storage
	SFTP          *SFTPStorage  `json:"sftp,omitempty"`          // optional SFTP storage
}

type LocalStorage struct {
	Path    string `json:"path"`    // filesystem path
	URLBase string `json:"urlBase"` // base public URL (for callback responses)
}

type S3Storage struct {
	Key    string `json:"key"`
	Bucket string `json:"bucket"`
	Dir    string `json:"dir"`
}

type SFTPStorage struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Path     string `json:"path"`
}

// --- Optional helper for normalized defaults ---

func (s *StorageSpec) Normalize() {
	if s.DirectHost && s.DirectHostDir == "" {
		s.DirectHostDir = "./uploads"
	}
	if s.Local != nil && s.Local.Path == "" {
		s.Local.Path = s.DirectHostDir
	}
}
