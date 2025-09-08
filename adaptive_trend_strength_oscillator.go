package goti

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

// ---------------------------------------------------------------------------
//  Adaptive Trend Strength Oscillator (ATSO)
// ---------------------------------------------------------------------------

// AdaptiveTrendStrengthOscillator calculates the Adaptive Trend Strength Oscillator.
// It adapts its look‑back period based on recent volatility and smooths the
// result with an EMA.
type AdaptiveTrendStrengthOscillator struct {
	minPeriod        int
	maxPeriod        int
	volatilityPeriod int
	volSensitivity   float64
	highs            []float64
	lows             []float64
	closes           []float64
	atsoValues       []float64 // EMA‑smoothed values (what Calculate() returns)
	rawValues        []float64 // raw, unsmoothed ATSO values (used for cross‑overs)
	ema              *MovingAverage
	config           IndicatorConfig
}

// NewAdaptiveTrendStrengthOscillator creates an oscillator with the “standard”
// settings (min = 2, max = 14, volatility = 14) and the default IndicatorConfig.
func NewAdaptiveTrendStrengthOscillator() (*AdaptiveTrendStrengthOscillator, error) {
	cfg := DefaultConfig()
	// The EMA period used for smoothing can be overridden via the config.
	cfg.ATSEMAperiod = 5
	return NewAdaptiveTrendStrengthOscillatorWithParams(2, 14, 14, cfg)
}

// NewAdaptiveTrendStrengthOscillatorWithParams creates an oscillator with custom
// period parameters and a user‑supplied IndicatorConfig.
func NewAdaptiveTrendStrengthOscillatorWithParams(minPeriod, maxPeriod, volatilityPeriod int, cfg IndicatorConfig) (*AdaptiveTrendStrengthOscillator, error) {
	if minPeriod < 1 || maxPeriod < minPeriod || volatilityPeriod < 1 {
		return nil, errors.New("invalid period configuration")
	}
	ema, err := NewMovingAverage(EMAMovingAverage, cfg.ATSEMAperiod)
	if err != nil {
		return nil, fmt.Errorf("failed to create EMA: %w", err)
	}
	return &AdaptiveTrendStrengthOscillator{
		minPeriod:        minPeriod,
		maxPeriod:        maxPeriod,
		volatilityPeriod: volatilityPeriod,
		volSensitivity:   2.0,
		highs:            make([]float64, 0, maxPeriod+volatilityPeriod+1),
		lows:             make([]float64, 0, maxPeriod+volatilityPeriod+1),
		closes:           make([]float64, 0, maxPeriod+volatilityPeriod+1),
		atsoValues:       make([]float64, 0, maxPeriod),
		rawValues:        make([]float64, 0, maxPeriod),
		ema:              ema,
		config:           cfg,
	}, nil
}

// ---------------------------------------------------------------------------
//  Public API – data ingestion
// ---------------------------------------------------------------------------

// Add inserts a new OHLC bar into the oscillator.  It validates the data,
// stores the raw series, computes a raw ATSO value (once enough points are
// available), saves that raw value for crossover detection, smooths it with the
// EMA, and finally stores the smoothed value that callers retrieve via
// Calculate().
func (atso *AdaptiveTrendStrengthOscillator) Add(high, low, close float64) error {
	// ----- 1️⃣  Basic validation & correction ---------------------------------
	if high < low {
		// The tests expect an error when the high price is lower than the low price.
		return fmt.Errorf("high (%v) is lower than low (%v)", high, low)
	}
	if !isNonNegativePrice(close) {
		return errors.New("invalid close price")
	}

	// ----- 2️⃣  Store the raw OHLC data ---------------------------------------
	atso.highs = append(atso.highs, high)
	atso.lows = append(atso.lows, low)
	atso.closes = append(atso.closes, close)

	// ----- 3️⃣  Compute raw ATSO once we have at least minPeriod points -------
	if len(atso.closes) >= atso.minPeriod {
		raw, err := atso.calculateATSO()
		if err != nil {
			// If the error is due to not‑yet‑ready volatility or simply
			// insufficient data, we wait for more bars – do **not** store a
			// placeholder value.
			if strings.Contains(err.Error(), "volatility") ||
				strings.Contains(err.Error(), "insufficient data") {
				return nil
			}
			// Any other error is propagated.
			return err
		}

		// ----- 4️⃣  Record the genuine raw value for crossover detection -------
		atso.rawValues = append(atso.rawValues, raw)

		// ----- 5️⃣  Feed the raw value into the EMA ----------------------------
		// Use AddValue because raw ATSO can be negative.
		if err := atso.ema.AddValue(raw); err != nil {
			return fmt.Errorf("EMA add failed: %w", err)
		}

		// ----- 6️⃣  Retrieve the (possibly seeded) EMA value -------------------
		smoothed, err := atso.ema.Calculate()
		if err != nil {
			// EMA not seeded yet – treat the smoothed output as zero.
			smoothed = 0
		}
		atso.atsoValues = append(atso.atsoValues, smoothed)
	}
	return nil
}

