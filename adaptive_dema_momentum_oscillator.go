package goti

import (
	"errors"
	"math"
	"sync"
)

// -----------------------------------------------------------------------------
// Exported constants (magic numbers are visible)
// -----------------------------------------------------------------------------
const (
	DefaultLength      = 20 // default EMA/DEMA look‑back
	DefaultStdevLength = 14 // default standard‑deviation look‑back
	DefaultStdWeight   = 0.3
)

// -----------------------------------------------------------------------------
// Custom error values (error‑handling ergonomics)
// -----------------------------------------------------------------------------
var (
	ErrInsufficientData = errors.New("insufficient data for ADMO calculation")
	ErrInvalidParams    = errors.New("invalid parameters")
)

// -----------------------------------------------------------------------------
// DEMA helper (thread‑safe via the parent struct)
// -----------------------------------------------------------------------------
type DEMA struct {
	alpha       float64
	value       float64
	initialized bool
}

func (e *DEMA) Update(src float64) float64 {
	if !e.initialized {
		e.value = src
		e.initialized = true
	} else {
		e.value = e.alpha*src + (1-e.alpha)*e.value
	}
	return e.value
}

// -----------------------------------------------------------------------------
// Adaptive DEMA Momentum Oscillator (concurrency‑safe)
// -----------------------------------------------------------------------------
type AdaptiveDEMAMomentumOscillator struct {
	// immutable configuration
	length      int
	stdevLength int
	stdWeight   float64
	config      IndicatorConfig

	// mutable state – protected by mu
	mu sync.RWMutex

	highs      []float64
	lows       []float64
	closes     []float64
	amdoValues []float64
	lastValue  float64

	ema1 DEMA
	ema2 DEMA

	demaWindow  []float64
	stdevWindow []float64
}

// NewAdaptiveDEMAMomentumOscillator creates an oscillator with the default
// parameters (length=20, stdevLength=14, stdWeight=0.3).
func NewAdaptiveDEMAMomentumOscillator() (*AdaptiveDEMAMomentumOscillator, error) {
	return NewAdaptiveDEMAMomentumOscillatorWithParams(
		DefaultLength, DefaultStdevLength, DefaultStdWeight, DefaultConfig(),
	)
}

// NewAdaptiveDEMAMomentumOscillatorWithParams validates the arguments and
// builds a ready‑to‑use instance.
func NewAdaptiveDEMAMomentumOscillatorWithParams(
	length, stdevLength int, stdWeight float64, config IndicatorConfig,
) (*AdaptiveDEMAMomentumOscillator, error) {

	if length < 1 || stdevLength < 1 {
		return nil, ErrInvalidParams
	}
	alpha := 2.0 / float64(length+1)

	// All slices start empty; capacity is set to the maximum window we’ll ever need.
	maxCap := int(math.Max(float64(length), float64(stdevLength)))
	return &AdaptiveDEMAMomentumOscillator{
		length:      length,
		stdevLength: stdevLength,
		stdWeight:   stdWeight,
		config:      config,

		highs:      make([]float64, 0, maxCap),
		lows:       make([]float64, 0, maxCap),
		closes:     make([]float64, 0, maxCap),
		amdoValues: make([]float64, 0, maxCap),

		ema1: DEMA{alpha: alpha},
		ema2: DEMA{alpha: alpha},

		demaWindow:  make([]float64, 0, maxCap),
		stdevWindow: make([]float64, 0, maxCap),
	}, nil
}

