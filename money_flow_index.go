// money_flow_index.go
// artifact_id: 5d11be5b-fe59-4f45-a569-109880e077bc
// artifact_version_id: 2d3e4f5a-6b7c-8d9e-0f1a-2b3c4d5e6f7a

package goti

import (
	"errors"
	"fmt"
)

// MoneyFlowIndex calculates the Money Flow Index
type MoneyFlowIndex struct {
	period    int
	highs     []float64
	lows      []float64
	closes    []float64
	volumes   []float64
	mfiValues []float64
	lastValue float64
	config    IndicatorConfig
}

// NewMoneyFlowIndex initializes with standard period (5) and default config
func NewMoneyFlowIndex() (*MoneyFlowIndex, error) {
	return NewMoneyFlowIndexWithParams(5, DefaultConfig())
}

// NewMoneyFlowIndexWithParams initializes with custom period and config
func NewMoneyFlowIndexWithParams(period int, config IndicatorConfig) (*MoneyFlowIndex, error) {
	if period < 1 {
		return nil, errors.New("period must be at least 1")
	}
	if config.MFIOverbought <= config.MFIOversold {
		return nil, errors.New("MFI overbought threshold must be greater than oversold")
	}
	return &MoneyFlowIndex{
		period:    period,
		highs:     make([]float64, 0, period+1),
		lows:      make([]float64, 0, period+1),
		closes:    make([]float64, 0, period+1),
		volumes:   make([]float64, 0, period+1),
		mfiValues: make([]float64, 0, period),
		config:    config,
	}, nil
}

// Add appends new price and volume data
func (mfi *MoneyFlowIndex) Add(high, low, close, volume float64) error {
	if high < low || !isNonNegativePrice(close) || !isValidVolume(volume) {
		return errors.New("invalid price or volume")
	}
	mfi.highs = append(mfi.highs, high)
	mfi.lows = append(mfi.lows, low)
	mfi.closes = append(mfi.closes, close)
	mfi.volumes = append(mfi.volumes, volume)
	if len(mfi.closes) >= mfi.period+1 {
		mfiValue, err := mfi.calculateMFI()
		if err != nil {
			return fmt.Errorf("calculateMFI failed: %w", err)
		}
		mfi.mfiValues = append(mfi.mfiValues, mfiValue)
		mfi.lastValue = mfiValue
	}
	mfi.trimSlices()
	return nil
}

// trimSlices limits slice sizes
func (mfi *MoneyFlowIndex) trimSlices() {
	if len(mfi.closes) > mfi.period+1 {
		mfi.highs = mfi.highs[len(mfi.highs)-mfi.period-1:]
		mfi.lows = mfi.lows[len(mfi.lows)-mfi.period-1:]
		mfi.closes = mfi.closes[len(mfi.closes)-mfi.period-1:]
		mfi.volumes = mfi.volumes[len(mfi.volumes)-mfi.period-1:]
	}
	if len(mfi.mfiValues) > mfi.period {
		mfi.mfiValues = mfi.mfiValues[len(mfi.mfiValues)-mfi.period:]
	}
}

// calculateMFI computes the Money Flow Index value
func (mfi *MoneyFlowIndex) calculateMFI() (float64, error) {
	if len(mfi.closes) < mfi.period+1 {
		return 0, fmt.Errorf("insufficient data: need %d, have %d", mfi.period+1, len(mfi.closes))
	}
	startIdx := len(mfi.closes) - mfi.period - 1
	if startIdx < 0 {
		return 0, fmt.Errorf("invalid start index: %d", startIdx)
	}
	highs := mfi.highs[startIdx:]
	lows := mfi.lows[startIdx:]
	closes := mfi.closes[startIdx:]
	volumes := mfi.volumes[startIdx:]
	if len(highs) < mfi.period || len(lows) < mfi.period || len(closes) < mfi.period || len(volumes) < mfi.period {
		return 0, fmt.Errorf("insufficient slice length for period %d", mfi.period)
	}
	positiveMF, negativeMF := 0.0, 0.0
	for i := 1; i <= mfi.period; i++ {
		if i >= len(closes) || i-1 < 0 {
			return 0, fmt.Errorf("invalid index for MFI calculation: i=%d, len(closes)=%d", i, len(closes))
		}
		typicalPrice := (highs[i] + lows[i] + closes[i]) / 3
		scaledVolume := volumes[i] / 300000 // Adjusted scaling
		rawMoneyFlow := typicalPrice * scaledVolume
		if closes[i] > closes[i-1] {
			positiveMF += rawMoneyFlow
		} else if closes[i] < closes[i-1] {
			negativeMF += rawMoneyFlow
		}
	}
	if positiveMF == 0 && negativeMF == 0 {
		return 50, nil
	}
	if negativeMF == 0 && positiveMF > 0 {
		return 100, nil
	}
	if positiveMF == 0 && negativeMF > 0 {
		return 10, nil
	}
	moneyRatio := positiveMF / negativeMF
	mfiValue := 100 - (100 / (1 + moneyRatio))
	return clamp(mfiValue, 0, 100), nil
}

// Calculate returns the current MFI value
func (mfi *MoneyFlowIndex) Calculate() (float64, error) {
	if len(mfi.mfiValues) == 0 {
		return 0, errors.New("no MFI data")
	}
	return mfi.lastValue, nil
}

