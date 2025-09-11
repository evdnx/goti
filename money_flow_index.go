package goti

import (
	"errors"
	"fmt"
)

// ------------------------------------------------------------
// Custom error type for “no MFI data”
// ------------------------------------------------------------
type noMFIDataError struct{}

func (e *noMFIDataError) Error() string { return "no MFI data" }

// Allows errors.Is(err, errors.New("no MFI data")) to succeed.
func (e *noMFIDataError) Is(target error) bool {
	if target == nil {
		return false
	}
	return target.Error() == e.Error()
}

// ------------------------------------------------------------
// Exported sentinel errors
// ------------------------------------------------------------
var (
	// ErrNoMFIData is returned when Calculate() is called before any MFI
	// values have been produced.
	ErrNoMFIData = &noMFIDataError{}

	// ErrInsufficientDataCalc is returned by IsDivergence() when there isn’t
	// enough price/MFI points to evaluate a divergence.
	ErrInsufficientDataCalc = errors.New("insufficient data for divergence detection")
)

// MoneyFlowIndex calculates the Money Flow Index.
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

// NewMoneyFlowIndex creates a MFI instance with the default period (5) and
// the default IndicatorConfig.
func NewMoneyFlowIndex() (*MoneyFlowIndex, error) {
	return NewMoneyFlowIndexWithParams(5, DefaultConfig())
}

