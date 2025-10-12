package models

type PixerveJWT struct {
	Issuer    string  `json:"iss"` // optional
	Subject   string  `json:"sub"`
	IssuedAt  int64   `json:"iat"`
	ExpiresAt int64   `json:"exp"`
	Job       JobSpec `json:"job"`
}

// Core job specification
type JobSpec struct {
	CompletionCallback string            `json:"completionCallback"` // callback URL
	CallbackHeaders    map[string]string `json:"callbackHeaders,omitempty"`
	Priority           int               `json:"priority"` // 0 = realtime, 1 = queued
	KeepOriginal       bool              `json:"keepOriginal"`

	// Formats requested for conversion
	Formats map[string]FormatSpec `json:"formats"` // e.g., jpg, webp, avif

	// Storage backends — each backend has its own key (random string mapped in PebbleDB)
	StorageKeys map[string]string `json:"storageKeys,omitempty"` // e.g., {"s3":"abc123", "sftp":"def456"}

	// Direct host storage
	DirectHost bool   `json:"directHost,omitempty"` // true if we want to serve via Pixerve HTTP
	SubDir     string `json:"subDir,omitempty"`     // tenant folder or logical subdir
}

// Encoding settings per format
type FormatSpec struct {
	Settings FormatSettings `json:"settings"`
	Sizes    [][]int        `json:"sizes"` // [[W,H],[square]]; single number = square
}

type FormatSettings struct {
	Quality int `json:"quality"` // 1–100
	Speed   int `json:"speed"`   // encoder speed/efficiency tradeoff
}
