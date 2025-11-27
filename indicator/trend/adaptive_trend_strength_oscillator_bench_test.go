package trend

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/evdnx/goti/config"
	"github.com/evdnx/goti/indicator/core"
)

/*
   Benchmark helpers
   -----------------
   All benchmarks reuse a deterministic pseudo‑random generator seeded with a fixed
   value so that runs are repeatable.  The `randVals` function builds a slice of
   `n` float64 values in the range [0,100).  The same helper is used for the
   MovingAverage, EMA and ATSO benchmarks.
*/

func randVals(n int) []float64 {
	r := rand.New(rand.NewSource(42))
	vals := make([]float64, n)
	for i := 0; i < n; i++ {
		vals[i] = r.Float64() * 100
	}
	return vals
}

/*
   AdaptiveTrendStrengthOscillator benchmarks
   -----------------------------------------
   Three aspects are benchmarked:
   1. Adding a full OHLC bar series (the common path).
   2. Computing volatility alone (private helper, exercised indirectly via Add).
   3. Mapping volatility to an adaptive period.

   The benchmark uses a realistic‑looking synthetic price series that slowly
   drifts upward, ensuring that the oscillator stays in the “ready” state after the
   warm‑up period.
*/

func generateOHLCSeries(n int) (highs, lows, closes []float64) {
	highs = make([]float64, n)
	lows = make([]float64, n)
	closes = make([]float64, n)

	base := 100.0
	for i := 0; i < n; i++ {
		// Simulate a gentle upward drift with random jitter.
		drift := float64(i) * 0.01
		jitter := rand.Float64()*0.5 - 0.25
		close := base + drift + jitter
		high := close + rand.Float64()*0.5
		low := close - rand.Float64()*0.5
		highs[i] = high
		lows[i] = low
		closes[i] = close
	}
	return
}

func benchmarkATSOAdd(b *testing.B, minP, maxP, volP int, emaPeriod int) {
	cfg := config.DefaultConfig()
	cfg.ATSEMAperiod = emaPeriod
	atso, err := NewAdaptiveTrendStrengthOscillatorWithParams(minP, maxP, volP, cfg)
	if err != nil {
		b.Fatalf("constructor error: %v", err)
	}
	highs, lows, closes := generateOHLCSeries(b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := atso.Add(highs[i], lows[i], closes[i]); err != nil {
			b.Fatalf("Add error at %d: %v", i, err)
		}
	}
}

func BenchmarkATSO_Add_SmallEMA(b *testing.B) {
	benchmarkATSOAdd(b, 2, 5, 3, 2) // tiny EMA period → more frequent EMA updates
}

func BenchmarkATSO_Add_MediumEMA(b *testing.B) {
	benchmarkATSOAdd(b, 2, 5, 3, 10)
}

func BenchmarkATSO_Add_LargeEMA(b *testing.B) {
	benchmarkATSOAdd(b, 2, 5, 3, 50)
}

/*
   Volatility‑mapping benchmark
   ---------------------------
   Isolate the `mapVolatilityToPeriod` logic to see how the linear scaling behaves.
*/

func BenchmarkATSO_MapVolatility(b *testing.B) {
	atso, _ := NewAdaptiveTrendStrengthOscillator()
	vols := []float64{0.001, 0.005, 0.01, 0.02, 0.04, 0.08}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = atso.mapVolatilityToPeriod(vols[i%len(vols)])
	}
}

/*
   Full end‑to‑end benchmark
   -------------------------
   Runs a realistic workload: generate a series, feed it to the oscillator,
   then read the latest smoothed value.  This mimics the typical usage pattern
   in a trading loop.
*/

func BenchmarkATSO_FullWorkflow(b *testing.B) {
	cfg := config.DefaultConfig()
	cfg.ATSEMAperiod = 10
	atso, _ := NewAdaptiveTrendStrengthOscillatorWithParams(2, 14, 10, cfg)
	highs, lows, closes := generateOHLCSeries(b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = atso.Add(highs[i], lows[i], closes[i])
		_, _ = atso.Calculate() // ignore error during warm‑up; we care about throughput
	}
}

/*
   Memory allocation reporting
   --------------------------
   All benchmarks use `-benchmem` when invoked, so the Go test harness will
   automatically report allocations per operation.  No extra instrumentation is
   required.
*/

