package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// PriceFetcher handles fetching BTC price from the CoinDesk API.
type PriceFetcher struct {
	apiKey string
	apiURL string
}

func NewPriceFetcher(apiKey, apiURL string) *PriceFetcher {
	return &PriceFetcher{
		apiKey: apiKey,
		apiURL: apiURL,
	}
}

// Fetch makes a HTTP GET request to the CoinDesk API with a 3-second timeout.
// Using context.WithTimeout prevents hanging requests.
func (pf *PriceFetcher) Fetch() (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(pf.apiURL, pf.apiKey), nil)
	if err != nil {
		return 0, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Data map[string]struct {
			Value float64 `json:"VALUE"`
		} `json:"Data"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return 0, err
	}

	return result.Data["BTC-USD"].Value, nil
}
