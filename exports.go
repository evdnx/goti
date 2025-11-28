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

// ---- MACD ----
type MACD = indicator.MACD

func NewMACD() (*indicator.MACD, error) {
	return indicator.NewMACD()
}

func NewMACDWithParams(fastPeriod, slowPeriod, signalPeriod int) (*indicator.MACD, error) {
	return indicator.NewMACDWithParams(fastPeriod, slowPeriod, signalPeriod)
}

// ---- Stochastic Oscillator ----
type StochasticOscillator = indicator.StochasticOscillator

func NewStochasticOscillator() (*indicator.StochasticOscillator, error) {
	return indicator.NewStochasticOscillator()
}

func NewStochasticOscillatorWithParams(kPeriod, dPeriod int) (*indicator.StochasticOscillator, error) {
	return indicator.NewStochasticOscillatorWithParams(kPeriod, dPeriod)
}

// ---- Commodity Channel Index ----
type CommodityChannelIndex = indicator.CommodityChannelIndex

func NewCommodityChannelIndex() (*indicator.CommodityChannelIndex, error) {
	return indicator.NewCommodityChannelIndex()
}

func NewCommodityChannelIndexWithParams(period int) (*indicator.CommodityChannelIndex, error) {
	return indicator.NewCommodityChannelIndexWithParams(period)
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

// ---- VWAP ----
type VWAP = indicator.VWAP

func NewVWAP() *indicator.VWAP {
	return indicator.NewVWAP()
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

// ---- Parabolic SAR ----
type ParabolicSAR = indicator.ParabolicSAR

func NewParabolicSAR() (*indicator.ParabolicSAR, error) {
	return indicator.NewParabolicSAR()
}

func NewParabolicSARWithParams(step, maxStep float64) (*indicator.ParabolicSAR, error) {
	return indicator.NewParabolicSARWithParams(step, maxStep)
}

// ---- Average True Range ----
type AverageTrueRange = indicator.AverageTrueRange
type ATROption = indicator.ATROption
type BollingerBands = indicator.BollingerBands

func WithCloseValidation(enabled bool) indicator.ATROption {
	return indicator.WithCloseValidation(enabled)
}

func NewAverageTrueRange() (*indicator.AverageTrueRange, error) {
	return indicator.NewAverageTrueRange()
}

func NewAverageTrueRangeWithParams(period int, opts ...indicator.ATROption) (*indicator.AverageTrueRange, error) {
	return indicator.NewAverageTrueRangeWithParams(period, opts...)
}

func NewBollingerBands() (*indicator.BollingerBands, error) {
	return indicator.NewBollingerBands()
}

func NewBollingerBandsWithParams(period int, multiplier float64) (*indicator.BollingerBands, error) {
	return indicator.NewBollingerBandsWithParams(period, multiplier)
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
type ScalpingIndicatorSuite = suite.ScalpingIndicatorSuite
type IndicatorSuite = suite.ScalpingIndicatorSuite

func NewScalpingIndicatorSuite() (*suite.ScalpingIndicatorSuite, error) {
	return suite.NewScalpingIndicatorSuite()
}

func NewScalpingIndicatorSuiteWithConfig(cfg config.IndicatorConfig) (*suite.ScalpingIndicatorSuite, error) {
	return suite.NewScalpingIndicatorSuiteWithConfig(cfg)
}

// Backwards-compatible aliases for callers expecting the old names.
func NewIndicatorSuite() (*suite.ScalpingIndicatorSuite, error) {
	return NewScalpingIndicatorSuite()
}

func NewIndicatorSuiteWithConfig(cfg config.IndicatorConfig) (*suite.ScalpingIndicatorSuite, error) {
	return NewScalpingIndicatorSuiteWithConfig(cfg)
}
