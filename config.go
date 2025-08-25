package goti

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
	RSIOverbought   float64 // RSI > this → overbought
	RSIOversold     float64 // RSI < this → oversold
	MFIOverbought   float64 // Money Flow Index overbought level
	MFIOversold     float64 // Money Flow Index oversold level
	AMDOOverbought  float64 // ADMO z‑score overbought threshold
	AMDOOversold    float64 // ADMO z‑score oversold threshold
	AMDOScaling     float64 // scaling factor used by some ADMO variants
	VWAOStrongTrend float64 // VWAO strong‑trend threshold
}

// DefaultConfig returns a sensible set of defaults for every indicator.
func DefaultConfig() IndicatorConfig {
	return IndicatorConfig{
		RSIOverbought:   70,
		RSIOversold:     30,
		MFIOverbought:   80,
		MFIOversold:     20,
		AMDOOverbought:  DefaultAMDOOverbought,
		AMDOOversold:    DefaultAMDOOversold,
		AMDOScaling:     50,
		VWAOStrongTrend: 70,
	}
}
