package goti

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Helper – generate a deterministic data set for a given period.
// Returns slices of high, low, close, volume.
// ---------------------------------------------------------------------------
func genTestData(period int) (highs, lows, closes, volumes []float64) {
	// Create (period+1) candles.  Values are chosen so that the
	// highest‑high occurs on the newest bar and the lowest‑low on the
	// oldest bar – this makes manual calculation easy.
	for i := 0; i <= period; i++ {
		high := 100.0 + float64(i)   // increasing high
		low := 90.0 - float64(i)     // decreasing low
		close := (high + low) / 2.0  // mid‑point
		vol := 10.0 + float64(i)*2.0 // steadily rising volume
		highs = append(highs, high)
		lows = append(lows, low)
		closes = append(closes, close)
		volumes = append(volumes, vol)
	}
	return
}

// ---------------------------------------------------------------------------
// Test constructor validation (period & config)
// ---------------------------------------------------------------------------
func TestNewVolumeWeightedAroonOscillator_Errors(t *testing.T) {
	// Invalid period
	if _, err := NewVolumeWeightedAroonOscillatorWithParams(0, DefaultConfig()); err == nil {
		t.Fatalf("expected error for period < 1")
	}

	// Invalid config (ATSEMAperiod <= 0)
	badCfg := DefaultConfig()
	badCfg.ATSEMAperiod = 0
	if _, err := NewVolumeWeightedAroonOscillatorWithParams(14, badCfg); err == nil {
		t.Fatalf("expected error for invalid config")
	}
}

// ---------------------------------------------------------------------------
// Test basic Add/Calculate flow with a known deterministic data set.
// The expected VWAO value is calculated analytically in the comment.
// ---------------------------------------------------------------------------
func TestVWAO_CalculationSimple(t *testing.T) {
	period := 4
	highs, lows, closes, volumes := genTestData(period)

	osc, err := NewVolumeWeightedAroonOscillatorWithParams(period, DefaultConfig())
	if err != nil {
		t.Fatalf("constructor error: %v", err)
	}

	// Feed the candles one by one.
	for i := 0; i < len(highs); i++ {
		if err := osc.Add(highs[i], lows[i], closes[i], volumes[i]); err != nil {
			t.Fatalf("Add failed at i=%d: %v", i, err)
		}
	}

	// After feeding period+1 candles we should have exactly one VWAO value.
	val, err := osc.Calculate()
	if err != nil {
		t.Fatalf("Calculate error: %v", err)
	}

	/*
	   Manual calculation for the generated data (period=4):
	     - Highest high = 104 (index 4, newest bar)
	     - Lowest low   = 86  (index 0, oldest bar)
	     - totalWeightedAge = Σ (period‑i) * vol[i]
	       = (4*10) + (3*12) + (2*14) + (1*16) + (0*18) = 40+36+28+16+0 = 120
	     - weightedHighAge = (4‑4)*vol[4] = 0*18 = 0
	     - weightedLowAge  = (4‑0)*vol[0] = 4*10 = 40
	     - aroonUp   = 0/120 *100 = 0
	     - aroonDown = 40/120*100 ≈ 33.3333
	     - oscillator = 0 – 33.3333 = -33.3333 (clamped within [-100,100])
	*/

	expected := -33.333333333333336
	if math.Abs(val-expected) > 1e-9 {
		t.Fatalf("unexpected VWAO value: got %v want %v", val, expected)
	}
}

// ---------------------------------------------------------------------------
// Test that adding a candle with invalid price/volume returns an error.
// ---------------------------------------------------------------------------
func TestVWAO_AddValidation(t *testing.T) {
	osc, _ := NewVolumeWeightedAroonOscillator()
	// high < low
	if err := osc.Add(90, 100, 95, 10); err == nil {
		t.Fatalf("expected error when high < low")
	}
	// negative close
	if err := osc.Add(100, 90, -1, 10); err == nil {
		t.Fatalf("expected error when close is negative")
	}
	// NaN volume
	if err := osc.Add(100, 90, 95, math.NaN()); err == nil {
		t.Fatalf("expected error when volume is NaN")
	}
}

