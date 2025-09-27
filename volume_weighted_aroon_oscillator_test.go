package goti

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Helper – deterministic data set.
//   - highs increase each bar (newest bar = highest high)
//   - lows **decrease** each bar (oldest bar = lowest low)
//   - volumes rise steadily.
//
// ---------------------------------------------------------------------------
func genDeterministicData(period int) (highs, lows, closes, volumes []float64) {
	for i := 0; i <= period; i++ {
		high := 100.0 + float64(i)   // increasing high
		low := 90.0 - float64(i)     // decreasing low → oldest bar is lowest
		close := (high + low) / 2.0  // midpoint
		vol := 10.0 + float64(i)*2.0 // rising volume
		highs = append(highs, high)
		lows = append(lows, low)
		closes = append(closes, close)
		volumes = append(volumes, vol)
	}
	return
}

// ---------------------------------------------------------------------------
// Helper – deterministic data where the *lowest low* occurs on the **oldest**
// bar (so the low‑index differs from the high‑index).  This matches the
// implementation’s “most recent lowest low” rule and yields a non‑zero VWAO.
// ---------------------------------------------------------------------------
func genCalcSimpleData(period int) (highs, lows, closes, volumes []float64) {
	for i := 0; i <= period; i++ {
		high := 100.0 + float64(i) // increasing high → newest bar highest
		low := 80.0 + float64(i)   // **increasing** low → oldest bar lowest
		close := (high + low) / 2.0
		vol := 10.0 + float64(i)*2.0
		highs = append(highs, high)
		lows = append(lows, low)
		closes = append(closes, close)
		volumes = append(volumes, vol)
	}
	return
}

// ---------------------------------------------------------------------------
// Constructor validation (period & config)
// ---------------------------------------------------------------------------
func TestNewVolumeWeightedAroonOscillator_Errors(t *testing.T) {
	if _, err := NewVolumeWeightedAroonOscillatorWithParams(0, DefaultConfig()); err == nil {
		t.Fatalf("expected error for period < 1")
	}
	bad := DefaultConfig()
	bad.ATSEMAperiod = 0
	if _, err := NewVolumeWeightedAroonOscillatorWithParams(14, bad); err == nil {
		t.Fatalf("expected error for invalid config")
	}
}

// ---------------------------------------------------------------------------
// Validation of Add (price/volume rules)
// ---------------------------------------------------------------------------
func TestVWAO_AddValidation(t *testing.T) {
	osc, _ := NewVolumeWeightedAroonOscillator()
	if err := osc.Add(90, 100, 95, 10); err == nil {
		t.Fatalf("expected error when high < low")
	}
	if err := osc.Add(100, 90, -1, 10); err == nil {
		t.Fatalf("expected error when close negative")
	}
	if err := osc.Add(100, 90, 95, math.NaN()); err == nil {
		t.Fatalf("expected error when volume NaN")
	}
}

// ---------------------------------------------------------------------------
// SetPeriod – ensure internal buffers shrink correctly.
// ---------------------------------------------------------------------------
func TestVWAO_SetPeriod(t *testing.T) {
	osc, _ := NewVolumeWeightedAroonOscillatorWithParams(3, DefaultConfig())
	h, l, c, v := genDeterministicData(3) // 4 candles
	for i := 0; i < len(h); i++ {
		if err := osc.Add(h[i], l[i], c[i], v[i]); err != nil {
			t.Fatalf("add: %v", err)
		}
	}
	if len(osc.GetVWAOValues()) != 1 {
		t.Fatalf("expected 1 VWAO before period change")
	}
	if err := osc.SetPeriod(2); err != nil {
		t.Fatalf("set period error: %v", err)
	}
	if osc.period != 2 {
		t.Fatalf("period not updated")
	}
	if len(osc.GetCloses()) > 3 {
		t.Fatalf("trimSlices failed")
	}
}

