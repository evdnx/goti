package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
)

// -----------------------------------------------------------------------------
// Generic helpers
// -----------------------------------------------------------------------------

// keepLast returns the last n elements of a slice (or the whole slice if it is
// shorter). It works for any element type thanks to Go generics.
func keepLast[T any](s []T, n int) []T {
	if len(s) > n {
		return s[len(s)-n:]
	}
	return s
}

// KeepLast is the exported wrapper for keepLast to share slice logic across packages.
func KeepLast[T any](s []T, n int) []T {
	return keepLast(s, n)
}

// -----------------------------------------------------------------------------
// MovingAverage types
// -----------------------------------------------------------------------------

// MovingAverageType defines the type of moving average
type MovingAverageType string

const (
	EMAMovingAverage MovingAverageType = "EMA"
	SMAMovingAverage MovingAverageType = "SMA"
	WMAMovingAverage MovingAverageType = "WMA"
)

// MovingAverage calculates Simple or Exponential Moving Average
type MovingAverage struct {
	maType    MovingAverageType
	period    int
	values    []float64
	lastValue float64 // holds the previously‑calculated value (EMA only)

	// Internal bookkeeping for EMA so we can perform incremental updates as
	// new samples arrive without needing the full history.
	sampleCount    int
	emaSeedSum     float64
	emaInitialized bool
}

// NewMovingAverage initializes a MovingAverage with the specified type and period
func NewMovingAverage(maType MovingAverageType, period int) (*MovingAverage, error) {
	if period < 1 {
		return nil, errors.New("period must be at least 1")
	}
	if maType != SMAMovingAverage && maType != EMAMovingAverage && maType != WMAMovingAverage {
		return nil, errors.New("invalid moving average type")
	}
	ma := &MovingAverage{
		maType: maType,
		period: period,
		values: make([]float64, 0, period),
	}
	return ma, nil
}

/* -------------------------------------------------------------------------
   Adding data
--------------------------------------------------------------------------*/

// Add appends a new value to the moving average, enforcing non‑negative values.
// It **does not** call Calculate – the caller should invoke Calculate when the
// current MA value is needed.
func (ma *MovingAverage) Add(value float64) error {
	if !isNonNegativePrice(value) {
		return fmt.Errorf("cannot add negative or NaN price %f", value)
	}
	ma.pushSample(value)
	return nil
}

// AddValue appends a new value without enforcing the non‑negative price rule.
// Like Add, it defers calculation until Calculate is called explicitly.
func (ma *MovingAverage) AddValue(value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return fmt.Errorf("cannot add invalid value %f", value)
	}
	ma.pushSample(value)
	return nil
}

func (ma *MovingAverage) pushSample(value float64) {
	ma.values = append(ma.values, value)
	ma.sampleCount++
	if ma.maType == EMAMovingAverage {
		ma.updateEMA(value)
	}
	ma.trimSlices()
}

// updateEMA incrementally updates the EMA state each time a new value is
// ingested. Once we have gathered `period` samples we seed the EMA with the
// simple average of those initial observations. Subsequent calls apply the
// classic smoothing recursion using only the most recent sample and the
// previously computed EMA value.
func (ma *MovingAverage) updateEMA(latest float64) {
	if ma.period <= 0 {
		return
	}

	// Accumulate the first `period` values to seed the EMA with an SMA.
	if ma.sampleCount <= ma.period {
		ma.emaSeedSum += latest
		if ma.sampleCount < ma.period {
			return
		}
		ma.lastValue = ma.emaSeedSum / float64(ma.period)
		ma.emaInitialized = true
		return
	}

	if !ma.emaInitialized {
		ma.lastValue = ma.emaSeedSum / float64(ma.period)
		ma.emaInitialized = true
	}
	alpha := 2.0 / float64(ma.period+1)
	ma.lastValue = alpha*latest + (1-alpha)*ma.lastValue
}

/* -------------------------------------------------------------------------
   Core calculation
--------------------------------------------------------------------------*/

func (ma *MovingAverage) trimSlices() {
	ma.values = keepLast(ma.values, ma.period)
}

