package indicator

import (
	"testing"
)

// ---------------------------------------------------------------------------
// Helper that exposes the unexported computeVWAO for benchmarking only.
// It lives in the *_test.go file, so it is invisible to the library itself.
// ---------------------------------------------------------------------------
func (v *VolumeWeightedAroonOscillator) benchCompute() (float64, error) {
	return v.computeVWAO()
}

// ---------------------------------------------------------------------------
// Benchmark: adding candles (validation + possible VWAO calculation).
// ---------------------------------------------------------------------------
func BenchmarkVWAO_Add(b *testing.B) {
	// Use a modest period so that a VWAO is computed on almost every Add.
	const period = 10
	osc, _ := NewVolumeWeightedAroonOscillatorWithParams(period, DefaultConfig())

	// Pre‑generate deterministic data to avoid allocation inside the loop.
	highs, lows, closes, vols := genDeterministicData(period)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Cycle through the pre‑generated slice so we never run out of data.
		idx := i % len(highs)
		if err := osc.Add(highs[idx], lows[idx], closes[idx], vols[idx]); err != nil {
			b.Fatalf("Add failed: %v", err)
		}
	}
}

// ---------------------------------------------------------------------------
// Parallel version of the Add benchmark.
// ---------------------------------------------------------------------------
func BenchmarkVWAO_AddParallel(b *testing.B) {
	const period = 10
	highs, lows, closes, vols := genDeterministicData(period)

	b.RunParallel(func(pb *testing.PB) {
		// Each goroutine gets its own oscillator instance – otherwise we would
		// race on the internal slices.
		osc, _ := NewVolumeWeightedAroonOscillatorWithParams(period, DefaultConfig())
		i := 0
		for pb.Next() {
			idx := i % len(highs)
			if err := osc.Add(highs[idx], lows[idx], closes[idx], vols[idx]); err != nil {
				b.Fatalf("Add failed: %v", err)
			}
			i++
		}
	})
}

// ---------------------------------------------------------------------------
// Benchmark that isolates the pure VWAO calculation (no slice trimming, no
// validation).  The oscillator is primed with a full window first.
// ---------------------------------------------------------------------------
func BenchmarkVWAO_ComputeOnly(b *testing.B) {
	const period = 14
	osc, _ := NewVolumeWeightedAroonOscillatorWithParams(period, DefaultConfig())

	// Fill the oscillator once so computeVWAO has a full window.
	highs, lows, closes, vols := genDeterministicData(period)
	for i := 0; i < len(highs); i++ {
		_ = osc.Add(highs[i], lows[i], closes[i], vols[i])
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := osc.benchCompute(); err != nil {
			b.Fatalf("compute error: %v", err)
		}
	}
}

// ---------------------------------------------------------------------------
// Benchmark Reset – fill then clear repeatedly.
// ---------------------------------------------------------------------------
func BenchmarkVWAO_Reset(b *testing.B) {
	const period = 20
	osc, _ := NewVolumeWeightedAroonOscillatorWithParams(period, DefaultConfig())
	highs, lows, closes, vols := genDeterministicData(period)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Fill the window (period+1 candles) – this mimics a realistic usage pattern.
		for j := 0; j <= period; j++ {
			_ = osc.Add(highs[j%len(highs)], lows[j%len(lows)], closes[j%len(closes)], vols[j%len(vols)])
		}
		osc.Reset()
	}
}

// ---------------------------------------------------------------------------
// Benchmark GetPlotData – generate a modest series and repeatedly request the
// plot structures.  This also touches timestamp generation.
// ---------------------------------------------------------------------------
func BenchmarkVWAO_GetPlotData(b *testing.B) {
	const period = 14
	osc, _ := NewVolumeWeightedAroonOscillatorWithParams(period, DefaultConfig())

	// Produce enough candles for a handful of VWAO values (say 100).
	const totalCandles = 100
	highs, lows, closes, vols := genDeterministicData(period)
	for i := 0; i < totalCandles; i++ {
		idx := i % len(highs)
		_ = osc.Add(highs[idx], lows[idx], closes[idx], vols[idx])
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = osc.GetPlotData(1_600_000_000, 60_000)
	}
}
