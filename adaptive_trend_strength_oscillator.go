// adaptive_trend_strength_oscillator.go
// artifact_id: 5f480763-62fd-4b69-a04c-f64a0d1f8f71
// artifact_version_id: 0b1c2d3e-4f5a-6b7c-8d9e-0f1a2b3c4d5e

package goti

import (
	"errors"
	"fmt"
	"math"
)

// AdaptiveTrendStrengthOscillator calculates the Adaptive Trend Strength Oscillator.
// It adapts its look‑back period based on recent volatility and smooths the
// result with an EMA.
type AdaptiveTrendStrengthOscillator struct {
	minPeriod        int
	maxPeriod        int
	volatilityPeriod int
	volSensitivity   float64
	highs            []float64
	lows             []float64
	closes           []float64
	atsoValues       []float64
	lastValue        float64
	ema              *MovingAverage
	config           IndicatorConfig
}

// NewAdaptiveTrendStrengthOscillator initializes with standard periods (2, 14, 14)
// and the default configuration.
func NewAdaptiveTrendStrengthOscillator() (*AdaptiveTrendStrengthOscillator, error) {
	// Expose the EMA period via config – callers can override it if needed.
	cfg := DefaultConfig()
	cfg.ATSEMAperiod = 5 // default EMA period for ATSO smoothing
	return NewAdaptiveTrendStrengthOscillatorWithParams(2, 14, 14, cfg)
}

// NewAdaptiveTrendStrengthOscillatorWithParams initializes with custom periods and config.
func NewAdaptiveTrendStrengthOscillatorWithParams(minPeriod, maxPeriod, volatilityPeriod int, config IndicatorConfig) (*AdaptiveTrendStrengthOscillator, error) {
	if minPeriod < 1 || maxPeriod < minPeriod || volatilityPeriod < 1 {
		return nil, errors.New("invalid periods")
	}
	ema, err := NewMovingAverage(EMA, config.ATSEMAperiod)
	if err != nil {
		return nil, fmt.Errorf("failed to create EMA: %w", err)
	}
	return &AdaptiveTrendStrengthOscillator{
		minPeriod:        minPeriod,
		maxPeriod:        maxPeriod,
		volatilityPeriod: volatilityPeriod,
		volSensitivity:   2.0,
		highs:            make([]float64, 0, maxPeriod+volatilityPeriod+1),
		lows:             make([]float64, 0, maxPeriod+volatilityPeriod+1),
		closes:           make([]float64, 0, maxPeriod+volatilityPeriod+1),
		atsoValues:       make([]float64, 0, maxPeriod),
		ema:              ema,
		config:           config,
	}, nil
}

// Add appends new price data (high, low, close) to the oscillator.
func (atso *AdaptiveTrendStrengthOscillator) Add(high, low, close float64) error {
	if high < low || !isNonNegativePrice(close) {
		return errors.New("invalid price")
	}
	atso.highs = append(atso.highs, high)
	atso.lows = append(atso.lows, low)
	atso.closes = append(atso.closes, close)

	if len(atso.closes) >= atso.maxPeriod+atso.volatilityPeriod+1 {
		atsoValue, err := atso.calculateATSO()
		if err != nil {
			return fmt.Errorf("calculateATSO failed: %w", err)
		}
		if err := atso.ema.AddValue(atsoValue); err != nil {
			return fmt.Errorf("ema.AddValue failed: %w", err)
		}
		smoothedValue, err := atso.ema.Calculate()
		if err == nil {
			atso.atsoValues = append(atso.atsoValues, smoothedValue)
			atso.lastValue = smoothedValue
		} else {
			atso.atsoValues = append(atso.atsoValues, atsoValue)
			atso.lastValue = atsoValue
		}
	}
	atso.trimSlices()
	return nil
}

// trimSlices limits the size of internal slices to avoid unbounded growth.
func (atso *AdaptiveTrendStrengthOscillator) trimSlices() {
	capacity := atso.maxPeriod + atso.volatilityPeriod + 1
	if len(atso.closes) > capacity {
		atso.highs = trimTail(atso.highs, capacity)
		atso.lows = trimTail(atso.lows, capacity)
		atso.closes = trimTail(atso.closes, capacity)
	}
	if len(atso.atsoValues) > atso.maxPeriod {
		atso.atsoValues = trimTail(atso.atsoValues, atso.maxPeriod)
	}
}

