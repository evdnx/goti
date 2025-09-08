package goti

import (
	"math/rand"
	"testing"
)

func nrandVals(n int) []float64 {
	r := rand.New(rand.NewSource(42))
	vals := make([]float64, n)
	for i := 0; i < n; i++ {
		vals[i] = r.Float64() * 100
	}
	return vals
}

/*
   MovingAverage benchmarks
   -----------------------
   We benchmark each supported type (SMA, EMA, WMA) for three different periods:
   small (5), medium (20) and large (200).  The benchmark measures the cost of
   repeatedly calling `Add` followed by `Calculate`.  The slice is trimmed on
   every `Add`, matching the production behaviour.
*/

func benchmarkMovingAverage(b *testing.B, maType MovingAverageType, period int) {
	ma, err := NewMovingAverage(maType, period)
	if err != nil {
		b.Fatalf("failed to create MovingAverage: %v", err)
	}
	values := nrandVals(b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := ma.Add(values[i%len(values)]); err != nil {
			b.Fatalf("Add error: %v", err)
		}
		_, err := ma.Calculate()
		if err != nil && i >= period-1 { // ignore “insufficient data” early on
			b.Fatalf("Calculate error: %v", err)
		}
	}
}

func BenchmarkMovingAverage_SMA_Small(b *testing.B)  { benchmarkMovingAverage(b, SMAMovingAverage, 5) }
func BenchmarkMovingAverage_SMA_Medium(b *testing.B) { benchmarkMovingAverage(b, SMAMovingAverage, 20) }
func BenchmarkMovingAverage_SMA_Large(b *testing.B)  { benchmarkMovingAverage(b, SMAMovingAverage, 200) }

func BenchmarkMovingAverage_EMA_Small(b *testing.B)  { benchmarkMovingAverage(b, EMAMovingAverage, 5) }
func BenchmarkMovingAverage_EMA_Medium(b *testing.B) { benchmarkMovingAverage(b, EMAMovingAverage, 20) }
func BenchmarkMovingAverage_EMA_Large(b *testing.B)  { benchmarkMovingAverage(b, EMAMovingAverage, 200) }

func BenchmarkMovingAverage_WMA_Small(b *testing.B)  { benchmarkMovingAverage(b, WMAMovingAverage, 5) }
func BenchmarkMovingAverage_WMA_Medium(b *testing.B) { benchmarkMovingAverage(b, WMAMovingAverage, 20) }
func BenchmarkMovingAverage_WMA_Large(b *testing.B)  { benchmarkMovingAverage(b, WMAMovingAverage, 200) }

/*
   EMA wrapper benchmarks
   ----------------------
   The `EMA` type stores all raw values internally and performs the seeding step
   (first EMA = SMA of the initial period).  Benchmarks cover three periods.
*/

func benchmarkEMA(b *testing.B, period int) {
	ema := NewEMA(period)
	values := nrandVals(b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := ema.Add(values[i%len(values)]); err != nil {
			b.Fatalf("EMA Add error: %v", err)
		}
		_, _ = ema.Calculate() // ignore “not seeded” error – it’s part of normal flow
	}
}

func BenchmarkEMA_Period5(b *testing.B)   { benchmarkEMA(b, 5) }
func BenchmarkEMA_Period20(b *testing.B)  { benchmarkEMA(b, 20) }
func BenchmarkEMA_Period200(b *testing.B) { benchmarkEMA(b, 200) }
