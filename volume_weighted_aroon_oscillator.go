// volume_weighted_aroon_oscillator.go
// artifact_id: 7a8b9c2d-3e4f-5a6b-7c8d-9e0f1a2b3c4d
// artifact_version_id: 6e9f0b2c-3d4e-4f5a-6b7c-8d9e0f1a2b3c

package goti

import (
	"errors"
	"fmt"
)

// VolumeWeightedAroonOscillator calculates the Volume-Weighted Aroon Oscillator
type VolumeWeightedAroonOscillator struct {
	period     int
	highs      []float64
	lows       []float64
	closes     []float64
	volumes    []float64
	vwaoValues []float64
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
		config:     config,
	}, nil
}

// Add appends new price and volume data to the oscillator
func (vwao *VolumeWeightedAroonOscillator) Add(high, low, close, volume float64) error {
	if high < low || !isNonNegativePrice(close) || !isValidVolume(volume) {
		return errors.New("invalid price or volume")
	}
	vwao.highs = append(vwao.highs, high)
	vwao.lows = append(vwao.lows, low)
	vwao.closes = append(vwao.closes, close)
	vwao.volumes = append(vwao.volumes, volume)
	if len(vwao.closes) >= vwao.period+1 {
		vwaoValue, err := vwao.calculateVWAO()
		if err != nil {
			return fmt.Errorf("calculateVWAO failed: %w", err)
		}
		vwao.vwaoValues = append(vwao.vwaoValues, vwaoValue)
		vwao.lastValue = vwaoValue
	}
	vwao.trimSlices()
	return nil
}

// trimSlices limits the size of data slices to prevent memory growth
func (vwao *VolumeWeightedAroonOscillator) trimSlices() {
	if len(vwao.closes) > vwao.period+1 {
		vwao.highs = vwao.highs[len(vwao.highs)-vwao.period-1:]
		vwao.lows = vwao.lows[len(vwao.lows)-vwao.period-1:]
		vwao.closes = vwao.closes[len(vwao.closes)-vwao.period-1:]
		vwao.volumes = vwao.volumes[len(vwao.volumes)-vwao.period-1:]
	}
	if len(vwao.vwaoValues) > vwao.period {
		vwao.vwaoValues = vwao.vwaoValues[len(vwao.vwaoValues)-vwao.period:]
	}
}

// calculateVWAO computes the Volume-Weighted Aroon Oscillator value
func (vwao *VolumeWeightedAroonOscillator) calculateVWAO() (float64, error) {
	if len(vwao.closes) < vwao.period+1 {
		return 0, fmt.Errorf("insufficient data: need %d, have %d", vwao.period+1, len(vwao.closes))
	}
	startIdx := len(vwao.closes) - vwao.period - 1
	if startIdx < 0 {
		return 0, fmt.Errorf("invalid start index: %d", startIdx)
	}
	highs := vwao.highs[startIdx:]
	lows := vwao.lows[startIdx:]
	volumes := vwao.volumes[startIdx:]
	if len(highs) < vwao.period || len(lows) < vwao.period || len(volumes) < vwao.period {
		return 0, fmt.Errorf("insufficient slice length for period %d", vwao.period)
	}

	maxHigh, minLow := highs[0], lows[0]
	maxHighIdx, minLowIdx := 0, 0
	totalVolume := 0.0
	weightedHighPeriods, weightedLowPeriods := 0.0, 0.0

	for i := 0; i <= vwao.period; i++ {
		if i >= len(highs) || i >= len(lows) || i >= len(volumes) {
			return 0, fmt.Errorf("invalid index for VWAO calculation: i=%d", i)
		}
		totalVolume += volumes[i]
		if highs[i] >= maxHigh {
			maxHigh = highs[i]
			maxHighIdx = i
		}
		if lows[i] <= minLow {
			minLow = lows[i]
			minLowIdx = i
		}
		weightedHighPeriods += float64(vwao.period-i) * volumes[i]
		weightedLowPeriods += float64(vwao.period-i) * volumes[i]
	}

	if totalVolume == 0 {
		return 0, errors.New("total volume is zero")
	}

	aroonUp := (float64(vwao.period-maxHighIdx) / float64(vwao.period)) * 100
	aroonDown := (float64(vwao.period-minLowIdx) / float64(vwao.period)) * 100
	vwaoValue := aroonUp - aroonDown
	return clamp(vwaoValue, -100, 100), nil
}

