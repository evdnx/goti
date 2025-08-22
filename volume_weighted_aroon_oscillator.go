package goti

import (
	"errors"
	"fmt"
	"math"
)

// VolumeWeightedAroonOscillator calculates the Volume-Weighted Aroon Oscillator
type VolumeWeightedAroonOscillator struct {
	period     int
	highs      []float64
	lows       []float64
	closes     []float64
	volumes    []float64
	vwaoValues []float64
	aroonUp    []float64
	aroonDown  []float64
	lastValue  float64
	config     IndicatorConfig
}

// NewVolumeWeightedAroonOscillator initializes with standard period (14)
func NewVolumeWeightedAroonOscillator() (*VolumeWeightedAroonOscillator, error) {
	return NewVolumeWeightedAroonOscillatorWithParams(14, DefaultConfig())
}

// NewVolumeWeightedAroonOscillatorWithParams initializes with custom period and config
func NewVolumeWeightedAroonOscillatorWithParams(period int, config IndicatorConfig) (*VolumeWeightedAroonOscillator, error) {
	if period < 1 {
		return nil, errors.New("period must be at least 1")
	}
	return &VolumeWeightedAroonOscillator{
		period:     period,
		highs:      make([]float64, 0, period+1),
		lows:       make([]float64, 0, period+1),
		closes:     make([]float64, 0, period+1),
		volumes:    make([]float64, 0, period+1),
		vwaoValues: make([]float64, 0, period),
		aroonUp:    make([]float64, 0, period),
		aroonDown:  make([]float64, 0, period),
		config:     config,
	}, nil
}

// Add appends new price and volume data
func (vwao *VolumeWeightedAroonOscillator) Add(high, low, close, volume float64) error {
	if high < low || !isValidPrice(close) || !isValidVolume(volume) {
		return errors.New("invalid price or volume")
	}
	vwao.highs = append(vwao.highs, high)
	vwao.lows = append(vwao.lows, low)
	vwao.closes = append(vwao.closes, close)
	vwao.volumes = append(vwao.volumes, volume)
	if len(vwao.closes) >= vwao.period+1 {
		vwaoValue, up, down, err := vwao.calculateVWAO()
		if err == nil {
			vwao.vwaoValues = append(vwao.vwaoValues, vwaoValue)
			vwao.aroonUp = append(vwao.aroonUp, up)
			vwao.aroonDown = append(vwao.aroonDown, down)
			vwao.lastValue = vwaoValue
		}
	}
	vwao.trimSlices()
	return nil
}

// trimSlices limits slice sizes
func (vwao *VolumeWeightedAroonOscillator) trimSlices() {
	if len(vwao.closes) > vwao.period+1 {
		vwao.highs = vwao.highs[len(vwao.highs)-vwao.period-1:]
		vwao.lows = vwao.lows[len(vwao.lows)-vwao.period-1:]
		vwao.closes = vwao.closes[len(vwao.closes)-vwao.period-1:]
		vwao.volumes = vwao.volumes[len(vwao.volumes)-vwao.period-1:]
	}
	if len(vwao.vwaoValues) > vwao.period {
		vwao.vwaoValues = vwao.vwaoValues[len(vwao.vwaoValues)-vwao.period:]
		vwao.aroonUp = vwao.aroonUp[len(vwao.aroonUp)-vwao.period:]
		vwao.aroonDown = vwao.aroonDown[len(vwao.aroonDown)-vwao.period:]
	}
}

