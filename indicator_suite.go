package goti

import (
	"fmt"
)

// IndicatorSuite combines multiple indicators for unified signals
type IndicatorSuite struct {
	amdo    *AdaptiveMomentumDivergenceOscillator
	atso    *AdaptiveTrendStrengthOscillator
	vwao    *VolumeWeightedAroonOscillator
	hma     *HullMovingAverage
	mfi     *MoneyFlowIndex
	rsi     *RelativeStrengthIndex
	atr     *AverageTrueRange
	config  IndicatorConfig
	weights map[string]float64
}

// NewIndicatorSuite initializes the suite with default parameters and weights
func NewIndicatorSuite() (*IndicatorSuite, error) {
	return NewIndicatorSuiteWithConfig(DefaultConfig())
}

// NewIndicatorSuiteWithConfig initializes with custom config
func NewIndicatorSuiteWithConfig(config IndicatorConfig) (*IndicatorSuite, error) {
	amdo, err := NewAdaptiveMomentumDivergenceOscillatorWithParams(5, 14, 14, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create AMDO: %w", err)
	}
	atso, err := NewAdaptiveTrendStrengthOscillatorWithParams(5, 14, 14, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create ATSO: %w", err)
	}
	vwao, err := NewVolumeWeightedAroonOscillatorWithParams(14, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create VWAO: %w", err)
	}
	hma, err := NewHullMovingAverageWithParams(9)
	if err != nil {
		return nil, fmt.Errorf("failed to create HMA: %w", err)
	}
	mfi, err := NewMoneyFlowIndexWithParams(14, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create MFI: %w", err)
	}
	rsi, err := NewRelativeStrengthIndexWithParams(14, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create RSI: %w", err)
	}
	atr, err := NewAverageTrueRangeWithParams(14)
	if err != nil {
		return nil, fmt.Errorf("failed to create ATR: %w", err)
	}
	return &IndicatorSuite{
		amdo:   amdo,
		atso:   atso,
		vwao:   vwao,
		hma:    hma,
		mfi:    mfi,
		rsi:    rsi,
		atr:    atr,
		config: config,
		weights: map[string]float64{
			"amdo": 1.0,
			"atso": 1.0,
			"vwao": 1.0,
			"hma":  1.5, // Higher weight for trend
			"mfi":  1.2, // Slightly higher for volume
			"rsi":  1.0,
			// ATR not included in weights as it's non-directional
		},
	}, nil
}

// Add appends new data to all indicators
func (suite *IndicatorSuite) Add(high, low, close, volume float64) error {
	if err := suite.amdo.Add(close); err != nil {
		return fmt.Errorf("AMDO add failed: %w", err)
	}
	if err := suite.atso.Add(high, low, close); err != nil {
		return fmt.Errorf("ATSO add failed: %w", err)
	}
	if err := suite.vwao.Add(high, low, close, volume); err != nil {
		return fmt.Errorf("VWAO add failed: %w", err)
	}
	if err := suite.hma.Add(close); err != nil {
		return fmt.Errorf("HMA add failed: %w", err)
	}
	if err := suite.mfi.Add(high, low, close, volume); err != nil {
		return fmt.Errorf("MFI add failed: %w", err)
	}
	if err := suite.rsi.Add(close); err != nil {
		return fmt.Errorf("RSI add failed: %w", err)
	}
	if err := suite.atr.Add(high, low, close); err != nil {
		return fmt.Errorf("ATR add failed: %w", err)
	}
	return nil
}

