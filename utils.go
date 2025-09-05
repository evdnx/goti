package goti

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
)

// MovingAverageType defines the type of moving average
type MovingAverageType string

const (
	EMA MovingAverageType = "EMA"
	SMA MovingAverageType = "SMA"
	WMA MovingAverageType = "WMA"
)

// MovingAverage calculates Simple or Exponential Moving Average
type MovingAverage struct {
	maType    MovingAverageType
	period    int
	values    []float64
	lastValue float64 // holds the previously‑calculated EMA (used for recursion)
}

// NewMovingAverage initializes a MovingAverage with the specified type and period
func NewMovingAverage(maType MovingAverageType, period int) (*MovingAverage, error) {
	if period < 1 {
		return nil, errors.New("period must be at least 1")
	}
	if maType != SMA && maType != EMA && maType != WMA {
		return nil, errors.New("invalid moving average type")
	}
	return &MovingAverage{
		maType: maType,
		period: period,
		values: make([]float64, 0, period),
	}, nil
}

/* -------------------------------------------------------------------------
   Adding data
--------------------------------------------------------------------------*/

// Add appends a new value to the moving average, enforcing non‑negative values.
// It **does not** call Calculate – the caller should invoke Calculate when the
// current MA value is needed.
func (ma *MovingAverage) Add(value float64) error {
	if !isNonNegativePrice(value) {
		return errors.New("invalid value")
	}
	ma.values = append(ma.values, value)
	ma.trimSlices()
	return nil
}

// AddValue appends a new value without enforcing the non‑negative price rule.
// Like Add, it defers calculation until Calculate is called explicitly.
func (ma *MovingAverage) AddValue(value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return errors.New("invalid value")
	}
	ma.values = append(ma.values, value)
	ma.trimSlices()
	return nil
}

/* -------------------------------------------------------------------------
   Core calculation
--------------------------------------------------------------------------*/

// trimSlices ensures the internal slice never exceeds the configured period.
func (ma *MovingAverage) trimSlices() {
	if len(ma.values) > ma.period {
		ma.values = ma.values[len(ma.values)-ma.period:]
	}
}

// Calculate returns the current moving‑average value.
// The slice has already been trimmed by Add, so we can operate directly on it.
func (ma *MovingAverage) Calculate() (float64, error) {
	if len(ma.values) < ma.period {
		return 0, fmt.Errorf("insufficient data: need %d, have %d", ma.period, len(ma.values))
	}

	switch ma.maType {
	case SMA:
		// Simple Moving Average – average the values we have.
		sum := 0.0
		for _, v := range ma.values {
			sum += v
		}
		return sum / float64(ma.period), nil

	case EMA:
		// Exponential Moving Average – uses the previously‑calculated EMA.
		ema, err := calculateEMA(ma.values, ma.period, ma.lastValue)
		if err != nil {
			return 0, err
		}
		// Store the EMA for the next recursive step.
		ma.lastValue = ema
		return ema, nil

	case WMA:
		// Weighted Moving Average.
		return calculateWMA(ma.values, ma.period)

	default:
		return 0, fmt.Errorf("unsupported moving‑average type %s", ma.maType)
	}
}

/* -------------------------------------------------------------------------
   Miscellaneous helpers
--------------------------------------------------------------------------*/

func (ma *MovingAverage) Reset() {
	ma.values = ma.values[:0]
	ma.lastValue = 0
}

func (ma *MovingAverage) SetPeriod(period int) error {
	if period < 1 {
		return errors.New("period must be at least 1")
	}
	ma.period = period
	ma.trimSlices()
	return nil
}

func (ma *MovingAverage) GetValues() []float64 {
	return copySlice(ma.values)
}

/* -------------------------------------------------------------------------
   Plotting utilities (unchanged)
--------------------------------------------------------------------------*/

type PlotData struct {
	Name      string    `json:"name"`
	X         []float64 `json:"x"`
	Y         []float64 `json:"y"`
	Type      string    `json:"type,omitempty"`
	Signal    string    `json:"signal,omitempty"`
	Timestamp []int64   `json:"timestamp,omitempty"`
}

func copySlice(src []float64) []float64 {
	if src == nil {
		return nil
	}
	dst := make([]float64, len(src))
	copy(dst, src)
	return dst
}

/* -------------------------------------------------------------------------
   Numeric helpers (unchanged)
--------------------------------------------------------------------------*/

