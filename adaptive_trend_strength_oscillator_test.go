package goti

import (
	"fmt"
	"math"
	"testing"
)

// Helper to build a simple ATSO instance with deterministic config.
func newTestATSO(t *testing.T) *AdaptiveTrendStrengthOscillator {
	cfg := DefaultConfig()
	// Use a very short EMA period so we can observe smoothing quickly.
	cfg.ATSEMAperiod = 2
	atso, err := NewAdaptiveTrendStrengthOscillatorWithParams(2, 5, 3, cfg)
	if err != nil {
		t.Fatalf("failed to create ATSO: %v", err)
	}
	return atso
}

// Feed a monotonic upward price series and verify that the oscillator produces
// a positive value after enough points.
func TestATSO_BullishTrend(t *testing.T) {
	atso := newTestATSO(t)

	// Generate 20 bars with steadily rising prices.
	high := 10.0
	low := 9.0
	close := 9.5
	for i := 0; i < 20; i++ {
		if err := atso.Add(high, low, close); err != nil {
			t.Fatalf("Add error at iteration %d: %v", i, err)
		}
		high += 1.0
		low += 1.0
		close += 1.0
	}

	val, err := atso.Calculate()
	if err != nil {
		t.Fatalf("Calculate error: %v", err)
	}
	if val <= 0 {
		t.Fatalf("expected positive ATSO value for bullish trend, got %v", val)
	}
}

// Feed a monotonic downward price series and verify a negative value.
func TestATSO_BearishTrend(t *testing.T) {
	atso := newTestATSO(t)

	high := 20.0
	low := 19.0
	close := 19.5
	for i := 0; i < 20; i++ {
		if err := atso.Add(high, low, close); err != nil {
			t.Fatalf("Add error at iteration %d: %v", i, err)
		}
		high -= 1.0
		low -= 1.0
		close -= 1.0
	}

	val, err := atso.Calculate()
	if err != nil {
		t.Fatalf("Calculate error: %v", err)
	}
	if val >= 0 {
		t.Fatalf("expected negative ATSO value for bearish trend, got %v", val)
	}
}

// Verify that bullish/bearish crossovers are detected correctly.
func TestATSO_Crossovers(t *testing.T) {
	atso := newTestATSO(t)

	// First feed a bearish series (negative ATSO)
	high, low, close := 30.0, 29.0, 29.5
	for i := 0; i < 12; i++ {
		if err := atso.Add(high, low, close); err != nil {
			t.Fatalf("Add error (bearish) %d: %v", i, err)
		}
		high -= 1.0
		low -= 1.0
		close -= 1.0
	}
	// Now switch to a bullish series (positive ATSO)
	high, low, close = 10.0, 9.0, 9.5
	for i := 0; i < 12; i++ {
		if err := atso.Add(high, low, close); err != nil {
			t.Fatalf("Add error (bullish) %d: %v", i, err)
		}
		high += 1.0
		low += 1.0
		close += 1.0
	}

	// After the switch we should have seen a bullish crossover.
	bull, err := atso.IsBullishCrossover()
	if err != nil {
		t.Fatalf("IsBullishCrossover error: %v", err)
	}
	if !bull {
		t.Fatalf("expected bullish crossover after trend reversal")
	}
	// And there should be no bearish crossover at the same moment.
	bear, err := atso.IsBearishCrossover()
	if err != nil {
		t.Fatalf("IsBearishCrossover error: %v", err)
	}
	if bear {
		t.Fatalf("did not expect a bearish crossover at this point")
	}
}

