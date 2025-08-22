package goti

import (
	"errors"
	"math"
)

// HullMovingAverage calculates the Hull Moving Average
type HullMovingAverage struct {
	period    int
	closes    []float64
	rawHMAs   []float64
	hmaValues []float64
	lastValue float64
}

// NewHullMovingAverage initializes with standard period (9)
func NewHullMovingAverage() (*HullMovingAverage, error) {
	return NewHullMovingAverageWithParams(9)
}

// NewHullMovingAverageWithParams initializes with custom period
func NewHullMovingAverageWithParams(period int) (*HullMovingAverage, error) {
	if period < 1 {
		return nil, errors.New("period must be at least 1")
	}
	return &HullMovingAverage{
		period:    period,
		closes:    make([]float64, 0, period*2),
		rawHMAs:   make([]float64, 0, period*2),
		hmaValues: make([]float64, 0, period),
	}, nil
}

// Add appends new price data
func (hma *HullMovingAverage) Add(close float64) error {
	if !isValidPrice(close) {
		return errors.New("invalid price")
	}
	hma.closes = append(hma.closes, close)
	if len(hma.closes) >= hma.period {
		wmaFull, err := calculateWMA(hma.closes[len(hma.closes)-hma.period:], hma.period)
		if err != nil {
			return err
		}
		wmaHalfPeriod := hma.period / 2
		if wmaHalfPeriod < 1 {
			wmaHalfPeriod = 1
		}
		wmaHalf, err := calculateWMA(hma.closes[len(hma.closes)-wmaHalfPeriod:], wmaHalfPeriod)
		if err != nil {
			return err
		}
		rawHMA := 2*wmaHalf - wmaFull
		hma.rawHMAs = append(hma.rawHMAs, rawHMA)
		sqrtPeriod := int(math.Sqrt(float64(hma.period)))
		if sqrtPeriod < 1 {
			sqrtPeriod = 1
		}
		if len(hma.rawHMAs) >= sqrtPeriod {
			hmaValue, err := calculateWMA(hma.rawHMAs[len(hma.rawHMAs)-sqrtPeriod:], sqrtPeriod)
			if err == nil {
				hma.hmaValues = append(hma.hmaValues, hmaValue)
				hma.lastValue = hmaValue
			}
		}
	}
	hma.trimSlices()
	return nil
}

// trimSlices limits slice sizes
func (hma *HullMovingAverage) trimSlices() {
	if len(hma.closes) > hma.period*2 {
		hma.closes = hma.closes[len(hma.closes)-hma.period*2:]
	}
	sqrtPeriod := int(math.Sqrt(float64(hma.period)))
	if sqrtPeriod < 1 {
		sqrtPeriod = 1
	}
	if len(hma.rawHMAs) > sqrtPeriod*2 {
		hma.rawHMAs = hma.rawHMAs[len(hma.rawHMAs)-sqrtPeriod*2:]
	}
	if len(hma.hmaValues) > hma.period {
		hma.hmaValues = hma.hmaValues[len(hma.hmaValues)-hma.period:]
	}
}

// Calculate returns the current HMA value
func (hma *HullMovingAverage) Calculate() (float64, error) {
	if len(hma.hmaValues) == 0 {
		return 0, errors.New("no HMA data")
	}
	return hma.lastValue, nil
}

// GetLastValue returns the last HMA value
func (hma *HullMovingAverage) GetLastValue() float64 {
	return hma.lastValue
}

// IsBullishCrossover checks if price crosses above HMA
func (hma *HullMovingAverage) IsBullishCrossover() (bool, error) {
	if len(hma.hmaValues) < 2 || len(hma.closes) < 2 {
		return false, errors.New("insufficient data for crossover")
	}
	currentHMA := hma.hmaValues[len(hma.hmaValues)-1]
	previousHMA := hma.hmaValues[len(hma.hmaValues)-2]
	currentClose := hma.closes[len(hma.closes)-1]
	previousClose := hma.closes[len(hma.closes)-2]
	return previousClose <= previousHMA && currentClose > currentHMA, nil
}