func clamp(value, min, max float64) float64 {
	if min == max {
		return min // avoid division‑by‑zero style issues
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// calculateStandardDeviation computes the standard deviation using Welford’s algorithm.
func calculateStandardDeviation(data []float64, mean float64) float64 {
	if len(data) == 0 {
		return 0
	}
	if mean == 0 {
		for _, v := range data {
			mean += v
		}
		mean /= float64(len(data))
	}
	var sumSq float64
	for _, v := range data {
		diff := v - mean
		sumSq += diff * diff
	}
	if len(data) < 2 {
		return 0
	}
	return math.Sqrt(sumSq / float64(len(data)-1))
}

/* -------------------------------------------------------------------------
   EMA / WMA implementations (unchanged)
--------------------------------------------------------------------------*/

// calculateEMA computes the Exponential Moving Average.
// If we have fewer than “period” samples we fall back to a simple
// average of the data we do have – this lets the first EMA value be
// produced without waiting for a full window.
func calculateEMA(data []float64, period int, prevEMA float64) (float64, error) {
	if len(data) == 0 {
		return 0, fmt.Errorf("no data for EMA")
	}
	// If we don’t have a full period yet, return the SMA of whatever we have.
	if len(data) < period {
		sum := 0.0
		for _, v := range data {
			sum += v
		}
		return sum / float64(len(data)), nil
	}
	// Full‑period case – unchanged from the original implementation.
	if len(data) == period && prevEMA == 0 {
		sum := 0.0
		for _, v := range data[len(data)-period:] {
			sum += v
		}
		return sum / float64(period), nil
	}
	smoothing := 2.0 / float64(period+1)
	current := data[len(data)-1]
	return smoothing*current + (1-smoothing)*prevEMA, nil
}

// calculateWMA computes the Weighted Moving Average.
func calculateWMA(data []float64, period int) (float64, error) {
	if len(data) < period {
		return 0, fmt.Errorf("insufficient data for WMA: need %d, have %d", period, len(data))
	}
	sum, weightSum := 0.0, 0.0
	for i := 0; i < period; i++ {
		weight := float64(period - i)
		sum += data[len(data)-period+i] * weight
		weightSum += weight
	}
	if weightSum == 0 {
		return 0, errors.New("zero weight sum in WMA calculation")
	}
	return sum / weightSum, nil
}

/* -------------------------------------------------------------------------
   Validation helpers (unchanged)
--------------------------------------------------------------------------*/

func isValidPrice(price float64) bool {
	return price > 0 && !math.IsNaN(price) && !math.IsInf(price, 0)
}

func isNonNegativePrice(price float64) bool {
	return price >= 0 && !math.IsNaN(price) && !math.IsInf(price, 0)
}

func isValidVolume(volume float64) bool {
	return volume >= 0 && !math.IsNaN(volume) && !math.IsInf(volume, 0)
}

/* -------------------------------------------------------------------------
   Timestamp / formatting utilities (unchanged)
--------------------------------------------------------------------------*/

func GenerateTimestamps(startTime int64, count int, interval int64) []int64 {
	if count <= 0 {
		return nil
	}
	ts := make([]int64, count)
	for i := 0; i < count; i++ {
		ts[i] = startTime + int64(i)*interval
	}
	return ts
}

func formatPlotDataJSON(data []PlotData) (string, error) {
	if len(data) == 0 {
		return "[]", nil
	}
	for _, d := range data {
		if len(d.X) != len(d.Y) {
			return "", fmt.Errorf("mismatched X and Y lengths for %s: %d vs %d", d.Name, len(d.X), len(d.Y))
		}
	}
	b, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal plot data: %w", err)
	}
	return string(b), nil
}

func formatPlotDataCSV(data []PlotData) (string, error) {
	if len(data) == 0 {
		return "", nil
	}
	var sb strings.Builder
	sb.WriteString("Name,X,Y,Type,Signal,Timestamp\n")
	for _, d := range data {
		if len(d.X) != len(d.Y) {
			return "", fmt.Errorf("mismatched X and Y lengths for %s: %d vs %d", d.Name, len(d.X), len(d.Y))
		}
		for i := 0; i < len(d.X); i++ {
			ts := ""
			if i < len(d.Timestamp) {
				ts = fmt.Sprintf("%d", d.Timestamp[i])
			}
			fmt.Fprintf(&sb, "%s,%f,%f,%s,%s,%s\n",
				d.Name, d.X[i], d.Y[i], d.Type, d.Signal, ts)
		}
	}
	return sb.String(), nil
}

/* -------------------------------------------------------------------------
   Misc numeric helper
--------------------------------------------------------------------------*/

func calculateSlope(y2, y1 float64) float64 {
	return y2 - y1
}