// calculateVWAO computes the Volume-Weighted Aroon Oscillator value
func (vwao *VolumeWeightedAroonOscillator) calculateVWAO() (float64, float64, float64, error) {
	if len(vwao.closes) < vwao.period+1 {
		return 0, 0, 0, fmt.Errorf("insufficient data: need %d, have %d", vwao.period+1, len(vwao.closes))
	}
	startIdx := len(vwao.closes) - vwao.period
	highs := vwao.highs[startIdx:]
	lows := vwao.lows[startIdx:]
	volumes := vwao.volumes[startIdx:]
	sumVolume := 0.0
	for _, v := range volumes {
		sumVolume += v
	}
	if sumVolume == 0 {
		// Fall back to standard Aroon if no volume
		maxHighIdx, minLowIdx := 0, 0
		for i := 1; i < vwao.period; i++ {
			if highs[i] > highs[maxHighIdx] {
				maxHighIdx = i
			}
			if lows[i] < lows[minLowIdx] {
				minLowIdx = i
			}
		}
		daysSinceHigh := vwao.period - 1 - maxHighIdx
		daysSinceLow := vwao.period - 1 - minLowIdx
		aroonUp := 100.0 * float64(vwao.period-daysSinceHigh) / float64(vwao.period)
		aroonDown := 100.0 * float64(vwao.period-daysSinceLow) / float64(vwao.period)
		return clamp(aroonUp-aroonDown, -100, 100), aroonUp, aroonDown, nil
	}
	// Find argmax of high*volume
	maxWeighted := -math.MaxFloat64
	daysSinceHigh := vwao.period - 1
	for i := 0; i < vwao.period; i++ {
		weighted := highs[i] * volumes[i]
		if weighted > maxWeighted {
			maxWeighted = weighted
			daysSinceHigh = vwao.period - 1 - i
		}
	}
	// Find argmin of low*volume
	minWeighted := math.MaxFloat64
	daysSinceLow := vwao.period - 1
	for i := 0; i < vwao.period; i++ {
		weighted := lows[i] * volumes[i]
		if weighted < minWeighted {
			minWeighted = weighted
			daysSinceLow = vwao.period - 1 - i
		}
	}
	aroonUp := 100.0 * float64(vwao.period-daysSinceHigh) / float64(vwao.period)
	aroonDown := 100.0 * float64(vwao.period-daysSinceLow) / float64(vwao.period)
	return clamp(aroonUp-aroonDown, -100, 100), aroonUp, aroonDown, nil
}

// Calculate returns the current VWAO value
func (vwao *VolumeWeightedAroonOscillator) Calculate() (float64, error) {
	if len(vwao.vwaoValues) == 0 {
		return 0, errors.New("no VWAO data")
	}
	return vwao.lastValue, nil
}

// GetLastValue returns the last VWAO value
func (vwao *VolumeWeightedAroonOscillator) GetLastValue() float64 {
	return vwao.lastValue
}

// IsBullishCrossover checks if VWAO crosses above zero
func (vwao *VolumeWeightedAroonOscillator) IsBullishCrossover() (bool, error) {
	if len(vwao.vwaoValues) < 2 {
		return false, errors.New("insufficient data for crossover")
	}
	current := vwao.vwaoValues[len(vwao.vwaoValues)-1]
	previous := vwao.vwaoValues[len(vwao.vwaoValues)-2]
	return previous <= 0 && current > 0, nil
}

// IsBearishCrossover checks if VWAO crosses below zero
func (vwao *VolumeWeightedAroonOscillator) IsBearishCrossover() (bool, error) {
	if len(vwao.vwaoValues) < 2 {
		return false, errors.New("insufficient data for crossover")
	}
	current := vwao.vwaoValues[len(vwao.vwaoValues)-1]
	previous := vwao.vwaoValues[len(vwao.vwaoValues)-2]
	return previous >= 0 && current < 0, nil
}

// IsStrongTrend checks for strong trend signals
func (vwao *VolumeWeightedAroonOscillator) IsStrongTrend() (bool, string, error) {
	if len(vwao.vwaoValues) < 1 {
		return false, "", errors.New("insufficient data for trend")
	}
	current := vwao.vwaoValues[len(vwao.vwaoValues)-1]
	if current > vwao.config.VWAOStrongTrend {
		return true, "Bullish", nil
	}
	if current < -vwao.config.VWAOStrongTrend {
		return true, "Bearish", nil
	}
	return false, "", nil
}

