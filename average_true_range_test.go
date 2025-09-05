package goti

import (
	"testing"
)

func TestAverageTrueRange(t *testing.T) {
	// Test initialization
	atr, err := NewAverageTrueRangeWithParams(14)
	if err != nil {
		t.Fatalf("NewAverageTrueRangeWithParams failed: %v", err)
	}

	// Test invalid period
	_, err = NewAverageTrueRangeWithParams(0)
	if err == nil || err.Error() != "period must be at least 1" {
		t.Errorf("Expected error for invalid period, got: %v", err)
	}

	// Test adding invalid price
	err = atr.Add(100, 101, 100)
	if err == nil || err.Error() != "invalid price" {
		t.Errorf("Expected error for high < low, got: %v", err)
	}
	err = atr.Add(101, 100, -1)
	if err == nil || err.Error() != "invalid price" {
		t.Errorf("Expected error for invalid close, got: %v", err)
	}

	// Test ATR calculation with sample data
	for i := 0; i < 15; i++ {
		high := float64(100 + i + 2)
		low := float64(100 + i - 2)
		close := float64(100 + i)
		atr.Add(high, low, close)
	}
	value, err := atr.Calculate()
	if err != nil {
		t.Errorf("Calculate failed: %v", err)
	}
	if value <= 0 {
		t.Errorf("Expected positive ATR, got: %f", value)
	}

	// Test insufficient data
	atr.Reset()
	atr.Add(101, 99, 100)
	_, err = atr.Calculate()
	if err == nil || err.Error() != "no ATR data" {
		t.Errorf("Expected error for no ATR data, got: %v", err)
	}
}
