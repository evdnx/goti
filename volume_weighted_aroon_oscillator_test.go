// volume_weighted_aroon_oscillator_test.go
// artifact_id: b9c0d1e2-2f3a-4b5c-6d7e-8f9a0b1c2d3e
// artifact_version_id: 7f0a1b2c-3d4e-4f5a-6b7c-8d9e0f1a2b3c

package goti

import (
	"math"
	"testing"
)

func TestVolumeWeightedAroonOscillator(t *testing.T) {
	// Test initialization
	config := DefaultConfig()
	vwao, err := NewVolumeWeightedAroonOscillatorWithParams(14, config)
	if err != nil {
		t.Fatalf("NewVolumeWeightedAroonOscillatorWithParams failed: %v", err)
	}

	// Test invalid period
	_, err = NewVolumeWeightedAroonOscillatorWithParams(0, config)
	if err == nil || err.Error() != "period must be at least 1" {
		t.Errorf("Expected error for invalid period, got: %v", err)
	}

	// Test adding invalid data
	err = vwao.Add(100, 101, 100, 1000)
	if err == nil || err.Error() != "invalid price or volume" {
		t.Errorf("Expected error for high < low, got: %v", err)
	}
	err = vwao.Add(101, 100, -1, 1000)
	if err == nil || err.Error() != "invalid price or volume" {
		t.Errorf("Expected error for invalid close, got: %v", err)
	}
	err = vwao.Add(101, 100, 100, -1)
	if err == nil || err.Error() != "invalid price or volume" {
		t.Errorf("Expected error for invalid volume, got: %v", err)
	}

	// Test VWAO calculation with sample data (all gains)
	for i := 0; i < 50; i++ {
		high := float64(100 + i*400 + 1)
		low := float64(100 + i*400 - 1)
		close := float64(100 + i*400)
		volume := 1000000.0
		err := vwao.Add(high, low, close, volume)
		if err != nil {
			t.Errorf("Add failed at index %d: %v", i, err)
		}
	}
	value, err := vwao.Calculate()
	if err != nil {
		t.Errorf("Calculate failed: %v", err)
	}
	if math.Abs(value-100) > 0.01 {
		t.Errorf("Expected VWAO near 100, got: %f", value)
	}

	// Test strong trend
	isStrong, err := vwao.IsStrongTrend()
	if err != nil {
		t.Errorf("IsStrongTrend failed: %v", err)
	}
	if !isStrong {
		t.Error("Expected strong trend")
	}

	// Test bullish crossover
	vwao.Reset()
	prices := make([]float64, 50)
	for i := 0; i < 45; i++ {
		prices[i] = float64(10000 - i*100)
	}
	prices[45] = 100
	prices[46] = 300
	prices[47] = 600
	prices[48] = 1000
	prices[49] = 1500
	for i, close := range prices {
		high := close + 1
		low := close - 1
		volume := 1000000.0
		err := vwao.Add(high, low, close, volume)
		if err != nil {
			t.Errorf("Add failed at index %d: %v", i, err)
		}
	}
	bullish, err := vwao.IsBullishCrossover()
	if err != nil {
		t.Errorf("IsBullishCrossover failed: %v", err)
	}
	if !bullish {
		t.Error("Expected bullish crossover")
	}

	// Test divergence (bullish: price up, VWAO oversold)
	vwao.Reset()
	prices = make([]float64, 50)
	for i := 0; i < 45; i++ {
		prices[i] = float64(10000 - i*100)
	}
	prices[45] = 100
	prices[46] = 300
	prices[47] = 600
	prices[48] = 1000
	prices[49] = 1500
	for i, close := range prices {
		high := close + 1
		low := close - 1
		volume := 1000000.0
		err := vwao.Add(high, low, close, volume)
		if err != nil {
			t.Errorf("Add failed at index %d: %v", i, err)
		}
	}
	isDivergence, signal, err := vwao.IsDivergence()
	if err != nil {
		t.Errorf("IsDivergence failed: %v", err)
	}
	if !isDivergence || signal != "Bullish" {
		t.Errorf("Expected bullish divergence, got: %v, %s", isDivergence, signal)
	}

	// Test insufficient data
	vwao.Reset()
	vwao.Add(101, 99, 100, 1000)
	_, err = vwao.Calculate()
	if err == nil || err.Error() != "no VWAO data" {
		t.Errorf("Expected error for no VWAO data, got: %v", err)
	}
}
