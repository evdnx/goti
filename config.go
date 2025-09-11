package goti

import "fmt"

// -----------------------------------------------------------------------------
// Exported constants (magic numbers made visible)
// -----------------------------------------------------------------------------
const (
	// Default ADMO z‑score thresholds.
	DefaultAMDOOverbought = 1.0  // above this → overbought
	DefaultAMDOOversold   = -1.0 // below this → oversold
)

// -----------------------------------------------------------------------------
// IndicatorConfig – central place for all tunable parameters
// -----------------------------------------------------------------------------
type IndicatorConfig struct {
	RSIOverbought float64 // RSI > this → overbought
	RSIOversold   float64 // RSI < this → oversold
	MFIOverbought float64 // Money Flow Index overbought level
	MFIOversold   float64 // Money Flow Index oversold level
	// MFIVolumeScale scales raw volume before it is multiplied by the typical price.
	// The historic default (300 000) is kept for backward compatibility.
	MFIVolumeScale float64

	AMDOOverbought  float64 // ADMO z‑score overbought threshold
	AMDOOversold    float64 // ADMO z‑score oversold threshold
	AMDOScaling     float64 // scaling factor used by some ADMO variants
	VWAOStrongTrend float64 // VWAO strong‑trend threshold

	// ATSEMAperiod is the EMA period used to smooth the Adaptive Trend
	// Strength Oscillator (ATSO).  The default matches the original hard‑coded
	// value of 5 but can be overridden by the caller.
	ATSEMAperiod int
}

// DefaultConfig returns a sensible set of defaults for every indicator.
func DefaultConfig() IndicatorConfig {
	return IndicatorConfig{
		RSIOverbought:   70,
		RSIOversold:     30,
		MFIOverbought:   80,
		MFIOversold:     20,
		MFIVolumeScale:  300_000, // historic default
		AMDOOverbought:  DefaultAMDOOverbought,
		AMDOOversold:    DefaultAMDOOversold,
		AMDOScaling:     50,
		VWAOStrongTrend: 70,
		ATSEMAperiod:    5,
	}
}

// -------------------------------------------------------------------
// Validate – checks that the configuration values are sensible.
// -------------------------------------------------------------------
func (c IndicatorConfig) Validate() error {
	// 0 or negative values are not allowed.
	if c.ATSEMAperiod <= 0 {
		return fmt.Errorf("ATSEMAperiod must be greater than 0, got %d", c.ATSEMAperiod)
	}

	// Upper‑bound sanity check – any value that is absurdly large is treated
	// as an error (covers the wrap‑around case when a negative literal is
	// forced into an unsigned type elsewhere).
	const maxReasonablePeriod = 1_000_000
	if c.ATSEMAperiod > maxReasonablePeriod {
		return fmt.Errorf(
			"ATSEMAperiod is unreasonably large (%d); must be ≤ %d",
			c.ATSEMAperiod,
			maxReasonablePeriod,
		)
	}
	return nil
}
