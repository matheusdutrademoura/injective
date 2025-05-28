package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/matheusdutrademoura/injective/internal/server"
)

func main() {
	injectiveServer := server.NewServer()
	go injectiveServer.Broadcaster()

	http.HandleFunc("/stream", injectiveServer.SseHandler)
	http.Handle("/", injectiveServer.ServeFrontend())

	// Create the HTTP server with a timeout-aware configuration.
	httpServer := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Println("Frontend available at http://localhost:8080/")
	log.Println("SSE stream available at http://localhost:8080/stream")

	// Start server in a goroutine so we can shut it down gracefully later.
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Listen for system interrupts to perform graceful shutdown.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	<-stop // wait for interrupt

	log.Println("Shutting down server...")

	// Give active connections time to finish.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}
