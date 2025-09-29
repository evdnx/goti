# GoTI Package

This Go package provides a set of robust technical indicators designed for trading and financial analysis. The indicators cover key market dynamics including momentum, trend, volume, and volatility, with a focus on adaptability, signal clarity, and memory efficiency. The package includes consistent error handling, visualization support, and customizable configurations.

## Indicators Overview

1. **Adaptive Momentum Divergence Oscillator (AMDO)**  
   An adaptive oscillator that detects momentum divergence by dynamically adjusting its period based on market volatility. Ideal for identifying potential reversals through zero-line crossovers and strong divergence thresholds (±50).

2. **Adaptive Trend Strength Oscillator (ATSO)**  
   Measures trend strength with volatility-based period adjustment and EMA smoothing for reduced noise. Supports bullish/bearish crossovers and customizable volatility sensitivity.

3. **Volume-Weighted Aroon Oscillator (VWAO)**  
   Enhances the Aroon Oscillator with volume weighting to detect trend direction changes. Provides zero-line crossovers, strong trend signals (±50), and individual Aroon Up/Down values.

4. **Hull Moving Average (HMA)**  
   A low-lag moving average that combines weighted moving averages with a square root transformation for smoother, faster trend signals. Supports price vs. HMA crossovers and trend direction detection.

5. **Money Flow Index (MFI)**  
   A volume-weighted momentum oscillator that identifies overbought/oversold conditions (80/20 thresholds) and divergence signals. Useful for assessing buying/selling pressure.

6. **Relative Strength Index (RSI)**  
   A momentum oscillator that measures the speed and change of price movements on a 0-100 scale. Identifies overbought (>70) and oversold (<30) conditions and supports divergence analysis for potential reversals.

7. **Average True Range (ATR)**  
   A volatility indicator that measures the average range of price movements, useful for assessing market volatility and setting risk management parameters like stop-loss levels.

## Features

- **Adaptivity**: Indicators like AMDO and ATSO dynamically adjust periods based on volatility for better performance in varying market conditions.
- **Crossover Signals**: Built-in methods for bullish/bearish crossovers and divergence detection.
- **Visualization**: `GetPlotData` method returns `PlotData` structs compatible with JSON/CSV output for charting libraries (e.g., Chart.js, Plotly), including signal annotations and timestamp support.
- **Memory Efficiency**: Pre-allocated slices and trimming to minimize memory usage.
- **Error Handling**: Consistent checks for invalid inputs and insufficient data.
- **Configurable Thresholds**: Customize overbought/oversold and divergence thresholds via `IndicatorConfig`.
- **Weighted Signals**: `IndicatorSuite` supports weighted signal aggregation for prioritizing reliable indicators.
- **Utils Support**: Relies on `utils.go` for common functions like `clamp`, `copySlice`, `calculateStandardDeviation`, and `FormatPlotDataJSON`.
- **Testing**: Unit tests for all indicators ensure reliability across edge cases (see test files, e.g., `relative_strength_index_test.go`).

## Installation

No external dependencies are required beyond standard Go libraries (e.g., `math`, `encoding/json`).

## Usage

### Basic Example (RSI)
```go
package main

import (
	"fmt"
	"path/to/goti"
)

func main() {
	// Initialize RSI
	rsi, err := goti.NewRelativeStrengthIndex()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Add data
	rsi.Add(100.5)
	rsi.Add(101.2)
	// ... add more data

	// Check signals
	bullish, _ := rsi.IsBullishCrossover()
	if bullish {
		fmt.Println("Bullish crossover detected!")
	}

	// Get plot data
	plotData := rsi.GetPlotData(1625097600000, 86400000) // Start time, daily interval
	jsonData, _ := goti.FormatPlotDataJSON(plotData)
	fmt.Println("Plot JSON:", jsonData)
}
```

### ATR Example
```go
atr, err := goti.NewAverageTrueRange()
if err != nil {
	fmt.Println("Error:", err)
	return
}
atr.Add(102.0, 99.0, 100.5)
atr.Add(103.0, 98.5, 101.0)
// ... add more data
value, _ := atr.Calculate()
fmt.Println("ATR Value:", value)
```

### Combined Suite Example
```go
config := goti.DefaultConfig()
config.RSIOverbought = 75 // Customize thresholds
suite, err := goti.NewIndicatorSuiteWithConfig(config)
if err != nil {
	fmt.Println("Error:", err)
	return
}

// Add data (high, low, close, volume)
suite.Add(102.0, 99.0, 100.5, 1000.0)
// ... add more data

signal, _ := suite.GetCombinedSignal()
fmt.Println("Combined Signal:", signal)
```

## Documentation

Each indicator includes:

- **Constructor**: `New<IndicatorName>()` or `New<IndicatorName>WithParams(...)`.
- **Add Method**: Appends new data and updates calculations.
- **Calculate/GetLastValue**: Retrieves the latest value.
- **Signal Methods**: E.g., `IsBullishCrossover()`, `IsBearishCrossover()`, `IsStrongDivergence()`.
- **Reset/SetPeriod**: Clears data or updates parameters.
- **GetPlotData**: Returns visualization data with annotations and timestamps.
- **Configurability**: Supports custom thresholds via `IndicatorConfig`.

For detailed methods, refer to the source code comments. Unit tests are available in `<indicator>_test.go` files to validate functionality.

## Testing

The package includes unit tests for all indicators, covering initialization, data addition, calculations, and edge cases. Run tests with:
```bash
go test ./...
```