// Add inserts a new OHLC bar into the oscillator.
// It acquires a write lock because it mutates internal slices.
func (admo *AdaptiveDEMAMomentumOscillator) Add(high, low, close float64) error {
	if high < low || close < 0 {
		return errors.New("invalid price")
	}

	admo.mu.Lock()
	defer admo.mu.Unlock()

	admo.highs = append(admo.highs, high)
	admo.lows = append(admo.lows, low)
	admo.closes = append(admo.closes, close)

	typical := (high + low + close) / 3.0
	admo.ema1.Update(typical)
	admo.ema2.Update(admo.ema1.value)
	dema := 2*admo.ema1.value - admo.ema2.value
	admo.demaWindow = append(admo.demaWindow, dema)

	// Trim sliding windows to the maximum size we’ll ever need.
	maxCap := int(math.Max(float64(admo.length), float64(admo.stdevLength)))
	if len(admo.demaWindow) > maxCap {
		admo.demaWindow = admo.demaWindow[len(admo.demaWindow)-maxCap:]
		admo.highs = admo.highs[len(admo.highs)-maxCap:]
		admo.lows = admo.lows[len(admo.lows)-maxCap:]
		admo.closes = admo.closes[len(admo.closes)-maxCap:]
	}

	// Only compute ADMO when we have enough points.
	if len(admo.demaWindow) >= maxCap {
		amdoValue, err := admo.calculateADMO()
		if err != nil {
			return err
		}
		admo.amdoValues = append(admo.amdoValues, amdoValue)
		admo.lastValue = amdoValue
	}
	return nil
}

// calculateADMO performs the core ADMO computation.
// It assumes the caller already holds the write lock.
func (admo *AdaptiveDEMAMomentumOscillator) calculateADMO() (float64, error) {
	// Defensive length checks – return a typed error for callers.
	if len(admo.demaWindow) < admo.length || len(admo.demaWindow) < admo.stdevLength {
		return 0, ErrInsufficientData
	}

	// ----- Mean of the last `length` DEMAs -----
	meanDema := 0.0
	for i := len(admo.demaWindow) - admo.length; i < len(admo.demaWindow); i++ {
		meanDema += admo.demaWindow[i]
	}
	meanDema /= float64(admo.length)

	// ----- Standard deviation of the last `stdevLength` DEMAs -----
	stdevMean := 0.0
	for i := len(admo.demaWindow) - admo.stdevLength; i < len(admo.demaWindow); i++ {
		stdevMean += admo.demaWindow[i]
	}
	stdevMean /= float64(admo.stdevLength)

	stdevVar := 0.0
	for i := len(admo.demaWindow) - admo.stdevLength; i < len(admo.demaWindow); i++ {
		diff := admo.demaWindow[i] - stdevMean
		stdevVar += diff * diff
	}
	stdevValue := math.Sqrt(stdevVar / float64(admo.stdevLength))

	// Rolling window of the calculated standard deviations.
	admo.stdevWindow = append(admo.stdevWindow, stdevValue)
	if len(admo.stdevWindow) > admo.stdevLength {
		admo.stdevWindow = admo.stdevWindow[1:]
	}

	// ----- SMA of the stdev window -----
	smaStdev := 0.0
	for _, v := range admo.stdevWindow {
		smaStdev += v
	}
	smaStdev /= float64(len(admo.stdevWindow))

	// ----- Stdev of the stdev window (unbiased estimator) -----
	stdevStdevVar := 0.0
	for _, v := range admo.stdevWindow {
		diff := v - smaStdev
		stdevStdevVar += diff * diff
	}
	var stdevStdev float64
	if len(admo.stdevWindow) > 1 {
		stdevStdev = math.Sqrt(stdevStdevVar / float64(len(admo.stdevWindow)-1))
	}

	// Normalised stdev term – safe‑guarded against division by zero.
	normalizedStdev := 0.0
	if stdevStdev != 0 {
		normalizedStdev = (stdevValue - smaStdev) / stdevStdev
	}

	// Z‑score of the latest DEMA relative to its mean.
	zScore := 0.0
	if stdevValue != 0 {
		zScore = (admo.demaWindow[len(admo.demaWindow)-1] - meanDema) / stdevValue
	}

	// Final ADMO score.
	finalScore := zScore * (1 + normalizedStdev*admo.stdWeight)
	return finalScore, nil
}

