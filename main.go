package main

import (
	"net/http"
	"pixerve/routes"
)

func main() {
	http.HandleFunc("/upload", routes.UploadHandler)
	http.ListenAndServe(":8080", nil)
}
