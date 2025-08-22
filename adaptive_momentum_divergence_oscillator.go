package goti

import (
	"errors"
	"fmt"
	"math"
)

// AdaptiveMomentumDivergenceOscillator calculates the Adaptive Momentum Divergence Oscillator
type AdaptiveMomentumDivergenceOscillator struct {
	minPeriod        int
	maxPeriod        int
	volatilityPeriod int
	closes           []float64
	momValues        []float64
	amdoValues       []float64
	lastValue        float64
	config           IndicatorConfig
}

// NewAdaptiveMomentumDivergenceOscillator initializes with standard periods (5, 14, 14)
func NewAdaptiveMomentumDivergenceOscillator() (*AdaptiveMomentumDivergenceOscillator, error) {
	return NewAdaptiveMomentumDivergenceOscillatorWithParams(5, 14, 14, DefaultConfig())
}

// NewAdaptiveMomentumDivergenceOscillatorWithParams initializes with custom periods and config
func NewAdaptiveMomentumDivergenceOscillatorWithParams(minPeriod, maxPeriod, volatilityPeriod int, config IndicatorConfig) (*AdaptiveMomentumDivergenceOscillator, error) {
	if minPeriod < 1 || maxPeriod < minPeriod || volatilityPeriod < 1 {
		return nil, errors.New("invalid periods")
	}
	return &AdaptiveMomentumDivergenceOscillator{
		minPeriod:        minPeriod,
		maxPeriod:        maxPeriod,
		volatilityPeriod: volatilityPeriod,
		closes:           make([]float64, 0, maxPeriod*2+volatilityPeriod),
		momValues:        make([]float64, 0, maxPeriod+1),
		amdoValues:       make([]float64, 0, maxPeriod),
		config:           config,
	}, nil
}

// Add appends new price data
func (amdo *AdaptiveMomentumDivergenceOscillator) Add(close float64) error {
	if !isValidPrice(close) {
		return errors.New("invalid price")
	}
	amdo.closes = append(amdo.closes, close)
	if len(amdo.closes) >= amdo.maxPeriod+1 {
		momentum := close - amdo.closes[len(amdo.closes)-amdo.maxPeriod-1]
		amdo.momValues = append(amdo.momValues, momentum)
		if len(amdo.momValues) >= amdo.maxPeriod {
			amdoValue, err := amdo.calculateAMDO()
			if err == nil {
				amdo.amdoValues = append(amdo.amdoValues, amdoValue)
				amdo.lastValue = amdoValue
			}
		}
	}
	amdo.trimSlices()
	return nil
}

// trimSlices limits slice sizes
func (amdo *AdaptiveMomentumDivergenceOscillator) trimSlices() {
	if len(amdo.closes) > amdo.maxPeriod*2+amdo.volatilityPeriod {
		amdo.closes = amdo.closes[len(amdo.closes)-amdo.maxPeriod*2-amdo.volatilityPeriod:]
	}
	if len(amdo.momValues) > amdo.maxPeriod+1 {
		amdo.momValues = amdo.momValues[len(amdo.momValues)-amdo.maxPeriod-1:]
	}
	if len(amdo.amdoValues) > amdo.maxPeriod {
		amdo.amdoValues = amdo.amdoValues[len(amdo.amdoValues)-amdo.maxPeriod:]
	}
}

// calculateAMDO computes the Adaptive Momentum Divergence Oscillator value
func (amdo *AdaptiveMomentumDivergenceOscillator) calculateAMDO() (float64, error) {
	if len(amdo.momValues) < amdo.maxPeriod {
		return 0, fmt.Errorf("insufficient momentum data: need %d, have %d", amdo.maxPeriod, len(amdo.momValues))
	}
	// Calculate volatility
	startIdx := len(amdo.closes) - amdo.volatilityPeriod
	volatility := calculateStandardDeviation(amdo.closes[startIdx:], 0)
	// Adjust period
	period := amdo.minPeriod + int(float64(amdo.maxPeriod-amdo.minPeriod)*math.Min(volatility, 1))
	if period > amdo.maxPeriod {
		period = amdo.maxPeriod
	}
	if period < amdo.minPeriod {
		period = amdo.minPeriod
	}
	if len(amdo.closes) < period || len(amdo.momValues) < period {
		return 0, fmt.Errorf("insufficient data for period %d: have closes %d, mom %d", period, len(amdo.closes), len(amdo.momValues))
	}
	startIdx = len(amdo.closes) - period
	closes := amdo.closes[startIdx:]
	moms := amdo.momValues[len(amdo.momValues)-period:]
	priceHigh, priceLow := closes[0], closes[0]
	momHigh, momLow := moms[0], moms[0]
	for _, p := range closes {
		if p > priceHigh {
			priceHigh = p
		}
		if p < priceLow {
			priceLow = p
		}
	}
	for _, m := range moms {
		if m > momHigh {
			momHigh = m
		}
		if m < momLow {
			momLow = m
		}
	}
	if priceHigh == priceLow || momHigh == momLow {
		return 0.001, nil // Small non-zero value to avoid division-by-zero
	}
	currentPrice := closes[len(closes)-1]
	currentMom := moms[len(moms)-1]
	normPrice := (currentPrice - priceLow) / (priceHigh - priceLow)
	normMom := (currentMom - momLow) / (momHigh - momLow)
	return clamp((normPrice-normMom)*100, -100, 100), nil
}

