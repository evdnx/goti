package goti

import (
	"errors"
	"math"
	"reflect"
	"testing"
)

// ---------------------------------------------------------------------------
// Helper – approximate equality for floating‑point numbers
// ---------------------------------------------------------------------------
func approxEqual(a, b float64) bool {
	const eps = 1e-6
	return math.Abs(a-b) <= eps
}

// ---------------------------------------------------------------------------
// Constructor tests
// ---------------------------------------------------------------------------
func TestNewHullMovingAverage(t *testing.T) {
	h, err := NewHullMovingAverage()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.period != 9 {
		t.Errorf("expected default period 9, got %d", h.period)
	}
	if len(h.closes) != 0 || len(h.rawHMAs) != 0 || len(h.hmaValues) != 0 {
		t.Error("expected all internal slices to be empty")
	}
}

// ---------------------------------------------------------------------------
// Add() – validation and basic calculation (period = 3)
// ---------------------------------------------------------------------------
func TestHullMovingAverage_AddAndCalculate_Period3(t *testing.T) {
	h, err := NewHullMovingAverageWithParams(3)
	if err != nil {
		t.Fatalf("constructor error: %v", err)
	}

	// Feed three closing prices – this is the first moment an HMA is produced.
	prices := []float64{10, 20, 30}
	for _, p := range prices {
		if err := h.Add(p); err != nil {
			t.Fatalf("Add(%v) failed: %v", p, err)
		}
	}

	/*
	   With the library’s WMA implementation the most recent price receives
	   the highest weight.  Therefore:

	     wmaFull = (10*3 + 20*2 + 30*1) / (3+2+1) = 100 / 6 ≈ 16.666667
	     wmaHalf = last 1 price = 30
	     rawHMA = 2*wmaHalf - wmaFull = 60 - 16.666667 = 43.333333

	   Since sqrt(period)=1, the final HMA equals rawHMA.
	*/
	expected := 43.3333333333

	val, err := h.Calculate()
	if err != nil {
		t.Fatalf("Calculate returned error: %v", err)
	}
	if !approxEqual(val, expected) {
		t.Errorf("HMA value mismatch: got %.9f, want %.9f", val, expected)
	}
}

// ---------------------------------------------------------------------------
// Add() – price validation
// ---------------------------------------------------------------------------
func TestHullMovingAverage_Add_InvalidPrice(t *testing.T) {
	h, _ := NewHullMovingAverage()
	invalid := []float64{-1, 0, math.NaN(), math.Inf(1)}
	for _, p := range invalid {
		if err := h.Add(p); err == nil {
			t.Errorf("expected error for price %v, got nil", p)
		}
	}
}

// ---------------------------------------------------------------------------
// Crossover detection – using manually‑crafted state
// ---------------------------------------------------------------------------
func TestHullMovingAverage_Crossovers(t *testing.T) {
	h, _ := NewHullMovingAverageWithParams(4)

	// Manually set slices to simulate a bullish crossover:
	//   previous close = 95, previous HMA = 100  (close <= HMA)
	//   current  close = 105, current  HMA = 102 (close > HMA)
	h.closes = []float64{95, 105}
	h.hmaValues = []float64{100, 102}
	h.lastValue = 102

	bull, err := h.IsBullishCrossover()
	if err != nil {
		t.Fatalf("bullish error: %v", err)
	}
	if !bull {
		t.Error("expected bullish crossover, got false")
	}
	bear, err := h.IsBearishCrossover()
	if err != nil {
		t.Fatalf("bearish error: %v", err)
	}
	if bear {
		t.Error("expected no bearish crossover, got true")
	}

	// Now flip the data for a bearish crossover:
	h.closes = []float64{105, 95}
	h.hmaValues = []float64{102, 100}
	h.lastValue = 100

	bull, _ = h.IsBullishCrossover()
	if bull {
		t.Error("expected no bullish crossover, got true")
	}
	bear, _ = h.IsBearishCrossover()
	if !bear {
		t.Error("expected bearish crossover, got false")
	}
}

// ---------------------------------------------------------------------------
// Trend direction
// ---------------------------------------------------------------------------
func TestHullMovingAverage_TrendDirection(t *testing.T) {
	h, _ := NewHullMovingAverageWithParams(5)

	// Upward trend
	h.hmaValues = []float64{10, 12}
	dir, err := h.GetTrendDirection()
	if err != nil {
		t.Fatalf("trend error: %v", err)
	}
	if dir != "Bullish" {
		t.Errorf("expected Bullish, got %s", dir)
	}

	// Downward trend
	h.hmaValues = []float64{15, 13}
	dir, _ = h.GetTrendDirection()
	if dir != "Bearish" {
		t.Errorf("expected Bearish, got %s", dir)
	}

	// Flat trend
	h.hmaValues = []float64{20, 20}
	dir, _ = h.GetTrendDirection()
	if dir != "Neutral" {
		t.Errorf("expected Neutral, got %s", dir)
	}
}

