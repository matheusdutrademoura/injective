package main

import (
	"log"
	"net/http"

	"github.com/matheusdutrademoura/injective/internal/server"
)

func main() {
	server := server.NewServer()
	go server.Broadcaster()

	http.HandleFunc("/stream", server.SseHandler)
	http.Handle("/", server.ServeFrontend())

	log.Println("Frontend started at http://localhost:8080/")
	log.Println("Stream started at http://localhost:8080/stream")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
