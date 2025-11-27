package volatility

import (
	"errors"
	"fmt"
	"math"

	"github.com/evdnx/goti/indicator/core"
)

// AverageTrueRange calculates the Average True Range (ATR).
type AverageTrueRange struct {
	period        int
	highs         []float64
	lows          []float64
	closes        []float64
	atrValues     []float64
	lastValue     float64
	validateClose bool // optional validation of close price against high/low
}

/*
   Constructors
   ------------

   NewAverageTrueRange creates an ATR calculator with the default period (14).

   NewAverageTrueRangeWithParams creates an ATR calculator with a custom period.
   Functional options can be supplied to tweak behaviour (e.g. disabling the
   close‑price validation).
*/

// NewAverageTrueRange creates an ATR calculator with the default period (14).
func NewAverageTrueRange() (*AverageTrueRange, error) {
	return NewAverageTrueRangeWithParams(14)
}

// NewAverageTrueRangeWithParams creates an ATR calculator with a custom period.
// Additional functional options can be passed to modify the instance.
func NewAverageTrueRangeWithParams(period int, opts ...ATROption) (*AverageTrueRange, error) {
	if period < 1 {
		return nil, errors.New("period must be at least 1")
	}
	atr := &AverageTrueRange{
		period:        period,
		highs:         make([]float64, 0, period+1),
		lows:          make([]float64, 0, period+1),
		closes:        make([]float64, 0, period+1),
		atrValues:     make([]float64, 0, period),
		validateClose: true, // enabled by default
	}
	for _, opt := range opts {
		opt(atr)
	}
	return atr, nil
}

/* ---------- Functional options ---------- */

// ATROption configures an AverageTrueRange instance.
type ATROption func(*AverageTrueRange)

// WithCloseValidation enables or disables the check that the close price lies
// between the high and low of the same candle.
func WithCloseValidation(enabled bool) ATROption {
	return func(a *AverageTrueRange) { a.validateClose = enabled }
}

/* ---------- Public API ---------- */

// AddCandle appends a new OHLC data point.
// It validates the inputs and, when enough data is present, updates the ATR series.
func (atr *AverageTrueRange) AddCandle(high, low, close float64) error {
	if high < low {
		return errors.New("high must be >= low")
	}
	if !core.IsValidPrice(high) || !core.IsValidPrice(low) {
		return errors.New("high/low contain invalid price")
	}
	if atr.validateClose && (close < low || close > high) {
		return fmt.Errorf("close price %.4f out of bounds [%.4f, %.4f]", close, low, high)
	}
	if !core.IsValidPrice(close) {
		return errors.New("invalid close price")
	}

	atr.highs = append(atr.highs, high)
	atr.lows = append(atr.lows, low)
	atr.closes = append(atr.closes, close)

	// Compute ATR once we have period+1 closing prices.
	if len(atr.closes) >= atr.period+1 {
		val, err := atr.calculateATR()
		if err != nil {
			return err
		}
		atr.atrValues = append(atr.atrValues, val)
		atr.lastValue = val
	}
	atr.trimSlices()
	return nil
}

// Calculate returns the most recent ATR value.
// An error is returned if the series has not yet produced any output.
func (atr *AverageTrueRange) Calculate() (float64, error) {
	if len(atr.atrValues) == 0 {
		return 0, fmt.Errorf("ATR not ready – need at least %d data points", atr.period+1)
	}
	return atr.lastValue, nil
}

// Reset clears all stored data and starts fresh.
func (atr *AverageTrueRange) Reset() {
	atr.highs = atr.highs[:0]
	atr.lows = atr.lows[:0]
	atr.closes = atr.closes[:0]
	atr.atrValues = atr.atrValues[:0]
	atr.lastValue = 0
}

// SetPeriod changes the look‑back period. All historic data is discarded because
// the previous window no longer aligns with the new period.
func (atr *AverageTrueRange) SetPeriod(period int) error {
	if period < 1 {
		return errors.New("period must be at least 1")
	}
	atr.period = period
	atr.Reset()
	return nil
}

/* ---------- Internal helpers ---------- */

// trimSlices ensures the internal slices never exceed the configured window.
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

// trueRange computes the true‑range for a given index (index refers to the
// position inside the internal slices, not the original data stream).
func (atr *AverageTrueRange) trueRange(idx int) float64 {
	highLow := atr.highs[idx] - atr.lows[idx]
	highPrevClose := math.Abs(atr.highs[idx] - atr.closes[idx-1])
	lowPrevClose := math.Abs(atr.lows[idx] - atr.closes[idx-1])
	return math.Max(highLow, math.Max(highPrevClose, lowPrevClose))
}

// calculateATR aggregates the true‑range over the configured period and returns
// the average.
func (atr *AverageTrueRange) calculateATR() (float64, error) {
	if len(atr.closes) < atr.period+1 {
		return 0, fmt.Errorf("insufficient data: need %d, have %d", atr.period+1, len(atr.closes))
	}
	start := len(atr.closes) - atr.period
	var sumTR float64
	for i := start; i < len(atr.closes); i++ {
		sumTR += atr.trueRange(i)
	}
	return sumTR / float64(atr.period), nil
}

/* ---------- Optional getters (defensive copies) ---------- */

func (atr *AverageTrueRange) GetATRValues() []float64 { return core.CopySlice(atr.atrValues) }
func (atr *AverageTrueRange) GetHighs() []float64     { return core.CopySlice(atr.highs) }
func (atr *AverageTrueRange) GetLows() []float64      { return core.CopySlice(atr.lows) }
func (atr *AverageTrueRange) GetCloses() []float64    { return core.CopySlice(atr.closes) }