// ---------------------------------------------------------------------------
// Reset clears everything
// ---------------------------------------------------------------------------
func TestHullMovingAverage_Reset(t *testing.T) {
	h, _ := NewHullMovingAverage()
	_ = h.Add(10)
	_ = h.Add(20)
	h.Reset()

	if len(h.closes) != 0 || len(h.rawHMAs) != 0 || len(h.hmaValues) != 0 {
		t.Error("reset did not clear internal slices")
	}
	if h.lastValue != 0 {
		t.Error("reset did not zero lastValue")
	}
}

// ---------------------------------------------------------------------------
// SetPeriod – changes period and trims slices appropriately
// ---------------------------------------------------------------------------
func TestHullMovingAverage_SetPeriod(t *testing.T) {
	h, _ := NewHullMovingAverageWithParams(4)

	// Fill slices beyond the limits for period=4
	for i := 0; i < 20; i++ {
		_ = h.Add(float64(i + 1))
	}
	origLenCloses := len(h.closes)

	if err := h.SetPeriod(2); err != nil {
		t.Fatalf("SetPeriod error: %v", err)
	}
	if h.period != 2 {
		t.Errorf("expected period 2, got %d", h.period)
	}
	if len(h.closes) >= origLenCloses {
		t.Error("expected closes slice to shrink after period change")
	}
}

// ---------------------------------------------------------------------------
// DetectSignals – verifies the signal vector matches expected crossovers
// ---------------------------------------------------------------------------
func TestHullMovingAverage_DetectSignals(t *testing.T) {
	h, _ := NewHullMovingAverageWithParams(3)

	// Construct a scenario with:
	//   index 0 : no signal
	//   index 1 : bullish crossover (price moves above HMA)
	//   index 2 : bearish crossover (price moves below HMA)
	h.closes = []float64{95, 105, 95}
	h.hmaValues = []float64{100, 102, 100}
	expected := []float64{0, 1, -1}

	sig := h.DetectSignals()
	if !reflect.DeepEqual(sig, expected) {
		t.Errorf("signals mismatch: got %v, want %v", sig, expected)
	}
}

// ---------------------------------------------------------------------------
// GetPlotData – basic sanity checks (lengths, timestamps)
// ---------------------------------------------------------------------------
func TestHullMovingAverage_GetPlotData(t *testing.T) {
	h, _ := NewHullMovingAverageWithParams(3)

	// Feed enough data to produce three HMA points.
	prices := []float64{10, 20, 30, 40, 50, 60}
	for _, p := range prices {
		_ = h.Add(p)
	}
	start := int64(1_600_000_000) // arbitrary epoch
	interval := int64(60)         // 1‑minute candles

	data := h.GetPlotData(start, interval)

	if len(data) != 3 {
		t.Fatalf("expected 3 PlotData series, got %d", len(data))
	}
	// All series must have identical X/Y lengths.
	for _, s := range data {
		if len(s.X) != len(s.Y) {
			t.Fatalf("series %s has mismatched X/Y lengths", s.Name)
		}
	}
	// Verify timestamps are correctly generated.
	expTS := GenerateTimestamps(start, len(h.hmaValues), interval)
	if !reflect.DeepEqual(data[0].Timestamp, expTS) {
		t.Errorf("timestamps mismatch: got %v, want %v", data[0].Timestamp, expTS)
	}
}

// ---------------------------------------------------------------------------
// Edge‑case errors
// ---------------------------------------------------------------------------
func TestHullMovingAverage_Errors(t *testing.T) {
	h, _ := NewHullMovingAverageWithParams(5)

	// No HMA yet → Calculate should return ErrInsufficientHMAData
	if _, err := h.Calculate(); !errors.Is(err, ErrInsufficientHMAData) {
		t.Errorf("expected ErrInsufficientHMAData, got %v", err)
	}

	// Not enough points for crossover detection
	if _, err := h.IsBullishCrossover(); !errors.Is(err, ErrInsufficientCrossData) {
		t.Errorf("expected ErrInsufficientCrossData, got %v", err)
	}
}
