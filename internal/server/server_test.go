package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/matheusdutrademoura/injective/internal/models"
)

// mockFlusherWriter implements http.ResponseWriter + http.Flusher for testing SSE
type mockFlusherWriter struct {
	httptest.ResponseRecorder
	flushed bool
}

func (m *mockFlusherWriter) Flush() {
	m.flushed = true
}

// TestSseHandlerBasic tests that SseHandler writes proper SSE events including missed updates.
func TestSseHandlerBasic(t *testing.T) {
	os.Setenv("COINDESK_API_KEY", "dummy")
	os.Setenv("COINDESK_API_URL", "http://localhost:9999/%s") // ou algo mockado

	s := NewServer()

	// Add some updates to buffer with timestamps in the past
	now := time.Now().UTC()
	s.updateBuffer.Add(models.PriceUpdate{Timestamp: now.Add(-10 * time.Second), Price: 100.0})
	s.updateBuffer.Add(models.PriceUpdate{Timestamp: now.Add(-5 * time.Second), Price: 200.0})

	// Create a request with since param to get missed updates
	req := httptest.NewRequest("GET", "/stream?since="+strconv.FormatInt(now.Add(-15*time.Second).Unix(), 10), nil)
	w := &mockFlusherWriter{ResponseRecorder: *httptest.NewRecorder()}

	// Create a cancellable context to stop the goroutine after test
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req = req.WithContext(ctx)

	// Run SseHandler in a goroutine since it listens on channel indefinitely
	done := make(chan struct{})
	go func() {
		s.SseHandler(w, req)
		close(done)
	}()

	// Wait a bit to let missed events be sent
	time.Sleep(100 * time.Millisecond)

	// Cancel the context to close client and end handler
	cancel()
	<-done

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d", resp.StatusCode)
	}

	// Check Content-Type header
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Expected Content-Type text/event-stream, got %q", ct)
	}

	// Check that response body contains missed updates as SSE data lines
	body := w.Body.String()
	if !strings.Contains(body, "data:") {
		t.Errorf("Expected SSE data events in response body, got %q", body)
	}

	// Check that flushed was called
	if !w.flushed {
		t.Errorf("Expected Flush to be called on ResponseWriter")
	}
}