// Test Reset clears internal state.
func TestATSO_Reset(t *testing.T) {
	atso := newTestATSO(t)

	// Populate with a few points.
	for i := 0; i < 8; i++ {
		if err := atso.Add(10+float64(i), 9+float64(i), 9.5+float64(i)); err != nil {
			t.Fatalf("Add error: %v", err)
		}
	}
	if len(atso.GetATSOValues()) == 0 {
		t.Fatalf("expected some ATSO values before reset")
	}
	atso.Reset()
	if len(atso.GetATSOValues()) != 0 ||
		len(atso.GetHighs()) != 0 ||
		len(atso.GetLows()) != 0 ||
		len(atso.GetCloses()) != 0 {
		t.Fatalf("Reset did not clear all internal slices")
	}
	if atso.GetLastValue() != 0 {
		t.Fatalf("Reset should zero out lastValue")
	}
}

// Verify that SetPeriods validates inputs and updates internal limits.
func TestATSO_SetPeriods(t *testing.T) {
	atso := newTestATSO(t)

	// Valid change.
	if err := atso.SetPeriods(3, 6, 2); err != nil {
		t.Fatalf("SetPeriods valid case failed: %v", err)
	}
	if atso.minPeriod != 3 || atso.maxPeriod != 6 || atso.volatilityPeriod != 2 {
		t.Fatalf("SetPeriods did not store new values")
	}

	// Invalid (min > max).
	if err := atso.SetPeriods(8, 5, 2); err == nil {
		t.Fatalf("expected error when minPeriod > maxPeriod")
	}
}

// Test volatility‑sensitivity setter.
func TestATSO_VolatilitySensitivity(t *testing.T) {
	atso := newTestATSO(t)

	if err := atso.SetVolatilitySensitivity(3.5); err != nil {
		t.Fatalf("failed to set valid sensitivity: %v", err)
	}
	if atso.volSensitivity != 3.5 {
		t.Fatalf("volSensitivity not updated")
	}
	if err := atso.SetVolatilitySensitivity(0); err == nil {
		t.Fatalf("expected error for non‑positive sensitivity")
	}
}

// Plot data generation sanity‑check (does not validate visual output, just structure).
func TestATSO_PlotData(t *testing.T) {
	atso := newTestATSO(t)

	// Feed a tiny deterministic series.
	for i := 0; i < 10; i++ {
		if err := atso.Add(10+float64(i), 9+float64(i), 9.5+float64(i)); err != nil {
			t.Fatalf("Add error: %v", err)
		}
	}
	data := atso.GetPlotData(1622505600000, 60000) // start timestamp + 1‑minute interval

	if len(data) != 2 {
		t.Fatalf("expected 2 PlotData series (ATSO + Signals), got %d", len(data))
	}
	if data[0].Name != "Adaptive Trend Strength Oscillator" {
		t.Fatalf("unexpected name for first series: %s", data[0].Name)
	}
	if data[1].Name != "Signals" {
		t.Fatalf("unexpected name for second series: %s", data[1].Name)
	}
	if len(data[0].X) != len(atso.atsoValues) {
		t.Fatalf("X length mismatch: %d vs %d", len(data[0].X), len(atso.atsoValues))
	}
	if len(data[1].Y) != len(atso.atsoValues) {
		t.Fatalf("Signals Y length mismatch")
	}
}

// Edge‑case: calling Calculate before any data should return an error.
func TestATSO_Calculate_NoData(t *testing.T) {
	atso, err := NewAdaptiveTrendStrengthOscillator()
	if err != nil {
		t.Fatalf("constructor error: %v", err)
	}
	if _, err := atso.Calculate(); err == nil {
		t.Fatalf("expected error when calculating with no data")
	}
}

// Ensure that adding an invalid price pair returns an error.
func TestATSO_Add_InvalidPrices(t *testing.T) {
	atso := newTestATSO(t)

	// high < low should be rejected.
	if err := atso.Add(5, 10, 7); err == nil {
		t.Fatalf("expected error when high < low")
	}
	// negative close should be rejected.
	if err := atso.Add(10, 5, -1); err == nil {
		t.Fatalf("expected error when close is negative")
	}
}

