package goti

import (
	"errors"
	"fmt"
	"math"
)

// AverageTrueRange calculates the Average True Range
type AverageTrueRange struct {
	period    int
	highs     []float64
	lows      []float64
	closes    []float64
	atrValues []float64
	lastValue float64
}

// NewAverageTrueRange initializes with standard period (14)
func NewAverageTrueRange() (*AverageTrueRange, error) {
	return NewAverageTrueRangeWithParams(14)
}

// NewAverageTrueRangeWithParams initializes with custom period
func NewAverageTrueRangeWithParams(period int) (*AverageTrueRange, error) {
	if period < 1 {
		return nil, errors.New("period must be at least 1")
	}
	return &AverageTrueRange{
		period:    period,
		highs:     make([]float64, 0, period+1),
		lows:      make([]float64, 0, period+1),
		closes:    make([]float64, 0, period+1),
		atrValues: make([]float64, 0, period),
	}, nil
}

// Add appends new price data
func (atr *AverageTrueRange) Add(high, low, close float64) error {
	if high < low || !isValidPrice(close) {
		return errors.New("invalid price")
	}
	atr.highs = append(atr.highs, high)
	atr.lows = append(atr.lows, low)
	atr.closes = append(atr.closes, close)
	if len(atr.closes) >= atr.period+1 {
		atrValue, err := atr.calculateATR()
		if err == nil {
			atr.atrValues = append(atr.atrValues, atrValue)
			atr.lastValue = atrValue
		}
	}
	atr.trimSlices()
	return nil
}

// trimSlices limits slice sizes
func (atr *AverageTrueRange) trimSlices() {
	if len(atr.closes) > atr.period+1 {
		atr.highs = atr.highs[len(atr.highs)-atr.period-1:]
		atr.lows = atr.lows[len(atr.lows)-atr.period-1:]
		atr.closes = atr.closes[len(atr.closes)-atr.period-1:]
	}
	if len(atr.atrValues) > atr.period {
		atr.atrValues = atr.atrValues[len(atr.atrValues)-atr.period:]
	}
}

// calculateATR computes the Average True Range value
func (atr *AverageTrueRange) calculateATR() (float64, error) {
	if len(atr.closes) < atr.period+1 {
		return 0, fmt.Errorf("insufficient data: need %d, have %d", atr.period+1, len(atr.closes))
	}
	sumTR := 0.0
	for i := 1; i <= atr.period; i++ {
		idx := len(atr.closes) - atr.period + i - 1
		highLow := atr.highs[idx] - atr.lows[idx]
		highPrevClose := math.Abs(atr.highs[idx] - atr.closes[idx-1])
		lowPrevClose := math.Abs(atr.lows[idx] - atr.closes[idx-1])
		tr := math.Max(highLow, math.Max(highPrevClose, lowPrevClose))
		sumTR += tr
	}
	return sumTR / float64(atr.period), nil
}

// Calculate returns the current ATR value
func (atr *AverageTrueRange) Calculate() (float64, error) {
	if len(atr.atrValues) == 0 {
		return 0, errors.New("no ATR data")
	}
	return atr.lastValue, nil
}

// GetLastValue returns the last ATR value
func (atr *AverageTrueRange) GetLastValue() float64 {
	return atr.lastValue
}

// Reset clears all data
func (atr *AverageTrueRange) Reset() {
	atr.highs = atr.highs[:0]
	atr.lows = atr.lows[:0]
	atr.closes = atr.closes[:0]
	atr.atrValues = atr.atrValues[:0]
	atr.lastValue = 0
}

// SetPeriod updates the period
func (atr *AverageTrueRange) SetPeriod(period int) error {
	if period < 1 {
		return errors.New("period must be at least 1")
	}
	atr.period = period
	atr.trimSlices()
	return nil
}

// GetHighs returns a copy of high prices
func (atr *AverageTrueRange) GetHighs() []float64 {
	return copySlice(atr.highs)
}

// GetLows returns a copy of low prices
func (atr *AverageTrueRange) GetLows() []float64 {
	return copySlice(atr.lows)
}

// GetCloses returns a copy of close prices
func (atr *AverageTrueRange) GetCloses() []float64 {
	return copySlice(atr.closes)
}

// GetATRValues returns a copy of ATR values
func (atr *AverageTrueRange) GetATRValues() []float64 {
	return copySlice(atr.atrValues)
}

// GetPlotData returns data for visualization
func (atr *AverageTrueRange) GetPlotData(startTime, interval int64) []PlotData {
	var plotData []PlotData
	if len(atr.atrValues) > 0 {
		x := make([]float64, len(atr.atrValues))
		timestamps := GenerateTimestamps(startTime, len(atr.atrValues), interval)
		for i := range atr.atrValues {
			x[i] = float64(i)
		}
		plotData = append(plotData, PlotData{
			Name:      "Average True Range",
			X:         x,
			Y:         atr.atrValues,
			Type:      "line",
			Timestamp: timestamps,
		})
	}
	return plotData
}
