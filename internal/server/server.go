package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/matheusdutrademoura/injective/internal/client"
	"github.com/matheusdutrademoura/injective/internal/fetcher"
	"github.com/matheusdutrademoura/injective/internal/models"
	"github.com/matheusdutrademoura/injective/internal/ringbuffer"
)

// Server ties all components together and handles HTTP requests
type Server struct {
	clientManager *client.ClientManager
	updateBuffer  *ringbuffer.RingBuffer
	priceFetcher  *fetcher.PriceFetcher
}

func NewServer() *Server {
	coindeskApiKey := os.Getenv("COINDESK_API_KEY")
	if coindeskApiKey == "" {
		log.Fatal("COINDESK_API_KEY env var not set")
	}
	coindeskApiURL := os.Getenv("COINDESK_API_URL")
	if coindeskApiURL == "" {
		log.Fatal("COINDESK_API_URL env var not set")
	}

	return &Server{
		clientManager: client.NewClientManager(),
		updateBuffer:  ringbuffer.NewRingBuffer(300, 5*time.Minute),
		priceFetcher:  fetcher.NewPriceFetcher(coindeskApiKey, coindeskApiURL),
	}
}

// Broadcaster continuously fetches price, stores in buffer, and broadcasts to clients.
// Runs in a separate goroutine.
func (s *Server) Broadcaster() {
	for {
		price, err := s.priceFetcher.Fetch()
		if err != nil {
			log.Printf("error fetching price after retries: %v", err)
			continue
		}

		update := models.PriceUpdate{
			Timestamp: time.Now().UTC(),
			Price:     price,
		}

		s.updateBuffer.Add(update)
		s.clientManager.Broadcast(update)

		time.Sleep(5 * time.Second)
	}
}

// SseHandler handles HTTP SSE connections.
// It streams missed updates based on ?since=timestamp and live updates thereafter.
func (s *Server) SseHandler(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	client := client.NewClientWithBuffer(1)
	s.clientManager.Register(client)

	// Unregister client on connection close or context cancellation
	go func() {
		<-r.Context().Done()
		s.clientManager.Unregister(client)
	}()

	sinceParam := r.URL.Query().Get("since")
	if sinceParam != "" {
		sinceUnix, err := strconv.ParseInt(sinceParam, 10, 64)
		if err == nil {
			sinceTime := time.Unix(sinceUnix, 0).UTC()
			missedUpdates := s.updateBuffer.Since(sinceTime)
			for _, update := range missedUpdates {
				data, _ := json.Marshal(update)
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			}
		}
	}

	for update := range client.Chan {
		data, _ := json.Marshal(update)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
}

func (s *Server) ServeFrontend() http.Handler {
	return http.FileServer(http.Dir(filepath.Join(".", "frontend")))
}