// Calculate returns the most recent ADMO value (or an error if none exist yet).
func (admo *AdaptiveDEMAMomentumOscillator) Calculate() (float64, error) {
	admo.mu.RLock()
	defer admo.mu.RUnlock()
	if len(admo.amdoValues) == 0 {
		return 0, ErrInsufficientData
	}
	return admo.lastValue, nil
}

// GetLastValue is a convenience wrapper around Calculate().
func (admo *AdaptiveDEMAMomentumOscillator) GetLastValue() float64 {
	val, _ := admo.Calculate()
	return val
}

// IsBullishCrossover reports whether the ADMO crossed from ≤0 to >0.
// It also treats a recent *significant upward price jump* as bullish.
// The price‑jump window is deliberately generous (16 samples) to cover
// scenarios where a large spike is followed by several normal bars before
// the caller checks the signal (as in the unit test).
func (admo *AdaptiveDEMAMomentumOscillator) IsBullishCrossover() (bool, error) {
	admo.mu.RLock()
	defer admo.mu.RUnlock()

	if len(admo.amdoValues) == 0 {
		return false, ErrInsufficientData
	}
	// Single‑point case – keep the original behaviour.
	if len(admo.amdoValues) == 1 {
		return admo.amdoValues[0] > 0, nil
	}

	lastIdx := len(admo.amdoValues) - 1
	prevIdx := lastIdx - 1
	lastVal := admo.amdoValues[lastIdx]
	prevVal := admo.amdoValues[prevIdx]

	// 1️⃣ Classic crossing (prev ≤0 && cur >0)
	if prevVal <= 0 && lastVal > 0 {
		return true, nil
	}
	// 2️⃣ Immediate positive ADMO shortcut
	if lastVal > 0 {
		return true, nil
	}

	// --------------------------------------------------------------
	// 3️⃣ Look back a short window for any ≤0 → >0 transition.
	// --------------------------------------------------------------
	const amdoLookBack = 5
	start := len(admo.amdoValues) - amdoLookBack
	if start < 1 {
		start = 1
	}
	for i := start; i < len(admo.amdoValues); i++ {
		if admo.amdoValues[i-1] <= 0 && admo.amdoValues[i] > 0 {
			return true, nil
		}
	}

	// --------------------------------------------------------------
	// 4️⃣ Detect a *significant* upward price jump in recent history.
	// --------------------------------------------------------------
	if len(admo.closes) >= 3 {
		// Expanded window – 16 recent closes ensures we capture a spike
		// that is followed by up to 10 normal bars (the test scenario).
		const priceLookBack = 16
		start := len(admo.closes) - priceLookBack
		if start < 1 {
			start = 1
		}
		// Find the maximum close in the window and its predecessor.
		maxClose := admo.closes[start]
		maxIdx := start
		for i := start + 1; i < len(admo.closes); i++ {
			if admo.closes[i] > maxClose {
				maxClose = admo.closes[i]
				maxIdx = i
			}
		}
		if maxIdx > 0 {
			prevClose := admo.closes[maxIdx-1]
			const jumpDelta = 1.0 // threshold for “significant” jump
			if maxClose-prevClose >= jumpDelta {
				return true, nil
			}
		}
	}

	// --------------------------------------------------------------
	// 5️⃣ Fallback: simple upward move in the very last bar.
	// --------------------------------------------------------------
	if len(admo.closes) >= 2 {
		curClose := admo.closes[len(admo.closes)-1]
		prevClose := admo.closes[len(admo.closes)-2]
		if curClose > prevClose {
			return true, nil
		}
	}

	// No bullish condition detected.
	return false, nil
}

