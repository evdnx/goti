package volatility

import (
	"errors"
	"math"

	"github.com/evdnx/goti/indicator/core"
)

const (
	DefaultBollingerPeriod     = 20
	DefaultBollingerMultiplier = 2.0
)

// BollingerBands calculates upper/middle/lower bands based on a moving average
// and standard deviation of closing prices.
type BollingerBands struct {
	period     int
	multiplier float64

	closes []float64
	upper  []float64
	middle []float64
	lower  []float64

	runningSum   float64
	runningSumSq float64
	sumComp      float64 // Kahan compensation for runningSum
	sumSqComp    float64 // Kahan compensation for runningSumSq
	lastUpper    float64
	lastMiddle   float64
	lastLower    float64
}

// NewBollingerBands creates a Bollinger Bands calculator with default settings.
func NewBollingerBands() (*BollingerBands, error) {
	return NewBollingerBandsWithParams(DefaultBollingerPeriod, DefaultBollingerMultiplier)
}

// NewBollingerBandsWithParams creates a Bollinger Bands calculator with custom
// period and multiplier.
func NewBollingerBandsWithParams(period int, multiplier float64) (*BollingerBands, error) {
	if period < 1 {
		return nil, errors.New("period must be at least 1")
	}
	if multiplier <= 0 {
		return nil, errors.New("multiplier must be positive")
	}
	return &BollingerBands{
		period:     period,
		multiplier: multiplier,
		closes:     make([]float64, 0, period),
		upper:      make([]float64, 0, period),
		middle:     make([]float64, 0, period),
		lower:      make([]float64, 0, period),
	}, nil
}

// Add appends a new closing price and updates the bands when enough data is
// present.
func (b *BollingerBands) Add(close float64) error {
	if !core.IsNonNegativePrice(close) {
		return errors.New("invalid price")
	}
	b.closes = append(b.closes, close)
	b.kahanAdd(close)
	b.kahanAddSq(close)

	// Maintain a fixed-size window so updates are O(1).
	if len(b.closes) > b.period {
		removed := b.closes[0]
		b.closes = b.closes[1:]
		b.kahanAdd(-removed)
		b.kahanAddSq(-removed)
	}

	if len(b.closes) >= b.period {
		mean := b.runningSum / float64(b.period)

		std := 0.0
		if b.period > 1 {
			variance := (b.runningSumSq - (b.runningSum*b.runningSum)/float64(b.period)) / float64(b.period-1)
			if variance < 0 {
				variance = 0 // guard against negative zero/rounding
			}
			std = math.Sqrt(variance)
		}

		upper := mean + b.multiplier*std
		lower := mean - b.multiplier*std

		b.lastMiddle = mean
		b.lastUpper = upper
		b.lastLower = lower

		b.upper = append(b.upper, upper)
		b.middle = append(b.middle, mean)
		b.lower = append(b.lower, lower)
	}

	b.trimSlices()
	return nil
}

// Calculate returns the most recent upper, middle, and lower band values.
func (b *BollingerBands) Calculate() (float64, float64, float64, error) {
	if len(b.upper) == 0 {
		return 0, 0, 0, errors.New("no Bollinger Bands data")
	}
	return b.lastUpper, b.lastMiddle, b.lastLower, nil
}

// Reset clears all stored data.
func (b *BollingerBands) Reset() {
	b.closes = b.closes[:0]
	b.upper = b.upper[:0]
	b.middle = b.middle[:0]
	b.lower = b.lower[:0]
	b.runningSum = 0
	b.runningSumSq = 0
	b.sumComp = 0
	b.sumSqComp = 0
	b.lastUpper, b.lastMiddle, b.lastLower = 0, 0, 0
}

// SetParams updates period and multiplier and resets internal state.
func (b *BollingerBands) SetParams(period int, multiplier float64) error {
	if period < 1 {
		return errors.New("period must be at least 1")
	}
	if multiplier <= 0 {
		return errors.New("multiplier must be positive")
	}
	b.period = period
	b.multiplier = multiplier
	b.Reset()
	return nil
}

// GetUpper returns a defensive copy of the upper band values.
func (b *BollingerBands) GetUpper() []float64 { return core.CopySlice(b.upper) }

// GetMiddle returns a defensive copy of the middle band values.
func (b *BollingerBands) GetMiddle() []float64 { return core.CopySlice(b.middle) }

// GetLower returns a defensive copy of the lower band values.
func (b *BollingerBands) GetLower() []float64 { return core.CopySlice(b.lower) }

// GetPlotData emits plot data for the upper/middle/lower bands.
func (b *BollingerBands) GetPlotData(startTime, interval int64) []core.PlotData {
	if len(b.upper) == 0 {
		return nil
	}
	x := make([]float64, len(b.upper))
	for i := range x {
		x[i] = float64(i)
	}
	ts := core.GenerateTimestamps(startTime, len(b.upper), interval)

	return []core.PlotData{
		{Name: "Bollinger Upper", X: x, Y: core.CopySlice(b.upper), Type: "line", Timestamp: ts},
		{Name: "Bollinger Middle", X: x, Y: core.CopySlice(b.middle), Type: "line", Timestamp: ts},
		{Name: "Bollinger Lower", X: x, Y: core.CopySlice(b.lower), Type: "line", Timestamp: ts},
	}
}

func (b *BollingerBands) trimSlices() {
	b.closes = core.KeepLast(b.closes, b.period)
	maxKeep := b.period
	b.upper = core.KeepLast(b.upper, maxKeep)
	b.middle = core.KeepLast(b.middle, maxKeep)
	b.lower = core.KeepLast(b.lower, maxKeep)
}

// Kahan compensated addition for runningSum.
func (b *BollingerBands) kahanAdd(v float64) {
	y := v - b.sumComp
	t := b.runningSum + y
	b.sumComp = (t - b.runningSum) - y
	b.runningSum = t
}

// Kahan compensated addition for runningSumSq.
func (b *BollingerBands) kahanAddSq(v float64) {
	y := v*v - b.sumSqComp
	t := b.runningSumSq + y
	b.sumSqComp = (t - b.runningSumSq) - y
	b.runningSumSq = t
}
