package goti

import (
	"errors"
	"fmt"
	"math"
)

// AdaptiveTrendStrengthOscillator calculates the Adaptive Trend Strength Oscillator
type AdaptiveTrendStrengthOscillator struct {
	minPeriod        int
	maxPeriod        int
	volatilityPeriod int
	volSensitivity   float64
	highs            []float64
	lows             []float64
	closes           []float64
	atsoValues       []float64
	lastValue        float64
	ema              *MovingAverage
	config           IndicatorConfig
}

// NewAdaptiveTrendStrengthOscillator initializes with standard periods (5, 14, 14)
func NewAdaptiveTrendStrengthOscillator() (*AdaptiveTrendStrengthOscillator, error) {
	return NewAdaptiveTrendStrengthOscillatorWithParams(5, 14, 14, DefaultConfig())
}

// NewAdaptiveTrendStrengthOscillatorWithParams initializes with custom periods and config
func NewAdaptiveTrendStrengthOscillatorWithParams(minPeriod, maxPeriod, volatilityPeriod int, config IndicatorConfig) (*AdaptiveTrendStrengthOscillator, error) {
	if minPeriod < 1 || maxPeriod < minPeriod || volatilityPeriod < 1 {
		return nil, errors.New("invalid periods")
	}
	ema, err := NewMovingAverage(EMA, maxPeriod)
	if err != nil {
		return nil, fmt.Errorf("failed to create EMA: %w", err)
	}
	return &AdaptiveTrendStrengthOscillator{
		minPeriod:        minPeriod,
		maxPeriod:        maxPeriod,
		volatilityPeriod: volatilityPeriod,
		volSensitivity:   1.0,
		highs:            make([]float64, 0, maxPeriod+volatilityPeriod+1),
		lows:             make([]float64, 0, maxPeriod+volatilityPeriod+1),
		closes:           make([]float64, 0, maxPeriod+volatilityPeriod+1),
		atsoValues:       make([]float64, 0, maxPeriod),
		ema:              ema,
		config:           config,
	}, nil
}

// Add appends new price data
func (atso *AdaptiveTrendStrengthOscillator) Add(high, low, close float64) error {
	if high < low || !isValidPrice(close) {
		return errors.New("invalid price")
	}
	atso.highs = append(atso.highs, high)
	atso.lows = append(atso.lows, low)
	atso.closes = append(atso.closes, close)
	if len(atso.closes) >= atso.maxPeriod+atso.volatilityPeriod+1 {
		atsoValue, err := atso.calculateATSO()
		if err != nil {
			return err
		}
		if err := atso.ema.Add(atsoValue); err != nil {
			return err
		}
		smoothedValue, err := atso.ema.Calculate()
		if err != nil {
			return err
		}
		atso.atsoValues = append(atso.atsoValues, smoothedValue)
		atso.lastValue = smoothedValue
	}
	atso.trimSlices()
	return nil
}

// trimSlices limits slice sizes
func (atso *AdaptiveTrendStrengthOscillator) trimSlices() {
	if len(atso.closes) > atso.maxPeriod+atso.volatilityPeriod+1 {
		atso.highs = atso.highs[len(atso.highs)-atso.maxPeriod-atso.volatilityPeriod-1:]
		atso.lows = atso.lows[len(atso.lows)-atso.maxPeriod-atso.volatilityPeriod-1:]
		atso.closes = atso.closes[len(atso.closes)-atso.maxPeriod-atso.volatilityPeriod-1:]
	}
	if len(atso.atsoValues) > atso.maxPeriod {
		atso.atsoValues = atso.atsoValues[len(atso.atsoValues)-atso.maxPeriod:]
	}
}

// calculateATSO computes the Adaptive Trend Strength Oscillator value
func (atso *AdaptiveTrendStrengthOscillator) calculateATSO() (float64, error) {
	if len(atso.closes) < atso.maxPeriod+atso.volatilityPeriod+1 {
		return 0, fmt.Errorf("insufficient data: need %d, have %d", atso.maxPeriod+atso.volatilityPeriod+1, len(atso.closes))
	}
	// Calculate volatility
	startIdx := len(atso.closes) - atso.volatilityPeriod
	volatility := calculateStandardDeviation(atso.closes[startIdx:], 0)
	// Adjust period with sensitivity
	period := atso.minPeriod + int(float64(atso.maxPeriod-atso.minPeriod)*math.Min(volatility*atso.volSensitivity, 1))
	if period > atso.maxPeriod {
		period = atso.maxPeriod
	}
	if period < atso.minPeriod {
		period = atso.minPeriod
	}
	if len(atso.closes) < period+1 {
		return 0, fmt.Errorf("insufficient data for period %d: have %d", period+1, len(atso.closes))
	}
	// Calculate trend strength
	startIdx = len(atso.highs) - period
	highs := atso.highs[startIdx:]
	lows := atso.lows[startIdx:]
	sumUp, sumDown := 0.0, 0.0
	for i := 1; i < period; i++ {
		highDiff := highs[i] - highs[i-1]
		lowDiff := lows[i-1] - lows[i]
		if highDiff > lowDiff && highDiff > 0 {
			sumUp += highDiff
		}
		if lowDiff > highDiff && lowDiff > 0 {
			sumDown += lowDiff
		}
	}
	trendStrength := (sumUp - sumDown) / float64(period)
	// Normalize by historical trend strength
	avgTrendStrength := 0.0
	count := 0
	for i := period; i < len(atso.closes); i++ {
		subSumUp, subSumDown := 0.0, 0.0
		for j := 1; j < period; j++ {
			if i-j-1 < 0 {
				break
			}
			highDiff := atso.highs[i-j] - atso.highs[i-j-1]
			lowDiff := atso.lows[i-j-1] - atso.lows[i-j]
			if highDiff > lowDiff && highDiff > 0 {
				subSumUp += highDiff
			}
			if lowDiff > highDiff && lowDiff > 0 {
				subSumDown += lowDiff
			}
		}
		if subSumUp != 0 || subSumDown != 0 {
			avgTrendStrength += (subSumUp - subSumDown) / float64(period)
			count++
		}
	}
	if count == 0 {
		return trendStrength * 100, nil
	}
	avgTrendStrength /= float64(count)
	if avgTrendStrength == 0 {
		return 0.001, nil // Avoid division-by-zero
	}
	return clamp((trendStrength/avgTrendStrength-1)*100, -100, 100), nil
}

