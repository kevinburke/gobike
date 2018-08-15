package main

import (
	"log"
	"net/http"

	"github.com/kevinburke/handlers"
)

func main() {
	fs := http.FileServer(http.Dir("docs"))
	http.Handle("/", handlers.Log(fs))
	handlers.Logger.Info("Starting server", "port", 8333, "protocol", "http")
	log.Fatal(http.ListenAndServe(":8333", nil))
}
