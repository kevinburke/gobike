package main

import (
	"log"
	"net/http"

	"github.com/kevinburke/handlers"
)

func main() {
	fs := http.FileServer(http.Dir("docs"))
	http.Handle("/", handlers.Log(fs))
	log.Println("Listening...")
	log.Fatal(http.ListenAndServe(":8333", nil))
}
