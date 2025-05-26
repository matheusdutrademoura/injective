package models

import "time"

// PriceUpdate represents a BTC price at a specific time
type PriceUpdate struct {
	Timestamp time.Time `json:"timestamp"`
	Price     float64   `json:"price"`
}
