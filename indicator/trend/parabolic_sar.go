package trend

import (
	"errors"
	"math"

	"github.com/evdnx/goti/indicator/core"
)

const (
	DefaultSARStep    = 0.02
	DefaultSARMaxStep = 0.2
)

// ParabolicSAR implements Wilder's Parabolic SAR (Stop and Reverse) indicator.
// It works incrementally from high/low data, tracking the current trend,
// extreme point (EP), acceleration factor (AF), and SAR value.
type ParabolicSAR struct {
	step    float64
	maxStep float64

	af          float64
	ep          float64
	sar         float64
	uptrend     bool
	initialized bool

	highs  []float64
	lows   []float64
	values []float64

	lastValue float64
}

// NewParabolicSAR creates a SAR calculator with default step (0.02) and
// maximum step (0.2).
func NewParabolicSAR() (*ParabolicSAR, error) {
	return NewParabolicSARWithParams(DefaultSARStep, DefaultSARMaxStep)
}

// NewParabolicSARWithParams allows custom acceleration parameters.
func NewParabolicSARWithParams(step, maxStep float64) (*ParabolicSAR, error) {
	if step <= 0 || maxStep <= 0 {
		return nil, errors.New("step parameters must be positive")
	}
	if step > maxStep {
		return nil, errors.New("step must be <= maxStep")
	}
	return &ParabolicSAR{
		step:    step,
		maxStep: maxStep,
		highs:   make([]float64, 0, 4),
		lows:    make([]float64, 0, 4),
		values:  make([]float64, 0, 16),
	}, nil
}

// Add appends a new candle (high/low). SAR values are produced once at least
// two candles have been seen.
func (p *ParabolicSAR) Add(high, low float64) error {
	if high < low || !core.IsNonNegativePrice(high) || !core.IsNonNegativePrice(low) {
		return errors.New("invalid price data")
	}
	p.highs = append(p.highs, high)
	p.lows = append(p.lows, low)

	switch len(p.highs) {
	case 1:
		// Need at least two points to start.
		return nil
	case 2:
		p.initializeTrend()
	default:
		p.updateSAR()
	}

	p.trimSlices()
	return nil
}

// Calculate returns the latest SAR value.
func (p *ParabolicSAR) Calculate() (float64, error) {
	if len(p.values) == 0 {
		return 0, errors.New("no SAR data")
	}
	return p.lastValue, nil
}

// IsUptrend reports the current trend direction.
func (p *ParabolicSAR) IsUptrend() bool { return p.uptrend }

// Reset clears internal state while preserving parameters.
func (p *ParabolicSAR) Reset() {
	p.af = 0
	p.ep = 0
	p.sar = 0
	p.uptrend = false
	p.initialized = false
	p.highs = p.highs[:0]
	p.lows = p.lows[:0]
	p.values = p.values[:0]
	p.lastValue = 0
}

// SetParams updates step parameters and resets the indicator.
func (p *ParabolicSAR) SetParams(step, maxStep float64) error {
	if step <= 0 || maxStep <= 0 {
		return errors.New("step parameters must be positive")
	}
	if step > maxStep {
		return errors.New("step must be <= maxStep")
	}
	p.step = step
	p.maxStep = maxStep
	p.Reset()
	return nil
}

// GetValues returns the SAR series (defensive copy).
func (p *ParabolicSAR) GetValues() []float64 { return core.CopySlice(p.values) }

// GetPlotData returns plot-friendly SAR points.
func (p *ParabolicSAR) GetPlotData(startTime, interval int64) []core.PlotData {
	if len(p.values) == 0 {
		return nil
	}
	x := make([]float64, len(p.values))
	for i := range x {
		x[i] = float64(i)
	}
	ts := core.GenerateTimestamps(startTime, len(p.values), interval)
	return []core.PlotData{{
		Name:      "Parabolic SAR",
		X:         x,
		Y:         p.values,
		Type:      "line",
		Timestamp: ts,
	}}
}

func (p *ParabolicSAR) initializeTrend() {
	if len(p.highs) < 2 {
		return
	}
	prevMid := (p.highs[len(p.highs)-2] + p.lows[len(p.lows)-2]) / 2
	currMid := (p.highs[len(p.highs)-1] + p.lows[len(p.lows)-1]) / 2
	p.uptrend = currMid >= prevMid
	if p.uptrend {
		p.ep = math.Max(p.highs[len(p.highs)-1], p.highs[len(p.highs)-2])
		p.sar = p.lows[len(p.lows)-2]
	} else {
		p.ep = math.Min(p.lows[len(p.lows)-1], p.lows[len(p.lows)-2])
		p.sar = p.highs[len(p.highs)-2]
	}
	p.af = p.step
	p.initialized = true
	p.values = append(p.values, p.sar)
	p.lastValue = p.sar
}

func (p *ParabolicSAR) updateSAR() {
	if !p.initialized || len(p.highs) < 3 {
		return
	}

	newSAR := p.sar + p.af*(p.ep-p.sar)

	prevLow := p.lows[len(p.lows)-2]
	prevHigh := p.highs[len(p.highs)-2]
	if len(p.lows) >= 3 {
		prevLow = math.Min(prevLow, p.lows[len(p.lows)-3])
		prevHigh = math.Max(prevHigh, p.highs[len(p.highs)-3])
	}

	if p.uptrend {
		newSAR = math.Min(newSAR, prevLow)
		if p.lows[len(p.lows)-1] < newSAR {
			// Reversal to downtrend.
			p.uptrend = false
			newSAR = p.ep
			p.ep = p.lows[len(p.lows)-1]
			p.af = p.step
		} else {
			if p.highs[len(p.highs)-1] > p.ep {
				p.ep = p.highs[len(p.highs)-1]
				p.af = math.Min(p.af+p.step, p.maxStep)
			}
		}
	} else {
		newSAR = math.Max(newSAR, prevHigh)
		if p.highs[len(p.highs)-1] > newSAR {
			// Reversal to uptrend.
			p.uptrend = true
			newSAR = p.ep
			p.ep = p.highs[len(p.highs)-1]
			p.af = p.step
		} else {
			if p.lows[len(p.lows)-1] < p.ep {
				p.ep = p.lows[len(p.lows)-1]
				p.af = math.Min(p.af+p.step, p.maxStep)
			}
		}
	}

	p.sar = newSAR
	p.values = append(p.values, newSAR)
	p.lastValue = newSAR
}

func (p *ParabolicSAR) trimSlices() {
	// Keep only the last few samples (enough for boundary checks).
	p.highs = core.KeepLast(p.highs, 4)
	p.lows = core.KeepLast(p.lows, 4)
	p.values = core.KeepLast(p.values, 256)
}
