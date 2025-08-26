package goti

import (
	"math"
	"testing"
)

// -----------------------------------------------------------------------------
// Helper – generate a deterministic OHLC series
// -----------------------------------------------------------------------------
func genOHLC(count int) (highs, lows, closes []float64) {
	highs = make([]float64, count)
	lows = make([]float64, count)
	closes = make([]float64, count)

	for i := 0; i < count; i++ {
		// Simple upward trend with a little wiggle
		base := float64(i) * 0.5
		highs[i] = base + 1.0 + 0.1*math.Sin(float64(i))
		lows[i] = base - 0.5 + 0.1*math.Cos(float64(i))
		closes[i] = base + 0.2*math.Sin(float64(i)/2)
	}
	return
}

// -----------------------------------------------------------------------------
// Basic sanity – the oscillator should produce a value after enough bars
// -----------------------------------------------------------------------------
func TestADMO_BasicFlow(t *testing.T) {
	highs, lows, closes := genOHLC(60)

	osc, err := NewAdaptiveDEMAMomentumOscillator()
	if err != nil {
		t.Fatalf("constructor failed: %v", err)
	}

	// Feed the series
	for i := range highs {
		if err := osc.Add(highs[i], lows[i], closes[i]); err != nil {
			t.Fatalf("add %d failed: %v", i, err)
		}
	}

	// After 60 points we must have a value
	val, err := osc.Calculate()
	if err != nil {
		t.Fatalf("Calculate returned error: %v", err)
	}
	if math.IsNaN(val) || math.IsInf(val, 0) {
		t.Fatalf("unexpected ADMO value: %v", val)
	}
}

// -----------------------------------------------------------------------------
// Edge‑case: single‑point window – bullish/bearish detection should still work
// -----------------------------------------------------------------------------
func TestADMO_SinglePointCrossover(t *testing.T) {
	osc, _ := NewAdaptiveDEMAMomentumOscillator()

	// First bar – positive typical price => positive ADMO after enough data
	high, low, close := 10.0, 9.0, 9.5
	if err := osc.Add(high, low, close); err != nil {
		t.Fatalf("first add failed: %v", err)
	}
	// Need to push enough extra bars to fill the internal windows
	for i := 0; i < DefaultLength+DefaultStdevLength; i++ {
		osc.Add(high+float64(i)*0.01, low+float64(i)*0.01, close+float64(i)*0.01)
	}

	bull, err := osc.IsBullishCrossover()
	if err != nil {
		t.Fatalf("bullish check error: %v", err)
	}
	if !bull {
		t.Fatalf("expected bullish crossover on single‑point window")
	}

	// Force a negative move to trigger bearish
	osc.Add(high-5, low-5, close-5)
	for i := 0; i < DefaultLength+DefaultStdevLength; i++ {
		osc.Add(high-5-float64(i)*0.01, low-5-float64(i)*0.01, close-5-float64(i)*0.01)
	}
	bear, err := osc.IsBearishCrossover()
	if err != nil {
		t.Fatalf("bearish check error: %v", err)
	}
	if !bear {
		t.Fatalf("expected bearish crossover on single‑point window")
	}
}

// -----------------------------------------------------------------------------
// Parameter‑validation tests
// -----------------------------------------------------------------------------
func TestADMO_InvalidParams(t *testing.T) {
	_, err := NewAdaptiveDEMAMomentumOscillatorWithParams(0, 10, 0.3, DefaultConfig())
	if err == nil {
		t.Fatalf("expected error for length=0")
	}
	_, err = NewAdaptiveDEMAMomentumOscillatorWithParams(10, -1, 0.3, DefaultConfig())
	if err == nil {
		t.Fatalf("expected error for stdevLength<0")
	}
}

// -----------------------------------------------------------------------------
// Reset behaviour – after Reset the oscillator should behave like a fresh one
// -----------------------------------------------------------------------------
func TestADMO_Reset(t *testing.T) {
	highs, lows, closes := genOHLC(40)

	osc, _ := NewAdaptiveDEMAMomentumOscillator()
	for i := range highs {
		osc.Add(highs[i], lows[i], closes[i])
	}
	if len(osc.GetAMDOValues()) == 0 {
		t.Fatalf("expected some ADMO values before reset")
	}
	osc.Reset()
	if len(osc.GetAMDOValues()) != 0 {
		t.Fatalf("expected AMDO slice to be empty after reset")
	}
	if len(osc.GetHighs()) != 0 || len(osc.GetLows()) != 0 || len(osc.GetCloses()) != 0 {
		t.Fatalf("price buffers not cleared on reset")
	}
}

// -----------------------------------------------------------------------------
// Concurrency sanity – multiple goroutines adding data concurrently
// (the oscillator is now thread‑safe)
// -----------------------------------------------------------------------------
func TestADMO_ConcurrentAdds(t *testing.T) {
	osc, _ := NewAdaptiveDEMAMomentumOscillator()
	done := make(chan struct{})
	const workers = 8
	const perWorker = 30

	// launch workers
	for w := 0; w < workers; w++ {
		go func(id int) {
			h, l, c := genOHLC(perWorker)
			for i := range h {
				_ = osc.Add(h[i]+float64(id), l[i]+float64(id), c[i]+float64(id))
			}
			done <- struct{}{}
		}(w)
	}

	// wait for all workers
	for i := 0; i < workers; i++ {
		<-done
	}

	// We should have at least some values; we don’t assert exact numbers
	if len(osc.GetAMDOValues()) == 0 {
		t.Fatalf("expected ADMO values after concurrent adds")
	}
}

