package goti

import (
	"errors"
	"testing"
)

// ---------------------------------------------------------------------------
// Helper – creates a fresh RSI with the *default* configuration.
// ---------------------------------------------------------------------------
func newDefaultRSI(t *testing.T) *RelativeStrengthIndex {
	rsi, err := NewRelativeStrengthIndex()
	if err != nil {
		t.Fatalf("unexpected error creating RSI: %v", err)
	}
	return rsi
}

// ---------------------------------------------------------------------------
// Construction & basic validation
// ---------------------------------------------------------------------------
func TestNewRelativeStrengthIndex_WithInvalidPeriod(t *testing.T) {
	_, err := NewRelativeStrengthIndexWithParams(0, DefaultConfig())
	if err == nil {
		t.Fatalf("expected error for period < 1")
	}
}

func TestNewRelativeStrengthIndex_WithBadConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RSIOverbought = 40
	cfg.RSIOversold = 60 // overbought <= oversold -> invalid
	_, err := NewRelativeStrengthIndexWithParams(5, cfg)
	if err == nil {
		t.Fatalf("expected error when overbought <= oversold")
	}
}

// ---------------------------------------------------------------------------
// Adding data & basic calculation
// ---------------------------------------------------------------------------
func TestRSI_FirstValue_UsesSimpleAverages(t *testing.T) {
	rsi := newDefaultRSI(t)

	// 5‑period RSI → need 6 closes to emit first value.
	prices := []float64{10, 11, 12, 13, 14, 15}
	for _, p := range prices {
		if err := rsi.Add(p); err != nil {
			t.Fatalf("Add failed: %v", err)
		}
	}

	val, err := rsi.Calculate()
	if err != nil {
		t.Fatalf("Calculate failed: %v", err)
	}
	// With a monotonic rise, avgLoss = 0 → RSI should be 100.
	if val != 100 {
		t.Fatalf("expected first RSI = 100 (pure up), got %v", val)
	}
}

func TestRSI_FirstValue_PureDownwardMovement(t *testing.T) {
	rsi := newDefaultRSI(t)

	prices := []float64{15, 14, 13, 12, 11, 10}
	for _, p := range prices {
		if err := rsi.Add(p); err != nil {
			t.Fatalf("Add failed: %v", err)
		}
	}

	val, err := rsi.Calculate()
	if err != nil {
		t.Fatalf("Calculate failed: %v", err)
	}
	if val != 0 {
		t.Fatalf("expected first RSI = 0 (pure down), got %v", val)
	}
}

func TestRSI_FirstValue_NeutralMovement(t *testing.T) {
	rsi := newDefaultRSI(t)

	// Flat price series → avgGain = avgLoss → RSI = 50
	prices := []float64{10, 10, 10, 10, 10, 10}
	for _, p := range prices {
		if err := rsi.Add(p); err != nil {
			t.Fatalf("Add failed: %v", err)
		}
	}

	val, err := rsi.Calculate()
	if err != nil {
		t.Fatalf("Calculate failed: %v", err)
	}
	if val != 50 {
		t.Fatalf("expected first RSI = 50 (no movement), got %v", val)
	}
}

// ---------------------------------------------------------------------------
// Wilder smoothing after the seed period
// ---------------------------------------------------------------------------
func TestRSI_WilderSmoothing_ContinuesCorrectly(t *testing.T) {
	rsi := newDefaultRSI(t)

	// Seed with a mixed series that yields a known first RSI.
	seed := []float64{44, 45, 46, 45, 44, 43} // 5‑period → first RSI after 6th point
	for _, p := range seed {
		if err := rsi.Add(p); err != nil {
			t.Fatalf("seed Add failed: %v", err)
		}
	}
	first, _ := rsi.Calculate()

	// Append a new price and verify the second value differs (i.e. smoothing applied).
	if err := rsi.Add(42); err != nil {
		t.Fatalf("second Add failed: %v", err)
	}
	second, err := rsi.Calculate()
	if err != nil {
		t.Fatalf("second Calculate failed: %v", err)
	}
	if first == second {
		t.Fatalf("expected RSI to change after smoothing, got same value %v", first)
	}
}

// ---------------------------------------------------------------------------
// Edge‑case handling (zero loss / zero gain)
// ---------------------------------------------------------------------------
func TestRSI_EdgeCases_ZeroLossOrGain(t *testing.T) {
	rsi := newDefaultRSI(t)

	// Build a scenario where avgLoss becomes zero after the seed.
	prices := []float64{10, 11, 12, 13, 14, 15} // all up → first RSI = 100
	for _, p := range prices {
		if err := rsi.Add(p); err != nil {
			t.Fatalf("Add failed: %v", err)
		}
	}
	val, _ := rsi.Calculate()
	if val != 100 {
		t.Fatalf("expected 100 after pure up, got %v", val)
	}

	// Now add a down tick – avgLoss will become non‑zero, RSI should drop.
	if err := rsi.Add(14); err != nil {
		t.Fatalf("Add down tick failed: %v", err)
	}
	val2, err := rsi.Calculate()
	if err != nil {
		t.Fatalf("Calculate after down tick failed: %v", err)
	}
	if val2 >= 100 {
		t.Fatalf("expected RSI to decrease after a loss, got %v", val2)
	}
}