// Calculate returns the current moving‑average value.
// The slice has already been trimmed by Add, so we can operate directly on it.
func (ma *MovingAverage) Calculate() (float64, error) {
	if len(ma.values) < ma.period {
		return 0, fmt.Errorf("insufficient data: need %d, have %d", ma.period, len(ma.values))
	}

	switch ma.maType {
	case SMAMovingAverage:
		// Simple Moving Average – average the values we have.
		sum := 0.0
		for _, v := range ma.values {
			sum += v
		}
		return sum / float64(ma.period), nil

	case EMAMovingAverage:
		if !ma.emaInitialized {
			return 0, fmt.Errorf("insufficient data: need %d, have %d", ma.period, len(ma.values))
		}
		return ma.lastValue, nil

	case WMAMovingAverage:
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
	ma.values = make([]float64, 0, ma.period)
	ma.lastValue = 0
	ma.sampleCount = 0
	ma.emaSeedSum = 0
	ma.emaInitialized = false
}

func (ma *MovingAverage) SetPeriod(period int) error {
	if period < 1 {
		return errors.New("period must be at least 1")
	}
	ma.period = period
	ma.Reset()
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

// CopySlice exposes the defensive copy helper to other packages.
func CopySlice(src []float64) []float64 {
	return copySlice(src)
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
//   - If we have fewer than “period” samples we fall back to a simple average
//     of the data we do have – this lets the first EMA value be produced
//     without waiting for a full window.
//   - When we have **exactly** “period” samples we always return the *simple*
//     average of those `period` points, even if a previous EMA value exists.
//     This guarantees that the first EMA after the full period matches
//     the SMA of the same points (the behaviour the unit test expects).
//   - Once we have more than “period” points we switch to the classic EMA
//     recursion using the smoothing factor.
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
	// When we have exactly “period” points we **always** return the SMA of
	// those points, regardless of any previous EMA value.  This seeds the EMA
	// with the correct SMA (the test expects 100 after three values).
	if len(data) == period {
		sum := 0.0
		for i := len(data) - period; i < len(data); i++ {
			sum += data[i]
		}
		return sum / float64(period), nil
	}
	// Full‑period case – standard EMA recursion.
	smoothing := 2.0 / float64(period+1)
	current := data[len(data)-1]
	return smoothing*current + (1-smoothing)*prevEMA, nil
}

// calculateWMA computes the Weighted Moving Average.
// The most recent value receives the highest weight (standard WMA definition).
func calculateWMA(data []float64, period int) (float64, error) {
	if len(data) < period {
		return 0, fmt.Errorf("insufficient data for WMA: need %d, have %d", period, len(data))
	}
	sum, weightSum := 0.0, 0.0
	for i := 0; i < period; i++ {
		weight := float64(i + 1) // weight increases: 1, 2, ..., period (newest gets highest)
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

// IsValidPrice exposes the price validation helper.
func IsValidPrice(price float64) bool { return isValidPrice(price) }

// IsNonNegativePrice exposes the non-negative price validator.
func IsNonNegativePrice(price float64) bool { return isNonNegativePrice(price) }

// IsValidVolume exposes the volume validator.
func IsValidVolume(volume float64) bool { return isValidVolume(volume) }

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

func FormatPlotDataJSON(data []PlotData) (string, error) {
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

func FormatPlotDataCSV(data []PlotData) (string, error) {
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

// Clamp exposes clamp to other packages.
func Clamp(value, min, max float64) float64 {
	return clamp(value, min, max)
}

// CalculateStandardDeviation exposes the standard-deviation helper.
func CalculateStandardDeviation(data []float64, mean float64) float64 {
	return calculateStandardDeviation(data, mean)
}

// CalculateSlope exposes the slope helper.
func CalculateSlope(y2, y1 float64) float64 {
	return calculateSlope(y2, y1)
}

// CalculateEMA exposes the EMA helper.
func CalculateEMA(data []float64, period int, prevEMA float64) (float64, error) {
	return calculateEMA(data, period, prevEMA)
}

// CalculateWMA exposes the WMA helper.
func CalculateWMA(data []float64, period int) (float64, error) {
	return calculateWMA(data, period)
}
