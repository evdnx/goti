package volume

import (
	"errors"

	"github.com/evdnx/goti/indicator/core"
)

// VWAP calculates the Volume Weighted Average Price using cumulative sums.
type VWAP struct {
	cumPV    float64 // cumulative price*volume
	cumVol   float64 // cumulative volume
	vwapVals []float64
	last     float64
}

// NewVWAP constructs a VWAP calculator with an empty state.
func NewVWAP() *VWAP {
	return &VWAP{
		vwapVals: make([]float64, 0, 64),
	}
}

// Add ingests a new OHLCV candle. Typical price is used for VWAP.
func (v *VWAP) Add(high, low, close, volume float64) error {
	if high < low || !core.IsNonNegativePrice(close) || !core.IsValidVolume(volume) {
		return errors.New("invalid price or volume")
	}
	typicalPrice := (high + low + close) / 3
	v.cumPV += typicalPrice * volume
	v.cumVol += volume

	if v.cumVol > 0 {
		v.last = v.cumPV / v.cumVol
		v.vwapVals = append(v.vwapVals, v.last)
		v.trimSlices()
	}
	return nil
}

// Calculate returns the current VWAP value.
func (v *VWAP) Calculate() (float64, error) {
	if len(v.vwapVals) == 0 || v.cumVol == 0 {
		return 0, errors.New("no VWAP data")
	}
	return v.last, nil
}

// Reset clears all accumulated state.
func (v *VWAP) Reset() {
	v.cumPV = 0
	v.cumVol = 0
	v.last = 0
	v.vwapVals = v.vwapVals[:0]
}

// GetValues returns the VWAP series (defensive copy).
func (v *VWAP) GetValues() []float64 { return core.CopySlice(v.vwapVals) }

// GetPlotData emits VWAP plot data aligned with the number of samples added.
func (v *VWAP) GetPlotData(startTime, interval int64) []core.PlotData {
	if len(v.vwapVals) == 0 {
		return nil
	}
	x := make([]float64, len(v.vwapVals))
	for i := range x {
		x[i] = float64(i)
	}
	ts := core.GenerateTimestamps(startTime, len(v.vwapVals), interval)
	return []core.PlotData{{
		Name:      "VWAP",
		X:         x,
		Y:         v.vwapVals,
		Type:      "line",
		Timestamp: ts,
	}}
}

func (v *VWAP) trimSlices() {
	const maxKeep = 1024
	v.vwapVals = core.KeepLast(v.vwapVals, maxKeep)
}
