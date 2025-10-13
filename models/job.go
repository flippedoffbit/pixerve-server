package models

type WriterJob struct {
	Type        string            // "s3" or "local"
	Credentials map[string]string // everything else, each write destination has different credentials and own write implimentatons
}

type ConversionJob struct {
	Encoder       string // encoder name
	Length, Width int    // dimensions
	Quality       int    // 1–100
	Speed         int    // encoder speed/efficiency tradeoff
}