// IsBearishCrossover checks if price crosses below HMA
func (hma *HullMovingAverage) IsBearishCrossover() (bool, error) {
	if len(hma.hmaValues) < 2 || len(hma.closes) < 2 {
		return false, errors.New("insufficient data for crossover")
	}
	currentHMA := hma.hmaValues[len(hma.hmaValues)-1]
	previousHMA := hma.hmaValues[len(hma.hmaValues)-2]
	currentClose := hma.closes[len(hma.closes)-1]
	previousClose := hma.closes[len(hma.closes)-2]
	return previousClose >= previousHMA && currentClose < currentHMA, nil
}

// GetTrendDirection returns the HMA trend direction
func (hma *HullMovingAverage) GetTrendDirection() (string, error) {
	if len(hma.hmaValues) < 2 {
		return "", errors.New("insufficient data for trend direction")
	}
	current := hma.hmaValues[len(hma.hmaValues)-1]
	previous := hma.hmaValues[len(hma.hmaValues)-2]
	if current > previous {
		return "Bullish", nil
	} else if current < previous {
		return "Bearish", nil
	}
	return "Neutral", nil
}

// Reset clears all data
func (hma *HullMovingAverage) Reset() {
	hma.closes = hma.closes[:0]
	hma.rawHMAs = hma.rawHMAs[:0]
	hma.hmaValues = hma.hmaValues[:0]
	hma.lastValue = 0
}

// SetPeriod updates the period
func (hma *HullMovingAverage) SetPeriod(period int) error {
	if period < 1 {
		return errors.New("period must be at least 1")
	}
	hma.period = period
	hma.trimSlices()
	return nil
}

// GetCloses returns a copy of close prices
func (hma *HullMovingAverage) GetCloses() []float64 {
	return copySlice(hma.closes)
}

// GetHMAValues returns a copy of HMA values
func (hma *HullMovingAverage) GetHMAValues() []float64 {
	return copySlice(hma.hmaValues)
}

// GetPlotData returns data for visualization with signal annotations
func (hma *HullMovingAverage) GetPlotData(startTime, interval int64) []PlotData {
	var plotData []PlotData
	if len(hma.hmaValues) > 0 {
		x := make([]float64, len(hma.hmaValues))
		signals := make([]float64, len(hma.hmaValues))
		timestamps := GenerateTimestamps(startTime, len(hma.hmaValues), interval)
		closesStartIdx := len(hma.closes) - len(hma.hmaValues)
		if closesStartIdx < 0 {
			closesStartIdx = 0
		}
		for i := range hma.hmaValues {
			x[i] = float64(i)
			if i > 0 && closesStartIdx+i < len(hma.closes) {
				if hma.closes[closesStartIdx+i-1] <= hma.hmaValues[i-1] && hma.closes[closesStartIdx+i] > hma.hmaValues[i] {
					signals[i] = 1
				} else if hma.closes[closesStartIdx+i-1] >= hma.hmaValues[i-1] && hma.closes[closesStartIdx+i] < hma.hmaValues[i] {
					signals[i] = -1
				}
			}
		}
		plotData = append(plotData, PlotData{
			Name:      "Hull Moving Average",
			X:         x,
			Y:         hma.hmaValues,
			Type:      "line",
			Timestamp: timestamps,
		})
		plotData = append(plotData, PlotData{
			Name:      "Price",
			X:         x,
			Y:         hma.closes[closesStartIdx:],
			Type:      "line",
			Timestamp: timestamps,
		})
		plotData = append(plotData, PlotData{
			Name:      "Signals",
			X:         x,
			Y:         signals,
			Type:      "scatter",
			Timestamp: timestamps,
		})
	}
	return plotData
}
