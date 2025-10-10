package main

import (
	"fmt"
	"log"
	"net/http"
	"pixerve/routes"

	pebble "github.com/cockroachdb/pebble"
)

func main() {
	http.HandleFunc("/upload", routes.UploadHandler)
	http.ListenAndServe(":8080", nil)

	db, err := pebble.Open("demo", &pebble.Options{})
	if err != nil {
		log.Fatal(err)
	}
	key := []byte("hello")
	if err := db.Set(key, []byte("world"), pebble.Sync); err != nil {
		log.Fatal(err)
	}
	value, closer, err := db.Get(key)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s %s\n", key, value)
	if err := closer.Close(); err != nil {
		log.Fatal(err)
	}
	if err := db.Close(); err != nil {
		log.Fatal(err)
	}

}
