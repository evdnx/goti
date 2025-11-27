package trend

import (
	"errors"
	"fmt"
	"math"

	"github.com/evdnx/goti/indicator/core"
)

// ---------------------------------------------------------------------------
// Sentinel errors – exported so callers can compare with errors.Is()
// ---------------------------------------------------------------------------
var (
	ErrInvalidPrice          = errors.New("price must be > 0")
	ErrInsufficientHMAData   = errors.New("no HMA data")
	ErrInsufficientCrossData = errors.New("insufficient data for crossover")
)

// HullMovingAverage calculates the Hull Moving Average (HMA)
type HullMovingAverage struct {
	period    int
	closes    []float64
	rawHMAs   []float64
	hmaValues []float64
	lastValue float64
}

// NewHullMovingAverage initializes with the standard period (9)
func NewHullMovingAverage() (*HullMovingAverage, error) {
	return NewHullMovingAverageWithParams(9)
}

// NewHullMovingAverageWithParams initializes with a custom period
func NewHullMovingAverageWithParams(period int) (*HullMovingAverage, error) {
	if period < 1 {
		return nil, fmt.Errorf("period must be at least 1, got %d", period)
	}
	return &HullMovingAverage{
		period:    period,
		closes:    make([]float64, 0, period*2),
		rawHMAs:   make([]float64, 0, period*2),
		hmaValues: make([]float64, 0, period),
	}, nil
}

// Add appends a new price datum and updates the HMA state.
// It validates the price, updates the internal buffers and, when enough
// data is present, computes the next HMA value.
func (hma *HullMovingAverage) Add(close float64) error {
	if !core.IsValidPrice(close) {
		return fmt.Errorf("%w: %v", ErrInvalidPrice, close)
	}
	hma.closes = append(hma.closes, close)

	// Only start calculations once we have at least `period` closing prices.
	if len(hma.closes) >= hma.period {
		// 1️⃣ Full‑period WMA
		wmaFull, err := core.CalculateWMA(hma.closes[len(hma.closes)-hma.period:], hma.period)
		if err != nil {
			return err
		}

		// 2️⃣ Half‑period WMA (minimum 1)
		wmaHalfPeriod := hma.period / 2
		if wmaHalfPeriod < 1 {
			wmaHalfPeriod = 1
		}
		wmaHalf, err := core.CalculateWMA(hma.closes[len(hma.closes)-wmaHalfPeriod:], wmaHalfPeriod)
		if err != nil {
			return err
		}

		// 3️⃣ Raw HMA = 2·WMA(half) – WMA(full)
		rawHMA := 2*wmaHalf - wmaFull
		hma.rawHMAs = append(hma.rawHMAs, rawHMA)

		// 4️⃣ Final HMA = WMA of the last √period raw values
		sqrtPeriod := int(math.Sqrt(float64(hma.period)))
		if sqrtPeriod < 1 {
			sqrtPeriod = 1
		}
		if len(hma.rawHMAs) >= sqrtPeriod {
			hmaValue, err := core.CalculateWMA(hma.rawHMAs[len(hma.rawHMAs)-sqrtPeriod:], sqrtPeriod)
			if err == nil {
				hma.hmaValues = append(hma.hmaValues, hmaValue)
				hma.lastValue = hmaValue
			}
		}
	}
	hma.trimSlices()
	return nil
}

// trimSlices limits the size of the internal slices to keep memory bounded.
// The chosen multipliers (×2 for closes/rawHMAs, ×period for hmaValues) match the
// original implementation while making the intent explicit.
func (hma *HullMovingAverage) trimSlices() {
	const maxClosesMultiplier = 2
	if len(hma.closes) > hma.period*maxClosesMultiplier {
		hma.closes = hma.closes[len(hma.closes)-hma.period*maxClosesMultiplier:]
	}

	sqrtPeriod := int(math.Sqrt(float64(hma.period)))
	if sqrtPeriod < 1 {
		sqrtPeriod = 1
	}
	if len(hma.rawHMAs) > sqrtPeriod*maxClosesMultiplier {
		hma.rawHMAs = hma.rawHMAs[len(hma.rawHMAs)-sqrtPeriod*maxClosesMultiplier:]
	}
	if len(hma.hmaValues) > hma.period {
		hma.hmaValues = hma.hmaValues[len(hma.hmaValues)-hma.period:]
	}
}

// Calculate returns the most recent HMA value.
// If no HMA has been produced yet, ErrInsufficientHMAData is returned.
func (hma *HullMovingAverage) Calculate() (float64, error) {
	if len(hma.hmaValues) == 0 {
		return 0, ErrInsufficientHMAData
	}
	return hma.lastValue, nil
}

// GetLastValue returns the last calculated HMA without an error check.
func (hma *HullMovingAverage) GetLastValue() float64 {
	return hma.lastValue
}

