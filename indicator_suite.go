package goti

import (
	"errors"
	"fmt"
)

// IndicatorSuite combines multiple technical indicators for a comprehensive market signal
type IndicatorSuite struct {
	rsi  *RelativeStrengthIndex           // Relative Strength Index
	mfi  *MoneyFlowIndex                  // Money Flow Index
	vwao *VolumeWeightedAroonOscillator   // Volume-Weighted Aroon Oscillator
	hma  *HullMovingAverage               // Hull Moving Average
	amdo *AdaptiveDEMAMomentumOscillator  // Adaptive DEMA Divergence Oscillator
	atso *AdaptiveTrendStrengthOscillator // Adaptive Trend Strength Oscillator
}

// NewIndicatorSuite initializes the suite with default parameters
func NewIndicatorSuite() (*IndicatorSuite, error) {
	return NewIndicatorSuiteWithConfig(DefaultConfig())
}

// NewIndicatorSuiteWithConfig initializes the suite with a custom configuration
func NewIndicatorSuiteWithConfig(config IndicatorConfig) (*IndicatorSuite, error) {
	// Initialize Relative Strength Index
	rsi, err := NewRelativeStrengthIndexWithParams(5, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create RSI: %w", err)
	}

	// Initialize Money Flow Index
	mfi, err := NewMoneyFlowIndexWithParams(5, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create MFI: %w", err)
	}

	// Initialize Volume-Weighted Aroon Oscillator
	vwao, err := NewVolumeWeightedAroonOscillatorWithParams(14, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create VWAO: %w", err)
	}

	// Initialize Hull Moving Average
	hma, err := NewHullMovingAverageWithParams(9)
	if err != nil {
		return nil, fmt.Errorf("failed to create HMA: %w", err)
	}

	// Initialize Adaptive DEMA Divergence Oscillator
	amdo, err := NewAdaptiveDEMAMomentumOscillatorWithParams(20, 14, 0.3, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create AMDO: %w", err)
	}

	// Initialize Adaptive Trend Strength Oscillator
	atso, err := NewAdaptiveTrendStrengthOscillatorWithParams(2, 14, 14, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create ATSO: %w", err)
	}

	return &IndicatorSuite{
		rsi:  rsi,
		mfi:  mfi,
		vwao: vwao,
		hma:  hma,
		amdo: amdo,
		atso: atso,
	}, nil
}

// Add appends new price and volume data to all indicators
func (suite *IndicatorSuite) Add(high, low, close, volume float64) error {
	if high < low || !isNonNegativePrice(close) || !isValidVolume(volume) {
		return errors.New("invalid price or volume")
	}

	if err := suite.rsi.Add(close); err != nil {
		return fmt.Errorf("RSI add failed: %w", err)
	}
	if err := suite.mfi.Add(high, low, close, volume); err != nil {
		return fmt.Errorf("MFI add failed: %w", err)
	}
	if err := suite.vwao.Add(high, low, close, volume); err != nil {
		return fmt.Errorf("VWAO add failed: %w", err)
	}
	if err := suite.hma.Add(close); err != nil {
		return fmt.Errorf("HMA add failed: %w", err)
	}
	if err := suite.amdo.Add(high, low, close); err != nil {
		return fmt.Errorf("AMDO add failed: %w", err)
	}
	if err := suite.atso.Add(high, low, close); err != nil {
		return fmt.Errorf("ATSO add failed: %w", err)
	}
	return nil
}

// GetCombinedSignal calculates a weighted bullish signal from all indicators
func (suite *IndicatorSuite) GetCombinedSignal() (string, error) {
	rsiBullish, err := suite.rsi.IsBullishCrossover()
	if err != nil {
		return "", fmt.Errorf("RSI bullish crossover check failed: %w", err)
	}
	mfiBullish, err := suite.mfi.IsBullishCrossover()
	if err != nil {
		return "", fmt.Errorf("MFI bullish crossover check failed: %w", err)
	}
	vwaoBullish, err := suite.vwao.IsBullishCrossover()
	if err != nil {
		return "", fmt.Errorf("VWAO bullish crossover check failed: %w", err)
	}
	hmaBullish, err := suite.hma.IsBullishCrossover()
	if err != nil {
		return "", fmt.Errorf("HMA bullish crossover check failed: %w", err)
	}
	amdoBullish, err := suite.amdo.IsBullishCrossover()
	if err != nil {
		return "", fmt.Errorf("AMDO bullish crossover check failed: %w", err)
	}
	atsoBullish := suite.atso.IsBullishCrossover()

	weightSum := 0.0
	if rsiBullish {
		weightSum += 1.0
	}
	if mfiBullish {
		weightSum += 1.2
	}
	if vwaoBullish {
		weightSum += 1.0
	}
	if hmaBullish {
		weightSum += 1.5
	}
	if amdoBullish {
		weightSum += 0.8
	}
	if atsoBullish {
		weightSum += 0.5
	}

	if weightSum >= 4.0 {
		return "Strong Bullish", nil
	}
	if weightSum >= 2.0 {
		return "Bullish", nil
	}
	if weightSum > 0 {
		return "Weak Bullish", nil
	}
	return "Neutral", nil
}