/*
   Additional ATSO benchmarks
   -------------------------

   * BenchmarkATSO_Reset – measures the cost of clearing internal buffers and
     re‑initialising the embedded EMA.  This is useful when the oscillator is
     reused across multiple back‑test runs.

   * BenchmarkATSO_GetPlotData – evaluates the overhead of building the two
     PlotData structures (raw and smoothed series) that are later marshalled to
     JSON/CSV.  The benchmark populates the oscillator with a modest amount of
     data (1 000 points) before each measurement to reflect a typical UI request.

   * BenchmarkATSO_RawVsSmoothed – isolates the cost of reading the raw and the
     EMA‑smoothed slices.  Both getters simply return copies of internal slices,
     so the benchmark focuses on the slice‑copy overhead.
*/

func benchmarkATSOReset(b *testing.B, minP, maxP, volP, emaPeriod int) {
	cfg := config.DefaultConfig()
	cfg.ATSEMAperiod = emaPeriod
	atso, err := NewAdaptiveTrendStrengthOscillatorWithParams(minP, maxP, volP, cfg)
	if err != nil {
		b.Fatalf("constructor error: %v", err)
	}
	// Warm‑up with a deterministic series so the oscillator has internal state.
	highs, lows, closes := generateOHLCSeries(500)
	for i := 0; i < 500; i++ {
		if err := atso.Add(highs[i], lows[i], closes[i]); err != nil {
			b.Fatalf("warm‑up Add error: %v", err)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := atso.Reset(); err != nil {
			b.Fatalf("Reset error: %v", err)
		}
	}
}

func BenchmarkATSO_Reset_SmallEMA(b *testing.B)  { benchmarkATSOReset(b, 2, 5, 3, 2) }
func BenchmarkATSO_Reset_MediumEMA(b *testing.B) { benchmarkATSOReset(b, 2, 5, 3, 10) }
func BenchmarkATSO_Reset_LargeEMA(b *testing.B)  { benchmarkATSOReset(b, 2, 5, 3, 50) }

func benchmarkATSOGetPlotData(b *testing.B, points int) {
	cfg := config.DefaultConfig()
	cfg.ATSEMAperiod = 10
	atso, _ := NewAdaptiveTrendStrengthOscillatorWithParams(2, 14, 10, cfg)

	highs, lows, closes := generateOHLCSeries(points)
	for i := 0; i < points; i++ {
		_ = atso.Add(highs[i], lows[i], closes[i])
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = atso.GetPlotData()
	}
}

func BenchmarkATSO_GetPlotData_1k(b *testing.B)   { benchmarkATSOGetPlotData(b, 1_000) }
func BenchmarkATSO_GetPlotData_10k(b *testing.B)  { benchmarkATSOGetPlotData(b, 10_000) }
func BenchmarkATSO_GetPlotData_100k(b *testing.B) { benchmarkATSOGetPlotData(b, 100_000) }

func benchmarkATSOReadSlices(b *testing.B, points int) {
	cfg := config.DefaultConfig()
	cfg.ATSEMAperiod = 10
	atso, _ := NewAdaptiveTrendStrengthOscillatorWithParams(2, 14, 10, cfg)

	highs, lows, closes := generateOHLCSeries(points)
	for i := 0; i < points; i++ {
		_ = atso.Add(highs[i], lows[i], closes[i])
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = atso.RawValues()
		_ = atso.SmoothedValues()
	}
}

func BenchmarkATSO_ReadSlices_1k(b *testing.B)   { benchmarkATSOReadSlices(b, 1_000) }
func BenchmarkATSO_ReadSlices_10k(b *testing.B)  { benchmarkATSOReadSlices(b, 10_000) }
func BenchmarkATSO_ReadSlices_100k(b *testing.B) { benchmarkATSOReadSlices(b, 100_000) }

/*
   Extra benchmarks – JSON/CSV rendering, timestamp generation, and tiny helpers
   ---------------------------------------------------------------------------
   These are optional but useful when you want to see how much time the
   serialization helpers and the trivial numeric utilities add to the overall
   processing pipeline.
*/

func benchmarkPlotDataJSON(b *testing.B, seriesCount int, pointsPerSeries int) {
	// Build a slice of PlotData with deterministic values.
	data := make([]PlotData, seriesCount)
	for i := 0; i < seriesCount; i++ {
		x := make([]float64, pointsPerSeries)
		y := make([]float64, pointsPerSeries)
		ts := make([]int64, pointsPerSeries)
		for j := 0; j < pointsPerSeries; j++ {
			x[j] = float64(j)
			y[j] = float64(i*j) + 0.5
			ts[j] = int64(1609459200 + j*60) // Jan 1 2021 00:00 + 1‑min steps
		}
		data[i] = PlotData{
			Name:      fmt.Sprintf("Series-%d", i),
			X:         x,
			Y:         y,
			Type:      "line",
			Signal:    "",
			Timestamp: ts,
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := core.FormatPlotDataJSON(data); err != nil {
			b.Fatalf("JSON format error: %v", err)
		}
	}
}

func BenchmarkPlotData_JSON_Small(b *testing.B)  { benchmarkPlotDataJSON(b, 2, 100) }
func BenchmarkPlotData_JSON_Medium(b *testing.B) { benchmarkPlotDataJSON(b, 5, 1_000) }
func BenchmarkPlotData_JSON_Large(b *testing.B)  { benchmarkPlotDataJSON(b, 10, 10_000) }

func benchmarkPlotDataCSV(b *testing.B, seriesCount int, pointsPerSeries int) {
	data := make([]PlotData, seriesCount)
	for i := 0; i < seriesCount; i++ {
		x := make([]float64, pointsPerSeries)
		y := make([]float64, pointsPerSeries)
		ts := make([]int64, pointsPerSeries)
		for j := 0; j < pointsPerSeries; j++ {
			x[j] = float64(j)
			y[j] = float64(i*j) + 0.5
			ts[j] = int64(1609459200 + j*60)
		}
		data[i] = PlotData{
			Name:      fmt.Sprintf("Series-%d", i),
			X:         x,
			Y:         y,
			Type:      "line",
			Signal:    "",
			Timestamp: ts,
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := core.FormatPlotDataCSV(data); err != nil {
			b.Fatalf("CSV format error: %v", err)
		}
	}
}

func BenchmarkPlotData_CSV_Small(b *testing.B)  { benchmarkPlotDataCSV(b, 2, 100) }
func BenchmarkPlotData_CSV_Medium(b *testing.B) { benchmarkPlotDataCSV(b, 5, 1_000) }
func BenchmarkPlotData_CSV_Large(b *testing.B)  { benchmarkPlotDataCSV(b, 10, 10_000) }

/*
   Timestamp generation benchmark
   -----------------------------
   `GenerateTimestamps` is a tiny utility but it’s called for every chart
   render, so it’s worth measuring its overhead at scale.
*/

func BenchmarkGenerateTimestamps_1k(b *testing.B) {
	start := time.Now().Unix()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = core.GenerateTimestamps(start, 1_000, 60)
	}
}
func BenchmarkGenerateTimestamps_10k(b *testing.B) {
	start := time.Now().Unix()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = core.GenerateTimestamps(start, 10_000, 60)
	}
}

/*
   Tiny numeric helper – calculateSlope
   -----------------------------------
   The function is a one‑liner, but benchmarking it shows the baseline cost of a
   pure‑Go arithmetic operation when called millions of times.
*/

func BenchmarkCalculateSlope(b *testing.B) {
	y1, y2 := 123.456, 789.012
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = core.CalculateSlope(y2, y1)
	}
}

