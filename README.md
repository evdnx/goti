**GoTI – Technical‑Analysis Library for Go**

A lightweight, dependency‑free Go package that implements a suite of popular technical‑analysis indicators.  
All indicators share a common design philosophy:

- **Thread‑safe where needed** – e.g. `AdaptiveDEMAMomentumOscillator`.
- **Memory‑bounded** – internal slices are trimmed to the minimum required capacity.
- **Consistent error handling** – sentinel errors (`ErrInsufficientData`, `ErrNoMFIData`, …) are exported for `errors.Is`.
- **Configurable thresholds** – via `IndicatorConfig`.
- **Ready‑for‑visualisation** – each indicator can emit `PlotData` structures that serialize to JSON/CSV.

---

## **Package layout**

- `github.com/evdnx/goti` – convenience façade that re-exports everything.
- `github.com/evdnx/goti/config` – shared thresholds and validation helpers.
- `github.com/evdnx/goti/indicator` – all indicator implementations, moving averages, and plotting utilities.
- `github.com/evdnx/goti/suite` – combined signal engine built from the individual indicators.

---

## **Table of Contents**

1. Installation
2. Configuration
3. Indicators
   - Relative Strength Index (RSI)
   - Money Flow Index (MFI)
   - Volume‑Weighted Aroon Oscillator (VWAO)
   - Hull Moving Average (HMA)
   - Average True Range (ATR)
   - Adaptive DEMA (Double Exponential Moving Average) Momentum Oscillator (ADMO)
   - Adaptive Trend Strength Oscillator (ATSO)
4. Indicator Suite
5. Utility Functions
6. Testing & Benchmarking
7. Serialization Helpers
8. License

---

## **Installation**

```go
go get github.com/evdnx/goti
```

The package only depends on the Go standard library. Use the top-level `goti` package for the façade, or import `indicator`, `config`, or `suite` directly when you want narrower dependencies.

---

## **Configuration**

All indicators accept an `IndicatorConfig` that bundles the tunable thresholds and a few misc parameters.

```go
cfg := goti.DefaultConfig()
cfg.RSIOverbought = 75          // raise overbought level for RSI
cfg.MFIOversold = 15           // tighter oversold for MFI
cfg.ATSEMAperiod = 10          // longer EMA smoothing for ATSO
```

Validate a config before use:

```go
if err := cfg.Validate(); err != nil {
    // handle mis‑configuration
}
```

---

## **Indicators**

Each indicator follows the same lifecycle:

```go
// 1️⃣ Create
ind, err := goti.New<IndicatorName>(/*optional params*/)

// 2️⃣ Feed data (Add / AddCandle)
err = ind.Add(/*price data*/)

// 3️⃣ Query results
value, err := ind.Calculate()
bull, err := ind.IsBullishCrossover()
```

### **Relative Strength Index (RSI)**

- **Package:** `relative_strength_index.go`
- **Default period:** 5
- **Key methods:** `Add`, `Calculate`, `IsBullishCrossover`, `IsBearishCrossover`, `IsDivergence`, `GetPlotData`

### **Money Flow Index (MFI)**

- **Package:** `money_flow_index.go`
- **Default period:** 5, volume‑scaled by `MFIVolumeScale` (default 300 000)
- **Sentinel error:** `ErrNoMFIData` (use `errors.Is`)

### **Volume‑Weighted Aroon Oscillator (VWAO)**

- **Package:** `volume_weighted_aroon_oscillator.go`
- **Default period:** 14
- **Strong‑trend threshold:** `VWAOStrongTrend` (default 70)

### **Hull Moving Average (HMA)**

- **Package:** `hull_moving_average.go`
- **Default period:** 9
- **Crossover helpers** (`IsBullishCrossover`, `IsBearishCrossover`) and trend detection (`GetTrendDirection`).

### **Average True Range (ATR)**

- **Package:** `average_true_range.go`
- **Default period:** 14
- **Functional option:** `WithCloseValidation(bool)` to disable the “close must lie between high/low” check.

### **Adaptive DEMA (Double Exponential Moving Average) Momentum Oscillator (ADMO)**

- **Package:** `adaptive_dema_momentum_oscillator.go`
- **Features:**
  - Dual EMA → DEMA → adaptive momentum calculation.
  - Zero‑line crossovers, significant‑price‑jump heuristics, and divergence detection.
  - Fully thread‑safe (`sync.RWMutex`).

