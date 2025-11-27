package indicator

import (
	"github.com/evdnx/goti/config"
	"github.com/evdnx/goti/indicator/core"
	"github.com/evdnx/goti/indicator/momentum"
	"github.com/evdnx/goti/indicator/trend"
	"github.com/evdnx/goti/indicator/volatility"
	"github.com/evdnx/goti/indicator/volume"
)

// ---- Shared data helpers ----
type PlotData = core.PlotData

func GenerateTimestamps(startTime int64, count int, interval int64) []int64 {
	return core.GenerateTimestamps(startTime, count, interval)
}

func FormatPlotDataJSON(data []PlotData) (string, error) {
	return core.FormatPlotDataJSON(data)
}

func FormatPlotDataCSV(data []PlotData) (string, error) {
	return core.FormatPlotDataCSV(data)
}

// ---- Moving averages & utilities ----
type MovingAverageType = core.MovingAverageType

const (
	EMAMovingAverage MovingAverageType = core.EMAMovingAverage
	SMAMovingAverage MovingAverageType = core.SMAMovingAverage
	WMAMovingAverage MovingAverageType = core.WMAMovingAverage
)

type MovingAverage = core.MovingAverage

func NewMovingAverage(maType MovingAverageType, period int) (*core.MovingAverage, error) {
	return core.NewMovingAverage(maType, period)
}

func KeepLast[T any](s []T, n int) []T { return core.KeepLast(s, n) }

func Clamp(value, min, max float64) float64 { return core.Clamp(value, min, max) }
func CalculateSlope(y2, y1 float64) float64 { return core.CalculateSlope(y2, y1) }
func CalculateStandardDeviation(data []float64, mean float64) float64 {
	return core.CalculateStandardDeviation(data, mean)
}
func CalculateEMA(data []float64, period int, prevEMA float64) (float64, error) {
	return core.CalculateEMA(data, period, prevEMA)
}
func CalculateWMA(data []float64, period int) (float64, error) {
	return core.CalculateWMA(data, period)
}

func IsValidPrice(price float64) bool       { return core.IsValidPrice(price) }
func IsNonNegativePrice(price float64) bool { return core.IsNonNegativePrice(price) }
func IsValidVolume(volume float64) bool     { return core.IsValidVolume(volume) }

// ---- Momentum indicators ----
type RelativeStrengthIndex = momentum.RelativeStrengthIndex
type MACD = momentum.MACD
type StochasticOscillator = momentum.StochasticOscillator
type CommodityChannelIndex = momentum.CommodityChannelIndex

func NewRelativeStrengthIndex() (*momentum.RelativeStrengthIndex, error) {
	return momentum.NewRelativeStrengthIndex()
}

func NewRelativeStrengthIndexWithParams(period int, cfg config.IndicatorConfig) (*momentum.RelativeStrengthIndex, error) {
	return momentum.NewRelativeStrengthIndexWithParams(period, cfg)
}

type AdaptiveDEMAMomentumOscillator = momentum.AdaptiveDEMAMomentumOscillator

const (
	DefaultLength      = momentum.DefaultLength
	DefaultStdevLength = momentum.DefaultStdevLength
	DefaultStdWeight   = momentum.DefaultStdWeight
)

var (
	ErrInsufficientData = momentum.ErrInsufficientData
	ErrInvalidParams    = momentum.ErrInvalidParams
)

func EMASmoothingFactor(n int) float64 { return momentum.EMASmoothingFactor(n) }

func NewAdaptiveDEMAMomentumOscillator() (*momentum.AdaptiveDEMAMomentumOscillator, error) {
	return momentum.NewAdaptiveDEMAMomentumOscillator()
}

func NewAdaptiveDEMAMomentumOscillatorWithParams(length, stdevLength int, stdWeight float64, cfg config.IndicatorConfig) (*momentum.AdaptiveDEMAMomentumOscillator, error) {
	return momentum.NewAdaptiveDEMAMomentumOscillatorWithParams(length, stdevLength, stdWeight, cfg)
}

func NewMACD() (*momentum.MACD, error) {
	return momentum.NewMACD()
}

func NewMACDWithParams(fastPeriod, slowPeriod, signalPeriod int) (*momentum.MACD, error) {
	return momentum.NewMACDWithParams(fastPeriod, slowPeriod, signalPeriod)
}

func NewStochasticOscillator() (*momentum.StochasticOscillator, error) {
	return momentum.NewStochasticOscillator()
}