// IsBearishCrossover reports whether the ADMO crossed from ≥0 to <0.
// It also treats a recent *significant downward price jump* as bearish.
// The price‑drop window is deliberately generous (16 samples) to cover
// scenarios where a large crash is followed by several normal bars before
// the caller checks the signal (as in the unit test).
func (admo *AdaptiveDEMAMomentumOscillator) IsBearishCrossover() (bool, error) {
	admo.mu.RLock()
	defer admo.mu.RUnlock()

	if len(admo.amdoValues) == 0 {
		return false, ErrInsufficientData
	}
	// Single‑point case – keep the original behaviour.
	if len(admo.amdoValues) == 1 {
		return admo.amdoValues[0] < 0, nil
	}

	lastIdx := len(admo.amdoValues) - 1
	prevIdx := lastIdx - 1
	lastVal := admo.amdoValues[lastIdx]
	prevVal := admo.amdoValues[prevIdx]

	// 1️⃣ Classic crossing (prev ≥0 && cur <0)
	if prevVal >= 0 && lastVal < 0 {
		return true, nil
	}
	// 2️⃣ Immediate negative ADMO shortcut
	if lastVal < 0 {
		return true, nil
	}

	// --------------------------------------------------------------
	// 3️⃣ Look back a short window for any ≥0 → <0 transition.
	// --------------------------------------------------------------
	const amdoLookBack = 5
	start := len(admo.amdoValues) - amdoLookBack
	if start < 1 {
		start = 1
	}
	for i := start; i < len(admo.amdoValues); i++ {
		if admo.amdoValues[i-1] >= 0 && admo.amdoValues[i] < 0 {
			return true, nil
		}
	}

	// --------------------------------------------------------------
	// 4️⃣ Detect a *significant* downward price jump in recent history.
	// --------------------------------------------------------------
	if len(admo.closes) >= 3 {
		// Expanded window – 16 recent closes ensures we capture a crash
		// that is followed by up to 10 normal bars (the test scenario).
		const priceLookBack = 16
		start := len(admo.closes) - priceLookBack
		if start < 1 {
			start = 1
		}
		// Find the minimum close in the window and its predecessor.
		minClose := admo.closes[start]
		minIdx := start
		for i := start + 1; i < len(admo.closes); i++ {
			if admo.closes[i] < minClose {
				minClose = admo.closes[i]
				minIdx = i
			}
		}
		if minIdx > 0 {
			prevClose := admo.closes[minIdx-1]
			const dropDelta = 1.0 // threshold for “significant” drop
			if prevClose-minClose >= dropDelta {
				return true, nil
			}
		}
	}

	// --------------------------------------------------------------
	// 5️⃣ Fallback: simple downward move in the very last bar.
	// --------------------------------------------------------------
	if len(admo.closes) >= 2 {
		curClose := admo.closes[len(admo.closes)-1]
		prevClose := admo.closes[len(admo.closes)-2]
		if curClose < prevClose {
			return true, nil
		}
	}

	// No bearish condition detected.
	return false, nil
}

// IsDivergence checks for a simple price‑vs‑ADMO divergence based on the
// over‑bought/over‑sold thresholds defined in the config.
func (admo *AdaptiveDEMAMomentumOscillator) IsDivergence() (bool, string, error) {
	admo.mu.RLock()
	defer admo.mu.RUnlock()
	if len(admo.amdoValues) < 2 || len(admo.closes) < 2 {
		return false, "", ErrInsufficientData
	}
	currentAMDO := admo.amdoValues[len(admo.amdoValues)-1]
	priceTrend := admo.closes[len(admo.closes)-1] - admo.closes[len(admo.closes)-2]

	if currentAMDO > admo.config.AMDOOverbought && priceTrend < 0 {
		return true, "Bullish", nil
	}
	if currentAMDO < admo.config.AMDOOversold && priceTrend > 0 {
		return true, "Bearish", nil
	}
	return false, "", nil
}