// ---------------------------------------------------------------------------
// Bullish crossover – price moves from below the oversold line to above it.
// ---------------------------------------------------------------------------
func TestRSI_BullishCrossoverDetection(t *testing.T) {
	rsi := newDefaultRSI(t)

	/*
	   Goal: generate a first RSI that is **≤ 30** (oversold) and a second RSI that
	   becomes **> 30**, thereby triggering a bullish crossover.

	   • The first 6 closes form a steep down‑trend → RSI ≈ 0.
	   • Two subsequent upward closes push the RSI above 30.
	   • Two extra “neutral” closes keep the two most‑recent RSI values in the
	     internal slice (the crossover helpers need at least two values).
	*/

	prices := []float64{
		100, 90, 80, 70, 60, // 5 descending closes
		50,     // sixth close → first RSI (still oversold / ~0)
		55, 60, // two upward closes → second RSI should cross >30
		62, 64, // extra points – preserve the two RSI values
	}

	for i, p := range prices {
		if err := rsi.Add(p); err != nil {
			t.Fatalf("Add failed at idx %d (price %.2f): %v", i, p, err)
		}
	}

	// Verify we really have ≥2 RSI values before checking the crossover.
	if len(rsi.GetRSIValues()) < 2 {
		t.Fatalf("expected ≥2 RSI values, got %d", len(rsi.GetRSIValues()))
	}

	cross, err := rsi.IsBullishCrossover()
	if err != nil {
		t.Fatalf("IsBullishCrossover returned error: %v", err)
	}
	if !cross {
		t.Fatalf("expected bullish crossover, got false – RSI values: %v", rsi.GetRSIValues())
	}
}

// ---------------------------------------------------------------------------
// Bearish crossover – price moves from above the overbought line to below it.
// ---------------------------------------------------------------------------
func TestRSI_BearishCrossoverDetection(t *testing.T) {
	rsi := newDefaultRSI(t)

	/*
	   Goal: first RSI **≥ 70** (overbought) then a second RSI **< 70**.

	   • Six ascending closes → first RSI ≈ 100.
	   • Two descending closes → second RSI should dip below 70.
	   • Two extra points keep the two latest RSI values alive.
	*/

	prices := []float64{
		10, 20, 30, 40, 50, // 5 ascending closes
		60,     // sixth close → first RSI (overbought / ~100)
		55, 50, // two downward closes → second RSI should cross <70
		48, 46, // extra points – preserve the two RSI values
	}

	for i, p := range prices {
		if err := rsi.Add(p); err != nil {
			t.Fatalf("Add failed at idx %d (price %.2f): %v", i, p, err)
		}
	}

	if len(rsi.GetRSIValues()) < 2 {
		t.Fatalf("expected ≥2 RSI values, got %d", len(rsi.GetRSIValues()))
	}

	cross, err := rsi.IsBearishCrossover()
	if err != nil {
		t.Fatalf("IsBearishCrossover returned error: %v", err)
	}
	if !cross {
		t.Fatalf("expected bearish crossover, got false – RSI values: %v", rsi.GetRSIValues())
	}
}

// ---------------------------------------------------------------------------
// Overbought / Oversold status reporting
// ---------------------------------------------------------------------------
func TestRSI_GetOverboughtOversold(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RSIOverbought = 70
	cfg.RSIOversold = 30
	rsi, _ := NewRelativeStrengthIndexWithParams(5, cfg)

	// Force an overbought condition.
	prices := []float64{10, 11, 12, 13, 14, 15}
	for _, p := range prices {
		if err := rsi.Add(p); err != nil {
			t.Fatalf("Add failed: %v", err)
		}
	}
	status, err := rsi.GetOverboughtOversold()
	if err != nil {
		t.Fatalf("GetOverboughtOversold error: %v", err)
	}
	if status != "Overbought" {
		t.Fatalf("expected Overbought, got %s", status)
	}

	// Force an oversold condition.
	rsi.Reset()
	prices = []float64{15, 14, 13, 12, 11, 10}
	for _, p := range prices {
		if err := rsi.Add(p); err != nil {
			t.Fatalf("Add failed: %v", err)
		}
	}
	status, err = rsi.GetOverboughtOversold()
	if err != nil {
		t.Fatalf("GetOverboughtOversold error: %v", err)
	}
	if status != "Oversold" {
		t.Fatalf("expected Oversold, got %s", status)
	}
}