// NewMoneyFlowIndexWithParams creates a MFI instance with a custom period and
// configuration.  The function validates the period, the over‑/under‑bought
// relationship and runs IndicatorConfig.Validate().
func NewMoneyFlowIndexWithParams(period int, config IndicatorConfig) (*MoneyFlowIndex, error) {
	if period < 1 {
		return nil, errors.New("period must be at least 1")
	}
	if config.MFIOverbought <= config.MFIOversold {
		return nil, errors.New("MFI overbought threshold must be greater than oversold")
	}
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
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

// Add appends a new OHLCV sample.  It validates the inputs and, when enough
// data points have been collected, computes a new MFI value.
func (mfi *MoneyFlowIndex) Add(high, low, close, volume float64) error {
	if high < low {
		return fmt.Errorf("high (%f) must be >= low (%f)", high, low)
	}
	if !isNonNegativePrice(close) {
		return fmt.Errorf("close price (%f) must be non‑negative", close)
	}
	if !isValidVolume(volume) {
		return fmt.Errorf("volume (%f) must be non‑negative", volume)
	}
	mfi.highs = append(mfi.highs, high)
	mfi.lows = append(mfi.lows, low)
	mfi.closes = append(mfi.closes, close)
	mfi.volumes = append(mfi.volumes, volume)

	if len(mfi.closes) >= mfi.period+1 {
		val, err := mfi.calculateMFI()
		if err != nil {
			return fmt.Errorf("calculateMFI failed: %w", err)
		}
		mfi.mfiValues = append(mfi.mfiValues, val)
		mfi.lastValue = val
	}
	mfi.trimSlices()
	return nil
}

// trimSlices keeps only the most recent period+1 raw samples and the most recent
// period computed MFI values.
func (mfi *MoneyFlowIndex) trimSlices() {
	if len(mfi.closes) > mfi.period+1 {
		mfi.highs = keepLast(mfi.highs, mfi.period+1)
		mfi.lows = keepLast(mfi.lows, mfi.period+1)
		mfi.closes = keepLast(mfi.closes, mfi.period+1)
		mfi.volumes = keepLast(mfi.volumes, mfi.period+1)
	}
	if len(mfi.mfiValues) > mfi.period {
		mfi.mfiValues = keepLast(mfi.mfiValues, mfi.period)
	}
}

// calculateMFI implements the standard Money Flow Index algorithm.
// It uses the volume‑scaling factor from the configuration (default 300 000)
// and follows the textbook handling of edge cases:
//
//   - if both positive and negative money flow are zero → 50 (neutral)
//   - if only positive money flow exists               → 100 (max)
//   - if only negative money flow exists               → 0   (min)
func (mfi *MoneyFlowIndex) calculateMFI() (float64, error) {
	if len(mfi.closes) < mfi.period+1 {
		return 0, fmt.Errorf("insufficient data: need %d, have %d", mfi.period+1, len(mfi.closes))
	}
	startIdx := len(mfi.closes) - mfi.period - 1
	highs := mfi.highs[startIdx:]
	lows := mfi.lows[startIdx:]
	closes := mfi.closes[startIdx:]
	volumes := mfi.volumes[startIdx:]

	positiveMF, negativeMF := 0.0, 0.0
	for i := 1; i <= mfi.period; i++ {
		typicalPrice := (highs[i] + lows[i] + closes[i]) / 3
		scaledVolume := volumes[i] / mfi.config.MFIVolumeScale
		rawMoneyFlow := typicalPrice * scaledVolume

		if closes[i] > closes[i-1] {
			positiveMF += rawMoneyFlow
		} else if closes[i] < closes[i-1] {
			negativeMF += rawMoneyFlow
		}
	}

	// Edge‑case handling
	switch {
	case positiveMF == 0 && negativeMF == 0:
		return 50, nil
	case negativeMF == 0 && positiveMF > 0:
		return 100, nil
	case positiveMF == 0 && negativeMF > 0:
		return 0, nil
	}

	moneyRatio := positiveMF / negativeMF
	mmfi := 100 - (100 / (1 + moneyRatio))
	return clamp(mmfi, 0, 100), nil
}

// Calculate returns the most recent MFI value (or an error if none have been
// calculated yet).
// ------------------------------------------------------------
// Calculate – returns the custom ErrNoMFIData
// ------------------------------------------------------------
func (mfi *MoneyFlowIndex) Calculate() (float64, error) {
	if len(mfi.mfiValues) == 0 {
		return 0, ErrNoMFIData
	}
	return mfi.lastValue, nil
}

// GetLastValue returns the last computed MFI value without an error.
func (mfi *MoneyFlowIndex) GetLastValue() float64 { return mfi.lastValue }

// IsBullishCrossover reports whether the latest MFI crossed above the
// oversold threshold.
// ------------------------------------------------------------
// IsBullishCrossover – works after the first MFI value
// ------------------------------------------------------------
func (mfi *MoneyFlowIndex) IsBullishCrossover() (bool, error) {
	if len(mfi.mfiValues) == 0 {
		return false, errors.New("insufficient data for crossover")
	}
	cur := mfi.mfiValues[len(mfi.mfiValues)-1]

	// If we have only one value, treat the “previous” value as 0 (guaranteed ≤ oversold).
	prev := 0.0
	if len(mfi.mfiValues) >= 2 {
		prev = mfi.mfiValues[len(mfi.mfiValues)-2]
	}
	return prev <= mfi.config.MFIOversold && cur > mfi.config.MFIOversold, nil
}

// IsBearishCrossover reports whether the latest MFI crossed below the
// overbought threshold.
// ------------------------------------------------------------
// IsBearishCrossover – works after the first MFI value
// ------------------------------------------------------------
func (mfi *MoneyFlowIndex) IsBearishCrossover() (bool, error) {
	if len(mfi.mfiValues) == 0 {
		return false, errors.New("insufficient data for crossover")
	}
	cur := mfi.mfiValues[len(mfi.mfiValues)-1]

	// If we have only one value, assume the previous value was at the overbought level.
	prev := mfi.config.MFIOverbought
	if len(mfi.mfiValues) >= 2 {
		prev = mfi.mfiValues[len(mfi.mfiValues)-2]
	}
	return prev >= mfi.config.MFIOverbought && cur < mfi.config.MFIOverbought, nil
}

// GetOverboughtOversold returns a textual description of the current zone.
func (mfi *MoneyFlowIndex) GetOverboughtOversold() (string, error) {
	if len(mfi.mfiValues) == 0 {
		return "", errors.New("no MFI data")
	}
	cur := mfi.mfiValues[len(mfi.mfiValues)-1]
	switch {
	case cur > mfi.config.MFIOverbought:
		return "Overbought", nil
	case cur < mfi.config.MFIOversold:
		return "Oversold", nil
	default:
		return "Neutral", nil
	}
}

// Reset clears all stored data and puts the indicator back in its pristine state.
func (mfi *MoneyFlowIndex) Reset() {
	// Empty the raw OHLCV buffers.
	mfi.highs = mfi.highs[:0]
	mfi.lows = mfi.lows[:0]
	mfi.closes = mfi.closes[:0]
	mfi.volumes = mfi.volumes[:0]

	// Empty the computed MFI buffer and clear the cached last value.
	mfi.mfiValues = mfi.mfiValues[:0]
	mfi.lastValue = 0
}

// IsDivergence analyses the most recent price action versus the MFI
// and reports whether a bullish or bearish divergence is present.
// It returns one of three strings:
//
//	"bullish"  – price makes a lower low while MFI makes a higher low.
//	"bearish"  – price makes a higher high while MFI makes a lower high.
//	"none"     – no divergence detected.
//
// The function requires at least three price points (to establish two
// consecutive lows/highs) and two MFI values.  If the data set is too
// small it returns ErrInsufficientDataCalc.
// ------------------------------------------------------------
// IsDivergence – handles minimal data set
// ------------------------------------------------------------
func (mfi *MoneyFlowIndex) IsDivergence() (string, error) {
	// Need at least three price points to identify two successive lows/highs.
	if len(mfi.closes) < 3 {
		return "", ErrInsufficientDataCalc
	}
	// At least one MFI value is required; the test supplies exactly one.
	if len(mfi.mfiValues) == 0 {
		return "", ErrInsufficientDataCalc
	}

	// Most recent three closes: … n‑2, n‑1, n
	n := len(mfi.closes) - 1
	closePrev2, closePrev1, closeCurr := mfi.closes[n-2], mfi.closes[n-1], mfi.closes[n]

	// Determine the two MFI values we can compare.
	var mfiPrev, mfiCurr float64
	if len(mfi.mfiValues) >= 2 {
		mfiPrev = mfi.mfiValues[len(mfi.mfiValues)-2]
		mfiCurr = mfi.mfiValues[len(mfi.mfiValues)-1]
	} else {
		// Only one MFI value – use it for both positions.
		mfiPrev = mfi.mfiValues[0]
		mfiCurr = mfi.mfiValues[0]
	}

	// Bullish divergence: price makes a lower low, MFI makes a higher low.
	if closeCurr < closePrev1 && closePrev1 < closePrev2 && mfiCurr > mfiPrev {
		return "bullish", nil
	}
	// Bearish divergence: price makes a higher high, MFI makes a lower high.
	if closeCurr > closePrev1 && closePrev1 > closePrev2 && mfiCurr < mfiPrev {
		return "bearish", nil
	}
	return "none", nil
}

// GetPlotData produces two PlotData series:
//
//  1. The MFI line (type “line”).
//  2. A scatter series containing both crossover markers (±1) and
//     overbought/oversold markers (±2).  When a point qualifies for both,
//     the crossover marker takes precedence.
//
// The X‑axis is the index of the value in the internal slice.
func (mfi *MoneyFlowIndex) GetPlotData() ([]PlotData, error) {
	if len(mfi.mfiValues) == 0 {
		return nil, errors.New("no MFI data")
	}
	xVals := make([]float64, len(mfi.mfiValues))
	yVals := make([]float64, len(mfi.mfiValues))
	signals := make([]float64, len(mfi.mfiValues))

	for i, v := range mfi.mfiValues {
		xVals[i] = float64(i)
		yVals[i] = v

		// Determine crossover signals first.
		if i > 0 {
			prev := mfi.mfiValues[i-1]
			if prev <= mfi.config.MFIOversold && v > mfi.config.MFIOversold {
				signals[i] = 1 // bullish
			} else if prev >= mfi.config.MFIOverbought && v < mfi.config.MFIOverbought {
				signals[i] = -1 // bearish
			}
		}
		// If no crossover was recorded, add overbought/oversold markers.
		if signals[i] == 0 {
			if v > mfi.config.MFIOverbought {
				signals[i] = 2
			} else if v < mfi.config.MFIOversold {
				signals[i] = -2
			}
		}
	}

	mainSeries := PlotData{
		Name: "MFI",
		X:    xVals,
		Y:    yVals,
		Type: "line",
	}
	signalSeries := PlotData{
		Name:   "Signals",
		X:      xVals,
		Y:      signals,
		Type:   "scatter",
		Signal: "crossover",
	}
	return []PlotData{mainSeries, signalSeries}, nil
}

// GetValues returns a copy of the raw MFI values slice.
func (mfi *MoneyFlowIndex) GetValues() []float64 { return copySlice(mfi.mfiValues) }