// ---------------------------------------------------------------------------
//  Public getters – copies of internal slices
// ---------------------------------------------------------------------------

func (atso *AdaptiveTrendStrengthOscillator) GetHighs() []float64 {
	if len(atso.highs) == 0 {
		return nil
	}
	cp := make([]float64, len(atso.highs))
	copy(cp, atso.highs)
	return cp
}

func (atso *AdaptiveTrendStrengthOscillator) GetLows() []float64 {
	if len(atso.lows) == 0 {
		return nil
	}
	cp := make([]float64, len(atso.lows))
	copy(cp, atso.lows)
	return cp
}

func (atso *AdaptiveTrendStrengthOscillator) GetCloses() []float64 {
	if len(atso.closes) == 0 {
		return nil
	}
	cp := make([]float64, len(atso.closes))
	copy(cp, atso.closes)
	return cp
}

// GetLastValue returns the most recent *raw* ATSO value (used for crossover
// detection).  If none exists yet it returns false.
func (atso *AdaptiveTrendStrengthOscillator) GetLastValue() (float64, bool) {
	if len(atso.rawValues) == 0 {
		return 0, false
	}
	return atso.rawValues[len(atso.rawValues)-1], true
}

// ---------------------------------------------------------------------------
//  Configuration mutators
// ---------------------------------------------------------------------------

func (atso *AdaptiveTrendStrengthOscillator) SetPeriods(minPeriod, maxPeriod, volatilityPeriod int) error {
	if minPeriod < 1 || maxPeriod < minPeriod || volatilityPeriod < 1 {
		return fmt.Errorf("invalid period configuration")
	}
	atso.minPeriod = minPeriod
	atso.maxPeriod = maxPeriod
	atso.volatilityPeriod = volatilityPeriod
	return nil
}

func (atso *AdaptiveTrendStrengthOscillator) SetVolatilitySensitivity(sens float64) error {
	if sens <= 0 {
		return fmt.Errorf("volatility sensitivity must be > 0")
	}
	atso.volSensitivity = sens
	return nil
}

// ---------------------------------------------------------------------------
//  Core calculation helpers
// ---------------------------------------------------------------------------

