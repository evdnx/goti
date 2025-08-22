package goti

// IndicatorConfig holds customizable parameters for indicators
type IndicatorConfig struct {
	MFIOverbought   float64
	MFIOversold     float64
	RSIOverbought   float64
	RSIOversold     float64
	AMDODivergence  float64
	VWAOStrongTrend float64
}

// DefaultConfig provides default threshold values
func DefaultConfig() IndicatorConfig {
	return IndicatorConfig{
		MFIOverbought:   80,
		MFIOversold:     20,
		RSIOverbought:   70,
		RSIOversold:     30,
		AMDODivergence:  50,
		VWAOStrongTrend: 50,
	}
}