// GetAroonUpDown returns the latest Aroon Up and Down values
func (vwao *VolumeWeightedAroonOscillator) GetAroonUpDown() (float64, float64, error) {
	if len(vwao.aroonUp) < 1 || len(vwao.aroonDown) < 1 {
		return 0, 0, errors.New("no Aroon Up/Down data")
	}
	return vwao.aroonUp[len(vwao.aroonUp)-1], vwao.aroonDown[len(vwao.aroonDown)-1], nil
}

// Reset clears all data
func (vwao *VolumeWeightedAroonOscillator) Reset() {
	vwao.highs = vwao.highs[:0]
	vwao.lows = vwao.lows[:0]
	vwao.closes = vwao.closes[:0]
	vwao.volumes = vwao.volumes[:0]
	vwao.vwaoValues = vwao.vwaoValues[:0]
	vwao.aroonUp = vwao.aroonUp[:0]
	vwao.aroonDown = vwao.aroonDown[:0]
	vwao.lastValue = 0
}

// SetPeriod updates the period
func (vwao *VolumeWeightedAroonOscillator) SetPeriod(period int) error {
	if period < 1 {
		return errors.New("period must be at least 1")
	}
	vwao.period = period
	vwao.trimSlices()
	return nil
}

// GetHighs returns a copy of high prices
func (vwao *VolumeWeightedAroonOscillator) GetHighs() []float64 {
	return copySlice(vwao.highs)
}

// GetLows returns a copy of low prices
func (vwao *VolumeWeightedAroonOscillator) GetLows() []float64 {
	return copySlice(vwao.lows)
}

// GetCloses returns a copy of close prices
func (vwao *VolumeWeightedAroonOscillator) GetCloses() []float64 {
	return copySlice(vwao.closes)
}

// GetVolumes returns a copy of volume values
func (vwao *VolumeWeightedAroonOscillator) GetVolumes() []float64 {
	return copySlice(vwao.volumes)
}

// GetVWAOValues returns a copy of VWAO values
func (vwao *VolumeWeightedAroonOscillator) GetVWAOValues() []float64 {
	return copySlice(vwao.vwaoValues)
}

// GetPlotData returns data for visualization with signal annotations
func (vwao *VolumeWeightedAroonOscillator) GetPlotData(startTime, interval int64) []PlotData {
	var plotData []PlotData
	if len(vwao.vwaoValues) > 0 {
		x := make([]float64, len(vwao.vwaoValues))
		signals := make([]float64, len(vwao.vwaoValues))
		timestamps := GenerateTimestamps(startTime, len(vwao.vwaoValues), interval)
		for i := range vwao.vwaoValues {
			x[i] = float64(i)
			if i > 0 {
				if vwao.vwaoValues[i-1] <= 0 && vwao.vwaoValues[i] > 0 {
					signals[i] = 1
				} else if vwao.vwaoValues[i-1] >= 0 && vwao.vwaoValues[i] < 0 {
					signals[i] = -1
				}
			}
			if vwao.vwaoValues[i] > vwao.config.VWAOStrongTrend {
				signals[i] = 2
			} else if vwao.vwaoValues[i] < -vwao.config.VWAOStrongTrend {
				signals[i] = -2
			}
		}
		plotData = append(plotData, PlotData{
			Name:      "Volume-Weighted Aroon Oscillator",
			X:         x,
			Y:         vwao.vwaoValues,
			Type:      "line",
			Timestamp: timestamps,
		})
		plotData = append(plotData, PlotData{
			Name:      "Aroon Up",
			X:         x,
			Y:         vwao.aroonUp,
			Type:      "line",
			Timestamp: timestamps,
		})
		plotData = append(plotData, PlotData{
			Name:      "Aroon Down",
			X:         x,
			Y:         vwao.aroonDown,
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
