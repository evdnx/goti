package goti

import (
	"errors"
	"fmt"
)

// RelativeStrengthIndex calculates the Relative Strength Index
type RelativeStrengthIndex struct {
	period    int
	closes    []float64
	rsiValues []float64
	lastValue float64
	config    IndicatorConfig
}

// NewRelativeStrengthIndex initializes with standard period (14) and default config
func NewRelativeStrengthIndex() (*RelativeStrengthIndex, error) {
	return NewRelativeStrengthIndexWithParams(14, DefaultConfig())
}

// NewRelativeStrengthIndexWithParams initializes with custom period and config
func NewRelativeStrengthIndexWithParams(period int, config IndicatorConfig) (*RelativeStrengthIndex, error) {
	if period < 1 {
		return nil, errors.New("period must be at least 1")
	}
	if config.RSIOverbought <= config.RSIOversold {
		return nil, errors.New("RSI overbought threshold must be greater than oversold")
	}
	return &RelativeStrengthIndex{
		period:    period,
		closes:    make([]float64, 0, period+1),
		rsiValues: make([]float64, 0, period),
		config:    config,
	}, nil
}

// Add appends new price data
func (rsi *RelativeStrengthIndex) Add(close float64) error {
	if !isValidPrice(close) {
		return errors.New("invalid price")
	}
	rsi.closes = append(rsi.closes, close)
	if len(rsi.closes) >= rsi.period+1 {
		rsiValue, err := rsi.calculateRSI()
		if err == nil {
			rsi.rsiValues = append(rsi.rsiValues, rsiValue)
			rsi.lastValue = rsiValue
		}
	}
	rsi.trimSlices()
	return nil
}

// trimSlices limits slice sizes
func (rsi *RelativeStrengthIndex) trimSlices() {
	if len(rsi.closes) > rsi.period+1 {
		rsi.closes = rsi.closes[len(rsi.closes)-rsi.period-1:]
	}
	if len(rsi.rsiValues) > rsi.period {
		rsi.rsiValues = rsi.rsiValues[len(rsi.rsiValues)-rsi.period:]
	}
}

// calculateRSI computes the Relative Strength Index value
func (rsi *RelativeStrengthIndex) calculateRSI() (float64, error) {
	if len(rsi.closes) < rsi.period+1 {
		return 0, fmt.Errorf("insufficient data: need %d, have %d", rsi.period+1, len(rsi.closes))
	}
	gain, loss := 0.0, 0.0
	// Calculate initial average gain/loss
	for i := 1; i <= rsi.period; i++ {
		diff := rsi.closes[len(rsi.closes)-rsi.period+i] - rsi.closes[len(rsi.closes)-rsi.period+i-1]
		if diff > 0 {
			gain += diff
		} else if diff < 0 {
			loss -= diff
		}
	}
	gain /= float64(rsi.period)
	loss /= float64(rsi.period)
	if loss == 0 {
		if gain == 0 {
			return 50, nil // Neutral case
		}
		return 100, nil // All gains
	}
	rs := gain / loss
	rsiValue := 100 - (100 / (1 + rs))
	return clamp(rsiValue, 0, 100), nil
}

// Calculate returns the current RSI value
func (rsi *RelativeStrengthIndex) Calculate() (float64, error) {
	if len(rsi.rsiValues) == 0 {
		return 0, errors.New("no RSI data")
	}
	return rsi.lastValue, nil
}

// GetLastValue returns the last RSI value
func (rsi *RelativeStrengthIndex) GetLastValue() float64 {
	return rsi.lastValue
}

// IsBullishCrossover checks if RSI crosses above oversold threshold
func (rsi *RelativeStrengthIndex) IsBullishCrossover() (bool, error) {
	if len(rsi.rsiValues) < 2 {
		return false, errors.New("insufficient data for crossover")
	}
	current := rsi.rsiValues[len(rsi.rsiValues)-1]
	previous := rsi.rsiValues[len(rsi.rsiValues)-2]
	return previous <= rsi.config.RSIOversold && current > rsi.config.RSIOversold, nil
}