// GetCombinedBearishSignal calculates a weighted bearish signal from all indicators
func (suite *IndicatorSuite) GetCombinedBearishSignal() (string, error) {
	rsiBearish, err := suite.rsi.IsBearishCrossover()
	if err != nil {
		return "", fmt.Errorf("RSI bearish crossover check failed: %w", err)
	}
	mfiBearish, err := suite.mfi.IsBearishCrossover()
	if err != nil {
		return "", fmt.Errorf("MFI bearish crossover check failed: %w", err)
	}
	vwaoBearish, err := suite.vwao.IsBearishCrossover()
	if err != nil {
		return "", fmt.Errorf("VWAO bearish crossover check failed: %w", err)
	}
	hmaBearish, err := suite.hma.IsBearishCrossover()
	if err != nil {
		return "", fmt.Errorf("HMA bearish crossover check failed: %w", err)
	}
	amdoBearish, err := suite.amdo.IsBearishCrossover()
	if err != nil {
		return "", fmt.Errorf("AMDO bearish crossover check failed: %w", err)
	}
	atsoBearish := suite.atso.IsBearishCrossover()

	weightSum := 0.0
	if rsiBearish {
		weightSum += 1.0
	}
	if mfiBearish {
		weightSum += 1.2
	}
	if vwaoBearish {
		weightSum += 1.0
	}
	if hmaBearish {
		weightSum += 1.5
	}
	if amdoBearish {
		weightSum += 0.8
	}
	if atsoBearish {
		weightSum += 0.5
	}

	if weightSum >= 4.0 {
		return "Strong Bearish", nil
	}
	if weightSum >= 2.0 {
		return "Bearish", nil
	}
	if weightSum > 0 {
		return "Weak Bearish", nil
	}
	return "Neutral", nil
}

// GetDivergenceSignals checks for divergence signals across all indicators
func (suite *IndicatorSuite) GetDivergenceSignals() (map[string]string, error) {
	result := make(map[string]string)
	rsiDiv, rsiSignal, err := suite.rsi.IsDivergence()
	if err != nil {
		return nil, fmt.Errorf("RSI divergence check failed: %w", err)
	}
	if rsiDiv {
		result["RSI"] = rsiSignal
	}
	mfiDiv, mfiSignal, err := suite.mfi.IsDivergence()
	if err != nil {
		return nil, fmt.Errorf("MFI divergence check failed: %w", err)
	}
	if mfiDiv {
		result["MFI"] = mfiSignal
	}
	amdoDiv, amdoSignal := suite.amdo.IsDivergence()
	if amdoDiv {
		result["AMDO"] = amdoSignal
	}
	return result, nil
}

// Reset clears all indicator data
func (suite *IndicatorSuite) Reset() {
	suite.rsi.Reset()
	suite.mfi.Reset()
	suite.vwao.Reset()
	suite.hma.Reset()
	suite.amdo.Reset()
	suite.atso.Reset()
}

// GetRSI returns the RSI indicator
func (suite *IndicatorSuite) GetRSI() *RelativeStrengthIndex {
	return suite.rsi
}

// GetMFI returns the MFI indicator
func (suite *IndicatorSuite) GetMFI() *MoneyFlowIndex {
	return suite.mfi
}

// GetVWAO returns the VWAO indicator
func (suite *IndicatorSuite) GetVWAO() *VolumeWeightedAroonOscillator {
	return suite.vwao
}

// GetHMA returns the HMA indicator
func (suite *IndicatorSuite) GetHMA() *HullMovingAverage {
	return suite.hma
}

// GetAMDO returns the AMDO indicator
func (suite *IndicatorSuite) GetAMDO() *AdaptiveDEMAMomentumOscillator {
	return suite.amdo
}

// GetATSO returns the ATSO indicator
func (suite *IndicatorSuite) GetATSO() *AdaptiveTrendStrengthOscillator {
	return suite.atso
}

// GetPlotData returns combined plot data from all indicators
func (suite *IndicatorSuite) GetPlotData(startTime, interval int64) []PlotData {
	var plotData []PlotData
	plotData = append(plotData, suite.rsi.GetPlotData(startTime, interval)...)
	plotData = append(plotData, suite.mfi.GetPlotData(startTime, interval)...)
	plotData = append(plotData, suite.vwao.GetPlotData(startTime, interval)...)
	plotData = append(plotData, suite.hma.GetPlotData(startTime, interval)...)
	plotData = append(plotData, suite.amdo.GetPlotData(startTime, interval)...)
	plotData = append(plotData, suite.atso.GetPlotData()...)
	return plotData
}
