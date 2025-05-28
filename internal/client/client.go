package client

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/matheusdutrademoura/injective/internal/models"
)

// Client represents a connected client receiving price updates via a channel.
type Client struct {
	ID   string
	Chan chan models.PriceUpdate
}

// NewClientWithBuffer creates a client with a buffered channel of the specified size.
// Buffering prevents slow clients from blocking the broadcaster immediately.
func NewClientWithBuffer(buffer int) *Client {
	return &Client{Chan: make(chan models.PriceUpdate, buffer)}
}

// ClientManager manages concurrent access to the map of connected clients.
type ClientManager struct {
	clients map[*Client]bool
	mutex   sync.Mutex
}

// clientCounter atomically generates unique client IDs for logging and identification.
var clientCounter int64

func NewClientManager() *ClientManager {
	return &ClientManager{clients: make(map[*Client]bool)}
}

// Register adds a new client to the manager, assigning it a unique ID.
// Uses atomic operations to safely generate IDs in concurrent environment.
func (cm *ClientManager) Register(c *Client) {
	c.ID = fmt.Sprintf("client-%d", atomic.AddInt64(&clientCounter, 1))
	cm.mutex.Lock()
	cm.clients[c] = true
	cm.mutex.Unlock()
	log.Printf("--> registered [%s]", c.ID)
}

// Unregister removes the client and closes its channel to signal disconnection.
// Closing the channel allows goroutines receiving from it to exit gracefully.
func (cm *ClientManager) Unregister(c *Client) {
	cm.mutex.Lock()
	_, exists := cm.clients[c]
	if exists {
		delete(cm.clients, c)
		log.Printf("<-- unregistered [%s]", c.ID)
	}
	cm.mutex.Unlock()
	close(c.Chan)
}

// Broadcast sends the price update to all registered clients.
// It uses non-blocking sends to avoid blocking the entire system on slow or stuck clients.
// Clients whose channel buffer is full are considered slow and are removed to maintain overall system health.
// Slow clients are collected while holding the lock and removed after releasing the lock to avoid deadlocks.
func (cm *ClientManager) Broadcast(update models.PriceUpdate) {
	cm.mutex.Lock()
	var slowClients []*Client

	for client := range cm.clients {
		select {
		case client.Chan <- update:
			// Successfully sent update
		default:
			// Client channel full; mark client for removal
			log.Printf("[!] dropping client [%s]", client.ID)
			slowClients = append(slowClients, client)
		}
	}
	cm.mutex.Unlock()

	// Unregister slow clients outside the lock to avoid deadlocks.
	for _, c := range slowClients {
		cm.Unregister(c)
	}
}