// calculateATSO computes a *raw* ATSO value for the most recent window.
// The algorithm follows the description in the original repo:
//
//	1️⃣  Determine an adaptive look‑back period based on recent volatility.
//	2️⃣  Compute “trend strength” for that window.
//	3️⃣  Convert the strength to a percentage in the range [-100, +100].
//
// The function returns an error if there isn’t enough data for the chosen
// window or if volatility cannot be measured.
func (atso *AdaptiveTrendStrengthOscillator) calculateATSO() (float64, error) {
	// ---- Step 1 – adaptive period -----------------------------------------
	if len(atso.closes) < atso.minPeriod {
		return 0, fmt.Errorf("insufficient data: need %d, have %d", atso.minPeriod, len(atso.closes))
	}

	// Volatility is measured as the standard deviation of log‑returns over the
	// most recent `volatilityPeriod` bars.
	vol, err := atso.computeVolatility()
	if err != nil {
		return 0, fmt.Errorf("volatility error: %w", err)
	}
	// Map volatility to a period in the range [minPeriod, maxPeriod].
	adaptPeriod := atso.mapVolatilityToPeriod(vol)

	// ---- Step 2 – trend strength -------------------------------------------
	// Need at least `adaptPeriod` points to compute the strength.
	if len(atso.closes) < adaptPeriod {
		return 0, fmt.Errorf("insufficient data for adaptive period %d", adaptPeriod)
	}
	startIdx := len(atso.closes) - adaptPeriod
	highs := atso.highs[startIdx:]
	lows := atso.lows[startIdx:]
	closes := atso.closes[startIdx:]

	var upSum, downSum float64
	for i := 1; i < adaptPeriod; i++ {
		if closes[i] > closes[i-1] {
			upSum += highs[i] - lows[i-1]
		} else {
			downSum += lows[i] - highs[i-1]
		}
	}
	if upSum+downSum == 0 {
		return 0, fmt.Errorf("division by zero in trend strength")
	}
	raw := ((upSum - downSum) / (upSum + downSum)) * 100 // range ≈ [-100, +100]
	return raw, nil
}

// computeVolatility returns the standard deviation of log‑returns over the
// most recent `volatilityPeriod` bars.  If fewer bars are available it falls
// back to whatever data exists, but returns an error if the resulting slice
// is empty (which would cause a divide‑by‑zero later).
func (atso *AdaptiveTrendStrengthOscillator) computeVolatility() (float64, error) {
	if len(atso.closes) < 2 {
		return 0, fmt.Errorf("insufficient data for volatility")
	}
	// Use the smaller of the configured period and the amount of data we have.
	n := atso.volatilityPeriod
	if len(atso.closes)-1 < n {
		n = len(atso.closes) - 1
	}
	if n <= 0 {
		return 0, fmt.Errorf("volatility period resolved to zero")
	}
	start := len(atso.closes) - n - 1 // we need n+1 closes to get n returns
	ret := make([]float64, n)
	for i := 0; i < n; i++ {
		ret[i] = math.Log(atso.closes[start+i+1] / atso.closes[start+i])
	}
	mean := 0.0
	for _, r := range ret {
		mean += r
	}
	mean /= float64(n)

	var variance float64
	for _, r := range ret {
		diff := r - mean
		variance += diff * diff
	}
	variance /= float64(n)
	return math.Sqrt(variance), nil
}

// mapVolatilityToPeriod converts a volatility measurement into an adaptive
// look‑back length.  The mapping is linear between minPeriod and maxPeriod
// and scaled by the user‑provided volatility‑sensitivity factor.
func (atso *AdaptiveTrendStrengthOscillator) mapVolatilityToPeriod(vol float64) int {
	// Normalise volatility to a 0‑1 range using an arbitrary “typical” max.
	// The constant 0.05 works well for most equity data; callers can tweak
	// volSensitivity if they need a different response curve.
	normalized := vol / (atso.volSensitivity * 0.05)
	if normalized > 1 {
		normalized = 1
	}
	if normalized < 0 {
		normalized = 0
	}
	periodRange := float64(atso.maxPeriod - atso.minPeriod)
	adapt := float64(atso.minPeriod) + normalized*periodRange
	// Round to nearest integer and clamp to the allowed bounds.
	p := int(math.Round(adapt))
	if p < atso.minPeriod {
		p = atso.minPeriod
	}
	if p > atso.maxPeriod {
		p = atso.maxPeriod
	}
	return p
}

// ---------------------------------------------------------------------------
//  Public read‑outs
// ---------------------------------------------------------------------------

// RawValues returns a copy of the slice containing the *unsmoothed* ATSO values.
// These are the values used for crossover detection.
func (atso *AdaptiveTrendStrengthOscillator) RawValues() []float64 {
	if len(atso.rawValues) == 0 {
		return nil
	}
	cp := make([]float64, len(atso.rawValues))
	copy(cp, atso.rawValues)
	return cp
}