// GetLastValue returns the last MFI value
func (mfi *MoneyFlowIndex) GetLastValue() float64 {
	return mfi.lastValue
}

// IsBullishCrossover checks if MFI crosses above oversold threshold
func (mfi *MoneyFlowIndex) IsBullishCrossover() (bool, error) {
	if len(mfi.mfiValues) < 2 {
		return false, errors.New("insufficient data for crossover")
	}
	current := mfi.mfiValues[len(mfi.mfiValues)-1]
	previous := mfi.mfiValues[len(mfi.mfiValues)-2]
	return previous <= mfi.config.MFIOversold && current > mfi.config.MFIOversold, nil
}

// IsBearishCrossover checks if MFI crosses below overbought threshold
func (mfi *MoneyFlowIndex) IsBearishCrossover() (bool, error) {
	if len(mfi.mfiValues) < 2 {
		return false, errors.New("insufficient data for crossover")
	}
	current := mfi.mfiValues[len(mfi.mfiValues)-1]
	previous := mfi.mfiValues[len(mfi.mfiValues)-2]
	return previous >= mfi.config.MFIOverbought && current < mfi.config.MFIOverbought, nil
}

// GetOverboughtOversold returns overbought/oversold status
func (mfi *MoneyFlowIndex) GetOverboughtOversold() (string, error) {
	if len(mfi.mfiValues) < 1 {
		return "", errors.New("no MFI data")
	}
	current := mfi.mfiValues[len(mfi.mfiValues)-1]
	if current > mfi.config.MFIOverbought {
		return "Overbought", nil
	}
	if current < mfi.config.MFIOversold {
		return "Oversold", nil
	}
	return "Neutral", nil
}

// IsDivergence checks for MFI divergence signals
func (mfi *MoneyFlowIndex) IsDivergence() (bool, string, error) {
	if len(mfi.mfiValues) < 2 || len(mfi.closes) < 2 {
		return false, "", errors.New("insufficient data for divergence")
	}
	currentMFI := mfi.mfiValues[len(mfi.mfiValues)-1]
	priceTrend := mfi.closes[len(mfi.closes)-1] - mfi.closes[len(mfi.closes)-2]
	if currentMFI > mfi.config.MFIOverbought && priceTrend < 0 {
		return true, "Bearish", nil
	}
	if currentMFI < mfi.config.MFIOversold && priceTrend > 0 {
		return true, "Bullish", nil
	}
	return false, "", nil
}

// Reset clears all data
func (mfi *MoneyFlowIndex) Reset() {
	mfi.highs = mfi.highs[:0]
	mfi.lows = mfi.lows[:0]
	mfi.closes = mfi.closes[:0]
	mfi.volumes = mfi.volumes[:0]
	mfi.mfiValues = mfi.mfiValues[:0]
	mfi.lastValue = 0
}

// SetPeriod updates the period
func (mfi *MoneyFlowIndex) SetPeriod(period int) error {
	if period < 1 {
		return errors.New("period must be at least 1")
	}
	mfi.period = period
	mfi.trimSlices()
	return nil
}

// GetHighs returns a copy of high prices
func (mfi *MoneyFlowIndex) GetHighs() []float64 {
	return copySlice(mfi.highs)
}

// GetLows returns a copy of low prices
func (mfi *MoneyFlowIndex) GetLows() []float64 {
	return copySlice(mfi.lows)
}

// GetCloses returns a copy of close prices
func (mfi *MoneyFlowIndex) GetCloses() []float64 {
	return copySlice(mfi.closes)
}

// GetVolumes returns a copy of volume values
func (mfi *MoneyFlowIndex) GetVolumes() []float64 {
	return copySlice(mfi.volumes)
}

// GetMFIValues returns a copy of MFI values
func (mfi *MoneyFlowIndex) GetMFIValues() []float64 {
	return copySlice(mfi.mfiValues)
}

// GetPlotData returns data for visualization with signal annotations
func (mfi *MoneyFlowIndex) GetPlotData(startTime, interval int64) []PlotData {
	var plotData []PlotData
	if len(mfi.mfiValues) > 0 {
		x := make([]float64, len(mfi.mfiValues))
		signals := make([]float64, len(mfi.mfiValues))
		timestamps := GenerateTimestamps(startTime, len(mfi.mfiValues), interval)
		for i := range mfi.mfiValues {
			x[i] = float64(i)
			if i > 0 {
				if mfi.mfiValues[i-1] <= mfi.config.MFIOversold && mfi.mfiValues[i] > mfi.config.MFIOversold {
					signals[i] = 1
				} else if mfi.mfiValues[i-1] >= mfi.config.MFIOverbought && mfi.mfiValues[i] < mfi.config.MFIOverbought {
					signals[i] = -1
				}
			}
			if mfi.mfiValues[i] > mfi.config.MFIOverbought {
				signals[i] = 2
			} else if mfi.mfiValues[i] < mfi.config.MFIOversold {
				signals[i] = -2
			}
		}
		plotData = append(plotData, PlotData{
			Name:      "Money Flow Index",
			X:         x,
			Y:         mfi.mfiValues,
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