// Calculate returns the current ATSO value
func (atso *AdaptiveTrendStrengthOscillator) Calculate() (float64, error) {
	if len(atso.atsoValues) == 0 {
		return 0, errors.New("no ATSO data")
	}
	return atso.lastValue, nil
}

// GetLastValue returns the last ATSO value
func (atso *AdaptiveTrendStrengthOscillator) GetLastValue() float64 {
	return atso.lastValue
}

// IsBullishCrossover checks if ATSO crosses above zero
func (atso *AdaptiveTrendStrengthOscillator) IsBullishCrossover() (bool, error) {
	if len(atso.atsoValues) < 2 {
		return false, errors.New("insufficient data for crossover")
	}
	current := atso.atsoValues[len(atso.atsoValues)-1]
	previous := atso.atsoValues[len(atso.atsoValues)-2]
	return previous <= 0 && current > 0, nil
}

// IsBearishCrossover checks if ATSO crosses below zero
func (atso *AdaptiveTrendStrengthOscillator) IsBearishCrossover() (bool, error) {
	if len(atso.atsoValues) < 2 {
		return false, errors.New("insufficient data for crossover")
	}
	current := atso.atsoValues[len(atso.atsoValues)-1]
	previous := atso.atsoValues[len(atso.atsoValues)-2]
	return previous >= 0 && current < 0, nil
}

// SetVolatilitySensitivity adjusts the volatility impact on period adaptation
func (atso *AdaptiveTrendStrengthOscillator) SetVolatilitySensitivity(sensitivity float64) error {
	if sensitivity <= 0 {
		return errors.New("sensitivity must be positive")
	}
	atso.volSensitivity = sensitivity
	return nil
}

// Reset clears all data
func (atso *AdaptiveTrendStrengthOscillator) Reset() {
	atso.highs = atso.highs[:0]
	atso.lows = atso.lows[:0]
	atso.closes = atso.closes[:0]
	atso.atsoValues = atso.atsoValues[:0]
	atso.ema.Reset()
	atso.lastValue = 0
}

// SetPeriods updates the periods
func (atso *AdaptiveTrendStrengthOscillator) SetPeriods(minPeriod, maxPeriod, volatilityPeriod int) error {
	if minPeriod < 1 || maxPeriod < minPeriod || volatilityPeriod < 1 {
		return errors.New("invalid periods")
	}
	atso.minPeriod, atso.maxPeriod, atso.volatilityPeriod = minPeriod, maxPeriod, volatilityPeriod
	if err := atso.ema.SetPeriod(maxPeriod); err != nil {
		return err
	}
	atso.trimSlices()
	return nil
}

// GetHighs returns a copy of high prices
func (atso *AdaptiveTrendStrengthOscillator) GetHighs() []float64 {
	return copySlice(atso.highs)
}

// GetLows returns a copy of low prices
func (atso *AdaptiveTrendStrengthOscillator) GetLows() []float64 {
	return copySlice(atso.lows)
}

// GetCloses returns a copy of close prices
func (atso *AdaptiveTrendStrengthOscillator) GetCloses() []float64 {
	return copySlice(atso.closes)
}

// GetATSOValues returns a copy of ATSO values
func (atso *AdaptiveTrendStrengthOscillator) GetATSOValues() []float64 {
	return copySlice(atso.atsoValues)
}

// GetPlotData returns data for visualization with signal annotations
func (atso *AdaptiveTrendStrengthOscillator) GetPlotData(startTime, interval int64) []PlotData {
	var plotData []PlotData
	if len(atso.atsoValues) > 0 {
		x := make([]float64, len(atso.atsoValues))
		signals := make([]float64, len(atso.atsoValues))
		timestamps := GenerateTimestamps(startTime, len(atso.atsoValues), interval)
		for i := range atso.atsoValues {
			x[i] = float64(i)
			if i > 0 {
				if atso.atsoValues[i-1] <= 0 && atso.atsoValues[i] > 0 {
					signals[i] = 1
				} else if atso.atsoValues[i-1] >= 0 && atso.atsoValues[i] < 0 {
					signals[i] = -1
				}
			}
		}
		plotData = append(plotData, PlotData{
			Name:      "Adaptive Trend Strength Oscillator",
			X:         x,
			Y:         atso.atsoValues,
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
