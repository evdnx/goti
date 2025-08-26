package goti

import (
	"testing"
)

// -----------------------------------------------------------------------------
// Benchmark: pure Add() throughput (no extra work)
// -----------------------------------------------------------------------------
func BenchmarkADMO_Add(b *testing.B) {
	osc, _ := NewAdaptiveDEMAMomentumOscillator()
	high, low, close := 10.0, 9.0, 9.5

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Slightly vary the input to avoid compiler optimisations
		_ = osc.Add(high+float64(i)*0.001, low+float64(i)*0.001, close+float64(i)*0.001)
	}
}

// -----------------------------------------------------------------------------
// Benchmark: Add() followed by Calculate() (typical usage pattern)
// -----------------------------------------------------------------------------
func BenchmarkADMO_Add_Calculate(b *testing.B) {
	osc, _ := NewAdaptiveDEMAMomentumOscillator()
	high, low, close := 10.0, 9.0, 9.5

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = osc.Add(high+float64(i)*0.001, low+float64(i)*0.001, close+float64(i)*0.001)
		_, _ = osc.Calculate()
	}
}

// -----------------------------------------------------------------------------
// Benchmark: full pipeline – Add → IsBullishCrossover → IsBearishCrossover
// (covers read‑lock paths)
// -----------------------------------------------------------------------------
func BenchmarkADMO_FullPipeline(b *testing.B) {
	osc, _ := NewAdaptiveDEMAMomentumOscillator()
	high, low, close := 10.0, 9.0, 9.5

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = osc.Add(high+float64(i)*0.001, low+float64(i)*0.001, close+float64(i)*0.001)
		_, _ = osc.IsBullishCrossover()
		_, _ = osc.IsBearishCrossover()
	}
}

// -----------------------------------------------------------------------------
// Benchmark with the default configuration (length=20, stdevLength=14)
// -----------------------------------------------------------------------------
func BenchmarkADMO_DefaultConfig(b *testing.B) {
	osc, _ := NewAdaptiveDEMAMomentumOscillator()
	high, low, close := 10.0, 9.0, 9.5

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = osc.Add(high+float64(i)*0.001, low+float64(i)*0.001, close+float64(i)*0.001)
		_, _ = osc.Calculate()
	}
}

// -----------------------------------------------------------------------------
// Benchmark with a *short* window (more reactive, more work per update)
// -----------------------------------------------------------------------------
func BenchmarkADMO_ShortWindow(b *testing.B) {
	osc, _ := NewAdaptiveDEMAMomentumOscillatorWithParams(5, 5, 0.5, DefaultConfig())
	high, low, close := 10.0, 9.0, 9.5

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = osc.Add(high+float64(i)*0.001, low+float64(i)*0.001, close+float64(i)*0.001)
		_, _ = osc.Calculate()
	}
}

// -----------------------------------------------------------------------------
// Benchmark with a *long* window (less frequent updates, larger slices)
// -----------------------------------------------------------------------------
func BenchmarkADMO_LongWindow(b *testing.B) {
	osc, _ := NewAdaptiveDEMAMomentumOscillatorWithParams(100, 80, 0.2, DefaultConfig())
	high, low, close := 10.0, 9.0, 9.5

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = osc.Add(high+float64(i)*0.001, low+float64(i)*0.001, close+float64(i)*0.001)
		_, _ = osc.Calculate()
	}
}

// -----------------------------------------------------------------------------
// Benchmark that also calls the crossover helpers (read‑lock path)
// -----------------------------------------------------------------------------
func BenchmarkADMO_WithCrossovers(b *testing.B) {
	osc, _ := NewAdaptiveDEMAMomentumOscillator()
	high, low, close := 10.0, 9.0, 9.5

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = osc.Add(high+float64(i)*0.001, low+float64(i)*0.001, close+float64(i)*0.001)
		_, _ = osc.IsBullishCrossover()
		_, _ = osc.IsBearishCrossover()
	}
}
