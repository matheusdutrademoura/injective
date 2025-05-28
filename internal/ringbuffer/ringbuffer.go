package ringbuffer

import (
	"sync"
	"time"

	"github.com/matheusdutrademoura/injective/internal/models"
)

// RingBuffer is a fixed-size circular buffer designed to store recent PriceUpdates with a time-based validity window.
//
// Internally, it uses a slice to store updates and two indexes:
//   - `head`: points to the position where the next update will be written.
//   - `count`: tracks how many valid entries exist in the buffer (<= capacity).
//
// When the buffer reaches capacity, the oldest data is overwritten by advancing the head index circularly.
// This ensures constant memory usage with O(1) insertions.
//
// The `Since()` method provides time-based filtering, returning updates that are:
//   - Not older than the given 'since' timestamp.
//   - Not expired according to the configured TTL (time-to-live).
//
// Use Case:
// Clients connecting via SSE can call `Since(t)` to retrieve missed updates on reconnects.
// The buffer holds only recent data and prevents memory from growing without limits.
//
// Thread safety is ensured using a mutex during reads and writes.

type RingBuffer struct {
	data  []models.PriceUpdate
	head  int
	count int
	ttl   time.Duration
	mutex sync.Mutex
}

func NewRingBuffer(size int, ttl time.Duration) *RingBuffer {
	return &RingBuffer{
		data: make([]models.PriceUpdate, size),
		ttl:  ttl,
	}
}

// Add inserts a new PriceUpdate into the ring buffer.
// If the buffer is full, the oldest entry is overwritten by advancing the head index.
func (rb *RingBuffer) Add(update models.PriceUpdate) {
	rb.mutex.Lock()
	defer rb.mutex.Unlock()

	rb.data[rb.head] = update
	rb.head = (rb.head + 1) % len(rb.data)

	if rb.count < len(rb.data) {
		rb.count++
	}
}

// Since returns all updates with timestamp >= since and that are still valid per TTL.
// This allows clients to fetch missed updates after a reconnect.
// It iterates only over currently stored items (count) and respects circular buffer indexing.
func (rb *RingBuffer) Since(since time.Time) []models.PriceUpdate {
	rb.mutex.Lock()
	defer rb.mutex.Unlock()

	now := time.Now().UTC()

	// Pre-allocate slice with capacity equal to current buffer count to avoid reallocations during appends.
	result := make([]models.PriceUpdate, 0, rb.count)

	// Calculate start index for iteration, considering the ring buffer wrap-around.
	start := (rb.head + len(rb.data) - rb.count) % len(rb.data)

	for i := 0; i < rb.count; i++ {
		idx := (start + i) % len(rb.data)
		u := rb.data[idx]

		// Skip updates older than 'since' parameter.
		if u.Timestamp.Before(since) {
			continue
		}

		// Skip updates expired according to TTL.
		if now.Sub(u.Timestamp) > rb.ttl {
			continue
		}

		result = append(result, u)
	}

	return result
}
