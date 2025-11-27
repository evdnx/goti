package indicator

import (
	"errors"
	"testing"
)

// ---------------------------------------------------------------------------
// Helper that builds a fully‑populated RSI (10 000 points) ready for the
// steady‑state benchmarks.  Returns the RSI and the slice of generated closes.
// ---------------------------------------------------------------------------
func prepRSI(b *testing.B) (*RelativeStrengthIndex, []float64) {
	rsi, err := NewRelativeStrengthIndex()
	if err != nil {
		b.Fatalf("failed to create RSI: %v", err)
	}

	// Generate a deterministic but slightly noisy price series.
	const total = 10_000
	closes := make([]float64, total)
	price := 100.0
	for i := 0; i < total; i++ {
		// Simple sinusoidal wiggle + small linear drift.
		price = price + 0.01*float64(i%5) - 0.005*float64(i%3)
		if price < 0 {
			price = 0.1 // guard against accidental negatives
		}
		closes[i] = price
		if err := rsi.Add(price); err != nil {
			b.Fatalf("Add failed at %d: %v", i, err)
		}
	}
	return rsi, closes
}

// ---------------------------------------------------------------------------
// Benchmark: adding a single price (includes slice trimming & validation).
// ---------------------------------------------------------------------------
func BenchmarkRSI_Add(b *testing.B) {
	// Fresh RSI for each iteration so we measure only the Add cost.
	for i := 0; i < b.N; i++ {
		rsi, _ := NewRelativeStrengthIndex()
		if err := rsi.Add(123.45); err != nil {
			b.Fatalf("Add error: %v", err)
		}
	}
}

// ---------------------------------------------------------------------------
// Benchmark: calculating the current RSI after the seed period.
// ---------------------------------------------------------------------------
func BenchmarkRSI_Calculate(b *testing.B) {
	rsi, _ := prepRSI(b)

	// Ensure we have at least one RSI value; otherwise Calculate would error.
	if _, err := rsi.Calculate(); err != nil {
		b.Fatalf("initial Calculate failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := rsi.Calculate(); err != nil {
			b.Fatalf("Calculate error: %v", err)
		}
	}
}

// ---------------------------------------------------------------------------
// Benchmark: typical live‑tick cycle – Add a new price then Calculate.
// ---------------------------------------------------------------------------
func BenchmarkRSI_AddAndCalc(b *testing.B) {
	rsi, _ := prepRSI(b)

	// Use a deterministic new price that varies with the loop counter.
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		price := 100.0 + float64(i%10)
		if err := rsi.Add(price); err != nil {
			b.Fatalf("Add error: %v", err)
		}
		if _, err := rsi.Calculate(); err != nil {
			b.Fatalf("Calculate error: %v", err)
		}
	}
}

// ---------------------------------------------------------------------------
// Benchmark: bullish‑crossover detection (requires ≥2 RSI values).
// ---------------------------------------------------------------------------
func BenchmarkRSI_IsBullishCrossover(b *testing.B) {
	rsi, _ := prepRSI(b)

	// Force a known state where a bullish crossover is possible.
	// We'll add a down‑trend to get RSI < oversold, then an up‑tick.
	if err := rsi.Add(10); err != nil {
		b.Fatalf("setup Add error: %v", err)
	}
	if err := rsi.Add(12); err != nil {
		b.Fatalf("setup Add error: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Alternate a tiny up/down move to keep the two‑RSI buffer alive.
		price := 11.0 + float64(i%2)
		if err := rsi.Add(price); err != nil {
			b.Fatalf("Add error: %v", err)
		}
		if _, err := rsi.IsBullishCrossover(); err != nil && !errors.Is(err, errors.New("insufficient data for crossover")) {
			b.Fatalf("IsBullishCrossover error: %v", err)
		}
	}
}

// ---------------------------------------------------------------------------
// Benchmark: bearish‑crossover detection.
// ---------------------------------------------------------------------------
func BenchmarkRSI_IsBearishCrossover(b *testing.B) {
	rsi, _ := prepRSI(b)

	// Seed with an up‑trend so the first RSI is overbought.
	if err := rsi.Add(200); err != nil {
		b.Fatalf("setup Add error: %v", err)
	}
	if err := rsi.Add(190); err != nil {
		b.Fatalf("setup Add error: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		price := 180.0 - float64(i%2) // slight down‑move each iteration
		if err := rsi.Add(price); err != nil {
			b.Fatalf("Add error: %v", err)
		}
		if _, err := rsi.IsBearishCrossover(); err != nil && !errors.Is(err, errors.New("insufficient data for crossover")) {
			b.Fatalf("IsBearishCrossover error: %v", err)
		}
	}
}

// ---------------------------------------------------------------------------
// Benchmark: generating PlotData (RSI line + signal scatter).
// ---------------------------------------------------------------------------
func BenchmarkRSI_GetPlotData(b *testing.B) {
	rsi, _ := prepRSI(b)

	// Use a fixed start timestamp and 1‑minute interval.
	const startTS = 1_640_995_200 // 2022‑01‑01 00:00:00 UTC
	const interval = int64(60)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rsi.GetPlotData(startTS, interval)
	}
}

// ---------------------------------------------------------------------------
// Benchmark: resetting the RSI (clears all slices & internal state).
// ---------------------------------------------------------------------------
func BenchmarkRSI_Reset(b *testing.B) {
	rsi, _ := prepRSI(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsi.Reset()
		// Re‑populate a tiny amount so the next Reset isn’t a no‑op (keeps the
		// benchmark realistic).
		if err := rsi.Add(100); err != nil {
			b.Fatalf("Add after Reset error: %v", err)
		}
	}
}
