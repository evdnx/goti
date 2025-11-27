package volume

import (
	"math/rand"
	"testing"
	"time"

	"github.com/evdnx/goti/config"
	"github.com/evdnx/goti/indicator/core"
)

/*
=====================================================================
 1️⃣ Moving‑Average benchmarks
=====================================================================
*/

func BenchmarkNewMovingAverage_EMA(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = core.NewMovingAverage(core.EMAMovingAverage, 20)
	}
}

func BenchmarkNewMovingAverage_SMA(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = core.NewMovingAverage(core.SMAMovingAverage, 20)
	}
}

// Feed a realistic stream of price data (≈10 k points) and call Calculate
// after each addition.  This mimics a live chart where the UI wants the
// latest MA on every tick.
func benchmarkMAAddAndCalculate(b *testing.B, typ core.MovingAverageType, period int) {
	ma, _ := core.NewMovingAverage(typ, period)

	// deterministic pseudo‑random data – fast and reproducible
	rng := rand.New(rand.NewSource(42))
	prices := make([]float64, 10000)
	for i := range prices {
		prices[i] = 50 + rng.Float64()*10 // 50‑60 range
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx := i % len(prices)
		_ = ma.Add(prices[idx])
		_, _ = ma.Calculate()
	}
}

func BenchmarkMovingAverage_AddCalculate_EMA_Period20(b *testing.B) {
	benchmarkMAAddAndCalculate(b, core.EMAMovingAverage, 20)
}
func BenchmarkMovingAverage_AddCalculate_SMA_Period20(b *testing.B) {
	benchmarkMAAddAndCalculate(b, core.SMAMovingAverage, 20)
}
func BenchmarkMovingAverage_AddCalculate_WMA_Period20(b *testing.B) {
	benchmarkMAAddAndCalculate(b, core.WMAMovingAverage, 20)
}

/*
=====================================================================
  2️⃣ Money‑Flow‑Index (MFI) benchmarks
=====================================================================
*/

// Helper that builds a ready‑to‑use MFI with a given period and a
// deterministic data generator.
func newBenchMFI(period int) *MoneyFlowIndex {
	mfi, err := NewMoneyFlowIndexWithParams(period, config.DefaultConfig())
	if err != nil {
		panic(err) // should never happen in a benchmark
	}
	return mfi
}

// Generate a slice of synthetic OHLCV rows.  The volume is scaled to avoid
// overflow and to keep the computation realistic.
func genOHLCV(n int) [][4]float64 {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	out := make([][4]float64, n)
	price := 100.0
	for i := 0; i < n; i++ {
		// random walk ±0.5%
		change := (rng.Float64() - 0.5) * 0.01
		price *= 1 + change
		high := price * (1 + rng.Float64()*0.005)
		low := price * (1 - rng.Float64()*0.005)
		close := price
		vol := 1000 + rng.Float64()*500 // modest volume
		out[i] = [4]float64{high, low, close, vol}
	}
	return out
}

// Benchmark adding N samples without ever calling Calculate.
// This measures the cost of validation and slice management.
func BenchmarkMFI_AddOnly_1000Samples(b *testing.B) {
	mfi := newBenchMFI(14)
	data := genOHLCV(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx := i % len(data)
		_ = mfi.Add(data[idx][0], data[idx][1], data[idx][2], data[idx][3])
	}
}

// Benchmark the full pipeline: Add → Calculate (once per sample)
// This reflects a real‑time UI that wants the latest MFI after each tick.
func BenchmarkMFI_AddCalculate_1000Samples(b *testing.B) {
	mfi := newBenchMFI(14)
	data := genOHLCV(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx := i % len(data)
		_ = mfi.Add(data[idx][0], data[idx][1], data[idx][2], data[idx][3])
		_, _ = mfi.Calculate()
	}
}

// Benchmark the crossover detection functions after a warm‑up phase.
func BenchmarkMFI_IsBullishCrossover(b *testing.B) {
	mfi := newBenchMFI(14)
	// Warm‑up with enough data to have at least one MFI value.
	for _, d := range genOHLCV(30) {
		_ = mfi.Add(d[0], d[1], d[2], d[3])
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mfi.IsBullishCrossover()
	}
}

func BenchmarkMFI_IsBearishCrossover(b *testing.B) {
	mfi := newBenchMFI(14)
	for _, d := range genOHLCV(30) {
		_ = mfi.Add(d[0], d[1], d[2], d[3])
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mfi.IsBearishCrossover()
	}
}

// Benchmark the classic divergence detection (the method you just fixed).
func BenchmarkMFI_IsDivergence(b *testing.B) {
	mfi := newBenchMFI(2) // short period to get MFI quickly
	// Feed a mix of up/down moves so the internal state always has ≥2 MFI values.
	for _, d := range genOHLCV(200) {
		_ = mfi.Add(d[0], d[1], d[2], d[3])
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mfi.IsDivergence()
	}
}

/*
=====================================================================
  3️⃣ Plot‑data generation benchmarks
=====================================================================
*/

func BenchmarkMFI_GetPlotData(b *testing.B) {
	mfi := newBenchMFI(14)
	for _, d := range genOHLCV(500) {
		_ = mfi.Add(d[0], d[1], d[2], d[3])
	}
	// Ensure at least one MFI value exists.
	_, _ = mfi.Calculate()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mfi.GetPlotData()
	}
}

/*
=====================================================================
  4️⃣ IndicatorConfig validation benchmark
=====================================================================
*/

func BenchmarkIndicatorConfig_Validate(b *testing.B) {
	cfg := config.DefaultConfig()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cfg.Validate()
	}
}

/*
=====================================================================
  5️⃣ Utility‑function micro‑benchmarks
=====================================================================
*/

func BenchmarkClamp(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = core.Clamp(float64(i%200)-100, -50, 150)
	}
}

func BenchmarkCalculateStandardDeviation(b *testing.B) {
	// Fixed slice of 1 024 random numbers.
	rng := rand.New(rand.NewSource(12345))
	data := make([]float64, 1024)
	for i := range data {
		data[i] = rng.NormFloat64()*10 + 100
	}
	mean := 100.0

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = core.CalculateStandardDeviation(data, mean)
	}
}
