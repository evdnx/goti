package momentum

import (
	"errors"

	"github.com/evdnx/goti/indicator/core"
)

const (
	DefaultStochasticKPeriod    = 14
	DefaultStochasticDPeriod    = 3
	DefaultStochasticOverbought = 80.0
	DefaultStochasticOversold   = 20.0
)

// StochasticOscillator implements a classic %K / %D stochastic oscillator.
// %K measures the current close relative to the recent high-low range, and
// %D is a moving average of %K.
type StochasticOscillator struct {
	kPeriod int
	dPeriod int

	highs  []float64
	lows   []float64
	closes []float64

	kValues []float64
	dValues []float64

	lastK float64
	lastD float64
}

// NewStochasticOscillator builds a stochastic oscillator with 14/3 defaults.
func NewStochasticOscillator() (*StochasticOscillator, error) {
	return NewStochasticOscillatorWithParams(DefaultStochasticKPeriod, DefaultStochasticDPeriod)
}

// NewStochasticOscillatorWithParams builds a stochastic oscillator with custom
// %K and %D periods.
func NewStochasticOscillatorWithParams(kPeriod, dPeriod int) (*StochasticOscillator, error) {
	if kPeriod < 1 || dPeriod < 1 {
		return nil, errors.New("periods must be at least 1")
	}
	return &StochasticOscillator{
		kPeriod: kPeriod,
		dPeriod: dPeriod,
		highs:   make([]float64, 0, kPeriod+1),
		lows:    make([]float64, 0, kPeriod+1),
		closes:  make([]float64, 0, kPeriod+1),
		kValues: make([]float64, 0, dPeriod),
		dValues: make([]float64, 0, dPeriod),
	}, nil
}

// Add ingests a new OHLC bar and updates the oscillator if possible.
func (s *StochasticOscillator) Add(high, low, close float64) error {
	if high < low || !core.IsNonNegativePrice(close) {
		return errors.New("invalid price data")
	}
	s.highs = append(s.highs, high)
	s.lows = append(s.lows, low)
	s.closes = append(s.closes, close)

	if len(s.closes) >= s.kPeriod {
		k := s.computeK()
		s.lastK = k
		s.kValues = append(s.kValues, k)

		if len(s.kValues) >= s.dPeriod {
			sum := 0.0
			for i := len(s.kValues) - s.dPeriod; i < len(s.kValues); i++ {
				sum += s.kValues[i]
			}
			s.lastD = sum / float64(s.dPeriod)
			s.dValues = append(s.dValues, s.lastD)
		}
	}

	s.trimSlices()
	return nil
}

// Calculate returns the latest %K and %D values.
func (s *StochasticOscillator) Calculate() (float64, float64, error) {
	if len(s.kValues) == 0 {
		return 0, 0, errors.New("no stochastic data")
	}
	if len(s.dValues) == 0 {
		return s.lastK, 0, errors.New("D line not ready")
	}
	return s.lastK, s.lastD, nil
}

// IsOverbought reports whether the current %K is above the common 80 level.
func (s *StochasticOscillator) IsOverbought() (bool, error) {
	if len(s.kValues) == 0 {
		return false, errors.New("no stochastic data")
	}
	return s.lastK > DefaultStochasticOverbought, nil
}

// IsOversold reports whether the current %K is below the common 20 level.
func (s *StochasticOscillator) IsOversold() (bool, error) {
	if len(s.kValues) == 0 {
		return false, errors.New("no stochastic data")
	}
	return s.lastK < DefaultStochasticOversold, nil
}

// Reset clears all stored samples and outputs.
func (s *StochasticOscillator) Reset() {
	s.highs = s.highs[:0]
	s.lows = s.lows[:0]
	s.closes = s.closes[:0]
	s.kValues = s.kValues[:0]
	s.dValues = s.dValues[:0]
	s.lastK, s.lastD = 0, 0
}

// SetPeriods updates %K and %D periods and resets the oscillator.
func (s *StochasticOscillator) SetPeriods(kPeriod, dPeriod int) error {
	if kPeriod < 1 || dPeriod < 1 {
		return errors.New("periods must be at least 1")
	}
	s.kPeriod = kPeriod
	s.dPeriod = dPeriod
	s.Reset()
	return nil
}

// GetKValues returns a defensive copy of the %K series.
func (s *StochasticOscillator) GetKValues() []float64 { return core.CopySlice(s.kValues) }

// GetDValues returns a defensive copy of the %D series.
func (s *StochasticOscillator) GetDValues() []float64 { return core.CopySlice(s.dValues) }

// GetPlotData emits plot-friendly series for %K and %D.
func (s *StochasticOscillator) GetPlotData(startTime, interval int64) []core.PlotData {
	if len(s.kValues) == 0 {
		return nil
	}
	x := make([]float64, len(s.kValues))
	for i := range x {
		x[i] = float64(i)
	}
	timestamps := core.GenerateTimestamps(startTime, len(s.kValues), interval)

	plots := []core.PlotData{
		{
			Name:      "%K",
			X:         x,
			Y:         s.kValues,
			Type:      "line",
			Timestamp: timestamps,
		},
	}
	if len(s.dValues) > 0 {
		plots = append(plots, core.PlotData{
			Name:      "%D",
			X:         x[len(x)-len(s.dValues):],
			Y:         s.dValues,
			Type:      "line",
			Timestamp: timestamps[len(timestamps)-len(s.dValues):],
		})
	}
	return plots
}

func (s *StochasticOscillator) computeK() float64 {
	start := len(s.closes) - s.kPeriod
	windowHigh := s.highs[start:]
	windowLow := s.lows[start:]

	highest := windowHigh[0]
	lowest := windowLow[0]
	for i := 1; i < s.kPeriod; i++ {
		if windowHigh[i] > highest {
			highest = windowHigh[i]
		}
		if windowLow[i] < lowest {
			lowest = windowLow[i]
		}
	}
	rangeHL := highest - lowest
	if rangeHL == 0 {
		return 0
	}
	close := s.closes[len(s.closes)-1]
	return ((close - lowest) / rangeHL) * 100
}

func (s *StochasticOscillator) trimSlices() {
	s.highs = core.KeepLast(s.highs, s.kPeriod+1)
	s.lows = core.KeepLast(s.lows, s.kPeriod+1)
	s.closes = core.KeepLast(s.closes, s.kPeriod+1)
	maxKeep := s.kPeriod + s.dPeriod
	s.kValues = core.KeepLast(s.kValues, maxKeep)
	s.dValues = core.KeepLast(s.dValues, maxKeep)
}