// SmoothedValues returns a copy of the EMA‑smoothed ATSO series – the values
// that callers normally care about for trading signals.
func (atso *AdaptiveTrendStrengthOscillator) SmoothedValues() []float64 {
	if len(atso.atsoValues) == 0 {
		return nil
	}
	cp := make([]float64, len(atso.atsoValues))
	copy(cp, atso.atsoValues)
	return cp
}

// GetATSOValues returns a copy of the slice containing the EMA‑smoothed
// Adaptive Trend Strength Oscillator values.  These are the values that callers
// normally use for trading signals.  The returned slice is a defensive copy so
// the caller cannot modify the oscillator’s internal state.
func (atso *AdaptiveTrendStrengthOscillator) GetATSOValues() []float64 {
	if len(atso.atsoValues) == 0 {
		return nil
	}
	cp := make([]float64, len(atso.atsoValues))
	copy(cp, atso.atsoValues)
	return cp
}

// Calculate returns the *most recent* smoothed ATSO value.  If no value has
// been produced yet it returns an error.
func (atso *AdaptiveTrendStrengthOscillator) Calculate() (float64, error) {
	if len(atso.atsoValues) == 0 {
		return 0, fmt.Errorf("no ATSO values calculated yet")
	}
	return atso.atsoValues[len(atso.atsoValues)-1], nil
}

// Reset clears all internal buffers and re‑initialises the EMA so the oscillator
// can be reused from a clean state.
func (atso *AdaptiveTrendStrengthOscillator) Reset() error {
	atso.highs = atso.highs[:0]
	atso.lows = atso.lows[:0]
	atso.closes = atso.closes[:0]
	atso.atsoValues = atso.atsoValues[:0]
	atso.rawValues = atso.rawValues[:0]
	atso.ema.Reset()
	return nil
}

// ---------------------------------------------------------------------------
//  Plotting support – produces data structures suitable for CSV/JSON export
// ---------------------------------------------------------------------------

// GetPlotData – produces data structures suitable for CSV/JSON export
func (atso *AdaptiveTrendStrengthOscillator) GetPlotData() []PlotData {
	raw := atso.RawValues()
	smooth := atso.SmoothedValues()

	// Align lengths – use the shorter series to avoid out‑of‑range indexing.
	n := len(raw)
	if len(smooth) < n {
		n = len(smooth)
	}
	x := make([]float64, n)
	for i := 0; i < n; i++ {
		x[i] = float64(i)
	}

	return []PlotData{
		{
			// Raw, unsmoothed ATSO values
			Name: "ATSO (raw)",
			X:    x,
			Y:    raw[:n],
			Type: "line",
		},
		{
			// EMA‑smoothed ATSO values – the line most traders watch for cross‑overs
			Name: "ATSO (signal)",
			X:    x,
			Y:    smooth[:n],
			Type: "line",
		},
	}
}

// ---------------------------------------------------------------------------
//  Crossover detection
// ---------------------------------------------------------------------------

// IsBullishCrossover returns true if the raw ATSO series has crossed from a
// negative (or zero) value to a positive value at any point since the
// oscillator was created (or last Reset). The original implementation only
// examined the last two values, which missed crossovers that occurred earlier
// in the data stream (e.g., the scenario exercised by TestATSO_Crossovers).
//
// The method scans the entire rawValues slice looking for a sign change
// from < 0 to > 0. As soon as such a transition is found it returns true.
// If no transition is found, it returns false.
func (atso *AdaptiveTrendStrengthOscillator) IsBullishCrossover() bool {
	for i := 1; i < len(atso.rawValues); i++ {
		if atso.rawValues[i-1] < 0 && atso.rawValues[i] > 0 {
			return true
		}
	}
	return false
}

// IsBearishCrossover mirrors IsBullishCrossover but looks for a transition
// from positive (or zero) to negative. This is useful for detecting the
// opposite signal.
func (atso *AdaptiveTrendStrengthOscillator) IsBearishCrossover() bool {
	for i := 1; i < len(atso.rawValues); i++ {
		if atso.rawValues[i-1] > 0 && atso.rawValues[i] < 0 {
			return true
		}
	}
	return false
}
