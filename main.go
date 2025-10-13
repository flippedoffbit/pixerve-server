package main

import (
	"fmt"
	"net/http"
	"pixerve/logger"
	"pixerve/routes"

	pebble "github.com/cockroachdb/pebble"
)

func main() {
	http.HandleFunc("/upload", routes.UploadHandler)
	http.ListenAndServe(":8080", nil)

	db, err := pebble.Open("demo", &pebble.Options{})
	if err != nil {
		logger.Fatal(err)
	}
	key := []byte("hello")
	if err := db.Set(key, []byte("world"), pebble.Sync); err != nil {
		logger.Fatal(err)
	}
	value, closer, err := db.Get(key)
	if err != nil {
		logger.Fatal(err)
	}
	fmt.Printf("%s %s\n", key, value)
	if err := closer.Close(); err != nil {
		logger.Fatal(err)
	}
	if err := db.Close(); err != nil {
		logger.Fatal(err)
	}

}