// Calculate returns the current VWAO value
func (vwao *VolumeWeightedAroonOscillator) Calculate() (float64, error) {
	if len(vwao.vwaoValues) == 0 {
		return 0, errors.New("no VWAO data")
	}
	return vwao.lastValue, nil
}

// GetLastValue returns the last calculated VWAO value
func (vwao *VolumeWeightedAroonOscillator) GetLastValue() float64 {
	return vwao.lastValue
}

// IsBullishCrossover checks if VWAO crosses above the strong trend threshold
func (vwao *VolumeWeightedAroonOscillator) IsBullishCrossover() (bool, error) {
	if len(vwao.vwaoValues) < 2 {
		return false, errors.New("insufficient data for crossover")
	}
	current := vwao.vwaoValues[len(vwao.vwaoValues)-1]
	previous := vwao.vwaoValues[len(vwao.vwaoValues)-2]
	return previous <= vwao.config.VWAOStrongTrend && current > vwao.config.VWAOStrongTrend, nil
}

// IsBearishCrossover checks if VWAO crosses below the negative strong trend threshold
func (vwao *VolumeWeightedAroonOscillator) IsBearishCrossover() (bool, error) {
	if len(vwao.vwaoValues) < 2 {
		return false, errors.New("insufficient data for crossover")
	}
	current := vwao.vwaoValues[len(vwao.vwaoValues)-1]
	previous := vwao.vwaoValues[len(vwao.vwaoValues)-2]
	return previous >= -vwao.config.VWAOStrongTrend && current < -vwao.config.VWAOStrongTrend, nil
}

// IsStrongTrend checks if VWAO indicates a strong trend
func (vwao *VolumeWeightedAroonOscillator) IsStrongTrend() (bool, error) {
	if len(vwao.vwaoValues) < 1 {
		return false, errors.New("no VWAO data")
	}
	current := vwao.vwaoValues[len(vwao.vwaoValues)-1]
	return current > vwao.config.VWAOStrongTrend || current < -vwao.config.VWAOStrongTrend, nil
}

// IsDivergence checks for VWAO divergence signals
func (vwao *VolumeWeightedAroonOscillator) IsDivergence() (bool, string, error) {
	if len(vwao.vwaoValues) < 2 || len(vwao.closes) < 2 {
		return false, "", errors.New("insufficient data for divergence")
	}
	currentVWAO := vwao.vwaoValues[len(vwao.vwaoValues)-1]
	priceTrend := vwao.closes[len(vwao.closes)-1] - vwao.closes[len(vwao.closes)-2]
	if currentVWAO > vwao.config.VWAOStrongTrend && priceTrend < 0 {
		return true, "Bearish", nil
	}
	if currentVWAO < -vwao.config.VWAOStrongTrend && priceTrend > 0 {
		return true, "Bullish", nil
	}
	return false, "", nil
}

// Reset clears all stored data
func (vwao *VolumeWeightedAroonOscillator) Reset() {
	vwao.highs = vwao.highs[:0]
	vwao.lows = vwao.lows[:0]
	vwao.closes = vwao.closes[:0]
	vwao.volumes = vwao.volumes[:0]
	vwao.vwaoValues = vwao.vwaoValues[:0]
	vwao.lastValue = 0
}

// SetPeriod updates the period for VWAO calculations
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
				if vwao.vwaoValues[i-1] <= vwao.config.VWAOStrongTrend && vwao.vwaoValues[i] > vwao.config.VWAOStrongTrend {
					signals[i] = 1
				} else if vwao.vwaoValues[i-1] >= -vwao.config.VWAOStrongTrend && vwao.vwaoValues[i] < -vwao.config.VWAOStrongTrend {
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
			Name:      "Volume Weighted Aroon Oscillator",
			X:         x,
			Y:         vwao.vwaoValues,
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