// Calculate returns the current AMDO value
func (amdo *AdaptiveMomentumDivergenceOscillator) Calculate() (float64, error) {
	if len(amdo.amdoValues) == 0 {
		return 0, errors.New("no AMDO data")
	}
	return amdo.lastValue, nil
}

// GetLastValue returns the last AMDO value
func (amdo *AdaptiveMomentumDivergenceOscillator) GetLastValue() float64 {
	return amdo.lastValue
}

// IsBullishCrossover checks if AMDO crosses above zero
func (amdo *AdaptiveMomentumDivergenceOscillator) IsBullishCrossover() (bool, error) {
	if len(amdo.amdoValues) < 2 {
		return false, errors.New("insufficient data for crossover")
	}
	current := amdo.amdoValues[len(amdo.amdoValues)-1]
	previous := amdo.amdoValues[len(amdo.amdoValues)-2]
	return previous <= 0 && current > 0, nil
}

// IsBearishCrossover checks if AMDO crosses below zero
func (amdo *AdaptiveMomentumDivergenceOscillator) IsBearishCrossover() (bool, error) {
	if len(amdo.amdoValues) < 2 {
		return false, errors.New("insufficient data for crossover")
	}
	current := amdo.amdoValues[len(amdo.amdoValues)-1]
	previous := amdo.amdoValues[len(amdo.amdoValues)-2]
	return previous >= 0 && current < 0, nil
}

// IsStrongDivergence checks for strong divergence signals
func (amdo *AdaptiveMomentumDivergenceOscillator) IsStrongDivergence() (bool, string, error) {
	if len(amdo.amdoValues) < 2 || len(amdo.closes) < 2 {
		return false, "", errors.New("insufficient data for divergence")
	}
	current := amdo.amdoValues[len(amdo.amdoValues)-1]
	priceTrend := amdo.closes[len(amdo.closes)-1] - amdo.closes[len(amdo.closes)-2]
	if current > amdo.config.AMDODivergence && priceTrend < 0 {
		return true, "Bullish", nil
	}
	if current < -amdo.config.AMDODivergence && priceTrend > 0 {
		return true, "Bearish", nil
	}
	return false, "", nil
}

// Reset clears all data
func (amdo *AdaptiveMomentumDivergenceOscillator) Reset() {
	amdo.closes = amdo.closes[:0]
	amdo.momValues = amdo.momValues[:0]
	amdo.amdoValues = amdo.amdoValues[:0]
	amdo.lastValue = 0
}

// SetPeriods updates the periods
func (amdo *AdaptiveMomentumDivergenceOscillator) SetPeriods(minPeriod, maxPeriod, volatilityPeriod int) error {
	if minPeriod < 1 || maxPeriod < minPeriod || volatilityPeriod < 1 {
		return errors.New("invalid periods")
	}
	amdo.minPeriod, amdo.maxPeriod, amdo.volatilityPeriod = minPeriod, maxPeriod, volatilityPeriod
	amdo.trimSlices()
	return nil
}

// GetCloses returns a copy of close prices
func (amdo *AdaptiveMomentumDivergenceOscillator) GetCloses() []float64 {
	return copySlice(amdo.closes)
}

// GetAMDOValues returns a copy of AMDO values
func (amdo *AdaptiveMomentumDivergenceOscillator) GetAMDOValues() []float64 {
	return copySlice(amdo.amdoValues)
}

// GetPlotData returns data for visualization
func (amdo *AdaptiveMomentumDivergenceOscillator) GetPlotData(startTime, interval int64) []PlotData {
	var plotData []PlotData
	if len(amdo.amdoValues) > 0 {
		x := make([]float64, len(amdo.amdoValues))
		signals := make([]float64, len(amdo.amdoValues))
		timestamps := GenerateTimestamps(startTime, len(amdo.amdoValues), interval)
		for i := range amdo.amdoValues {
			x[i] = float64(i)
			if i > 0 {
				if amdo.amdoValues[i-1] <= 0 && amdo.amdoValues[i] > 0 {
					signals[i] = 1 // Bullish crossover
				} else if amdo.amdoValues[i-1] >= 0 && amdo.amdoValues[i] < 0 {
					signals[i] = -1 // Bearish crossover
				}
				if len(amdo.closes) > i+1 {
					priceTrend := amdo.closes[len(amdo.closes)-len(amdo.amdoValues)+i] - amdo.closes[len(amdo.closes)-len(amdo.amdoValues)+i-1]
					if amdo.amdoValues[i] > amdo.config.AMDODivergence && priceTrend < 0 {
						signals[i] = 2 // Bullish divergence
					} else if amdo.amdoValues[i] < -amdo.config.AMDODivergence && priceTrend > 0 {
						signals[i] = -2 // Bearish divergence
					}
				}
			}
		}
		plotData = append(plotData, PlotData{
			Name:      "Adaptive Momentum Divergence Oscillator",
			X:         x,
			Y:         amdo.amdoValues,
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