// ---------------------------------------------------------------------------
// Reset – use a tiny period so we can actually compute a VWAO after reset.
// ---------------------------------------------------------------------------
func TestVWAO_Reset(t *testing.T) {
	osc, _ := NewVolumeWeightedAroonOscillatorWithParams(2, DefaultConfig())
	h, l, c, v := genDeterministicData(2) // 3 candles
	for i := 0; i < len(h); i++ {
		_ = osc.Add(h[i], l[i], c[i], v[i])
	}
	osc.Reset()
	if len(osc.GetCloses()) != 0 || len(osc.GetVWAOValues()) != 0 {
		t.Fatalf("reset did not clear state")
	}
	// Feed a fresh series (again 3 candles) – we should now have a value.
	for i := 0; i < len(h); i++ {
		if err := osc.Add(h[i], l[i], c[i], v[i]); err != nil {
			t.Fatalf("add after reset: %v", err)
		}
	}
	if _, err := osc.Calculate(); err != nil {
		t.Fatalf("calculate after reset failed: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Plot data – use a tiny period so we actually generate values.
// ---------------------------------------------------------------------------
func TestVWAO_GetPlotData(t *testing.T) {
	osc, _ := NewVolumeWeightedAroonOscillatorWithParams(2, DefaultConfig())
	h, l, c, v := genDeterministicData(2) // 3 candles → one VWAO
	for i := 0; i < len(h); i++ {
		_ = osc.Add(h[i], l[i], c[i], v[i])
	}
	plot := osc.GetPlotData(1_600_000_000, 60_000)
	if len(plot) != 2 {
		t.Fatalf("expected 2 PlotData series, got %d", len(plot))
	}
	if len(plot[0].Y) != len(plot[0].Timestamp) {
		t.Fatalf("timestamp length mismatch for main series")
	}
	if len(plot[1].Y) != len(plot[1].Timestamp) {
		t.Fatalf("timestamp length mismatch for signals series")
	}
}

// ---------------------------------------------------------------------------
// Getters must return copies (mutating the returned slice must not affect the
// internal state).
// ---------------------------------------------------------------------------
func TestVWAO_GettersCopySafety(t *testing.T) {
	osc, _ := NewVolumeWeightedAroonOscillatorWithParams(2, DefaultConfig())
	h, l, c, v := genDeterministicData(2)
	for i := 0; i < len(h); i++ {
		_ = osc.Add(h[i], l[i], c[i], v[i])
	}
	ret := osc.GetHighs()
	ret[0] = -999
	if osc.highs[0] == -999 {
		t.Fatalf("internal slice exposed via getter")
	}
}

func TestVWAO_Clamping(t *testing.T) {
	// Construct a scenario where weightedHighAge >> totalWeightedAge,
	// which would yield an aroonUp > 100 before clamping.
	osc, _ := NewVolumeWeightedAroonOscillatorWithParams(1, DefaultConfig())

	// First candle – just to fill the slice.
	_ = osc.Add(100, 90, 95, 1)

	// Second candle – extremely high volume on the high bar.
	// highIdx = 1 (newest), lowIdx = 0 (oldest).
	_ = osc.Add(200, 80, 150, 1000)

	val, err := osc.Calculate()
	if err != nil {
		t.Fatalf("calculate error: %v", err)
	}
	if val > 100 || val < -100 {
		t.Fatalf("value not clamped to [-100,100]: %v", val)
	}
}

// ---------------------------------------------------------------------------
// Simple calculation – now uses the data pattern above so the expected value
// (-33.33…) is produced.
// ---------------------------------------------------------------------------
func TestVWAO_CalculationSimple(t *testing.T) {
	period := 4
	highs, lows, closes, vols := genCalcSimpleData(period)

	osc, err := NewVolumeWeightedAroonOscillatorWithParams(period, DefaultConfig())
	if err != nil {
		t.Fatalf("ctor error: %v", err)
	}
	for i := 0; i < len(highs); i++ {
		if err := osc.Add(highs[i], lows[i], closes[i], vols[i]); err != nil {
			t.Fatalf("add %d: %v", i, err)
		}
	}
	val, err := osc.Calculate()
	if err != nil {
		t.Fatalf("calculate error: %v", err)
	}

	/*
	   Manual calc (period=4)
	     highest high = 104 (newest bar, idx 4)
	     lowest low   = 80  (oldest bar, idx 0)
	     totalWeightedAge = Σ (4‑i)*vol[i] = 40+36+28+16+0 = 120
	     weightedHighAge = (4‑4)*vol[4] = 0
	     weightedLowAge  = (4‑0)*vol[0] = 40
	     aroonUp   = 0/120*100 = 0
	     aroonDown = 40/120*100 ≈ 33.3333
	     oscillator = -33.3333
	*/
	expected := -33.333333333333336
	if math.Abs(val-expected) > 1e-9 {
		t.Fatalf("unexpected VWAO: got %v want %v", val, expected)
	}
}

// ---------------------------------------------------------------------------
// Zero‑volume error – all three candles have volume 0, so the total weighted
// volume is zero and the third Add must return an error.
// ---------------------------------------------------------------------------
func TestVWAO_ZeroVolumeError(t *testing.T) {
	osc, _ := NewVolumeWeightedAroonOscillatorWithParams(2, DefaultConfig())

	// Three candles, all zero volume.
	osc.Add(110, 100, 105, 0) // i0
	osc.Add(111, 101, 106, 0) // i1

	// The third Add triggers computeVWAO and should surface the zero‑volume error.
	if err := osc.Add(112, 102, 107, 0); err == nil {
		t.Fatalf("expected error on zero‑volume compute")
	}
	if len(osc.GetVWAOValues()) != 0 {
		t.Fatalf("no VWAO should have been stored")
	}
}

// ---------------------------------------------------------------------------
// Signal helpers – we bypass the heavy‑lifting of generating a real crossing
// by injecting the desired VWAO values directly (the struct is in the same
// package, so we can touch the private field).  This isolates the helper
// logic from the oscillator calculation.
// ---------------------------------------------------------------------------
func TestVWAO_SignalHelpers(t *testing.T) {
	cfg := DefaultConfig()
	cfg.VWAOStrongTrend = 10 // low threshold to make the logic obvious

	osc, _ := NewVolumeWeightedAroonOscillatorWithParams(2, cfg)

	// Manually set two consecutive VWAO values that cross the threshold:
	//   previous = 5  (≤ 10)
	//   current  = 15 (> 10)
	osc.vwaoValues = []float64{5, 15}
	osc.lastValue = 15

	// Bullish crossover should be true.
	bull, err := osc.IsBullishCrossover()
	if err != nil {
		t.Fatalf("bullish err: %v", err)
	}
	if !bull {
		t.Fatalf("expected bullish crossover")
	}

	// Bearish crossover should be false.
	bear, err := osc.IsBearishCrossover()
	if err != nil {
		t.Fatalf("bearish err: %v", err)
	}
	if bear {
		t.Fatalf("did not expect bearish crossover")
	}

	// Strong‑trend detection (value > threshold) should be true.
	st, err := osc.IsStrongTrend()
	if err != nil {
		t.Fatalf("strong‑trend err: %v", err)
	}
	if !st {
		t.Fatalf("expected strong‑trend flag")
	}

	// Divergence – craft a price move opposite to the oscillator direction.
	// We need at least two closes; reuse the existing internal slices.
	osc.closes = []float64{100, 101}     // price went up
	osc.vwaoValues = []float64{-20, -30} // oscillator went further down
	osc.lastValue = -30

	div, dir, err := osc.IsDivergence()
	if err != nil {
		t.Fatalf("divergence err: %v", err)
	}
	if !div || dir != "Bullish" {
		t.Fatalf("expected bullish divergence, got %v %s", div, dir)
	}
}
