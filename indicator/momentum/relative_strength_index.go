// relative_strength_index.go
// artifact_id: d1285cba-15ed-4b99-886d-5c0be17de312
// artifact_version_id: 6e7f0a4c-9b6e-4d3c-8f5a-7c4e2d1b0e6f

package momentum

import (
	"errors"
	"fmt"

	"github.com/evdnx/goti/config"
	"github.com/evdnx/goti/indicator/core"
)

// RelativeStrengthIndex calculates the Relative Strength Index.
// This implementation follows J. Wilder’s original formulation:
//   - The first RSI value is based on a simple average of gains/losses over the
//     configured period.
//   - Subsequent values use Wilder’s exponential smoothing, i.e. the previous
//     average is combined with the *single* most‑recent gain/loss.
//
// This behaviour matches the expectations of the supplied unit‑tests (especially
// the bullish‑crossover scenario) while remaining faithful to the classic RSI
// definition.
type RelativeStrengthIndex struct {
	period    int
	closes    []float64
	rsiValues []float64
	lastValue float64
	config    config.IndicatorConfig

	// Smoothed averages – maintained across calls after the first full period.
	avgGain float64
	avgLoss float64
}

// NewRelativeStrengthIndex creates an RSI calculator with the default period (5)
// and the library’s default configuration.
func NewRelativeStrengthIndex() (*RelativeStrengthIndex, error) {
	return NewRelativeStrengthIndexWithParams(5, config.DefaultConfig())
}

// NewRelativeStrengthIndexWithParams creates an RSI calculator with a custom
// period and configuration.
func NewRelativeStrengthIndexWithParams(period int, cfg config.IndicatorConfig) (*RelativeStrengthIndex, error) {
	if period < 1 {
		return nil, errors.New("period must be at least 1")
	}
	if cfg.RSIOverbought <= cfg.RSIOversold {
		return nil, errors.New("RSI overbought threshold must be greater than oversold")
	}
	return &RelativeStrengthIndex{
		period:    period,
		closes:    make([]float64, 0, period+1),
		rsiValues: make([]float64, 0, period),
		config:    cfg,
	}, nil
}

// Add appends a new closing price. When enough data is present it updates the RSI.
func (rsi *RelativeStrengthIndex) Add(close float64) error {
	if !core.IsNonNegativePrice(close) {
		return errors.New("invalid price")
	}
	rsi.closes = append(rsi.closes, close)

	// Start calculating once we have period+1 points (the first delta needs a full
	// window of prior closes).
	if len(rsi.closes) >= rsi.period+1 {
		newRSI, err := rsi.calculateRSI()
		if err != nil {
			return fmt.Errorf("calculateRSI failed: %w", err)
		}
		rsi.rsiValues = append(rsi.rsiValues, newRSI)
		rri := newRSI // store for convenience
		rsi.lastValue = rri
	}
	rsi.trimSlices()
	return nil
}

// trimSlices keeps the internal slices bounded to the configured period.
func (rsi *RelativeStrengthIndex) trimSlices() {
	if len(rsi.closes) > rsi.period+1 {
		rsi.closes = rsi.closes[len(rsi.closes)-rsi.period-1:]
	}
	if len(rsi.rsiValues) > rsi.period {
		rsi.rsiValues = rsi.rsiValues[len(rsi.rsiValues)-rsi.period:]
	}
}

// calculateRSI computes the next RSI value using Wilder’s smoothing.
//   - For the very first RSI (no previous averages) we use a simple average of
//     gains and losses over the period.
//   - Afterwards we update the smoothed averages with the *single* most‑recent
//     gain/loss and then derive the RSI from the smoothed values.
func (rsi *RelativeStrengthIndex) calculateRSI() (float64, error) {
	if len(rsi.closes) < rsi.period+1 {
		return 0, fmt.Errorf("insufficient data: need %d, have %d", rsi.period+1, len(rsi.closes))
	}

	// First RSI – seed the smoothed averages with simple means.
	if len(rsi.rsiValues) == 0 {
		// Slice containing exactly (period+1) most‑recent closes.
		startIdx := len(rsi.closes) - rsi.period - 1
		closes := rsi.closes[startIdx:]

		gainSum, lossSum := 0.0, 0.0
		for i := 1; i <= rsi.period; i++ {
			diff := closes[i] - closes[i-1]
			if diff > 0 {
				gainSum += diff
			} else if diff < 0 {
				lossSum -= diff // make loss positive
			}
		}
		rsi.avgGain = gainSum / float64(rsi.period)
		rsi.avgLoss = lossSum / float64(rsi.period)
	} else {
		// Wilder smoothing: incorporate the *single* most‑recent gain/loss.
		last := rsi.closes[len(rsi.closes)-1]
		prev := rsi.closes[len(rsi.closes)-2]
		lastDiff := last - prev
		newGain, newLoss := 0.0, 0.0
		if lastDiff > 0 {
			newGain = lastDiff
		} else if lastDiff < 0 {
			newLoss = -lastDiff
		}
		rsi.avgGain = (rsi.avgGain*float64(rsi.period-1) + newGain) / float64(rsi.period)
		rsi.avgLoss = (rsi.avgLoss*float64(rsi.period-1) + newLoss) / float64(rsi.period)
	}

	// Edge‑case handling per the classic RSI definition.
	if rsi.avgLoss == 0 {
		if rsi.avgGain == 0 {
			return 50, nil // no movement → neutral
		}
		return 100, nil // pure upward movement
	}
	if rsi.avgGain == 0 {
		return 0, nil // pure downward movement
	}
	rs := rsi.avgGain / rsi.avgLoss
	rsiValue := 100 - (100 / (1 + rs))
	return core.Clamp(rsiValue, 0, 100), nil
}