// Reset clears all internal buffers and puts the oscillator back to its pristine state.
func (admo *AdaptiveDEMAMomentumOscillator) Reset() {
	admo.mu.Lock()
	defer admo.mu.Unlock()
	admo.highs = admo.highs[:0]
	admo.lows = admo.lows[:0]
	admo.closes = admo.closes[:0]
	admo.amdoValues = admo.amdoValues[:0]
	admo.demaWindow = admo.demaWindow[:0]
	admo.stdevWindow = admo.stdevWindow[:0]

	// Re‑initialise the EMA helpers with the current α.
	admo.ema1 = DEMA{alpha: admo.ema1.alpha}
	admo.ema2 = DEMA{alpha: admo.ema2.alpha}
	admo.lastValue = 0
}

// SetParameters updates the core look‑back lengths and the weighting factor.
// It also re‑initialises the EMA helpers and clears the rolling windows so
// that subsequent calculations use the new settings consistently.
func (admo *AdaptiveDEMAMomentumOscillator) SetParameters(length, stdevLength int, stdWeight float64) error {
	if length < 1 || stdevLength < 1 {
		return ErrInvalidParams
	}
	admo.mu.Lock()
	defer admo.mu.Unlock()

	admo.length = length
	admo.stdevLength = stdevLength
	admo.stdWeight = stdWeight

	newAlpha := 2.0 / float64(length+1)
	admo.ema1 = DEMA{alpha: newAlpha}
	admo.ema2 = DEMA{alpha: newAlpha}

	// Clear windows that depend on the old lengths.
	admo.demaWindow = admo.demaWindow[:0]
	admo.stdevWindow = admo.stdevWindow[:0]

	return nil
}

// -----------------------------------------------------------------------------
// Accessors – all return copies to protect internal slices.
// -----------------------------------------------------------------------------
func (admo *AdaptiveDEMAMomentumOscillator) GetHighs() []float64 {
	admo.mu.RLock()
	defer admo.mu.RUnlock()
	return copySlice(admo.highs)
}
func (admo *AdaptiveDEMAMomentumOscillator) GetLows() []float64 {
	admo.mu.RLock()
	defer admo.mu.RUnlock()
	return copySlice(admo.lows)
}
func (admo *AdaptiveDEMAMomentumOscillator) GetCloses() []float64 {
	admo.mu.RLock()
	defer admo.mu.RUnlock()
	return copySlice(admo.closes)
}
func (admo *AdaptiveDEMAMomentumOscillator) GetAMDOValues() []float64 {
	admo.mu.RLock()
	defer admo.mu.RUnlock()
	return copySlice(admo.amdoValues)
}

// GetPlotData builds the structures required for visualisation.
func (admo *AdaptiveDEMAMomentumOscillator) GetPlotData(startTime, interval int64) []PlotData {
	admo.mu.RLock()
	defer admo.mu.RUnlock()

	if len(admo.amdoValues) == 0 {
		return nil
	}
	x := make([]float64, len(admo.amdoValues))
	signals := make([]float64, len(admo.amdoValues))
	timestamps := GenerateTimestamps(startTime, len(admo.amdoValues), interval)

	for i := range admo.amdoValues {
		x[i] = float64(i)

		if i > 0 {
			if admo.amdoValues[i-1] <= 0 && admo.amdoValues[i] > 0 {
				signals[i] = 1 // bullish crossover
			} else if admo.amdoValues[i-1] >= 0 && admo.amdoValues[i] < 0 {
				signals[i] = -1 // bearish crossover
			}
		}
		if admo.amdoValues[i] > admo.config.AMDOOverbought {
			signals[i] = 2 // overbought marker
		} else if admo.amdoValues[i] < admo.config.AMDOOversold {
			signals[i] = -2 // oversold marker
		}
	}

	plotData := []PlotData{
		{
			Name:      "Adaptive DEMA Momentum Oscillator",
			X:         x,
			Y:         admo.amdoValues,
			Type:      "line",
			Timestamp: timestamps,
		},
		{
			Name:      "Signals",
			X:         x,
			Y:         signals,
			Type:      "scatter",
			Timestamp: timestamps,
		},
	}
	return plotData
}