/*
   MovingAverage “calculate‑only” benchmark
   ---------------------------------------
   In many real‑world loops the data slice is already filled and the hot path
   consists mainly of `Calculate`.  This benchmark isolates that cost.
*/

func benchmarkMovingAverageCalcOnly(b *testing.B, maType core.MovingAverageType, period int) {
	ma, err := core.NewMovingAverage(maType, period)
	if err != nil {
		b.Fatalf("NewMovingAverage error: %v", err)
	}
	// Pre‑fill with enough data to make Calculate succeed.
	vals := randVals(period * 2)
	for _, v := range vals {
		if err := ma.Add(v); err != nil {
			b.Fatalf("pre‑fill Add error: %v", err)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ma.Calculate(); err != nil {
			b.Fatalf("Calculate error: %v", err)
		}
	}
}

func BenchmarkMovingAverage_CalcOnly_SMA_Small(b *testing.B) {
	benchmarkMovingAverageCalcOnly(b, core.SMAMovingAverage, 5)
}
func BenchmarkMovingAverage_CalcOnly_EMA_Medium(b *testing.B) {
	benchmarkMovingAverageCalcOnly(b, core.EMAMovingAverage, 20)
}
func BenchmarkMovingAverage_CalcOnly_WMA_Large(b *testing.B) {
	benchmarkMovingAverageCalcOnly(b, core.WMAMovingAverage, 200)
}
