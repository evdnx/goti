package goti

import "fmt"

// EMA implements a simple Exponential Moving Average.
// It stores all added values so that the very first EMA can be seeded
// with the Simple Moving Average of the first `period` points.
type EMA struct {
	period  int
	values  []float64 // raw values that have been added
	prevEMA float64   // last EMA value (zero until we have enough data)
	seeded  bool      // true once we have produced the first EMA
}

// NewEMA creates a fresh EMA ready to accept values and returns it as a
// MovingAverage interface.  Returning the interface directly lets callers
// assign the result to variables typed as MovingAverage without a compile‑time
// mismatch.
func NewEMA(period int) *EMA {
	if period <= 0 {
		panic(fmt.Sprintf("EMA period must be > 0, got %d", period))
	}
	return &EMA{
		period: period,
		values: make([]float64, 0, period*2), // modest pre‑allocation
	}
}

// Add inserts a new raw value into the EMA calculation.
// It never returns an error (the surrounding code already validates the
// input), but we keep the signature consistent with the existing calls.
func (e *EMA) Add(v float64) error {
	e.values = append(e.values, v)

	// If we haven’t seeded yet and we now have at least `period` points,
	// compute the simple moving average and store it as the first EMA.
	if !e.seeded && len(e.values) >= e.period {
		sum := 0.0
		for _, x := range e.values[len(e.values)-e.period:] {
			sum += x
		}
		e.prevEMA = sum / float64(e.period)
		e.seeded = true
		return nil
	}

	// If we’re already seeded, update the EMA using the classic formula.
	if e.seeded {
		alpha := 2.0 / float64(e.period+1)
		e.prevEMA = alpha*v + (1-alpha)*e.prevEMA
	}
	return nil
}

// Calculate returns the current EMA value.
// If the EMA has not been seeded yet (i.e. fewer than `period` values have
// been added), we return an error so callers can decide how to handle the
// “not ready” state.
func (e *EMA) Calculate() (float64, error) {
	if !e.seeded {
		return 0, fmt.Errorf("EMA not initialized – need at least %d values", e.period)
	}
	return e.prevEMA, nil
}