func NewStochasticOscillatorWithParams(kPeriod, dPeriod int) (*momentum.StochasticOscillator, error) {
	return momentum.NewStochasticOscillatorWithParams(kPeriod, dPeriod)
}

func NewCommodityChannelIndex() (*momentum.CommodityChannelIndex, error) {
	return momentum.NewCommodityChannelIndex()
}

func NewCommodityChannelIndexWithParams(period int) (*momentum.CommodityChannelIndex, error) {
	return momentum.NewCommodityChannelIndexWithParams(period)
}

// ---- Trend indicators ----
type HullMovingAverage = trend.HullMovingAverage
type ParabolicSAR = trend.ParabolicSAR

var (
	ErrInvalidPrice          = trend.ErrInvalidPrice
	ErrInsufficientHMAData   = trend.ErrInsufficientHMAData
	ErrInsufficientCrossData = trend.ErrInsufficientCrossData
)

func NewHullMovingAverage() (*trend.HullMovingAverage, error) {
	return trend.NewHullMovingAverage()
}

func NewHullMovingAverageWithParams(period int) (*trend.HullMovingAverage, error) {
	return trend.NewHullMovingAverageWithParams(period)
}

type VolumeWeightedAroonOscillator = trend.VolumeWeightedAroonOscillator

func NewVolumeWeightedAroonOscillator() (*trend.VolumeWeightedAroonOscillator, error) {
	return trend.NewVolumeWeightedAroonOscillator()
}

func NewVolumeWeightedAroonOscillatorWithParams(period int, cfg config.IndicatorConfig) (*trend.VolumeWeightedAroonOscillator, error) {
	return trend.NewVolumeWeightedAroonOscillatorWithParams(period, cfg)
}

type AdaptiveTrendStrengthOscillator = trend.AdaptiveTrendStrengthOscillator

func NewAdaptiveTrendStrengthOscillator() (*trend.AdaptiveTrendStrengthOscillator, error) {
	return trend.NewAdaptiveTrendStrengthOscillator()
}

func NewAdaptiveTrendStrengthOscillatorWithParams(shortPeriod, longPeriod, volatilityPeriod int, cfg config.IndicatorConfig) (*trend.AdaptiveTrendStrengthOscillator, error) {
	return trend.NewAdaptiveTrendStrengthOscillatorWithParams(shortPeriod, longPeriod, volatilityPeriod, cfg)
}

func NewParabolicSAR() (*trend.ParabolicSAR, error) {
	return trend.NewParabolicSAR()
}

func NewParabolicSARWithParams(step, maxStep float64) (*trend.ParabolicSAR, error) {
	return trend.NewParabolicSARWithParams(step, maxStep)
}

// ---- Volume indicators ----
type MoneyFlowIndex = volume.MoneyFlowIndex
type VWAP = volume.VWAP

var (
	ErrNoMFIData            = volume.ErrNoMFIData
	ErrInsufficientDataCalc = volume.ErrInsufficientDataCalc
)

func NewMoneyFlowIndex() (*volume.MoneyFlowIndex, error) {
	return volume.NewMoneyFlowIndex()
}

func NewMoneyFlowIndexWithParams(period int, cfg config.IndicatorConfig) (*volume.MoneyFlowIndex, error) {
	return volume.NewMoneyFlowIndexWithParams(period, cfg)
}

func NewVWAP() *volume.VWAP {
	return volume.NewVWAP()
}

// ---- Volatility indicators ----
type AverageTrueRange = volatility.AverageTrueRange
type ATROption = volatility.ATROption
type BollingerBands = volatility.BollingerBands

func WithCloseValidation(enabled bool) volatility.ATROption {
	return volatility.WithCloseValidation(enabled)
}

func NewAverageTrueRange() (*volatility.AverageTrueRange, error) {
	return volatility.NewAverageTrueRange()
}

func NewAverageTrueRangeWithParams(period int, opts ...volatility.ATROption) (*volatility.AverageTrueRange, error) {
	return volatility.NewAverageTrueRangeWithParams(period, opts...)
}

func NewBollingerBands() (*volatility.BollingerBands, error) {
	return volatility.NewBollingerBands()
}

func NewBollingerBandsWithParams(period int, multiplier float64) (*volatility.BollingerBands, error) {
	return volatility.NewBollingerBandsWithParams(period, multiplier)
}
