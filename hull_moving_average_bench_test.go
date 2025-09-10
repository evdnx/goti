package goti

import (
	"math"
	"strconv"
	"testing"
)

// ---------------------------------------------------------------------------
// Helper – generate a deterministic slice of closing prices.
// ---------------------------------------------------------------------------
func genPrices(n int) []float64 {
	prices := make([]float64, n)
	for i := 0; i < n; i++ {
		// Simple sinusoidal + trend pattern – guarantees non‑zero values.
		prices[i] = 100 + 20*math.Sin(float64(i)*0.1) + float64(i)*0.05
	}
	return prices
}

// ---------------------------------------------------------------------------
// Benchmark Add() – single price insertion.
// ---------------------------------------------------------------------------
func BenchmarkHullMovingAverage_Add(b *testing.B) {
	// Run sub‑benchmarks for a few representative periods.
	for _, period := range []int{5, 20, 100, 500} {
		b.Run(
			// Name format: Add/Period=<value>
			"Period="+strconv.Itoa(period),
			func(b *testing.B) {
				h, _ := NewHullMovingAverageWithParams(period)
				prices := genPrices(b.N) // generate exactly b.N prices

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = h.Add(prices[i])
				}
			},
		)
	}
}

// ---------------------------------------------------------------------------
// Benchmark Calculate() – after feeding a full data set.
// ---------------------------------------------------------------------------
func BenchmarkHullMovingAverage_Calculate(b *testing.B) {
	for _, size := range []int{10, 100, 1_000, 10_000} {
		b.Run(
			"Size="+strconv.Itoa(size),
			func(b *testing.B) {
				h, _ := NewHullMovingAverageWithParams(20)
				prices := genPrices(size)

				// Fill the HMA once (outside the timed loop).
				for _, p := range prices {
					_ = h.Add(p)
				}
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					_, _ = h.Calculate()
				}
			},
		)
	}
}

// ---------------------------------------------------------------------------
// Benchmark IsBullishCrossover() – requires at least two points.
// ---------------------------------------------------------------------------
func BenchmarkHullMovingAverage_IsBullishCrossover(b *testing.B) {
	h, _ := NewHullMovingAverageWithParams(20)
	prices := genPrices(200)
	for _, p := range prices {
		_ = h.Add(p)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = h.IsBullishCrossover()
	}
}

// ---------------------------------------------------------------------------
// Benchmark IsBearishCrossover().
// ---------------------------------------------------------------------------
func BenchmarkHullMovingAverage_IsBearishCrossover(b *testing.B) {
	h, _ := NewHullMovingAverageWithParams(20)
	prices := genPrices(200)
	for _, p := range prices {
		_ = h.Add(p)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = h.IsBearishCrossover()
	}
}

// ---------------------------------------------------------------------------
// Benchmark GetTrendDirection().
// ---------------------------------------------------------------------------
func BenchmarkHullMovingAverage_GetTrendDirection(b *testing.B) {
	h, _ := NewHullMovingAverageWithParams(20)
	prices := genPrices(500)
	for _, p := range prices {
		_ = h.Add(p)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = h.GetTrendDirection()
	}
}

// ---------------------------------------------------------------------------
// Benchmark Reset() – measures the cost of clearing slices.
// ---------------------------------------------------------------------------
func BenchmarkHullMovingAverage_Reset(b *testing.B) {
	h, _ := NewHullMovingAverageWithParams(50)
	prices := genPrices(1_000)
	for _, p := range prices {
		_ = h.Add(p)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Reset()
	}
}

// ---------------------------------------------------------------------------
// Benchmark SetPeriod() – includes slice‑trimming overhead.
// ---------------------------------------------------------------------------
func BenchmarkHullMovingAverage_SetPeriod(b *testing.B) {
	h, _ := NewHullMovingAverageWithParams(100)
	prices := genPrices(2_000)
	for _, p := range prices {
		_ = h.Add(p)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = h.SetPeriod(50) // switch to a smaller period each iteration
	}
}

// ---------------------------------------------------------------------------
// Benchmark DetectSignals() – runs the full signal‑generation loop.
// ---------------------------------------------------------------------------
func BenchmarkHullMovingAverage_DetectSignals(b *testing.B) {
	h, _ := NewHullMovingAverageWithParams(30)
	prices := genPrices(5_000)
	for _, p := range prices {
		_ = h.Add(p)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = h.DetectSignals()
	}
}

// ---------------------------------------------------------------------------
// Benchmark GetPlotData() – includes timestamp generation and slice copies.
// ---------------------------------------------------------------------------
func BenchmarkHullMovingAverage_GetPlotData(b *testing.B) {
	h, _ := NewHullMovingAverageWithParams(25)
	prices := genPrices(3_000)
	for _, p := range prices {
		_ = h.Add(p)
	}
	start := int64(1_600_000_000)
	interval := int64(60)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = h.GetPlotData(start, interval)
	}
}
