package client

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/matheusdutrademoura/injective/internal/models"
)

// Client represents a connected client receiving updates
type Client struct {
	ID   string
	Chan chan models.PriceUpdate
}

// NewClientWithBuffer creates a client with a buffered channel.
func NewClientWithBuffer(buffer int) *Client {
	return &Client{Chan: make(chan models.PriceUpdate, buffer)}
}

// keep the original NewClient for convenience with unbuffered channel if needed
func NewClient() *Client {
	return NewClientWithBuffer(0)
}

// ClientManager manages all connected clients safely
type ClientManager struct {
	clients map[*Client]bool
	mutex   sync.Mutex
}

var clientCounter int64

func NewClientManager() *ClientManager {
	return &ClientManager{clients: make(map[*Client]bool)}
}

// Register adds a client to the manager.
// ID is generated atomically to uniquely identify clients in logs.
func (cm *ClientManager) Register(c *Client) {
	c.ID = fmt.Sprintf("client-%d", atomic.AddInt64(&clientCounter, 1))
	cm.mutex.Lock()
	cm.clients[c] = true
	cm.mutex.Unlock()
	log.Printf("--> registered [%s]", c.ID)
}

// Unregister removes a client and closes its channel.
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

// Broadcast sends the update to all clients without blocking.
// Clients whose channel is full are considered slow and removed.
// To avoid deadlock, slow clients are collected first and removed after releasing the mutex.
func (cm *ClientManager) Broadcast(update models.PriceUpdate) {
	cm.mutex.Lock()
	var slowClients []*Client

	for client := range cm.clients {
		select {
		case client.Chan <- update:
			// sent successfully
		default:
			// client channel full, likely stuck or too slow
			log.Printf("[!] dropping client [%s]", client.ID)
			slowClients = append(slowClients, client)
		}
	}
	cm.mutex.Unlock()

	// Unregister slow clients outside the lock to avoid deadlock
	for _, c := range slowClients {
		cm.Unregister(c)
	}
}
