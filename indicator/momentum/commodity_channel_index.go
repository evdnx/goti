package momentum

import (
	"errors"
	"math"

	"github.com/evdnx/goti/indicator/core"
)

const (
	DefaultCCIPeriod     = 20
	DefaultCCIOverbought = 100.0
	DefaultCCIOversold   = -100.0
	cciConstant          = 0.015
)

// CommodityChannelIndex implements the CCI indicator.
// It uses typical price [(H+L+C)/3], a simple moving average of typical prices,
// and the mean deviation around that average.
type CommodityChannelIndex struct {
	period int

	typicalPrices []float64
	cciValues     []float64
	lastValue     float64
}

// NewCommodityChannelIndex builds a CCI with the default 20-period window.
func NewCommodityChannelIndex() (*CommodityChannelIndex, error) {
	return NewCommodityChannelIndexWithParams(DefaultCCIPeriod)
}

// NewCommodityChannelIndexWithParams allows a custom period.
func NewCommodityChannelIndexWithParams(period int) (*CommodityChannelIndex, error) {
	if period < 1 {
		return nil, errors.New("period must be at least 1")
	}
	return &CommodityChannelIndex{
		period:        period,
		typicalPrices: make([]float64, 0, period),
		cciValues:     make([]float64, 0, period),
	}, nil
}

// Add ingests a new OHLC bar and updates the CCI when enough data exists.
func (c *CommodityChannelIndex) Add(high, low, close float64) error {
	if high < low || !core.IsNonNegativePrice(close) {
		return errors.New("invalid price data")
	}
	tp := (high + low + close) / 3
	c.typicalPrices = append(c.typicalPrices, tp)

	if len(c.typicalPrices) >= c.period {
		c.lastValue = c.computeCCI()
		c.cciValues = append(c.cciValues, c.lastValue)
	}
	c.trimSlices()
	return nil
}

// Calculate returns the most recent CCI value.
func (c *CommodityChannelIndex) Calculate() (float64, error) {
	if len(c.cciValues) == 0 {
		return 0, errors.New("no CCI data")
	}
	return c.lastValue, nil
}

// IsOverbought reports whether CCI is above +100.
func (c *CommodityChannelIndex) IsOverbought() (bool, error) {
	if len(c.cciValues) == 0 {
		return false, errors.New("no CCI data")
	}
	return c.lastValue > DefaultCCIOverbought, nil
}

// IsOversold reports whether CCI is below -100.
func (c *CommodityChannelIndex) IsOversold() (bool, error) {
	if len(c.cciValues) == 0 {
		return false, errors.New("no CCI data")
	}
	return c.lastValue < DefaultCCIOversold, nil
}

// Reset clears all stored data.
func (c *CommodityChannelIndex) Reset() {
	c.typicalPrices = c.typicalPrices[:0]
	c.cciValues = c.cciValues[:0]
	c.lastValue = 0
}

// SetPeriod updates the lookback window and resets the indicator.
func (c *CommodityChannelIndex) SetPeriod(period int) error {
	if period < 1 {
		return errors.New("period must be at least 1")
	}
	c.period = period
	c.Reset()
	return nil
}

// GetValues returns the CCI series (defensive copy).
func (c *CommodityChannelIndex) GetValues() []float64 { return core.CopySlice(c.cciValues) }

// GetPlotData returns plot data for the CCI line.
func (c *CommodityChannelIndex) GetPlotData(startTime, interval int64) []core.PlotData {
	if len(c.cciValues) == 0 {
		return nil
	}
	x := make([]float64, len(c.cciValues))
	for i := range x {
		x[i] = float64(i)
	}
	ts := core.GenerateTimestamps(startTime, len(c.cciValues), interval)
	return []core.PlotData{{
		Name:      "CCI",
		X:         x,
		Y:         c.cciValues,
		Type:      "line",
		Timestamp: ts,
	}}
}

func (c *CommodityChannelIndex) computeCCI() float64 {
	start := len(c.typicalPrices) - c.period
	window := c.typicalPrices[start:]

	sum := 0.0
	for _, v := range window {
		sum += v
	}
	ma := sum / float64(c.period)

	var devSum float64
	for _, v := range window {
		devSum += math.Abs(v - ma)
	}
	meanDev := devSum / float64(c.period)
	if meanDev == 0 {
		return 0
	}
	return (window[len(window)-1] - ma) / (cciConstant * meanDev)
}

func (c *CommodityChannelIndex) trimSlices() {
	c.typicalPrices = core.KeepLast(c.typicalPrices, c.period)
	c.cciValues = core.KeepLast(c.cciValues, c.period)
}
