package momentum

import (
	"errors"
	"fmt"

	"github.com/evdnx/goti/indicator/core"
)

const (
	DefaultMACDFastPeriod   = 12
	DefaultMACDSlowPeriod   = 26
	DefaultMACDSignalPeriod = 9
)

// MACD implements the Moving Average Convergence Divergence indicator.
// It tracks the MACD line (fast EMA - slow EMA), the signal line (EMA of MACD),
// and the histogram (MACD - signal).
type MACD struct {
	fastPeriod   int
	slowPeriod   int
	signalPeriod int

	fastEMA   *core.MovingAverage
	slowEMA   *core.MovingAverage
	signalEMA *core.MovingAverage

	macdValues      []float64
	signalValues    []float64
	histogramValues []float64

	lastMACD   float64
	lastSignal float64
	lastHist   float64
}

// NewMACD creates a MACD with the standard 12/26/9 periods.
func NewMACD() (*MACD, error) {
	return NewMACDWithParams(DefaultMACDFastPeriod, DefaultMACDSlowPeriod, DefaultMACDSignalPeriod)
}

// NewMACDWithParams creates a MACD with custom fast/slow/signal periods.
func NewMACDWithParams(fastPeriod, slowPeriod, signalPeriod int) (*MACD, error) {
	if fastPeriod < 1 || slowPeriod < 1 || signalPeriod < 1 {
		return nil, errors.New("periods must be at least 1")
	}
	if fastPeriod >= slowPeriod {
		return nil, errors.New("fast period must be less than slow period")
	}

	fast, err := core.NewMovingAverage(core.EMAMovingAverage, fastPeriod)
	if err != nil {
		return nil, fmt.Errorf("failed to create fast EMA: %w", err)
	}
	slow, err := core.NewMovingAverage(core.EMAMovingAverage, slowPeriod)
	if err != nil {
		return nil, fmt.Errorf("failed to create slow EMA: %w", err)
	}
	signal, err := core.NewMovingAverage(core.EMAMovingAverage, signalPeriod)
	if err != nil {
		return nil, fmt.Errorf("failed to create signal EMA: %w", err)
	}

	return &MACD{
		fastPeriod:      fastPeriod,
		slowPeriod:      slowPeriod,
		signalPeriod:    signalPeriod,
		fastEMA:         fast,
		slowEMA:         slow,
		signalEMA:       signal,
		macdValues:      make([]float64, 0, signalPeriod),
		signalValues:    make([]float64, 0, signalPeriod),
		histogramValues: make([]float64, 0, signalPeriod),
	}, nil
}

// Add ingests a new closing price and updates the MACD series when possible.
func (m *MACD) Add(close float64) error {
	if !core.IsNonNegativePrice(close) {
		return errors.New("invalid price")
	}
	if err := m.fastEMA.Add(close); err != nil {
		return err
	}
	if err := m.slowEMA.Add(close); err != nil {
		return err
	}

	fast, errFast := m.fastEMA.Calculate()
	slow, errSlow := m.slowEMA.Calculate()
	if errFast == nil && errSlow == nil {
		macd := fast - slow
		m.lastMACD = macd
		m.macdValues = append(m.macdValues, macd)

		// Signal EMA can accept negative values, so we use AddValue.
		_ = m.signalEMA.AddValue(macd)
		if sig, err := m.signalEMA.Calculate(); err == nil {
			m.lastSignal = sig
			m.signalValues = append(m.signalValues, sig)

			hist := macd - sig
			m.lastHist = hist
			m.histogramValues = append(m.histogramValues, hist)
		}
	}

	m.trimSlices()
	return nil
}

// Calculate returns the latest MACD, signal, and histogram values.
func (m *MACD) Calculate() (float64, float64, float64, error) {
	if len(m.macdValues) == 0 {
		return 0, 0, 0, errors.New("no MACD data")
	}
	if len(m.signalValues) == 0 {
		return m.lastMACD, 0, 0, errors.New("signal line not ready")
	}
	return m.lastMACD, m.lastSignal, m.lastHist, nil
}

// Reset clears all internal state and re-seeds the EMAs.
func (m *MACD) Reset() {
	m.fastEMA.Reset()
	m.slowEMA.Reset()
	m.signalEMA.Reset()
	m.macdValues = m.macdValues[:0]
	m.signalValues = m.signalValues[:0]
	m.histogramValues = m.histogramValues[:0]
	m.lastMACD, m.lastSignal, m.lastHist = 0, 0, 0
}

// SetPeriods updates the fast/slow/signal periods and resets internal state.
func (m *MACD) SetPeriods(fastPeriod, slowPeriod, signalPeriod int) error {
	if fastPeriod < 1 || slowPeriod < 1 || signalPeriod < 1 {
		return errors.New("periods must be at least 1")
	}
	if fastPeriod >= slowPeriod {
		return errors.New("fast period must be less than slow period")
	}
	m.fastPeriod = fastPeriod
	m.slowPeriod = slowPeriod
	m.signalPeriod = signalPeriod

	fast, err := core.NewMovingAverage(core.EMAMovingAverage, fastPeriod)
	if err != nil {
		return err
	}
	slow, err := core.NewMovingAverage(core.EMAMovingAverage, slowPeriod)
	if err != nil {
		return err
	}
	signal, err := core.NewMovingAverage(core.EMAMovingAverage, signalPeriod)
	if err != nil {
		return err
	}
	m.fastEMA = fast
	m.slowEMA = slow
	m.signalEMA = signal
	m.Reset()
	return nil
}

// GetMACDValues returns a defensive copy of the MACD line values.
func (m *MACD) GetMACDValues() []float64 { return core.CopySlice(m.macdValues) }

// GetSignalValues returns a defensive copy of the signal line.
func (m *MACD) GetSignalValues() []float64 { return core.CopySlice(m.signalValues) }

// GetHistogramValues returns a defensive copy of the histogram.
func (m *MACD) GetHistogramValues() []float64 {
	return core.CopySlice(m.histogramValues)
}

// GetPlotData returns plot-friendly data for the MACD, signal, and histogram.
func (m *MACD) GetPlotData(startTime, interval int64) []core.PlotData {
	if len(m.macdValues) == 0 {
		return nil
	}
	x := make([]float64, len(m.macdValues))
	for i := range x {
		x[i] = float64(i)
	}
	timestamps := core.GenerateTimestamps(startTime, len(m.macdValues), interval)

	plots := []core.PlotData{
		{
			Name:      "MACD",
			X:         x,
			Y:         m.macdValues,
			Type:      "line",
			Timestamp: timestamps,
		},
	}
	if len(m.signalValues) > 0 {
		plots = append(plots, core.PlotData{
			Name:      "Signal",
			X:         x[len(x)-len(m.signalValues):],
			Y:         m.signalValues,
			Type:      "line",
			Timestamp: timestamps[len(timestamps)-len(m.signalValues):],
		})
	}
	if len(m.histogramValues) > 0 {
		plots = append(plots, core.PlotData{
			Name:      "Histogram",
			X:         x[len(x)-len(m.histogramValues):],
			Y:         m.histogramValues,
			Type:      "bar",
			Timestamp: timestamps[len(timestamps)-len(m.histogramValues):],
		})
	}
	return plots
}

func (m *MACD) trimSlices() {
	maxKeep := m.slowPeriod + m.signalPeriod
	m.macdValues = core.KeepLast(m.macdValues, maxKeep)
	m.signalValues = core.KeepLast(m.signalValues, maxKeep)
	m.histogramValues = core.KeepLast(m.histogramValues, maxKeep)
}
