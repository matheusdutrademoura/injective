package fetcher_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matheusdutrademoura/injective/internal/fetcher"
)

// TestFetchSuccess tests that Fetch returns the correct price on a successful response.
func TestFetchSuccess(t *testing.T) {
	// Setup a mock HTTP server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"Data": {
				"BTC-USD": {
					"VALUE": 45000.55
				}
			}
		}`))
	}))
	defer mockServer.Close()

	pf := fetcher.NewPriceFetcher("dummy-api-key", mockServer.URL+"?apikey=%s")

	price, err := pf.Fetch()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := 45000.55
	if price != expected {
		t.Errorf("expected price %.2f, got %.2f", expected, price)
	}
}

// TestFetchMalformedJSON tests Fetch behavior when JSON response is malformed.
func TestFetchMalformedJSON(t *testing.T) {
	// Mock server returns invalid JSON
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json}`))
	}))
	defer mockServer.Close()

	pf := fetcher.NewPriceFetcher("dummy-api-key", mockServer.URL+"?apikey=%s")

	_, err := pf.Fetch()
	if err == nil {
		t.Fatal("expected error due to malformed JSON, got nil")
	}
}

// TestFetchHTTPError tests Fetch behavior when HTTP response status is not 200 OK.
func TestFetchHTTPError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	pf := fetcher.NewPriceFetcher("dummy-api-key", mockServer.URL+"?apikey=%s")

	_, err := pf.Fetch()
	if err == nil {
		t.Fatal("expected error due to HTTP error status, got nil")
	}
}
