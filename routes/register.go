package routes

import (
	"net/http"

	"github.com/cockroachdb/pebble"
)

func RegisterS3Backend(w http.ResponseWriter, r *http.Request) {
	pebble.Open("s3backend", &pebble.Options{})
}
