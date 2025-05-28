package client

import (
	"testing"
	"time"

	"github.com/matheusdutrademoura/injective/internal/models"
)

// TestClientManager_RegisterAndUnregister verifies that clients can be registered and unregistered properly.
func TestClientManager_RegisterAndUnregister(t *testing.T) {
	cm := NewClientManager()

	// Create new client and register
	c := NewClientWithBuffer(1)
	cm.Register(c)

	if c.ID == "" {
		t.Errorf("expected client ID to be set")
	}

	// Check if client is in manager
	cm.mutex.Lock()
	_, exists := cm.clients[c]
	cm.mutex.Unlock()
	if !exists {
		t.Errorf("client was not registered properly")
	}

	// Unregister client
	cm.Unregister(c)

	// Check if client was removed
	cm.mutex.Lock()
	_, exists = cm.clients[c]
	cm.mutex.Unlock()
	if exists {
		t.Errorf("client was not unregistered properly")
	}

	// Check if channel is closed (sending on closed channel panics)
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic when sending on closed channel")
		}
	}()
	c.Chan <- models.PriceUpdate{} // should panic because channel is closed
}

// TestBroadcast_SendsToClients tests that Broadcast sends updates to registered clients.
func TestBroadcast_SendsToClients(t *testing.T) {
	cm := NewClientManager()

	// Create and register clients
	client1 := NewClientWithBuffer(1)
	client2 := NewClientWithBuffer(1)
	cm.Register(client1)
	cm.Register(client2)

	update := models.PriceUpdate{
		Price:     123.45,
		Timestamp: time.Now().UTC(),
	}

	// Broadcast update
	cm.Broadcast(update)

	// Verify clients receive the update
	select {
	case msg := <-client1.Chan:
		if msg.Price != update.Price {
			t.Errorf("client1 received wrong price")
		}
	case <-time.After(time.Second):
		t.Errorf("client1 did not receive broadcast")
	}

	select {
	case msg := <-client2.Chan:
		if msg.Price != update.Price {
			t.Errorf("client2 received wrong price")
		}
	case <-time.After(time.Second):
		t.Errorf("client2 did not receive broadcast")
	}
}

// TestBroadcast_DropsSlowClients tests that slow clients with full channels are dropped.
func TestBroadcast_DropsSlowClients(t *testing.T) {
	cm := NewClientManager()

	// Create client with very small channel buffer to simulate slow client
	c := &Client{
		Chan: make(chan models.PriceUpdate, 1),
	}
	cm.Register(c)

	// Fill the channel to block send
	c.Chan <- models.PriceUpdate{}

	update := models.PriceUpdate{
		Price:     999.99,
		Timestamp: time.Now().UTC(),
	}

	// Broadcast should drop the slow client and unregister it
	cm.Broadcast(update)

	// Check client was removed
	cm.mutex.Lock()
	_, exists := cm.clients[c]
	cm.mutex.Unlock()
	if exists {
		t.Errorf("slow client was not dropped")
	}

	// Channel should be closed
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic when sending on closed channel")
		}
	}()
	c.Chan <- update // sending to closed channel should panic
}

// TestClientManagerRace checks for race conditions by concurrently registering and unregistering clients.
func TestClientManagerRace(t *testing.T) {
	cm := NewClientManager()
	stop := make(chan struct{})

	// Goroutine to register clients
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				c := NewClientWithBuffer(1)
				cm.Register(c)
			}
		}
	}()

	// Goroutine to unregister clients
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				cm.mutex.Lock()
				for c := range cm.clients {
					cm.mutex.Unlock()
					cm.Unregister(c)
					cm.mutex.Lock()
				}
				cm.mutex.Unlock()
			}
		}
	}()

	time.Sleep(500 * time.Millisecond)
	close(stop)
}