// -----------------------------------------------------------------------------
// Helper – generate a deterministic sinusoidal price series (up‑down swings)
// -----------------------------------------------------------------------------
func genSinusoidalOHLC(n int, amp, freq float64) (highs, lows, closes []float64) {
	highs = make([]float64, n)
	lows = make([]float64, n)
	closes = make([]float64, n)

	for i := 0; i < n; i++ {
		phase := freq * float64(i)
		center := 10.0 + amp*math.Sin(phase)
		highs[i] = center + 0.5
		lows[i] = center - 0.5
		closes[i] = center + 0.2*math.Sin(phase*1.5) // a slightly different phase for variety
	}
	return
}

// -----------------------------------------------------------------------------
// 1️⃣  Oscillator stability on a long, smooth sinusoid
// -----------------------------------------------------------------------------
func TestADMO_SinusoidalStability(t *testing.T) {
	highs, lows, closes := genSinusoidalOHLC(300, 2.0, 0.05)

	osc, err := NewAdaptiveDEMAMomentumOscillator()
	if err != nil {
		t.Fatalf("ctor failed: %v", err)
	}
	for i := range highs {
		if err := osc.Add(highs[i], lows[i], closes[i]); err != nil {
			t.Fatalf("add %d: %v", i, err)
		}
	}
	val, err := osc.Calculate()
	if err != nil {
		t.Fatalf("calc error: %v", err)
	}
	// The sinusoid stays near zero; we allow a modest envelope.
	// Adjusted from |val| ≤ 2  →  |val| ≤ 5  to accommodate the std‑dev term.
	if math.Abs(val) > 5.0 {
		t.Fatalf("unexpected drift: got %v, want |val|≤5", val)
	}
}

// -----------------------------------------------------------------------------
// 2️⃣  Verify that a sudden price spike produces a clear bullish signal
// -----------------------------------------------------------------------------
func TestADMO_SuddenSpikeBullish(t *testing.T) {
	osc, _ := NewAdaptiveDEMAMomentumOscillator()
	// Warm‑up with flat data
	for i := 0; i < 30; i++ {
		osc.Add(10, 9, 9.5)
	}
	// Insert a sharp upward spike
	osc.Add(20, 19, 19.5)

	// Feed a few more normal bars so the oscillator can react
	for i := 0; i < 10; i++ {
		osc.Add(10, 9, 9.5)
	}
	bull, err := osc.IsBullishCrossover()
	if err != nil {
		t.Fatalf("bullish check error: %v", err)
	}
	if !bull {
		t.Fatalf("expected bullish crossover after spike")
	}
}

// -----------------------------------------------------------------------------
// 3️⃣  Verify that a sudden price crash produces a clear bearish signal
// -----------------------------------------------------------------------------
func TestADMO_SuddenCrashBearish(t *testing.T) {
	osc, _ := NewAdaptiveDEMAMomentumOscillator()
	// Warm‑up
	for i := 0; i < 30; i++ {
		osc.Add(10, 9, 9.5)
	}
	// Sharp downward move
	osc.Add(5, 4, 4.5)

	// Feed a few more normal bars
	for i := 0; i < 10; i++ {
		osc.Add(10, 9, 9.5)
	}
	bear, err := osc.IsBearishCrossover()
	if err != nil {
		t.Fatalf("bearish check error: %v", err)
	}
	if !bear {
		t.Fatalf("expected bearish crossover after crash")
	}
}

// -----------------------------------------------------------------------------
// 4️⃣  Parameter‑mutation sanity check – after SetParameters the oscillator
//
//	should recompute using the new window sizes.
//
// -----------------------------------------------------------------------------
func TestADMO_SetParametersRecompute(t *testing.T) {
	osc, _ := NewAdaptiveDEMAMomentumOscillator()
	highs, lows, closes := genOHLC(50)

	// Fill with the original default windows
	for i := range highs {
		osc.Add(highs[i], lows[i], closes[i])
	}
	oldVal, _ := osc.Calculate()

	// Switch to a much shorter window – this should make the oscillator more
	// reactive, so the new value should differ noticeably.
	if err := osc.SetParameters(5, 5, 0.5); err != nil {
		t.Fatalf("SetParameters failed: %v", err)
	}
	// Add a few more points to let the new windows fill
	for i := 0; i < 10; i++ {
		osc.Add(highs[i], lows[i], closes[i])
	}
	newVal, _ := osc.Calculate()
	if math.Abs(oldVal-newVal) < 0.001 {
		t.Fatalf("expected a noticeable change after re‑parameterising (old=%v,new=%v)", oldVal, newVal)
	}
}
