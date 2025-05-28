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

const (
	updateInterval   = 5 * time.Second                     // How often we fetch new price data
	historyWindow    = 1 * time.Hour                       // How much historical data we want to keep with fixed memory usage
	maxBufferEntries = int(historyWindow / updateInterval) // 3600s / 5s = 720 entries
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
		priceFetcher:  fetcher.NewPriceFetcher(coindeskApiKey, coindeskApiURL),
		updateBuffer:  ringbuffer.NewRingBuffer(maxBufferEntries, historyWindow),
	}
}

// Broadcaster runs in a goroutine, continuously fetching prices,
// storing them in the ring buffer, and broadcasting to all clients.
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

		time.Sleep(updateInterval)
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

	// We use a buffer size of 1 to avoid blocking the broadcaster on slow clients.
	// If the client is too slow to consume updates, the connection will be dropped.
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
		if err != nil {
			log.Printf("invalid 'since' param: %v", err)
			return
		}

		sinceTime := time.Unix(sinceUnix, 0).UTC()
		missedUpdates := s.updateBuffer.Since(sinceTime)
		for _, update := range missedUpdates {
			data, _ := json.Marshal(update)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}

	}

	for update := range client.Chan {
		data, _ := json.Marshal(update)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
}

// ServeFrontend serves static files from the ./frontend directory.
func (s *Server) ServeFrontend() http.Handler {
	return http.FileServer(http.Dir(filepath.Join(".", "frontend")))
}
