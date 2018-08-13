package main

import (
	"log"
	"net/http"
)

func main() {
	fs := http.FileServer(http.Dir("docs"))
	http.Handle("/", fs)
	log.Println("Listening...")
	log.Fatal(http.ListenAndServe(":8333", nil))
}