// ---------------------------------------------------------------------------
// Divergence detection
// ---------------------------------------------------------------------------
func TestRSI_Divergence_Bearish(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RSIOverbought = 70
	rsi, _ := NewRelativeStrengthIndexWithParams(5, cfg)

	// Create a situation where RSI is high but price drops.
	prices := []float64{10, 11, 12, 13, 14, 15, // rising → high RSI
		14, // price down one tick, RSI still high
	}
	for _, p := range prices {
		if err := rsi.Add(p); err != nil {
			t.Fatalf("Add failed: %v", err)
		}
	}
	ok, typ, err := rsi.IsDivergence()
	if err != nil {
		t.Fatalf("IsDivergence error: %v", err)
	}
	if !ok || typ != "Bearish" {
		t.Fatalf("expected Bearish divergence, got ok=%v type=%s", ok, typ)
	}
}

func TestRSI_Divergence_Bullish(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RSIOversold = 30
	rsi, _ := NewRelativeStrengthIndexWithParams(5, cfg)

	// RSI low but price rises.
	prices := []float64{15, 14, 13, 12, 11, 10, // falling → low RSI
		11, // price up one tick, RSI still low
	}
	for _, p := range prices {
		if err := rsi.Add(p); err != nil {
			t.Fatalf("Add failed: %v", err)
		}
	}
	ok, typ, err := rsi.IsDivergence()
	if err != nil {
		t.Fatalf("IsDivergence error: %v", err)
	}
	if !ok || typ != "Bullish" {
		t.Fatalf("expected Bullish divergence, got ok=%v type=%s", ok, typ)
	}
}

// ---------------------------------------------------------------------------
// Period change handling
// ---------------------------------------------------------------------------
func TestRSI_SetPeriod_ResetsState(t *testing.T) {
	rsi := newDefaultRSI(t)

	// Fill with some data.
	prices := []float64{10, 11, 12, 13, 14, 15}
	for _, p := range prices {
		_ = rsi.Add(p)
	}
	if len(rsi.GetRSIValues()) == 0 {
		t.Fatalf("expected at least one RSI value before period change")
	}

	// Change period – internal averages should reset.
	if err := rsi.SetPeriod(10); err != nil {
		t.Fatalf("SetPeriod error: %v", err)
	}
	if rsi.avgGain != 0 || rsi.avgLoss != 0 {
		t.Fatalf("expected avgGain/avgLoss to be cleared after period change")
	}
}

// ---------------------------------------------------------------------------
// Slice trimming logic (internal bounds)
// ---------------------------------------------------------------------------
func TestRSI_SliceTrimming(t *testing.T) {
	rsi := newDefaultRSI(t)

	// Feed more than period+1 closes.
	for i := 0; i < 20; i++ {
		_ = rsi.Add(float64(i))
	}
	if len(rsi.GetCloses()) != rsi.period+1 {
		t.Fatalf("expected closes slice length %d, got %d", rsi.period+1, len(rsi.GetCloses()))
	}
	if len(rsi.GetRSIValues()) > rsi.period {
		t.Fatalf("RSI values slice exceeded period bound")
	}
}

// ---------------------------------------------------------------------------
// Invalid input handling
// ---------------------------------------------------------------------------
func TestRSI_Add_InvalidPrice(t *testing.T) {
	rsi := newDefaultRSI(t)

	err := rsi.Add(-5) // negative price should be rejected
	if err == nil {
		t.Fatalf("expected error for negative price")
	}
	if !errors.Is(err, errors.New("invalid price")) {
		// The exact error string isn’t crucial; just ensure we got an error.
	}
}

// ---------------------------------------------------------------------------
// Plot data generation sanity check
// ---------------------------------------------------------------------------
func TestRSI_GetPlotData(t *testing.T) {
	rsi := newDefaultRSI(t)

	// Populate enough points to have at least one RSI value.
	for i := 0; i < 7; i++ {
		_ = rsi.Add(float64(10 + i))
	}
	data := rsi.GetPlotData(1609459200, 60) // arbitrary start timestamp, 1‑min interval

	if len(data) != 2 {
		t.Fatalf("expected two PlotData series (RSI + Signals), got %d", len(data))
	}
	if data[0].Name != "Relative Strength Index" || data[1].Name != "Signals" {
		t.Fatalf("unexpected PlotData names: %v, %v", data[0].Name, data[1].Name)
	}
	if len(data[0].Y) != len(rsi.GetRSIValues()) {
		t.Fatalf("RSI PlotData length mismatch")
	}
}
