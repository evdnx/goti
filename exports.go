package goti

import (
	"github.com/evdnx/goti/config"
	"github.com/evdnx/goti/indicator"
	"github.com/evdnx/goti/suite"
)

// ---- Shared data helpers ----
type PlotData = indicator.PlotData

func GenerateTimestamps(startTime int64, count int, interval int64) []int64 {
	return indicator.GenerateTimestamps(startTime, count, interval)
}

func FormatPlotDataJSON(data []indicator.PlotData) (string, error) {
	return indicator.FormatPlotDataJSON(data)
}

func FormatPlotDataCSV(data []indicator.PlotData) (string, error) {
	return indicator.FormatPlotDataCSV(data)
}

// ---- Moving averages ----
type MovingAverageType = indicator.MovingAverageType

const (
	EMAMovingAverage MovingAverageType = indicator.EMAMovingAverage
	SMAMovingAverage MovingAverageType = indicator.SMAMovingAverage
	WMAMovingAverage MovingAverageType = indicator.WMAMovingAverage
)

type MovingAverage = indicator.MovingAverage

func NewMovingAverage(maType indicator.MovingAverageType, period int) (*indicator.MovingAverage, error) {
	return indicator.NewMovingAverage(maType, period)
}

// ---- RSI ----
type RelativeStrengthIndex = indicator.RelativeStrengthIndex

func NewRelativeStrengthIndex() (*indicator.RelativeStrengthIndex, error) {
	return indicator.NewRelativeStrengthIndex()
}

func NewRelativeStrengthIndexWithParams(period int, cfg config.IndicatorConfig) (*indicator.RelativeStrengthIndex, error) {
	return indicator.NewRelativeStrengthIndexWithParams(period, cfg)
}

// ---- Money Flow Index ----
type MoneyFlowIndex = indicator.MoneyFlowIndex

var (
	ErrNoMFIData            = indicator.ErrNoMFIData
	ErrInsufficientDataCalc = indicator.ErrInsufficientDataCalc
)

func NewMoneyFlowIndex() (*indicator.MoneyFlowIndex, error) {
	return indicator.NewMoneyFlowIndex()
}

func NewMoneyFlowIndexWithParams(period int, cfg config.IndicatorConfig) (*indicator.MoneyFlowIndex, error) {
	return indicator.NewMoneyFlowIndexWithParams(period, cfg)
}

// ---- Volume Weighted Aroon Oscillator ----
type VolumeWeightedAroonOscillator = indicator.VolumeWeightedAroonOscillator

func NewVolumeWeightedAroonOscillator() (*indicator.VolumeWeightedAroonOscillator, error) {
	return indicator.NewVolumeWeightedAroonOscillator()
}

func NewVolumeWeightedAroonOscillatorWithParams(period int, cfg config.IndicatorConfig) (*indicator.VolumeWeightedAroonOscillator, error) {
	return indicator.NewVolumeWeightedAroonOscillatorWithParams(period, cfg)
}

// ---- Hull Moving Average ----
type HullMovingAverage = indicator.HullMovingAverage

func NewHullMovingAverage() (*indicator.HullMovingAverage, error) {
	return indicator.NewHullMovingAverage()
}

func NewHullMovingAverageWithParams(period int) (*indicator.HullMovingAverage, error) {
	return indicator.NewHullMovingAverageWithParams(period)
}

// ---- Average True Range ----
type AverageTrueRange = indicator.AverageTrueRange
type ATROption = indicator.ATROption

func WithCloseValidation(enabled bool) indicator.ATROption {
	return indicator.WithCloseValidation(enabled)
}

func NewAverageTrueRange() (*indicator.AverageTrueRange, error) {
	return indicator.NewAverageTrueRange()
}

func NewAverageTrueRangeWithParams(period int, opts ...indicator.ATROption) (*indicator.AverageTrueRange, error) {
	return indicator.NewAverageTrueRangeWithParams(period, opts...)
}

// ---- Adaptive DEMA Momentum Oscillator ----
type AdaptiveDEMAMomentumOscillator = indicator.AdaptiveDEMAMomentumOscillator

const (
	DefaultLength      = indicator.DefaultLength
	DefaultStdevLength = indicator.DefaultStdevLength
	DefaultStdWeight   = indicator.DefaultStdWeight
)

var (
	ErrInsufficientData = indicator.ErrInsufficientData
	ErrInvalidParams    = indicator.ErrInvalidParams
)

func EMASmoothingFactor(n int) float64 { return indicator.EMASmoothingFactor(n) }

func NewAdaptiveDEMAMomentumOscillator() (*indicator.AdaptiveDEMAMomentumOscillator, error) {
	return indicator.NewAdaptiveDEMAMomentumOscillator()
}

func NewAdaptiveDEMAMomentumOscillatorWithParams(length, stdevLength int, stdWeight float64, cfg config.IndicatorConfig) (*indicator.AdaptiveDEMAMomentumOscillator, error) {
	return indicator.NewAdaptiveDEMAMomentumOscillatorWithParams(length, stdevLength, stdWeight, cfg)
}

// ---- Adaptive Trend Strength Oscillator ----
type AdaptiveTrendStrengthOscillator = indicator.AdaptiveTrendStrengthOscillator

func NewAdaptiveTrendStrengthOscillator() (*indicator.AdaptiveTrendStrengthOscillator, error) {
	return indicator.NewAdaptiveTrendStrengthOscillator()
}

func NewAdaptiveTrendStrengthOscillatorWithParams(shortPeriod, longPeriod, volatilityPeriod int, cfg config.IndicatorConfig) (*indicator.AdaptiveTrendStrengthOscillator, error) {
	return indicator.NewAdaptiveTrendStrengthOscillatorWithParams(shortPeriod, longPeriod, volatilityPeriod, cfg)
}

// ---- Indicator suite ----
type IndicatorSuite = suite.IndicatorSuite

func NewIndicatorSuite() (*suite.IndicatorSuite, error) {
	return suite.NewIndicatorSuite()
}

func NewIndicatorSuiteWithConfig(cfg config.IndicatorConfig) (*suite.IndicatorSuite, error) {
	return suite.NewIndicatorSuiteWithConfig(cfg)
}