// IsBullishCrossover reports whether the latest price crossed above the HMA.
func (hma *HullMovingAverage) IsBullishCrossover() (bool, error) {
	if len(hma.hmaValues) < 2 || len(hma.closes) < 2 {
		return false, ErrInsufficientCrossData
	}
	currHMA := hma.hmaValues[len(hma.hmaValues)-1]
	prevHMA := hma.hmaValues[len(hma.hmaValues)-2]
	currClose := hma.closes[len(hma.closes)-1]
	prevClose := hma.closes[len(hma.closes)-2]
	return prevClose <= prevHMA && currClose > currHMA, nil
}

// IsBearishCrossover reports whether the latest price crossed below the HMA.
func (hma *HullMovingAverage) IsBearishCrossover() (bool, error) {
	if len(hma.hmaValues) < 2 || len(hma.closes) < 2 {
		return false, ErrInsufficientCrossData
	}
	currHMA := hma.hmaValues[len(hma.hmaValues)-1]
	prevHMA := hma.hmaValues[len(hma.hmaValues)-2]
	currClose := hma.closes[len(hma.closes)-1]
	prevClose := hma.closes[len(hma.closes)-2]
	return prevClose >= prevHMA && currClose < currHMA, nil
}

// GetTrendDirection returns a textual description of the HMA’s short‑term trend.
func (hma *HullMovingAverage) GetTrendDirection() (string, error) {
	if len(hma.hmaValues) < 2 {
		return "", ErrInsufficientHMAData
	}
	curr := hma.hmaValues[len(hma.hmaValues)-1]
	prev := hma.hmaValues[len(hma.hmaValues)-2]
	switch {
	case curr > prev:
		return "Bullish", nil
	case curr < prev:
		return "Bearish", nil
	default:
		return "Neutral", nil
	}
}

// Reset clears all stored data.
func (hma *HullMovingAverage) Reset() {
	hma.closes = hma.closes[:0]
	hma.rawHMAs = hma.rawHMAs[:0]
	hma.hmaValues = hma.hmaValues[:0]
	hma.lastValue = 0
}

// SetPeriod updates the HMA period and trims buffers accordingly.
func (hma *HullMovingAverage) SetPeriod(period int) error {
	if period < 1 {
		return fmt.Errorf("period must be at least 1, got %d", period)
	}
	hma.period = period
	hma.trimSlices()
	return nil
}

// GetCloses returns a copy of the stored close prices.
func (hma *HullMovingAverage) GetCloses() []float64 {
	return core.CopySlice(hma.closes)
}

// GetHMAValues returns a copy of the computed HMA series.
func (hma *HullMovingAverage) GetHMAValues() []float64 {
	return core.CopySlice(hma.hmaValues)
}

// DetectSignals walks the HMA series and produces a slice where:
//
//	 1  → bullish crossover
//	-1  → bearish crossover
//	 0  → no signal
func (hma *HullMovingAverage) DetectSignals() []float64 {
	signals := make([]float64, len(hma.hmaValues))

	// Align the closes slice with the HMA slice.
	offset := len(hma.closes) - len(hma.hmaValues)
	if offset < 0 {
		offset = 0
	}
	for i := 1; i < len(hma.hmaValues); i++ {
		prevClose := hma.closes[offset+i-1]
		curClose := hma.closes[offset+i]
		prevHMA := hma.hmaValues[i-1]
		curHMA := hma.hmaValues[i]

		if prevClose <= prevHMA && curClose > curHMA {
			signals[i] = 1
		} else if prevClose >= prevHMA && curClose < curHMA {
			signals[i] = -1
		}
	}
	return signals
}

// GetPlotData builds the three PlotData series (HMA, price, signals)
// ready for JSON/CSV export.  Timestamps are generated from the supplied
// start time and interval.
func (hma *HullMovingAverage) GetPlotData(startTime, interval int64) []core.PlotData {
	if len(hma.hmaValues) == 0 {
		return nil
	}

	x := make([]float64, len(hma.hmaValues))
	for i := range x {
		x[i] = float64(i)
	}
	timestamps := core.GenerateTimestamps(startTime, len(hma.hmaValues), interval)

	// Align closes with the HMA slice.
	closesStartIdx := len(hma.closes) - len(hma.hmaValues)
	if closesStartIdx < 0 {
		closesStartIdx = 0
	}
	priceSeries := hma.closes[closesStartIdx:]

	signals := hma.DetectSignals()

	plotData := []core.PlotData{
		{
			Name:      "Hull Moving Average",
			X:         x,
			Y:         hma.hmaValues,
			Type:      "line",
			Timestamp: timestamps,
		},
		{
			Name:      "Price",
			X:         x,
			Y:         priceSeries,
			Type:      "line",
			Timestamp: timestamps,
		},
		{
			Name:      "Signals",
			X:         x,
			Y:         signals,
			Type:      "scatter",
			Timestamp: timestamps,
		},
	}
	return plotData
}
