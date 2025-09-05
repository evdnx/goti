package goti

import (
	"testing"
)

func TestHullMovingAverage(t *testing.T) {
	// Test initialization
	hma, err := NewHullMovingAverageWithParams(9)
	if err != nil {
		t.Fatalf("NewHullMovingAverageWithParams failed: %v", err)
	}

	// Test invalid period
	_, err = NewHullMovingAverageWithParams(0)
	if err == nil || err.Error() != "period must be at least 1" {
		t.Errorf("Expected error for invalid period, got: %v", err)
	}

	// Test adding invalid price
	err = hma.Add(-1)
	if err == nil || err.Error() != "invalid price" {
		t.Errorf("Expected error for invalid price, got: %v", err)
	}

	// Test HMA calculation with sample data (uptrend)
	prices := make([]float64, 20)
	for i := 0; i < 20; i++ {
		prices[i] = float64(100 + i)
	}
	for _, price := range prices {
		hma.Add(price)
	}
	value, err := hma.Calculate()
	if err != nil {
		t.Errorf("Calculate failed: %v", err)
	}
	if value <= 100 {
		t.Errorf("Expected HMA > 100 for uptrend, got: %f", value)
	}

	// Test bullish crossover
	hma.Reset()
	for i := 0; i < 18; i++ {
		hma.Add(float64(100 - i)) // Downtrend
	}
	hma.Add(92)
	hma.Add(93) // Price crosses above HMA
	bullish, err := hma.IsBullishCrossover()
	if err != nil {
		t.Errorf("IsBullishCrossover failed: %v", err)
	}
	if !bullish {
		t.Error("Expected bullish crossover")
	}

	// Test insufficient data
	hma.Reset()
	hma.Add(100)
	_, err = hma.Calculate()
	if err == nil || err.Error() != "no HMA data" {
		t.Errorf("Expected error for no HMA data, got: %v", err)
	}
}