// IsBearishCrossover checks if RSI crosses below overbought threshold
func (rsi *RelativeStrengthIndex) IsBearishCrossover() (bool, error) {
	if len(rsi.rsiValues) < 2 {
		return false, errors.New("insufficient data for crossover")
	}
	current := rsi.rsiValues[len(rsi.rsiValues)-1]
	previous := rsi.rsiValues[len(rsi.rsiValues)-2]
	return previous >= rsi.config.RSIOverbought && current < rsi.config.RSIOverbought, nil
}

// GetOverboughtOversold returns overbought/oversold status
func (rsi *RelativeStrengthIndex) GetOverboughtOversold() (string, error) {
	if len(rsi.rsiValues) < 1 {
		return "", errors.New("no RSI data")
	}
	current := rsi.rsiValues[len(rsi.rsiValues)-1]
	if current > rsi.config.RSIOverbought {
		return "Overbought", nil
	}
	if current < rsi.config.RSIOversold {
		return "Oversold", nil
	}
	return "Neutral", nil
}

// IsDivergence checks for RSI divergence signals
func (rsi *RelativeStrengthIndex) IsDivergence() (bool, string, error) {
	if len(rsi.rsiValues) < 2 || len(rsi.closes) < 2 {
		return false, "", errors.New("insufficient data for divergence")
	}
	currentRSI := rsi.rsiValues[len(rsi.rsiValues)-1]
	priceTrend := rsi.closes[len(rsi.closes)-1] - rsi.closes[len(rsi.closes)-2]
	if currentRSI > rsi.config.RSIOverbought && priceTrend < 0 {
		return true, "Bearish", nil
	}
	if currentRSI < rsi.config.RSIOversold && priceTrend > 0 {
		return true, "Bullish", nil
	}
	return false, "", nil
}

// Reset clears all data
func (rsi *RelativeStrengthIndex) Reset() {
	rsi.closes = rsi.closes[:0]
	rsi.rsiValues = rsi.rsiValues[:0]
	rsi.lastValue = 0
}

// SetPeriod updates the period
func (rsi *RelativeStrengthIndex) SetPeriod(period int) error {
	if period < 1 {
		return errors.New("period must be at least 1")
	}
	rsi.period = period
	rsi.trimSlices()
	return nil
}

// GetCloses returns a copy of close prices
func (rsi *RelativeStrengthIndex) GetCloses() []float64 {
	return copySlice(rsi.closes)
}

// GetRSIValues returns a copy of RSI values
func (rsi *RelativeStrengthIndex) GetRSIValues() []float64 {
	return copySlice(rsi.rsiValues)
}

// GetPlotData returns data for visualization with signal annotations
func (rsi *RelativeStrengthIndex) GetPlotData(startTime, interval int64) []PlotData {
	var plotData []PlotData
	if len(rsi.rsiValues) > 0 {
		x := make([]float64, len(rsi.rsiValues))
		signals := make([]float64, len(rsi.rsiValues))
		timestamps := GenerateTimestamps(startTime, len(rsi.rsiValues), interval)
		for i := range rsi.rsiValues {
			x[i] = float64(i)
			if i > 0 {
				if rsi.rsiValues[i-1] <= rsi.config.RSIOversold && rsi.rsiValues[i] > rsi.config.RSIOversold {
					signals[i] = 1
				} else if rsi.rsiValues[i-1] >= rsi.config.RSIOverbought && rsi.rsiValues[i] < rsi.config.RSIOverbought {
					signals[i] = -1
				}
			}
			if rsi.rsiValues[i] > rsi.config.RSIOverbought {
				signals[i] = 2
			} else if rsi.rsiValues[i] < rsi.config.RSIOversold {
				signals[i] = -2
			}
		}
		plotData = append(plotData, PlotData{
			Name:      "Relative Strength Index",
			X:         x,
			Y:         rsi.rsiValues,
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
