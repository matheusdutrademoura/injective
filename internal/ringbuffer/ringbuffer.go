package ringbuffer

import (
	"sync"
	"time"

	"github.com/matheusdutrademoura/injective/internal/models"
)

// RingBuffer stores a fixed-size buffer of PriceUpdates with TTL expiration.
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

// Add inserts a new PriceUpdate, advancing the head if buffer is full.
func (rb *RingBuffer) Add(update models.PriceUpdate) {
	rb.mutex.Lock()
	defer rb.mutex.Unlock()

	idx := (rb.head + rb.count) % len(rb.data)
	rb.data[idx] = update

	if rb.count < len(rb.data) {
		rb.count++
	} else {
		rb.head = (rb.head + 1) % len(rb.data)
	}
}

// Since returns all PriceUpdates since the given timestamp that are still valid under TTL.
func (rb *RingBuffer) Since(since time.Time) []models.PriceUpdate {
	rb.mutex.Lock()
	defer rb.mutex.Unlock()

	var result []models.PriceUpdate
	now := time.Now().UTC()

	for i := 0; i < rb.count; i++ {
		idx := (rb.head + i) % len(rb.data)
		u := rb.data[idx]

		if u.Timestamp.Before(since) {
			continue
		}
		if now.Sub(u.Timestamp) > rb.ttl {
			continue
		}
		result = append(result, u)
	}

	return result
}
