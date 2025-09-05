// money_flow_index_test.go
// artifact_id: 42d5becf-fc89-4fed-ae1f-a1f3e12f6158
// artifact_version_id: 7b9d0e4c-8a5e-4f3c-9g7b-6c3d9f2e1a6f

package goti

import (
	"math"
	"testing"
)

func TestMoneyFlowIndex(t *testing.T) {
	// Test initialization
	config := DefaultConfig()
	mfi, err := NewMoneyFlowIndexWithParams(5, config)
	if err != nil {
		t.Fatalf("NewMoneyFlowIndexWithParams failed: %v", err)
	}

	// Test invalid period
	_, err = NewMoneyFlowIndexWithParams(0, config)
	if err == nil || err.Error() != "period must be at least 1" {
		t.Errorf("Expected error for invalid period, got: %v", err)
	}

	// Test invalid config
	invalidConfig := DefaultConfig()
	invalidConfig.MFIOverbought = 20
	invalidConfig.MFIOversold = 80
	_, err = NewMoneyFlowIndexWithParams(5, invalidConfig)
	if err == nil || err.Error() != "MFI overbought threshold must be greater than oversold" {
		t.Errorf("Expected error for invalid config, got: %v", err)
	}

	// Test adding invalid data
	err = mfi.Add(100, 101, 100, 1000)
	if err == nil || err.Error() != "invalid price or volume" {
		t.Errorf("Expected error for high < low, got: %v", err)
	}
	err = mfi.Add(101, 100, -1, 1000)
	if err == nil || err.Error() != "invalid price or volume" {
		t.Errorf("Expected error for invalid close, got: %v", err)
	}
	err = mfi.Add(101, 100, 100, -1)
	if err == nil || err.Error() != "invalid price or volume" {
		t.Errorf("Expected error for invalid volume, got: %v", err)
	}

	// Test MFI calculation with sample data (all gains)
	for i := 0; i < 50; i++ {
		high := float64(100 + i*10 + 1)
		low := float64(100 + i*10 - 1)
		close := float64(100 + i*10)
		volume := 1000000.0
		err := mfi.Add(high, low, close, volume)
		if err != nil {
			t.Errorf("Add failed at index %d: %v", i, err)
		}
	}
	value, err := mfi.Calculate()
	if err != nil {
		t.Errorf("Calculate failed: %v", err)
	}
	if math.Abs(value-100) > 0.01 {
		t.Errorf("Expected MFI near 100, got: %f", value)
	}

	// Test oversold crossover
	mfi.Reset()
	for i := 0; i < 45; i++ {
		high := float64(10000 - i*100 + 1)
		low := float64(10000 - i*100 - 1)
		close := float64(10000 - i*100)
		volume := 1000000.0
		err := mfi.Add(high, low, close, volume)
		if err != nil {
			t.Errorf("Add failed at index %d: %v", i, err)
		}
		t.Logf("MFI after price %d: %f", i, mfi.GetLastValue())
	}
	mfi.Add(101, 99, 100, 1000000)
	mfi.Add(301, 299, 300, 1000000)
	mfi.Add(601, 599, 600, 1000000)
	mfi.Add(1001, 999, 1000, 1000000)
	mfi.Add(1501, 1499, 1500, 1000000)
	bullish, err := mfi.IsBullishCrossover()
	if err != nil {
		t.Errorf("IsBullishCrossover failed: %v", err)
	}
	if !bullish {
		t.Error("Expected bullish crossover")
	}

	// Test divergence (bullish: price up, MFI oversold)
	mfi.Reset()
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
		err := mfi.Add(high, low, close, volume)
		if err != nil {
			t.Errorf("Add failed at index %d: %v", i, err)
		}
		t.Logf("MFI after price %d: %f", i, mfi.GetLastValue())
	}
	isDivergence, signal, err := mfi.IsDivergence()
	if err != nil {
		t.Errorf("IsDivergence failed: %v", err)
	}
	if !isDivergence || signal != "Bullish" {
		t.Errorf("Expected bullish divergence, got: %v, %s", isDivergence, signal)
	}

	// Test insufficient data
	mfi.Reset()
	mfi.Add(101, 99, 100, 1000)
	_, err = mfi.Calculate()
	if err == nil || err.Error() != "no MFI data" {
		t.Errorf("Expected error for no MFI data, got: %v", err)
	}
}
