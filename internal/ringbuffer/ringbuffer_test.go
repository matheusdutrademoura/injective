package ringbuffer_test

import (
	"testing"
	"time"

	"github.com/matheusdutrademoura/injective/internal/models"
	"github.com/matheusdutrademoura/injective/internal/ringbuffer"
)

// TestAddAndSince tests the Add and Since methods of RingBuffer.
func TestAddAndSince(t *testing.T) {
	ttl := 2 * time.Second
	rb := ringbuffer.NewRingBuffer(3, ttl)

	now := time.Now().UTC()

	// Add 3 updates within the TTL window
	for i := 0; i < 3; i++ {
		rb.Add(models.PriceUpdate{
			Timestamp: now.Add(time.Duration(i) * time.Second),
			Price:     float64(100 + i),
		})
	}

	// Retrieve all updates since zero time, expect 3 entries
	updates := rb.Since(time.Time{})
	if len(updates) != 3 {
		t.Errorf("expected 3 updates, got %d", len(updates))
	}

	// Sleep to expire the TTL for the first update
	time.Sleep(ttl + 100*time.Millisecond)

	// Now only 2 updates should remain valid
	updates = rb.Since(time.Time{})
	if len(updates) != 2 {
		t.Errorf("expected 2 updates after TTL expire, got %d", len(updates))
	}

	// Add 2 more updates to force overwrite in the ring buffer
	for i := 3; i < 5; i++ {
		rb.Add(models.PriceUpdate{
			Timestamp: now.Add(time.Duration(i) * time.Second),
			Price:     float64(100 + i),
		})
	}

	// The buffer capacity is 3, so only 3 most recent updates should remain
	updates = rb.Since(time.Time{})
	if len(updates) != 3 {
		t.Errorf("expected 3 updates after overwrite, got %d", len(updates))
	}

	// Test filtering by timestamp, only updates after now+3s should be returned
	filtered := rb.Since(now.Add(3 * time.Second))
	for _, u := range filtered {
		if u.Timestamp.Before(now.Add(3 * time.Second)) {
			t.Errorf("filtered update timestamp %v is before filter time", u.Timestamp)
		}
	}
}

// TestRingBufferRace checks for race conditions by concurrently adding and reading updates from the RingBuffer.
func TestRingBufferRace(t *testing.T) {
	ttl := 2 * time.Second
	rb := ringbuffer.NewRingBuffer(10, ttl)
	now := time.Now().UTC()
	stop := make(chan struct{})

	// Writer goroutine
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				rb.Add(models.PriceUpdate{
					Timestamp: now,
					Price:     float64(time.Now().UnixNano()),
				})
			}
		}
	}()

	// Reader goroutine
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				_ = rb.Since(now.Add(-time.Second))
			}
		}
	}()

	time.Sleep(500 * time.Millisecond)
	close(stop)
}
