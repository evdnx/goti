// volume_weighted_aroon_oscillator.go
//
// Genuine Volume‑Weighted Aroon Oscillator (VWAO)
// ------------------------------------------------------------
// This implementation builds on the classic Aroon‑Oscillator (Aroon‑Up
// – Aroon‑Down) but incorporates the traded volume of each bar.  The
// idea is to weight the “time‑since‑extreme” component by the volume
// that actually occurred on the bar where the extreme price was hit.
// The resulting oscillator reacts both to price extremes and to the
// amount of market activity behind those extremes.
//
// Public API (constructors, Add, Calculate, signal helpers, etc.) stays
// identical to the original version, ensuring drop‑in compatibility.
//
// Author: Lumo (Proton) – 2025
//

package goti

import (
	"errors"
	"fmt"
)

// VolumeWeightedAroonOscillator calculates a volume‑weighted Aroon Oscillator.
type VolumeWeightedAroonOscillator struct {
	period     int
	highs      []float64
	lows       []float64
	closes     []float64
	volumes    []float64
	vwaoValues []float64
	lastValue  float64
	config     IndicatorConfig
}

// NewVolumeWeightedAroonOscillator creates a VWAO with the default period (14)
// and the library’s default configuration.
func NewVolumeWeightedAroonOscillator() (*VolumeWeightedAroonOscillator, error) {
	return NewVolumeWeightedAroonOscillatorWithParams(14, DefaultConfig())
}