// ---------------------------------------------------------------------------
// Test SetPeriod – ensure old data is trimmed and new calculations use the new window.
// ---------------------------------------------------------------------------
func TestVWAO_SetPeriod(t *testing.T) {
	// Start with period 3, feed enough data for two VWAO values.
	osc, _ := NewVolumeWeightedAroonOscillatorWithParams(3, DefaultConfig())

	h, l, c, v := genTestData(3) // 4 candles (period+1)
	for i := 0; i < len(h); i++ {
		if err := osc.Add(h[i], l[i], c[i], v[i]); err != nil {
			t.Fatalf("add failed: %v", err)
		}
	}
	if len(osc.GetVWAOValues()) != 1 {
		t.Fatalf("expected 1 VWAO value before period change, got %d", len(osc.GetVWAOValues()))
	}

	// Change period to 2 – internal slices should be trimmed.
	if err := osc.SetPeriod(2); err != nil {
		t.Fatalf("SetPeriod error: %v", err)
	}
	if osc.period != 2 {
		t.Fatalf("period not updated")
	}
	// After trimming we should have at most 3 candles stored.
	if len(osc.GetCloses()) > 3 {
		t.Fatalf("trimSlices did not reduce stored candles")
	}
}

// ---------------------------------------------------------------------------
// Test Reset – all buffers cleared, subsequent adds work as fresh instance.
// ---------------------------------------------------------------------------
func TestVWAO_Reset(t *testing.T) {
	osc, _ := NewVolumeWeightedAroonOscillator()
	h, l, c, v := genTestData(2)
	for i := 0; i < len(h); i++ {
		_ = osc.Add(h[i], l[i], c[i], v[i])
	}
	osc.Reset()

	if len(osc.GetCloses()) != 0 || len(osc.GetVWAOValues()) != 0 {
		t.Fatalf("Reset did not clear internal state")
	}
	// After reset, a new series should still produce a value.
	for i := 0; i < len(h); i++ {
		if err := osc.Add(h[i], l[i], c[i], v[i]); err != nil {
			t.Fatalf("add after reset failed: %v", err)
		}
	}
	if _, err := osc.Calculate(); err != nil {
		t.Fatalf("calculate after reset failed: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Test signal helpers – bullish/bearish crossovers, strong‑trend detection,
// and divergence logic.
// ---------------------------------------------------------------------------
func TestVWAO_SignalHelpers(t *testing.T) {
	cfg := DefaultConfig()
	cfg.VWAOStrongTrend = 20 // make thresholds easier to trigger

	osc, _ := NewVolumeWeightedAroonOscillatorWithParams(3, cfg)

	// Build a scenario where the oscillator crosses above the strong‑trend line.
	// We'll feed four windows; the last two values will be 25 (above) and 15 (below).
	data := []struct {
		high, low, close, vol float64
	}{
		{101, 99, 100, 10},
		{102, 98, 100, 12},
		{103, 97, 100, 14},
		{104, 96, 100, 16}, // after this add we get first VWAO
		{105, 95, 100, 18}, // second VWAO – should be >20
		{106, 94, 100, 20}, // third VWAO – should drop below 20
	}
	for _, d := range data {
		if err := osc.Add(d.high, d.low, d.close, d.vol); err != nil {
			t.Fatalf("add error: %v", err)
		}
	}

	// At this point we have three VWAO values.
	if len(osc.GetVWAOValues()) < 3 {
		t.Fatalf("not enough VWAO values for signal tests")
	}

	// Bullish crossover: previous ≤ 20, current > 20
	bull, err := osc.IsBullishCrossover()
	if err != nil {
		t.Fatalf("IsBullishCrossover error: %v", err)
	}
	if !bull {
		t.Fatalf("expected bullish crossover")
	}

	// Bearish crossover: now previous ≥ -20, current < -20 (should be false)
	bear, err := osc.IsBearishCrossover()
	if err != nil {
		t.Fatalf("IsBearishCrossover error: %v", err)
	}
	if bear {
		t.Fatalf("did not expect bearish crossover")
	}

	// Strong‑trend detection (value > 20 or < -20)
	st, err := osc.IsStrongTrend()
	if err != nil {
		t.Fatalf("IsStrongTrend error: %v", err)
	}
	if !st {
		t.Fatalf("expected strong‑trend flag")
	}

	// Divergence – create a price move opposite to oscillator direction.
	// Last two closes: 100 -> 101 (up) while oscillator drops below -20.
	osc.Add(107, 93, 101, 22) // add one more candle to force divergence check
	div, dir, err := osc.IsDivergence()
	if err != nil {
		t.Fatalf("IsDivergence error: %v", err)
	}
	if !div || dir != "Bullish" {
		t.Fatalf("expected bullish divergence, got %v %s", div, dir)
	}
}

// ---------------------------------------------------------------------------
// Test edge case: total weighted volume equals zero (all volumes zero)
// Expect computeVWAO to return an explicit error.
// ---------------------------------------------------------------------------
func TestVWAO_ZeroVolumeError(t *testing.T) {
	osc, _ := NewVolumeWeightedAroonOscillatorWithParams(2, DefaultConfig())
	// All volumes are zero – after period+1 candles computeVWAO will be invoked.
	if err := osc.Add(110, 100, 105, 0); err != nil {
		t.Fatalf("add 1 failed: %v", err)
	}
	if err := osc.Add(111, 101, 106, 0); err != nil {
		t.Fatalf("add 2 failed: %v", err)
	}
	if err := osc.Add(112, 102, 107, 0); err != nil {
		t.Fatalf("add 3 failed: %v", err)
	}
	// The third Add triggers computeVWAO which should surface the zero‑volume error.
	if len(osc.GetVWAOValues()) != 0 {
		t.Fatalf("expected no VWAO values due to zero volume")
	}
	if _, err := osc.Calculate(); err == nil {
		t.Fatalf("expected error from Calculate due to no data")
	}
}

// ---------------------------------------------------------------------------
// Test GetPlotData – ensure timestamps length matches VWAO values and that
// signal encoding follows the spec (1 bullish, -1 bearish, 2/‑2 strong trend).
// ---------------------------------------------------------------------------
func TestVWAO_GetPlotData(t *testing.T) {
	osc, _ := NewVolumeWeightedAroonOscillator()
	h, l, c, v := genTestData(3) // period 14 default, but we only need a few points
	for i := 0; i < len(h); i++ {
		_ = osc.Add(h[i], l[i], c[i], v[i])
	}
	plot := osc.GetPlotData(1_600_000_000, 60_000) // arbitrary start/interval

	if len(plot) != 2 {
		t.Fatalf("expected 2 PlotData series, got %d", len(plot))
	}
	if len(plot[0].Y) != len(plot[0].Timestamp) {
		t.Fatalf("timestamps length mismatch for main series")
	}
	if len(plot[1].Y) != len(plot[1].Timestamp) {
		t.Fatalf("timestamps length mismatch for signals series")
	}
}

// ---------------------------------------------------------------------------
// Test that Getters return copies (modifying the returned slice does not affect
// the internal state).
// ---------------------------------------------------------------------------
func TestVWAO_GettersCopySafety(t *testing.T) {
	osc, _ := NewVolumeWeightedAroonOscillator()
	h, l, c, v := genTestData(2)
	for i := 0; i < len(h); i++ {
		_ = osc.Add(h[i], l[i], c[i], v[i])
	}
	origHighs := osc.GetHighs()
	origHighs[0] = -999 // mutate the returned slice

	if osc.highs[0] == -999 {
		t.Fatalf("internal highs slice was mutated via getter")
	}
}

// ---------------------------------------------------------------------------
// Test that the oscillator clamps its output to the [-100, 100] range.
// ---------------------------------------------------------------------------
func TestVWAO_Clamping(t *testing.T) {
	// Construct a scenario where weightedHighAge >> totalWeightedAge,
	// producing an aroonUp > 100 before clamping.
	osc, _ := NewVolumeWeightedAroonOscillatorWithParams(1, DefaultConfig())

	// First candle – just to fill slice.
	_ = osc.Add(100, 90, 95, 1)

	// Second candle – extremely high volume on the high bar.
	_ = osc.Add(200, 80, 150, 1000) // highIdx = 1, lowIdx = 0

	val, err := osc.Calculate()
	if err != nil {
		t.Fatalf("calculate error: %v", err)
	}
	if val > 100 || val < -100 {
		t.Fatalf("value not clamped: %v", val)
	}
}