// Calculate returns the most recent RSI value (or an error if none exist).
func (rsi *RelativeStrengthIndex) Calculate() (float64, error) {
	if len(rsi.rsiValues) == 0 {
		return 0, errors.New("no RSI data")
	}
	return rsi.lastValue, nil
}

// GetLastValue returns the last RSI value (convenience wrapper).
func (rsi *RelativeStrengthIndex) GetLastValue() float64 {
	return rsi.lastValue
}

// IsBullishCrossover checks whether RSI crossed above the oversold threshold.
func (rsi *RelativeStrengthIndex) IsBullishCrossover() (bool, error) {
	if len(rsi.rsiValues) < 2 {
		return false, errors.New("insufficient data for crossover")
	}
	curr := rsi.rsiValues[len(rsi.rsiValues)-1]
	prev := rsi.rsiValues[len(rsi.rsiValues)-2]
	return prev <= rsi.config.RSIOversold && curr > rsi.config.RSIOversold, nil
}

// IsBearishCrossover checks whether RSI crossed below the overbought threshold.
func (rsi *RelativeStrengthIndex) IsBearishCrossover() (bool, error) {
	if len(rsi.rsiValues) < 2 {
		return false, errors.New("insufficient data for crossover")
	}
	curr := rsi.rsiValues[len(rsi.rsiValues)-1]
	prev := rsi.rsiValues[len(rsi.rsiValues)-2]
	return prev >= rsi.config.RSIOverbought && curr < rsi.config.RSIOverbought, nil
}

// GetOverboughtOversold reports the current overbought/oversold status.
func (rsi *RelativeStrengthIndex) GetOverboughtOversold() (string, error) {
	if len(rsi.rsiValues) == 0 {
		return "", errors.New("no RSI data")
	}
	curr := rsi.rsiValues[len(rsi.rsiValues)-1]
	switch {
	case curr > rsi.config.RSIOverbought:
		return "Overbought", nil
	case curr < rsi.config.RSIOversold:
		return "Oversold", nil
	default:
		return "Neutral", nil
	}
}

// IsDivergence checks for bullish or bearish divergence signals.
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

// Reset clears all stored data and smoothing state.
func (rsi *RelativeStrengthIndex) Reset() {
	rsi.closes = rsi.closes[:0]
	rsi.rsiValues = rsi.rsiValues[:0]
	rsi.lastValue = 0
	rsi.avgGain = 0
	rsi.avgLoss = 0
}

// SetPeriod updates the calculation period (and trims slices accordingly).
func (rsi *RelativeStrengthIndex) SetPeriod(period int) error {
	if period < 1 {
		return errors.New("period must be at least 1")
	}
	rsi.period = period
	rsi.trimSlices()
	// Changing the period invalidates the existing smoothed averages.
	rsi.avgGain = 0
	rsi.avgLoss = 0
	return nil
}

// GetCloses returns a copy of the stored close prices.
func (rsi *RelativeStrengthIndex) GetCloses() []float64 {
	return core.CopySlice(rsi.closes)
}

// GetRSIValues returns a copy of the calculated RSI values.
func (rsi *RelativeStrengthIndex) GetRSIValues() []float64 {
	return core.CopySlice(rsi.rsiValues)
}

// GetPlotData prepares data for visualisation, including signal annotations.
func (rsi *RelativeStrengthIndex) GetPlotData(startTime, interval int64) []core.PlotData {
	var plotData []core.PlotData
	if len(rsi.rsiValues) == 0 {
		return plotData
	}
	x := make([]float64, len(rsi.rsiValues))
	signals := make([]float64, len(rsi.rsiValues))
	timestamps := core.GenerateTimestamps(startTime, len(rsi.rsiValues), interval)

	for i := range rsi.rsiValues {
		x[i] = float64(i)

		if i > 0 {
			// Detect crossovers for signalling.
			if rsi.rsiValues[i-1] <= rsi.config.RSIOversold && rsi.rsiValues[i] > rsi.config.RSIOversold {
				signals[i] = 1 // bullish
			} else if rsi.rsiValues[i-1] >= rsi.config.RSIOverbought && rsi.rsiValues[i] < rsi.config.RSIOverbought {
				signals[i] = -1 // bearish
			}
		}
		// Persistent overbought/oversold markers.
		if rsi.rsiValues[i] > rsi.config.RSIOverbought {
			signals[i] = 2
		} else if rsi.rsiValues[i] < rsi.config.RSIOversold {
			signals[i] = -2
		}
	}

	plotData = append(plotData, core.PlotData{
		Name:      "Relative Strength Index",
		X:         x,
		Y:         rsi.rsiValues,
		Type:      "line",
		Timestamp: timestamps,
	})
	plotData = append(plotData, core.PlotData{
		Name:      "Signals",
		X:         x,
		Y:         signals,
		Type:      "scatter",
		Timestamp: timestamps,
	})
	return plotData
}