### **Adaptive Trend Strength Oscillator (ATSO)**

- **Package:** `adaptive_trend_strength_oscillator.go`
- **Adaptive period** based on recent volatility, EMA‑smoothed output.
- **Crossover detection** scans the entire raw series for sign changes (improved over the original “last‑two‑points only” logic).

---

## **Indicator Suite**

`IndicatorSuite` aggregates all six indicators and provides a weighted‑signal engine.

```go
suite, err := goti.NewIndicatorSuiteWithConfig(cfg)
suite.Add(high, low, close, volume) // feeds every sub‑indicator
signal, err := suite.GetCombinedSignal() // “Strong Bullish”, “Neutral”, etc.
```

The suite also offers:

- `GetCombinedBearishSignal()`
- `GetDivergenceSignals()` – returns a map of indicator‑specific divergences.
- `Reset()` – clears every sub‑indicator while preserving the config.

---

## **Utility Functions**

**FunctionDescription**`keepLast[T any](s []T, n int) []T`Return the last *n* elements of a slice (generic).  
`GenerateTimestamps(start, count, interval int64) []int64`Produce Unix‑epoch timestamps for chart axes.  
`FormatPlotDataJSON(data []PlotData) (string, error)`Marshal a slice of `PlotData` to JSON (validated lengths).  
`FormatPlotDataCSV(data []PlotData) (string, error)`Serialize `PlotData` to CSV.  
`clamp(v, min, max float64) float64`Clamp a value to a closed interval.  
`calculateEMA`, `calculateWMA`, `calculateStandardDeviation`Core numeric kernels used by the indicators.

All helpers are deliberately **publicly exported** only when needed; the rest remain package‑private.

---

## **Testing & Benchmarking**

The repository ships with a comprehensive test matrix (`*_test.go`) covering:

- Correctness of each indicator (edge cases, period changes, resets).
- Defensive‑copy safety of getters.
- Divergence and crossover detection logic.

Run the full suite:

```go
go test ./...
```

Benchmarks are located in `*_bench_test.go`. Example:

```go
go test -bench=. -benchmem ./...
```

Typical benchmark categories:

- **Add** – cost of ingesting a single price/candle.
- **Full pipeline** – `Add` + `Calculate`.
- **Signal detection** – `IsBullishCrossover`, `IsDivergence`, etc.
- **Plot data generation** – JSON/CSV serialization overhead.

Feel free to add new benchmarks for custom workloads; the helper functions (`genOHLC`, `randVals`, etc.) are deterministic to keep results reproducible.

---

## **Serialization Helpers**

`PlotData` is the canonical container for charting libraries:

```go
type PlotData struct {
    Name      string    `json:"name"`
    X         []float64 `json:"x"`
    Y         []float64 `json:"y"`
    Type      string    `json:"type,omitempty"`   // "line" or "scatter"
    Signal    string    `json:"signal,omitempty"` // optional label for scatter series
    Timestamp []int64   `json:"timestamp,omitempty"`
}
```

Both JSON and CSV exporters validate that `len(X) == len(Y)` and return a clear error if the invariant is broken.

---

## **License**

This library is released under the **MIT License**. Feel free to fork, modify, and use it in commercial projects.

---

### **Quick‑start Example**

```go
package main

import (
    "fmt"
    "github.com/evdnx/goti"
)

func main() {
    // Initialise a suite with custom thresholds.
    cfg := goti.DefaultConfig()
    cfg.RSIOverbought = 80
    cfg.RSIOversold = 20
    suite, _ := goti.NewIndicatorSuiteWithConfig(cfg)

    // Simulate a stream of OHLCV data.
    data := []struct{ h, l, c, v float64 }{
        {101, 99, 100, 1_000},
        {102, 98, 101, 1_200},
        // …
    }

    for _, d := range data {
        if err := suite.Add(d.h, d.l, d.c, d.v); err != nil {
            panic(err)
        }
    }

    // Get a combined signal.
    signal, _ := suite.GetCombinedSignal()
    fmt.Println("Combined signal:", signal)

    // Export chart data for the UI.
    plot := suite.GetPlotData(1625097600000, 60_000) // start‑time, 1‑min interval
    json, _ := goti.FormatPlotDataJSON(plot)
    fmt.Println(json)
}
```