// Verify that the internal EMA is correctly seeded with SMA when first EMA calc occurs.
func TestATSO_EMASeed(t *testing.T) {
	// Use a tiny EMA period so we can inspect the seeded value easily.
	cfg := DefaultConfig()
	cfg.ATSEMAperiod = 3
	atso, err := NewAdaptiveTrendStrengthOscillatorWithParams(2, 5, 3, cfg)
	if err != nil {
		t.Fatalf("constructor error: %v", err)
	}

	/*
	   We need enough points to trigger the first ATSO calculation.
	   Required length = maxPeriod + volatilityPeriod + 1
	   = 5 + 3 + 1 = 9
	*/
	prices := []struct {
		high  float64
		low   float64
		close float64
	}{
		{10, 9, 9.5},
		{11, 10, 10.5},
		{12, 11, 11.5},
		{13, 12, 12.5},
		{14, 13, 13.5},
		{15, 14, 14.5},
		{16, 15, 15.5},
		{17, 16, 16.5},
		{18, 17, 17.5},
	}
	for i, p := range prices {
		if err := atso.Add(p.high, p.low, p.close); err != nil {
			t.Fatalf("Add error at index %d: %v", i, err)
		}
	}

	// At this point the EMA inside ATSO should have been seeded with the SMA of the first three ATSO values.
	// Retrieve the raw (unsmoothed) ATSO value that was just added.
	// (We don’t need the raw value for the assertions, so we omit it.)

	// The EMA should now hold a smoothed value. Because we used a period of 3,
	// the smoothing factor α = 2/(3+1) = 0.5.
	// The first EMA value is simply the SMA of the first three raw ATSO values.
	// To verify, we reconstruct those three raw values manually.

	// Helper to compute raw ATSO for a given window (mirrors the private method logic).
	computeRaw := func(startIdx, period int) (float64, error) {
		if startIdx+period > len(atso.closes) {
			return 0, fmt.Errorf("window out of range")
		}
		highs := atso.highs[startIdx : startIdx+period]
		lows := atso.lows[startIdx : startIdx+period]
		closes := atso.closes[startIdx : startIdx+period]

		// 1️⃣ Adaptive period (simplified: we just use the provided period here because volatility
		//    calculation is deterministic for our monotonic data set).
		adaptPeriod := period

		// 2️⃣ Trend strength for the window.
		var upSum, downSum float64
		for i := 1; i < adaptPeriod; i++ {
			if closes[i] > closes[i-1] {
				upSum += highs[i] - lows[i-1]
			} else {
				downSum += lows[i] - highs[i-1]
			}
		}
		if upSum+downSum == 0 {
			return 0, fmt.Errorf("division by zero in trend strength")
		}
		raw := ((upSum - downSum) / (upSum + downSum)) * 100
		return raw, nil
	}

	// Compute the first three raw ATSO values.
	raw0, err := computeRaw(0, 2) // period = minPeriod (2) for the very first window
	if err != nil {
		t.Fatalf("computeRaw0 error: %v", err)
	}
	raw1, err := computeRaw(1, 2)
	if err != nil {
		t.Fatalf("computeRaw1 error: %v", err)
	}
	raw2, err := computeRaw(2, 2)
	if err != nil {
		t.Fatalf("computeRaw2 error: %v", err)
	}

	// SMA of the first three raw values.
	sma := (raw0 + raw1 + raw2) / 3.0

	// The EMA stored inside the ATSO should equal this SMA after the first calculation.
	emaVal, err := atso.ema.Calculate()
	if err != nil {
		t.Fatalf("EMA Calculate error: %v", err)
	}
	if math.Abs(emaVal-sma) > 1e-9 {
		t.Fatalf("EMA seed mismatch: got %v, expected SMA %v", emaVal, sma)
	}

	// Finally, confirm that the smoothed value exposed by Calculate()
	// matches the EMA we just retrieved.
	calcVal, err := atso.Calculate()
	if err != nil {
		t.Fatalf("ATSO Calculate error: %v", err)
	}
	if math.Abs(calcVal-emaVal) > 1e-9 {
		t.Fatalf("ATSO Calculate returned %v, but EMA is %v", calcVal, emaVal)
	}
}
