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
	SMA MovingAverageType = "SMA"
	EMA MovingAverageType = "EMA"
)

// MovingAverage calculates Simple or Exponential Moving Average
type MovingAverage struct {
	maType    MovingAverageType
	period    int
	values    []float64
	lastValue float64
}

// NewMovingAverage initializes a MovingAverage with specified type and period
func NewMovingAverage(maType MovingAverageType, period int) (*MovingAverage, error) {
	if period < 1 {
		return nil, errors.New("period must be at least 1")
	}
	if maType != SMA && maType != EMA {
		return nil, errors.New("invalid moving average type")
	}
	return &MovingAverage{
		maType: maType,
		period: period,
		values: make([]float64, 0, period),
	}, nil
}

// Add appends a new value to the moving average, enforcing non-negative values
func (ma *MovingAverage) Add(value float64) error {
	if !isNonNegativePrice(value) {
		return errors.New("invalid value")
	}
	ma.values = append(ma.values, value)
	if len(ma.values) >= ma.period {
		maValue, err := ma.calculate()
		if err != nil {
			return fmt.Errorf("calculate failed: %w", err)
		}
		ma.lastValue = maValue
	}
	ma.trimSlices()
	return nil
}

// AddValue appends a new value without enforcing non-negative constraint
func (ma *MovingAverage) AddValue(value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return errors.New("invalid value")
	}
	ma.values = append(ma.values, value)
	if len(ma.values) >= ma.period {
		maValue, err := ma.calculate()
		if err != nil {
			return fmt.Errorf("calculate failed: %w", err)
		}
		ma.lastValue = maValue
	}
	ma.trimSlices()
	return nil
}

// trimSlices limits slice size
func (ma *MovingAverage) trimSlices() {
	if len(ma.values) > ma.period {
		ma.values = ma.values[len(ma.values)-ma.period:]
	}
}

// calculate computes the moving average value
func (ma *MovingAverage) calculate() (float64, error) {
	if len(ma.values) < ma.period {
		return 0, fmt.Errorf("insufficient data: need %d, have %d", ma.period, len(ma.values))
	}
	if ma.maType == SMA {
		sum := 0.0
		for _, v := range ma.values[len(ma.values)-ma.period:] {
			sum += v
		}
		return sum / float64(ma.period), nil
	}
	// EMA calculation
	return calculateEMA(ma.values[len(ma.values)-ma.period:], ma.period, ma.lastValue)
}

// Calculate returns the current moving average value
func (ma *MovingAverage) Calculate() (float64, error) {
	if len(ma.values) == 0 || ma.lastValue == 0 {
		return 0, errors.New("no moving average data")
	}
	return ma.lastValue, nil
}

// Reset clears all data
func (ma *MovingAverage) Reset() {
	ma.values = ma.values[:0]
	ma.lastValue = 0
}

// SetPeriod updates the period
func (ma *MovingAverage) SetPeriod(period int) error {
	if period < 1 {
		return errors.New("period must be at least 1")
	}
	ma.period = period
	ma.trimSlices()
	return nil
}

// GetValues returns a copy of the values
func (ma *MovingAverage) GetValues() []float64 {
	return copySlice(ma.values)
}

// PlotData holds data for visualization
type PlotData struct {
	Name      string    `json:"name"`
	X         []float64 `json:"x"`
	Y         []float64 `json:"y"`
	Type      string    `json:"type,omitempty"`
	Signal    string    `json:"signal,omitempty"`
	Timestamp []int64   `json:"timestamp,omitempty"`
}

// copySlice creates a copy of a float64 slice
func copySlice(src []float64) []float64 {
	if src == nil {
		return nil
	}
	dst := make([]float64, len(src))
	copy(dst, src)
	return dst
}

// clamp restricts a value to a specified range
func clamp(value, min, max float64) float64 {
	if min == max {
		return min // Avoid invalid range
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// calculateStandardDeviation computes the standard deviation using Welford’s algorithm
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
	var sumSquaredDiff float64
	for _, v := range data {
		diff := v - mean
		sumSquaredDiff += diff * diff
	}
	if len(data) < 2 {
		return 0 // Avoid division by zero
	}
	return math.Sqrt(sumSquaredDiff / float64(len(data)-1))
}

// calculateEMA computes the Exponential Moving Average
func calculateEMA(data []float64, period int, prevEMA float64) (float64, error) {
	if len(data) < period {
		return 0, fmt.Errorf("insufficient data for EMA: need %d, have %d", period, len(data))
	}
	if len(data) == period && prevEMA == 0 {
		// Initial EMA is SMA
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

// calculateWMA computes the Weighted Moving Average
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

// isValidPrice checks if a price is valid (strictly positive)
func isValidPrice(price float64) bool {
	return price > 0 && !math.IsNaN(price) && !math.IsInf(price, 0)
}

// isNonNegativePrice checks if a price is valid (non-negative)
func isNonNegativePrice(price float64) bool {
	return price >= 0 && !math.IsNaN(price) && !math.IsInf(price, 0)
}

// isValidVolume checks if a volume is valid
func isValidVolume(volume float64) bool {
	return volume >= 0 && !math.IsNaN(volume) && !math.IsInf(volume, 0)
}

// GenerateTimestamps creates timestamps for plotting
func GenerateTimestamps(startTime int64, count int, interval int64) []int64 {
	if count <= 0 {
		return nil
	}
	timestamps := make([]int64, count)
	for i := 0; i < count; i++ {
		timestamps[i] = startTime + int64(i)*interval
	}
	return timestamps
}

// formatPlotDataJSON converts PlotData to JSON
func formatPlotDataJSON(data []PlotData) (string, error) {
	if len(data) == 0 {
		return "[]", nil
	}
	for _, d := range data {
		if len(d.X) != len(d.Y) {
			return "", fmt.Errorf("mismatched X and Y lengths for %s: %d vs %d", d.Name, len(d.X), len(d.Y))
		}
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal plot data: %w", err)
	}
	return string(jsonData), nil
}

// formatPlotDataCSV converts PlotData to CSV
func formatPlotDataCSV(data []PlotData) (string, error) {
	if len(data) == 0 {
		return "", nil
	}
	var builder strings.Builder
	builder.WriteString("Name,X,Y,Type,Signal,Timestamp\n")
	for _, d := range data {
		if len(d.X) != len(d.Y) {
			return "", fmt.Errorf("mismatched X and Y lengths for %s: %d vs %d", d.Name, len(d.X), len(d.Y))
		}
		for i := 0; i < len(d.X); i++ {
			timestamp := ""
			if i < len(d.Timestamp) {
				timestamp = fmt.Sprintf("%d", d.Timestamp[i])
			}
			fmt.Fprintf(&builder, "%s,%f,%f,%s,%s,%s\n", d.Name, d.X[i], d.Y[i], d.Type, d.Signal, timestamp)
		}
	}
	return builder.String(), nil
}

// calculateSlope computes the slope between two points
func calculateSlope(y2, y1 float64) float64 {
	return y2 - y1
}