// GetCombinedSignal returns a combined trading signal
func (suite *IndicatorSuite) GetCombinedSignal() (string, error) {
	bullishWeight := 0.0
	bearishWeight := 0.0

	amdoBullish, err := suite.amdo.IsBullishCrossover()
	if err != nil && err.Error() != "insufficient data for crossover" {
		return "", err
	}
	if amdoBullish {
		bullishWeight += suite.weights["amdo"]
	}
	amdoBearish, err := suite.amdo.IsBearishCrossover()
	if err != nil && err.Error() != "insufficient data for crossover" {
		return "", err
	}
	if amdoBearish {
		bearishWeight += suite.weights["amdo"]
	}

	atsoBullish, err := suite.atso.IsBullishCrossover()
	if err != nil && err.Error() != "insufficient data for crossover" {
		return "", err
	}
	if atsoBullish {
		bullishWeight += suite.weights["atso"]
	}
	atsoBearish, err := suite.atso.IsBearishCrossover()
	if err != nil && err.Error() != "insufficient data for crossover" {
		return "", err
	}
	if atsoBearish {
		bearishWeight += suite.weights["atso"]
	}

	vwaoBullish, err := suite.vwao.IsBullishCrossover()
	if err != nil && err.Error() != "insufficient data for crossover" {
		return "", err
	}
	if vwaoBullish {
		bullishWeight += suite.weights["vwao"]
	}
	vwaoBearish, err := suite.vwao.IsBearishCrossover()
	if err != nil && err.Error() != "insufficient data for crossover" {
		return "", err
	}
	if vwaoBearish {
		bearishWeight += suite.weights["vwao"]
	}

	hmaBullish, err := suite.hma.IsBullishCrossover()
	if err != nil && err.Error() != "insufficient data for crossover" {
		return "", err
	}
	if hmaBullish {
		bullishWeight += suite.weights["hma"]
	}
	hmaBearish, err := suite.hma.IsBearishCrossover()
	if err != nil && err.Error() != "insufficient data for crossover" {
		return "", err
	}
	if hmaBearish {
		bearishWeight += suite.weights["hma"]
	}

	mfiBullish, err := suite.mfi.IsBullishCrossover()
	if err != nil && err.Error() != "insufficient data for crossover" {
		return "", err
	}
	if mfiBullish {
		bullishWeight += suite.weights["mfi"]
	}
	mfiBearish, err := suite.mfi.IsBearishCrossover()
	if err != nil && err.Error() != "insufficient data for crossover" {
		return "", err
	}
	if mfiBearish {
		bearishWeight += suite.weights["mfi"]
	}

	rsiBullish, err := suite.rsi.IsBullishCrossover()
	if err != nil && err.Error() != "insufficient data for crossover" {
		return "", err
	}
	if rsiBullish {
		bullishWeight += suite.weights["rsi"]
	}
	rsiBearish, err := suite.rsi.IsBearishCrossover()
	if err != nil && err.Error() != "insufficient data for crossover" {
		return "", err
	}
	if rsiBearish {
		bearishWeight += suite.weights["rsi"]
	}

	// ATR is non-directional, so not included in signal calculation
	if bullishWeight >= 4.0 {
		return "Strong Bullish", nil
	}
	if bearishWeight >= 4.0 {
		return "Strong Bearish", nil
	}
	if bullishWeight >= 2.5 {
		return "Bullish", nil
	}
	if bearishWeight >= 2.5 {
		return "Bearish", nil
	}
	return "Neutral", nil
}

// Reset clears all indicators
func (suite *IndicatorSuite) Reset() {
	suite.amdo.Reset()
	suite.atso.Reset()
	suite.vwao.Reset()
	suite.hma.Reset()
	suite.mfi.Reset()
	suite.rsi.Reset()
	suite.atr.Reset()
}

// GetPlotData returns combined plot data
func (suite *IndicatorSuite) GetPlotData(startTime, interval int64) []PlotData {
	var plotData []PlotData
	plotData = append(plotData, suite.amdo.GetPlotData(startTime, interval)...)
	plotData = append(plotData, suite.atso.GetPlotData(startTime, interval)...)
	plotData = append(plotData, suite.vwao.GetPlotData(startTime, interval)...)
	plotData = append(plotData, suite.hma.GetPlotData(startTime, interval)...)
	plotData = append(plotData, suite.mfi.GetPlotData(startTime, interval)...)
	plotData = append(plotData, suite.rsi.GetPlotData(startTime, interval)...)
	plotData = append(plotData, suite.atr.GetPlotData(startTime, interval)...)
	return plotData
}
