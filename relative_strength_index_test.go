package goti

import (
	"math"
	"testing"
)

func TestRelativeStrengthIndex(t *testing.T) {
	// Test initialization
	config := DefaultConfig()
	rsi, err := NewRelativeStrengthIndexWithParams(5, config)
	if err != nil {
		t.Fatalf("NewRelativeStrengthIndexWithParams failed: %v", err)
	}

	// Test invalid period
	_, err = NewRelativeStrengthIndexWithParams(0, config)
	if err == nil || err.Error() != "period must be at least 1" {
		t.Errorf("Expected error for invalid period, got: %v", err)
	}

	// Test invalid config
	invalidConfig := DefaultConfig()
	invalidConfig.RSIOverbought = 30
	invalidConfig.RSIOversold = 70
	_, err = NewRelativeStrengthIndexWithParams(5, invalidConfig)
	if err == nil || err.Error() != "RSI overbought threshold must be greater than oversold" {
		t.Errorf("Expected error for invalid config, got: %v", err)
	}

	// Test adding invalid price
	err = rsi.Add(-1)
	if err == nil || err.Error() != "invalid price" {
		t.Errorf("Expected error for invalid price, got: %v", err)
	}

	// Test RSI calculation with sample data (all gains)
	prices := make([]float64, 45)
	for i := 0; i < 45; i++ {
		prices[i] = float64(100 + i*10)
	}
	for i, price := range prices {
		err := rsi.Add(price)
		if err != nil {
			t.Errorf("Add failed at index %d: %v", i, err)
		}
	}
	value, err := rsi.Calculate()
	if err != nil {
		t.Errorf("Calculate failed: %v", err)
	}
	if math.Abs(value-100) > 0.01 {
		t.Errorf("Expected RSI near 100, got: %f", value)
	}

	// Test oversold crossover
	rsi.Reset()
	prices = make([]float64, 45)
	for i := 0; i < 40; i++ {
		prices[i] = float64(10000 - i*100)
	}
	prices[40] = 100
	prices[41] = 300
	prices[42] = 600
	prices[43] = 1000
	prices[44] = 1500
	for i, price := range prices {
		err := rsi.Add(price)
		if err != nil {
			t.Errorf("Add failed at index %d: %v", i, err)
		}
		t.Logf("RSI after price %d: %f", i, rsi.GetLastValue())
	}
	bullish, err := rsi.IsBullishCrossover()
	if err != nil {
		t.Errorf("IsBullishCrossover failed: %v", err)
	}
	if !bullish {
		t.Error("Expected bullish crossover")
	}

	// Test divergence (bullish: price up, RSI oversold)
	rsi.Reset()
	prices = make([]float64, 45)
	for i := 0; i < 40; i++ {
		prices[i] = float64(10000 - i*100)
	}
	prices[40] = 100
	prices[41] = 300
	prices[42] = 600
	prices[43] = 1000
	prices[44] = 1500
	for i, price := range prices {
		err := rsi.Add(price)
		if err != nil {
			t.Errorf("Add failed at index %d: %v", i, err)
		}
		t.Logf("RSI after price %d: %f", i, rsi.GetLastValue())
	}
	isDivergence, signal, err := rsi.IsDivergence()
	if err != nil {
		t.Errorf("IsDivergence failed: %v", err)
	}
	if !isDivergence || signal != "Bullish" {
		t.Errorf("Expected bullish divergence, got: %v, %s", isDivergence, signal)
	}

	// Test insufficient data
	rsi.Reset()
	rsi.Add(100)
	_, err = rsi.Calculate()
	if err == nil || err.Error() != "no RSI data" {
		t.Errorf("Expected error for no RSI data, got: %v", err)
	}
}