// calculateATSO orchestrates the three‑step computation:
// 1️⃣ adaptive period based on volatility,
// 2️⃣ raw trend‑strength for the current window,
// 3️⃣ normalization against historic averages.
func (atso *AdaptiveTrendStrengthOscillator) calculateATSO() (float64, error) {
	period, err := atso.adaptivePeriod()
	if err != nil {
		return 0, err
	}

	curStrength, err := atso.trendStrength(period)
	if err != nil {
		return 0, err
	}

	return atso.normalize(curStrength, period)
}

// adaptivePeriod determines the look‑back period adjusted for recent volatility.
func (atso *AdaptiveTrendStrengthOscillator) adaptivePeriod() (int, error) {
	if len(atso.closes) < atso.maxPeriod+atso.volatilityPeriod+1 {
		return 0, fmt.Errorf(
			"insufficient data for volatility: need %d, have %d",
			atso.maxPeriod+atso.volatilityPeriod+1, len(atso.closes),
		)
	}
	startIdx := len(atso.closes) - atso.volatilityPeriod
	volatility := calculateStandardDeviation(atso.closes[startIdx:], 0)
	if volatility == 0 {
		volatility = 0.001
	}
	// Linear interpolation between minPeriod and maxPeriod based on volatility.
	p := atso.minPeriod + int(math.Min(volatility*atso.volSensitivity, 1)*float64(atso.maxPeriod-atso.minPeriod))
	if p > atso.maxPeriod {
		p = atso.maxPeriod
	}
	if p < atso.minPeriod {
		p = atso.minPeriod
	}
	return p, nil
}

// trendStrength computes the raw (un‑scaled) strength for a given look‑back period.
func (atso *AdaptiveTrendStrengthOscillator) trendStrength(period int) (float64, error) {
	if len(atso.closes) < period+1 {
		return 0, fmt.Errorf("insufficient data for period %d: have %d", period+1, len(atso.closes))
	}
	startIdx := len(atso.highs) - period
	highs := atso.highs[startIdx:]
	lows := atso.lows[startIdx:]

	sumUp, sumDown := 0.0, 0.0
	for i := 1; i < period; i++ {
		highDiff := highs[i] - highs[i-1]
		lowDiff := lows[i-1] - lows[i]
		if highDiff > lowDiff && highDiff > 0 {
			sumUp += highDiff
		}
		if lowDiff > highDiff && lowDiff > 0 {
			sumDown += lowDiff
		}
	}
	trendStrength := (sumUp - sumDown) / float64(period)
	if trendStrength == 0 {
		trendStrength = 0.001
	}
	// Preserve the original scaling factor (7000) used by the author.
	return trendStrength * 7000, nil
}

// normalize scales the raw strength against a historic average, clamps the result,
// and returns the final ATSO value.
func (atso *AdaptiveTrendStrengthOscillator) normalize(raw float64, period int) (float64, error) {
	// Compute historic average trend‑strength.
	var avg float64
	var count int
	for i := period; i < len(atso.closes); i++ {
		var up, down float64
		for j := 1; j < period; j++ {
			hi := atso.highs[i-j] - atso.highs[i-j-1]
			lo := atso.lows[i-j-1] - atso.lows[i-j]
			if hi > lo && hi > 0 {
				up += hi
			}
			if lo > hi && lo > 0 {
				down += lo
			}
		}
		if up != 0 || down != 0 {
			avg += (up - down) / float64(period)
			count++
		}
	}
	if count == 0 {
		// No historic data to compare against – fall back to the raw value.
		return raw, nil
	}
	avg /= float64(count)
	if avg == 0 {
		return 0, fmt.Errorf("average trend strength is zero – cannot normalize")
	}
	return clamp((raw/avg-1)*100, -100, 100), nil
}

// Calculate returns the most recent ATSO value (smoothed by EMA).
func (atso *AdaptiveTrendStrengthOscillator) Calculate() (float64, error) {
	if len(atso.atsoValues) == 0 {
		return 0, errors.New("no ATSO data")
	}
	return atso.lastValue, nil
}

// GetLastValue returns the last (unsmoothed) ATSO value.
func (atso *AdaptiveTrendStrengthOscillator) GetLastValue() float64 {
	return atso.lastValue
}

// IsBullishCrossover checks whether ATSO crossed above zero on the latest update.
func (atso *AdaptiveTrendStrengthOscillator) IsBullishCrossover() (bool, error) {
	if len(atso.atsoValues) < 2 {
		return false, errors.New("insufficient data for crossover")
	}
	current := atso.atsoValues[len(atso.atsoValues)-1]
	previous := atso.atsoValues[len(atso.atsoValues)-2]
	return previous <= 0 && current > 0, nil
}