// NewVolumeWeightedAroonOscillatorWithParams creates a VWAO with a custom period
// and configuration.
func NewVolumeWeightedAroonOscillatorWithParams(period int, cfg IndicatorConfig) (*VolumeWeightedAroonOscillator, error) {
	if period < 1 {
		return nil, errors.New("period must be at least 1")
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return &VolumeWeightedAroonOscillator{
		period:     period,
		highs:      make([]float64, 0, period+1),
		lows:       make([]float64, 0, period+1),
		closes:     make([]float64, 0, period+1),
		volumes:    make([]float64, 0, period+1),
		vwaoValues: make([]float64, 0, period),
		config:     cfg,
	}, nil
}

// Add inserts a new candle (high, low, close) together with its volume.
// Validation mirrors the rest of the library: prices must be non‑negative,
// high ≥ low, and volume must be a valid number.
func (v *VolumeWeightedAroonOscillator) Add(high, low, close, volume float64) error {
	if high < low || !isNonNegativePrice(close) || !isValidVolume(volume) {
		return errors.New("invalid price or volume")
	}
	v.highs = append(v.highs, high)
	v.lows = append(v.lows, low)
	v.closes = append(v.closes, close)
	v.volumes = append(v.volumes, volume)

	// Compute a new VWAO once we have enough points (period+1 candles).
	if len(v.closes) >= v.period+1 {
		val, err := v.computeVWAO()
		if err != nil {
			return fmt.Errorf("computeVWAO failed: %w", err)
		}
		v.vwaoValues = append(v.vwaoValues, val)
		v.lastValue = val
	}
	v.trimSlices()
	return nil
}

// trimSlices caps the stored slices to the maximum size required for the
// next calculation, preventing unbounded memory growth.
func (v *VolumeWeightedAroonOscillator) trimSlices() {
	if len(v.closes) > v.period+1 {
		v.highs = v.highs[len(v.highs)-v.period-1:]
		v.lows = v.lows[len(v.lows)-v.period-1:]
		v.closes = v.closes[len(v.closes)-v.period-1:]
		v.volumes = v.volumes[len(v.volumes)-v.period-1:]
	}
	if len(v.vwaoValues) > v.period {
		v.vwaoValues = v.vwaoValues[len(v.vwaoValues)-v.period:]
	}
}

// computeVWAO performs the genuine volume‑weighted Aroon‑Oscillator
// calculation.
//
// Algorithm:
//  1. Look at the last (period+1) bars.
//  2. Identify the most recent highest high and lowest low and their indices.
//  3. Compute the total volume‑weighted “age” of the window:
//     Σ (period‑i) * volume[i]   for i = 0 … period
//  4. Weight the age of the high and low by the volume that occurred on the
//     bar where the extreme price was observed.
//     weightedHighAge = (period‑highIdx) * volume[highIdx]
//     weightedLowAge  = (period‑lowIdx)  * volume[lowIdx]
//  5. Derive volume‑weighted Aroon percentages:
//     aroonUp   = (weightedHighAge / totalWeightedAge) * 100
//     aroonDown = (weightedLowAge  / totalWeightedAge) * 100
//  6. Oscillator = aroonUp – aroonDown, clamped to [-100, 100].
//
// This yields a metric that rises when a strong high appears on heavy volume
// (and falls when a strong low appears on heavy volume), while still respecting
// the classic Aroon time‑decay intuition.
func (v *VolumeWeightedAroonOscillator) computeVWAO() (float64, error) {
	if len(v.closes) < v.period+1 {
		return 0, fmt.Errorf("insufficient data: need %d, have %d", v.period+1, len(v.closes))
	}

	// Slice the window that will be examined.
	start := len(v.closes) - v.period - 1
	highs := v.highs[start:]
	lows := v.lows[start:]
	vols := v.volumes[start:]

	// Locate the most recent highest high and lowest low.
	maxHighIdx, minLowIdx := 0, 0
	maxHigh, minLow := highs[0], lows[0]
	var totalWeightedAge float64

	for i := 0; i <= v.period; i++ {
		if highs[i] > maxHigh {
			maxHigh = highs[i]
			maxHighIdx = i
		}
		if lows[i] < minLow {
			minLow = lows[i]
			minLowIdx = i
		}
		// Age weighting: newer bars have larger (period‑i) factor.
		totalWeightedAge += float64(v.period-i) * vols[i]
	}
	if totalWeightedAge == 0 {
		return 0, errors.New("total weighted volume is zero")
	}

	// Volume‑weighted ages for the extremes.
	weightedHighAge := float64(v.period-maxHighIdx) * vols[maxHighIdx]
	weightedLowAge := float64(v.period-minLowIdx) * vols[minLowIdx]

	// Convert to classic Aroon percentages, but using volume‑weighted ages.
	aroonUp := (weightedHighAge / totalWeightedAge) * 100
	aroonDown := (weightedLowAge / totalWeightedAge) * 100

	osc := aroonUp - aroonDown
	return clamp(osc, -100, 100), nil
}

// Calculate returns the most recent VWAO value (or an error if none have been computed).
func (v *VolumeWeightedAroonOscillator) Calculate() (float64, error) {
	if len(v.vwaoValues) == 0 {
		return 0, errors.New("no VWAO data")
	}
	return v.lastValue, nil
}

// GetLastValue is a convenience wrapper that never errors – useful for UI polling.
func (v *VolumeWeightedAroonOscillator) GetLastValue() float64 { return v.lastValue }

// ---------- Signal helpers (unchanged semantics) ----------
func (v *VolumeWeightedAroonOscillator) IsBullishCrossover() (bool, error) {
	if len(v.vwaoValues) < 2 {
		return false, errors.New("insufficient data for crossover")
	}
	prev, cur := v.vwaoValues[len(v.vwaoValues)-2], v.vwaoValues[len(v.vwaoValues)-1]
	return prev <= v.config.VWAOStrongTrend && cur > v.config.VWAOStrongTrend, nil
}

func (v *VolumeWeightedAroonOscillator) IsBearishCrossover() (bool, error) {
	if len(v.vwaoValues) < 2 {
		return false, errors.New("insufficient data for crossover")
	}
	prev, cur := v.vwaoValues[len(v.vwaoValues)-2], v.vwaoValues[len(v.vwaoValues)-1]
	return prev >= -v.config.VWAOStrongTrend && cur < -v.config.VWAOStrongTrend, nil
}

func (v *VolumeWeightedAroonOscillator) IsStrongTrend() (bool, error) {
	if len(v.vwaoValues) == 0 {
		return false, errors.New("no VWAO data")
	}
	cur := v.vwaoValues[len(v.vwaoValues)-1]
	return cur > v.config.VWAOStrongTrend || cur < -v.config.VWAOStrongTrend, nil
}

func (v *VolumeWeightedAroonOscillator) IsDivergence() (bool, string, error) {
	if len(v.vwaoValues) < 2 || len(v.closes) < 2 {
		return false, "", errors.New("insufficient data for divergence")
	}
	curVWAO := v.vwaoValues[len(v.vwaoValues)-1]
	priceDelta := v.closes[len(v.closes)-1] - v.closes[len(v.closes)-2]

	if curVWAO > v.config.VWAOStrongTrend && priceDelta < 0 {
		return true, "Bearish", nil
	}
	if curVWAO < -v.config.VWAOStrongTrend && priceDelta > 0 {
		return true, "Bullish", nil
	}
	return false, "", nil
}

// Reset clears all internal buffers – handy for back‑testing loops.
func (v *VolumeWeightedAroonOscillator) Reset() {
	v.highs = v.highs[:0]
	v.lows = v.lows[:0]
	v.closes = v.closes[:0]
	v.volumes = v.volumes[:0]
	v.vwaoValues = v.vwaoValues[:0]
	v.lastValue = 0
}

// SetPeriod changes the look‑back window and trims any excess data.
func (v *VolumeWeightedAroonOscillator) SetPeriod(p int) error {
	if p < 1 {
		return errors.New("period must be at least 1")
	}
	v.period = p
	v.trimSlices()
	return nil
}

// ---------- Accessors (return copies) ----------
func (v *VolumeWeightedAroonOscillator) GetHighs() []float64   { return copySlice(v.highs) }
func (v *VolumeWeightedAroonOscillator) GetLows() []float64    { return copySlice(v.lows) }
func (v *VolumeWeightedAroonOscillator) GetCloses() []float64  { return copySlice(v.closes) }
func (v *VolumeWeightedAroonOscillator) GetVolumes() []float64 { return copySlice(v.volumes) }
func (v *VolumeWeightedAroonOscillator) GetVWAOValues() []float64 {
	return copySlice(v.vwaoValues)
}

// ---------- Plotting helper ----------
func (v *VolumeWeightedAroonOscillator) GetPlotData(startTime, interval int64) []PlotData {
	if len(v.vwaoValues) == 0 {
		return nil
	}
	x := make([]float64, len(v.vwaoValues))
	signals := make([]float64, len(v.vwaoValues))
	ts := GenerateTimestamps(startTime, len(v.vwaoValues), interval)

	for i := range v.vwaoValues {
		x[i] = float64(i)
		if i > 0 {
			if v.vwaoValues[i-1] <= v.config.VWAOStrongTrend && v.vwaoValues[i] > v.config.VWAOStrongTrend {
				signals[i] = 1 // bullish crossover
			} else if v.vwaoValues[i-1] >= -v.config.VWAOStrongTrend && v.vwaoValues[i] < -v.config.VWAOStrongTrend {
				signals[i] = -1 // bearish crossover
			}
		}
		// Highlight strong‑trend zones.
		if v.vwaoValues[i] > v.config.VWAOStrongTrend {
			signals[i] = 2
		} else if v.vwaoValues[i] < -v.config.VWAOStrongTrend {
			signals[i] = -2
		}
	}
	return []PlotData{
		{
			Name:      "Volume Weighted Aroon Oscillator",
			X:         x,
			Y:         v.vwaoValues,
			Type:      "line",
			Timestamp: ts,
		},
		{
			Name:      "Signals",
			X:         x,
			Y:         signals,
			Type:      "scatter",
			Timestamp: ts,
		},
	}
}
