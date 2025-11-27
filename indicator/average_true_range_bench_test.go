package indicator

import (
	"math/rand"
	"testing"
	"time"
)

/*
-------------------------------------------------------------

	Helper – generate a reproducible random OHLC slice
	-------------------------------------------------------------
*/
func randOHLC(n int, seed int64) (highs, lows, closes []float64) {
	r := rand.New(rand.NewSource(seed))
	highs = make([]float64, n)
	lows = make([]float64, n)
	closes = make([]float64, n)

	price := 100.0
	for i := 0; i < n; i++ {
		// simulate a modest random walk
		step := r.NormFloat64() * 0.5
		price += step
		if price < 1 {
			price = 1
		}
		high := price + r.Float64()*0.5
		low := price - r.Float64()*0.5
		close := low + r.Float64()*(high-low)

		highs[i] = high
		lows[i] = low
		closes[i] = close
	}
	return
}

/*
-------------------------------------------------------------

	Benchmark: Adding a single candle (core path)
	-------------------------------------------------------------
*/
func benchmarkAddCandle(b *testing.B, period int) {
	atr, _ := NewAverageTrueRangeWithParams(period)
	highs, lows, closes := randOHLC(b.N, time.Now().UnixNano())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = atr.AddCandle(highs[i], lows[i], closes[i])
	}
}

func BenchmarkAddCandle_Period5(b *testing.B)   { benchmarkAddCandle(b, 5) }
func BenchmarkAddCandle_Period14(b *testing.B)  { benchmarkAddCandle(b, 14) }
func BenchmarkAddCandle_Period50(b *testing.B)  { benchmarkAddCandle(b, 50) }
func BenchmarkAddCandle_Period200(b *testing.B) { benchmarkAddCandle(b, 200) }

/*
-------------------------------------------------------------

	Benchmark: Full pipeline – feed N candles then call Calculate
	-------------------------------------------------------------
*/
func benchmarkFullPipeline(b *testing.B, period, dataSize int) {
	highs, lows, closes := randOHLC(dataSize, 42)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		atr, _ := NewAverageTrueRangeWithParams(period)
		for j := 0; j < dataSize; j++ {
			_ = atr.AddCandle(highs[j], lows[j], closes[j])
		}
		_, _ = atr.Calculate() // ignore error – dataSize is always > period+1
	}
}

func BenchmarkFull_14_1000(b *testing.B)   { benchmarkFullPipeline(b, 14, 1000) }
func BenchmarkFull_14_5000(b *testing.B)   { benchmarkFullPipeline(b, 14, 5000) }
func BenchmarkFull_50_5000(b *testing.B)   { benchmarkFullPipeline(b, 50, 5000) }
func BenchmarkFull_200_10000(b *testing.B) { benchmarkFullPipeline(b, 200, 10000) }

/*
-------------------------------------------------------------

	Benchmark: Reset – repeatedly fill then reset
	-------------------------------------------------------------
*/
func benchmarkReset(b *testing.B, period, batchSize int) {
	highs, lows, closes := randOHLC(batchSize, 123)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		atr, _ := NewAverageTrueRangeWithParams(period)
		for j := 0; j < batchSize; j++ {
			_ = atr.AddCandle(highs[j], lows[j], closes[j])
		}
		atr.Reset()
	}
}

func BenchmarkReset_14_1000(b *testing.B) { benchmarkReset(b, 14, 1000) }
func BenchmarkReset_50_5000(b *testing.B) { benchmarkReset(b, 50, 5000) }

/*
-------------------------------------------------------------

	Benchmark: SetPeriod – change period after a warm‑up run
	-------------------------------------------------------------
*/
func BenchmarkSetPeriod(b *testing.B) {
	const warmSize = 2000
	highs, lows, closes := randOHLC(warmSize, 777)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		atr, _ := NewAverageTrueRangeWithParams(14)
		// Warm‑up with the original period
		for j := 0; j < warmSize; j++ {
			_ = atr.AddCandle(highs[j], lows[j], closes[j])
		}
		// Switch to a larger period – this forces a Reset internally
		_ = atr.SetPeriod(50)
	}
}

/*
-------------------------------------------------------------

	Benchmark: Getters – ensure defensive copies are cheap
	-------------------------------------------------------------
*/
func BenchmarkGetters(b *testing.B) {
	const size = 5000
	atr, _ := NewAverageTrueRangeWithParams(14)
	highs, lows, closes := randOHLC(size, 555)

	// Fill once – the cost of the getters is measured separately
	for i := 0; i < size; i++ {
		_ = atr.AddCandle(highs[i], lows[i], closes[i])
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = atr.GetHighs()
		_ = atr.GetLows()
		_ = atr.GetCloses()
		_ = atr.GetATRValues()
	}
}

/*
-------------------------------------------------------------

	Benchmark: Calculate – repeated reads after warm‑up
	-------------------------------------------------------------
*/
func BenchmarkCalculateRepeated(b *testing.B) {
	const size = 3000
	atr, _ := NewAverageTrueRangeWithParams(14)
	highs, lows, closes := randOHLC(size, 999)

	for i := 0; i < size; i++ {
		_ = atr.AddCandle(highs[i], lows[i], closes[i])
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = atr.Calculate()
	}
}