// IsBearishCrossover checks whether ATSO crossed below zero on the latest update.
func (atso *AdaptiveTrendStrengthOscillator) IsBearishCrossover() (bool, error) {
	if len(atso.atsoValues) < 2 {
		return false, errors.New("insufficient data for crossover")
	}
	current := atso.atsoValues[len(atso.atsoValues)-1]
	previous := atso.atsoValues[len(atso.atsoValues)-2]
	return previous >= 0 && current < 0, nil
}

// SetVolatilitySensitivity adjusts the volatility impact on period adaptation.
func (atso *AdaptiveTrendStrengthOscillator) SetVolatilitySensitivity(sensitivity float64) error {
	if sensitivity <= 0 {
		return errors.New("sensitivity must be positive")
	}
	atso.volSensitivity = sensitivity
	return nil
}

// Reset clears all stored price data and EMA state.
func (atso *AdaptiveTrendStrengthOscillator) Reset() {
	atso.highs = atso.highs[:0]
	atso.lows = atso.lows[:0]
	atso.closes = atso.closes[:0]
	atso.atsoValues = atso.atsoValues[:0]
	atso.ema.Reset()
	atso.lastValue = 0
}

// SetPeriods updates the look‑back periods. The EMA period remains fixed
// (configurable via IndicatorConfig.ATSEMAperiod).
func (atso *AdaptiveTrendStrengthOscillator) SetPeriods(minPeriod, maxPeriod, volatilityPeriod int) error {
	if minPeriod < 1 || maxPeriod < minPeriod || volatilityPeriod < 1 {
		return errors.New("invalid periods")
	}
	atso.minPeriod, atso.maxPeriod, atso.volatilityPeriod = minPeriod, maxPeriod, volatilityPeriod
	// EMA period is unchanged; we only need to trim slices to match new windows.
	atso.trimSlices()
	return nil
}

// GetHighs returns a copy of the stored high prices.
func (atso *AdaptiveTrendStrengthOscillator) GetHighs() []float64 {
	return copySlice(atso.highs)
}

// GetLows returns a copy of the stored low prices.
func (atso *AdaptiveTrendStrengthOscillator) GetLows() []float64 {
	return copySlice(atso.lows)
}

// GetCloses returns a copy of the stored close prices.
func (atso *AdaptiveTrendStrengthOscillator) GetCloses() []float64 {
	return copySlice(atso.closes)
}

// GetATSOValues returns a copy of the computed ATSO values.
func (atso *AdaptiveTrendStrengthOscillator) GetATSOValues() []float64 {
	return copySlice(atso.atsoValues)
}

// GetPlotData prepares data for visualisation, optionally annotating bullish/bearish signals.
func (atso *AdaptiveTrendStrengthOscillator) GetPlotData(startTime, interval int64) []PlotData {
	var plotData []PlotData
	if len(atso.atsoValues) > 0 {
		x := make([]float64, len(atso.atsoValues))
		signals := make([]float64, len(atso.atsoValues))
		timestamps := GenerateTimestamps(startTime, len(atso.atsoValues), interval)

		for i := range atso.atsoValues {
			x[i] = float64(i)
			if i > 0 {
				if atso.atsoValues[i-1] <= 0 && atso.atsoValues[i] > 0 {
					signals[i] = 1 // bullish
				} else if atso.atsoValues[i-1] >= 0 && atso.atsoValues[i] < 0 {
					signals[i] = -1 // bearish
				}
			}
		}
		plotData = append(plotData, PlotData{
			Name:      "Adaptive Trend Strength Oscillator",
			X:         x,
			Y:         atso.atsoValues,
			Type:      "line",
			Timestamp: timestamps,
		})
		plotData = append(plotData, PlotData{
			Name:      "Signals",
			X:         x,
			Y:         signals,
			Type:      "scatter",
			Timestamp: timestamps,
		})
	}
	return plotData
}

/*
   -------------------------------------------------------------------------
   Helper utilities that are generic enough to live in this file.
   They are deliberately small to keep the core logic readable.
   -------------------------------------------------------------------------
*/

// trimTail returns the last `maxLen` elements of a slice (or the whole slice if shorter).
func trimTail[T any](s []T, maxLen int) []T {
	if len(s) > maxLen {
		return s[len(s)-maxLen:]
	}
	return s
}
