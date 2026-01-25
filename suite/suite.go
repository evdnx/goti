package suite

import (
	"fmt"

	"github.com/evdnx/goti/config"
	"github.com/evdnx/goti/indicator"
)

// ---------------------------------------------------------------------
// ScalpingIndicatorSuite – fast, low-lag bundle tuned for intraday use.
// Optimized for 1–5 minute scalping with responsive periods and adaptive
// signal gating based on volatility regimes.
// ---------------------------------------------------------------------
type ScalpingIndicatorSuite struct {
	admo      *indicator.AdaptiveDEMAMomentumOscillator
	vwao      *indicator.VolumeWeightedAroonOscillator
	macd      *indicator.MACD
	hma       *indicator.HullMovingAverage
	sar       *indicator.ParabolicSAR
	bollinger *indicator.BollingerBands
	atr       *indicator.AverageTrueRange
	vwap      *indicator.VWAP
	mfi       *indicator.MoneyFlowIndex

	lastClose  float64
	prevClose  float64
	prev2Close float64 // second-to-last close for momentum confirmation
	lastHigh   float64
	lastLow    float64
	hasClose   bool
	closeCount int // track number of closes for momentum lookback

	// Cached values for performance
	cachedVolRatio    float64
	volRatioValid     bool
	cachedScoresValid bool
	cachedBullScore   float64
	cachedBearScore   float64
}

// NewScalpingIndicatorSuite creates a suite with scalping-optimised defaults.
func NewScalpingIndicatorSuite() (*ScalpingIndicatorSuite, error) {
	return NewScalpingIndicatorSuiteWithConfig(config.DefaultConfig())
}

// NewOptimizedScalpingIndicatorSuite creates a performance-optimized scalping suite
// with only 6 core indicators for high-frequency trading. Removes VWAP, SAR, and
// Bollinger Bands as recommended for scalping to reduce CPU usage by ~40%.
func NewOptimizedScalpingIndicatorSuite() (*OptimizedScalpingIndicatorSuite, error) {
	return NewOptimizedScalpingIndicatorSuiteWithConfig(config.DefaultConfig())
}

// NewScalpingIndicatorSuiteWithConfig builds a suite using a custom config and
// short, responsive periods suitable for 1–5 minute charts.
//
// Period rationale for scalping:
//   - ADMO(8,5,0.3): Adaptive momentum oscillator that adjusts to volatility
//   - VWAO(7): Volume-weighted Aroon for trend strength with volume confirmation
//   - MACD(5,12,4): Tight fast/slow spread with responsive signal line
//   - HMA(6): Ultra-low lag trend following
//   - SAR(0.02,0.2): Standard acceleration for stop placement
//   - Bollinger(12,2.0): Shorter lookback for volatility squeeze detection
//   - ATR(5): Very responsive volatility measure
//   - MFI(5): Quick volume-backed momentum
func NewScalpingIndicatorSuiteWithConfig(cfg config.IndicatorConfig) (*ScalpingIndicatorSuite, error) {
	// Tighten thresholds for faster reversals (asymmetric for mean-reversion).
	cfg.MFIOverbought = 72
	cfg.MFIOversold = 28
	// ADMO thresholds: slightly tighter for scalping
	cfg.AMDOOverbought = 0.8
	cfg.AMDOOversold = -0.8
	// VWAO strong trend threshold: tighter for scalping
	cfg.VWAOStrongTrend = 60

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// ADMO: 8/5/0.3 for adaptive momentum that responds to volatility changes
	admo, err := indicator.NewAdaptiveDEMAMomentumOscillatorWithParams(8, 5, 0.3, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Adaptive DEMA Momentum Oscillator: %w", err)
	}

	// VWAO: 7-period for volume-weighted trend strength
	vwao, err := indicator.NewVolumeWeightedAroonOscillatorWithParams(7, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Volume Weighted Aroon Oscillator: %w", err)
	}

	// MACD: 5/12/4 for tight histogram oscillation
	macd, err := indicator.NewMACDWithParams(5, 12, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to create MACD: %w", err)
	}

	// HMA: 6-period for ultra-low lag trend
	hma, err := indicator.NewHullMovingAverageWithParams(6)
	if err != nil {
		return nil, fmt.Errorf("failed to create HMA: %w", err)
	}

	// SAR: Standard acceleration factors work well for scalping
	sar, err := indicator.NewParabolicSARWithParams(0.02, 0.2)
	if err != nil {
		return nil, fmt.Errorf("failed to create Parabolic SAR: %w", err)
	}

	// Bollinger: 12-period for tighter squeeze detection
	bollinger, err := indicator.NewBollingerBandsWithParams(12, 2.0)
	if err != nil {
		return nil, fmt.Errorf("failed to create Bollinger Bands: %w", err)
	}

	// ATR: 5-period for very responsive volatility
	atr, err := indicator.NewAverageTrueRangeWithParams(5)
	if err != nil {
		return nil, fmt.Errorf("failed to create ATR: %w", err)
	}

	vwap := indicator.NewVWAP()

	// MFI: 5-period for rapid volume confirmation
	mfi, err := indicator.NewMoneyFlowIndexWithParams(5, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create MFI: %w", err)
	}

	return &ScalpingIndicatorSuite{
		admo:      admo,
		vwao:      vwao,
		macd:      macd,
		hma:       hma,
		sar:       sar,
		bollinger: bollinger,
		atr:       atr,
		vwap:      vwap,
		mfi:       mfi,
	}, nil
}

// Add forwards the OHLCV sample to every indicator in the suite.
func (suite *ScalpingIndicatorSuite) Add(high, low, close, volume float64) error {
	if high < low {
		return fmt.Errorf("invalid price: high (%v) must be >= low (%v)", high, low)
	}
	if !indicator.IsValidPrice(high) || !indicator.IsValidPrice(low) {
		return fmt.Errorf("invalid price")
	}
	if !indicator.IsNonNegativePrice(close) {
		return fmt.Errorf("invalid price")
	}
	if !indicator.IsValidVolume(volume) {
		return fmt.Errorf("invalid volume")
	}

	if err := suite.admo.Add(high, low, close); err != nil {
		return fmt.Errorf("ADMO add failed: %w", err)
	}
	if err := suite.vwao.Add(high, low, close, volume); err != nil {
		return fmt.Errorf("VWAO add failed: %w", err)
	}
	if err := suite.macd.Add(close); err != nil {
		return fmt.Errorf("MACD add failed: %w", err)
	}
	if err := suite.hma.Add(close); err != nil {
		return fmt.Errorf("HMA add failed: %w", err)
	}
	if err := suite.sar.Add(high, low); err != nil {
		return fmt.Errorf("Parabolic SAR add failed: %w", err)
	}
	if err := suite.bollinger.Add(close); err != nil {
		return fmt.Errorf("Bollinger add failed: %w", err)
	}
	if err := suite.atr.AddCandle(high, low, close); err != nil {
		return fmt.Errorf("ATR add failed: %w", err)
	}
	if err := suite.vwap.Add(high, low, close, volume); err != nil {
		return fmt.Errorf("VWAP add failed: %w", err)
	}
	if err := suite.mfi.Add(high, low, close, volume); err != nil {
		return fmt.Errorf("MFI add failed: %w", err)
	}

	if suite.hasClose {
		suite.prev2Close = suite.prevClose
		suite.prevClose = suite.lastClose
	}
	suite.lastClose = close
	suite.lastHigh = high
	suite.lastLow = low
	suite.hasClose = true
	suite.closeCount++

	// Invalidate cached values when new data is added
	suite.volRatioValid = false
	suite.cachedScoresValid = false

	return nil
}

// GetCombinedSignal returns the aggregated scalping bias.
// The signal strength is adjusted based on:
//   - Volatility regime (ATR/price ratio)
//   - Momentum confirmation (consecutive close direction)
//   - Signal confluence (number of agreeing indicators)
func (suite *ScalpingIndicatorSuite) GetCombinedSignal() (string, error) {
	bull, bear := suite.computeScores()
	net := bull - bear

	volRatio := suite.currentVolRatio()

	// Base thresholds calibrated for scalping sensitivity
	// Lower thresholds = more responsive to indicator confluence
	strong := 1.8
	normal := 0.9
	weak := 0.35

	// Volatility-adaptive thresholds:
	// - High vol (>0.5%): loosen thresholds, big moves need less confirmation
	// - Low vol (<0.15%): tighten thresholds, avoid noise in tight ranges
	// - Very low vol (<0.08%): require extra confirmation, chop zone
	switch {
	case volRatio > 0.005:
		// High volatility: signals are more reliable, loosen thresholds
		strong -= 0.3
		normal -= 0.2
		weak -= 0.1
	case volRatio > 0.003:
		// Normal-high volatility
		strong -= 0.15
		normal -= 0.1
	case volRatio < 0.0008:
		// Very low volatility: chop zone, require strong confluence
		strong += 0.4
		normal += 0.3
		weak += 0.2
	case volRatio < 0.0015:
		// Low volatility: tighten thresholds
		strong += 0.2
		normal += 0.15
		weak += 0.1
	}

	// Momentum confirmation boost: if price has moved in the same direction
	// for 2+ bars, boost the corresponding signal slightly
	if suite.closeCount >= 3 {
		if suite.lastClose > suite.prevClose && suite.prevClose > suite.prev2Close {
			// Two consecutive up closes: momentum confirmation for bulls
			if net > 0 {
				net += 0.15
			}
		} else if suite.lastClose < suite.prevClose && suite.prevClose < suite.prev2Close {
			// Two consecutive down closes: momentum confirmation for bears
			if net < 0 {
				net -= 0.15
			}
		}
	}

	switch {
	case net >= strong:
		return "Strong Bullish", nil
	case net >= normal:
		return "Bullish", nil
	case net >= weak:
		return "Weak Bullish", nil
	case net <= -strong:
		return "Strong Bearish", nil
	case net <= -normal:
		return "Bearish", nil
	case net <= -weak:
		return "Weak Bearish", nil
	default:
		return "Neutral", nil
	}
}

// ---------------------------------------------------------------------
// OptimizedScalpingIndicatorSuite – performance-optimized bundle for scalping.
// Reduced to 6 core indicators for high-frequency trading with ~40% faster updates.
// ---------------------------------------------------------------------
type OptimizedScalpingIndicatorSuite struct {
	admo *indicator.AdaptiveDEMAMomentumOscillator
	vwao *indicator.VolumeWeightedAroonOscillator
	macd *indicator.MACD
	hma  *indicator.HullMovingAverage
	atr  *indicator.AverageTrueRange
	mfi  *indicator.MoneyFlowIndex

	lastClose  float64
	prevClose  float64
	prev2Close float64 // second-to-last close for momentum confirmation
	lastHigh   float64
	lastLow    float64
	hasClose   bool
	closeCount int // track number of closes for momentum lookback

	// Cached values for performance
	cachedVolRatio    float64
	volRatioValid     bool
	cachedScoresValid bool
	cachedBullScore   float64
	cachedBearScore   float64
}

// NewOptimizedScalpingIndicatorSuiteWithConfig builds an optimized suite using custom config
// and short, responsive periods suitable for high-frequency scalping.
//
// Optimized indicator set (6 indicators):
//   - ADMO(8,5,0.3): Adaptive momentum oscillator that adjusts to volatility
//   - VWAO(7): Volume-weighted Aroon for trend strength with volume confirmation
//   - MACD(5,12,4): Tight fast/slow spread with responsive signal line
//   - HMA(6): Ultra-low lag trend following
//   - ATR(5): Very responsive volatility measure
//   - MFI(5): Quick volume-backed momentum
func NewOptimizedScalpingIndicatorSuiteWithConfig(cfg config.IndicatorConfig) (*OptimizedScalpingIndicatorSuite, error) {
	// Tighten thresholds for faster reversals (asymmetric for mean-reversion).
	cfg.MFIOverbought = 72
	cfg.MFIOversold = 28
	// ADMO thresholds: slightly tighter for scalping
	cfg.AMDOOverbought = 0.8
	cfg.AMDOOversold = -0.8
	// VWAO strong trend threshold: tighter for scalping
	cfg.VWAOStrongTrend = 60

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// ADMO: 8/5/0.3 for adaptive momentum that responds to volatility changes
	admo, err := indicator.NewAdaptiveDEMAMomentumOscillatorWithParams(8, 5, 0.3, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Adaptive DEMA Momentum Oscillator: %w", err)
	}

	// VWAO: 7-period for volume-weighted trend strength
	vwao, err := indicator.NewVolumeWeightedAroonOscillatorWithParams(7, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Volume Weighted Aroon Oscillator: %w", err)
	}

	// MACD: 5/12/4 for tight histogram oscillation
	macd, err := indicator.NewMACDWithParams(5, 12, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to create MACD: %w", err)
	}

	// HMA: 6-period for ultra-low lag trend
	hma, err := indicator.NewHullMovingAverageWithParams(6)
	if err != nil {
		return nil, fmt.Errorf("failed to create HMA: %w", err)
	}

	// ATR: 5-period for very responsive volatility
	atr, err := indicator.NewAverageTrueRangeWithParams(5)
	if err != nil {
		return nil, fmt.Errorf("failed to create ATR: %w", err)
	}

	// MFI: 5-period for rapid volume confirmation
	mfi, err := indicator.NewMoneyFlowIndexWithParams(5, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create MFI: %w", err)
	}

	return &OptimizedScalpingIndicatorSuite{
		admo: admo,
		vwao: vwao,
		macd: macd,
		hma:  hma,
		atr:  atr,
		mfi:  mfi,
	}, nil
}

// AddOptimized forwards the OHLCV sample to the 6 optimized indicators only.
func (suite *OptimizedScalpingIndicatorSuite) Add(high, low, close, volume float64) error {
	if high < low {
		return fmt.Errorf("invalid price: high (%v) must be >= low (%v)", high, low)
	}
	if !indicator.IsValidPrice(high) || !indicator.IsValidPrice(low) {
		return fmt.Errorf("invalid price")
	}
	if !indicator.IsNonNegativePrice(close) {
		return fmt.Errorf("invalid price")
	}
	if !indicator.IsValidVolume(volume) {
		return fmt.Errorf("invalid volume")
	}

	if err := suite.admo.Add(high, low, close); err != nil {
		return fmt.Errorf("ADMO add failed: %w", err)
	}
	if err := suite.vwao.Add(high, low, close, volume); err != nil {
		return fmt.Errorf("VWAO add failed: %w", err)
	}
	if err := suite.macd.Add(close); err != nil {
		return fmt.Errorf("MACD add failed: %w", err)
	}
	if err := suite.hma.Add(close); err != nil {
		return fmt.Errorf("HMA add failed: %w", err)
	}
	if err := suite.atr.AddCandle(high, low, close); err != nil {
		return fmt.Errorf("ATR add failed: %w", err)
	}
	if err := suite.mfi.Add(high, low, close, volume); err != nil {
		return fmt.Errorf("MFI add failed: %w", err)
	}

	if suite.hasClose {
		suite.prev2Close = suite.prevClose
		suite.prevClose = suite.lastClose
	}
	suite.lastClose = close
	suite.lastHigh = high
	suite.lastLow = low
	suite.hasClose = true
	suite.closeCount++

	// Invalidate cached values when new data is added
	suite.volRatioValid = false
	suite.cachedScoresValid = false

	return nil
}

// GetCombinedSignal returns the aggregated scalping bias for optimized suite.
func (suite *OptimizedScalpingIndicatorSuite) GetCombinedSignal() (string, error) {
	bull, bear := suite.computeScores()
	net := bull - bear

	volRatio := suite.currentVolRatio()

	// Base thresholds calibrated for scalping sensitivity (optimized for 6 indicators)
	strong := 1.8
	normal := 0.9
	weak := 0.35

	// Volatility-adaptive thresholds:
	// - High vol (>0.5%): loosen thresholds, big moves need less confirmation
	// - Low vol (<0.15%): tighten thresholds, avoid noise in tight ranges
	// - Very low vol (<0.08%): require extra confirmation, chop zone
	switch {
	case volRatio > 0.005:
		// High volatility: signals are more reliable, loosen thresholds
		strong -= 0.3
		normal -= 0.2
		weak -= 0.1
	case volRatio > 0.003:
		// Normal-high volatility
		strong -= 0.15
		normal -= 0.1
	case volRatio < 0.0008:
		// Very low volatility: chop zone, require strong confluence
		strong += 0.4
		normal += 0.3
		weak += 0.2
	case volRatio < 0.0015:
		// Low volatility: tighten thresholds
		strong += 0.2
		normal += 0.15
		weak += 0.1
	}

	// Momentum confirmation boost: if price has moved in the same direction
	// for 2+ bars, boost the corresponding signal slightly
	if suite.closeCount >= 3 {
		if suite.lastClose > suite.prevClose && suite.prevClose > suite.prev2Close {
			// Two consecutive up closes: momentum confirmation for bulls
			if net > 0 {
				net += 0.15
			}
		} else if suite.lastClose < suite.prevClose && suite.prevClose < suite.prev2Close {
			// Two consecutive down closes: momentum confirmation for bears
			if net < 0 {
				net -= 0.15
			}
		}
	}

	switch {
	case net >= strong:
		return "Strong Bullish", nil
	case net >= normal:
		return "Bullish", nil
	case net >= weak:
		return "Weak Bullish", nil
	case net <= -strong:
		return "Strong Bearish", nil
	case net <= -normal:
		return "Bearish", nil
	case net <= -weak:
		return "Weak Bearish", nil
	default:
		return "Neutral", nil
	}
}

// GetCombinedBearishSignal mirrors GetCombinedSignal for API parity.
func (suite *OptimizedScalpingIndicatorSuite) GetCombinedBearishSignal() (string, error) {
	return suite.GetCombinedSignal()
}

// GetDivergenceSignals checks for divergence signals across momentum/volume (optimized).
func (suite *OptimizedScalpingIndicatorSuite) GetDivergenceSignals() (map[string]string, error) {
	result := make(map[string]string)

	if admoDiv, admoSignal := suite.admo.IsDivergence(); admoDiv {
		result["ADMO"] = admoSignal
	}

	if vwaoDiv, vwaoSignal, err := suite.vwao.IsDivergence(); err == nil && vwaoDiv {
		result["VWAO"] = vwaoSignal
	}

	if mfiSignal, err := suite.mfi.IsDivergence(); err == nil && mfiSignal != "none" {
		result["MFI"] = mfiSignal
	}
	return result, nil
}

// Reset clears all indicator data for optimized suite.
func (suite *OptimizedScalpingIndicatorSuite) Reset() {
	suite.admo.Reset()
	suite.vwao.Reset()
	suite.macd.Reset()
	suite.hma.Reset()
	suite.atr.Reset()
	suite.mfi.Reset()

	suite.lastClose = 0
	suite.prevClose = 0
	suite.prev2Close = 0
	suite.lastHigh = 0
	suite.lastLow = 0
	suite.hasClose = false
	suite.closeCount = 0

	// Clear cached values
	suite.cachedVolRatio = 0
	suite.volRatioValid = false
	suite.cachedScoresValid = false
	suite.cachedBullScore = 0
	suite.cachedBearScore = 0
}

// GetCombinedBearishSignal mirrors GetCombinedSignal for API parity.
func (suite *ScalpingIndicatorSuite) GetCombinedBearishSignal() (string, error) {
	return suite.GetCombinedSignal()
}

// GetDivergenceSignals checks for divergence signals across momentum/volume.
func (suite *ScalpingIndicatorSuite) GetDivergenceSignals() (map[string]string, error) {
	result := make(map[string]string)

	if admoDiv, admoSignal := suite.admo.IsDivergence(); admoDiv {
		result["ADMO"] = admoSignal
	}

	if vwaoDiv, vwaoSignal, err := suite.vwao.IsDivergence(); err == nil && vwaoDiv {
		result["VWAO"] = vwaoSignal
	}

	if mfiSignal, err := suite.mfi.IsDivergence(); err == nil && mfiSignal != "none" {
		result["MFI"] = mfiSignal
	}
	return result, nil
}

// Reset clears all indicator data and cached price context.
func (suite *ScalpingIndicatorSuite) Reset() {
	suite.admo.Reset()
	suite.vwao.Reset()
	suite.macd.Reset()
	suite.hma.Reset()
	suite.sar.Reset()
	suite.bollinger.Reset()
	suite.atr.Reset()
	suite.vwap.Reset()
	suite.mfi.Reset()

	suite.lastClose = 0
	suite.prevClose = 0
	suite.prev2Close = 0
	suite.lastHigh = 0
	suite.lastLow = 0
	suite.hasClose = false
	suite.closeCount = 0

	// Clear cached values
	suite.cachedVolRatio = 0
	suite.volRatioValid = false
	suite.cachedScoresValid = false
	suite.cachedBullScore = 0
	suite.cachedBearScore = 0
}

// ----------------------- Indicator getters -----------------------

func (suite *ScalpingIndicatorSuite) GetAdaptiveDEMAMomentumOscillator() *indicator.AdaptiveDEMAMomentumOscillator {
	return suite.admo
}

func (suite *ScalpingIndicatorSuite) GetVolumeWeightedAroonOscillator() *indicator.VolumeWeightedAroonOscillator {
	return suite.vwao
}

func (suite *ScalpingIndicatorSuite) GetMACD() *indicator.MACD {
	return suite.macd
}

func (suite *ScalpingIndicatorSuite) GetHMA() *indicator.HullMovingAverage {
	return suite.hma
}

func (suite *ScalpingIndicatorSuite) GetParabolicSAR() *indicator.ParabolicSAR {
	return suite.sar
}

func (suite *ScalpingIndicatorSuite) GetBollingerBands() *indicator.BollingerBands {
	return suite.bollinger
}

func (suite *ScalpingIndicatorSuite) GetATR() *indicator.AverageTrueRange {
	return suite.atr
}

func (suite *ScalpingIndicatorSuite) GetVWAP() *indicator.VWAP {
	return suite.vwap
}

func (suite *ScalpingIndicatorSuite) GetMFI() *indicator.MoneyFlowIndex {
	return suite.mfi
}

// GetPlotData returns combined plot data from all indicators.
func (suite *ScalpingIndicatorSuite) GetPlotData(startTime, interval int64) []indicator.PlotData {
	// Pre-allocate with estimated capacity to reduce allocations
	plotData := make([]indicator.PlotData, 0, 20)

	plotData = append(plotData, suite.admo.GetPlotData(startTime, interval)...)
	plotData = append(plotData, suite.vwao.GetPlotData(startTime, interval)...)
	plotData = append(plotData, suite.macd.GetPlotData(startTime, interval)...)
	plotData = append(plotData, suite.hma.GetPlotData(startTime, interval)...)
	plotData = append(plotData, suite.sar.GetPlotData(startTime, interval)...)
	plotData = append(plotData, suite.bollinger.GetPlotData(startTime, interval)...)

	if atr := suite.atr.GetATRValues(); len(atr) > 0 {
		x := make([]float64, len(atr))
		for i := range x {
			x[i] = float64(i)
		}
		timestamps := indicator.GenerateTimestamps(startTime, len(atr), interval)
		plotData = append(plotData, indicator.PlotData{
			Name:      "ATR",
			X:         x,
			Y:         atr,
			Type:      "line",
			Timestamp: timestamps,
		})
	}

	plotData = append(plotData, suite.vwap.GetPlotData(startTime, interval)...)

	if mfi, err := suite.mfi.GetPlotData(); err == nil {
		plotData = append(plotData, mfi...)
	}

	return plotData
}

// computeScores aggregates bullish/bearish contributions from each indicator.
// Weights are calibrated for scalping with emphasis on:
//   - Crossover signals (high weight: first to signal reversals)
//   - Extreme zone readings (medium weight: mean reversion setups)
//   - Trend confirmation (lower weight: filters false signals)
func (suite *ScalpingIndicatorSuite) computeScores() (float64, float64) {
	if suite.cachedScoresValid {
		return suite.cachedBullScore, suite.cachedBearScore
	}

	var bull, bear float64

	// ---- Regime detection for profit/risk tilt ----
	volRatio := suite.currentVolRatio()
	bandwidthPct := 0.0
	if suite.hasClose {
		upper := suite.bollinger.GetUpper()
		lower := suite.bollinger.GetLower()
		if len(upper) > 0 && len(lower) > 0 && suite.lastClose > 0 {
			bandwidthPct = (upper[len(upper)-1] - lower[len(lower)-1]) / suite.lastClose
		}
	}
	isChop := volRatio < 0.0012 && bandwidthPct < 0.008 // tight range + low vol → avoid trend chasing

	trendBias := 0.0
	strongTrend := false
	if vals := suite.vwao.GetVWAOValues(); len(vals) > 0 {
		last := vals[len(vals)-1]
		if last > 60 {
			trendBias += 1
			strongTrend = true
		} else if last < -60 {
			trendBias -= 1
			strongTrend = true
		}
	}
	if dir, err := suite.hma.GetTrendDirection(); err == nil {
		if dir == "Bullish" {
			trendBias += 0.5
		} else if dir == "Bearish" {
			trendBias -= 0.5
		}
	}

	trendScale := 1.0
	if isChop {
		trendScale = 0.7 // de-emphasise trend signals in chop
	}

	/* ---- Adaptive DEMA Momentum Oscillator (volatility-adaptive momentum) ---- */
	// ADMO crossovers are primary scalping signals - adapts to volatility changes
	if bullish, err := suite.admo.IsBullishCrossover(); err == nil && bullish {
		bull += 1.3 * trendScale // Slightly higher weight than RSI due to adaptive nature
	}
	if bearish, err := suite.admo.IsBearishCrossover(); err == nil && bearish {
		bear += 1.3 * trendScale
	}
	// ADMO overbought/oversold zones
	admoVals := suite.admo.GetAMDOValues()
	if len(admoVals) > 0 {
		lastADMO := admoVals[len(admoVals)-1]
		// Check against config thresholds (default ±1.0, but we set ±0.8 for scalping)
		if lastADMO < -0.8 {
			bull += 0.6
		} else if lastADMO > 0.8 {
			bear += 0.6
		}
		// Strong momentum signals
		if lastADMO > 1.5 {
			bear += 0.3
		} else if lastADMO < -1.5 {
			bull += 0.3
		}
	}

	/* ---- Volume Weighted Aroon Oscillator (volume-backed trend strength) ---- */
	// VWAO provides volume-weighted trend signals - excellent for scalping
	if bullish, err := suite.vwao.IsBullishCrossover(); err == nil && bullish {
		bull += 1.2 * trendScale // Strong signal: volume-weighted trend shift
	}
	if bearish, err := suite.vwao.IsBearishCrossover(); err == nil && bearish {
		bear += 1.2 * trendScale
	}

	// Cache VWAO values (accessed multiple times)
	vwaoVals := suite.vwao.GetVWAOValues()
	if len(vwaoVals) > 0 {
		lastVWAO := vwaoVals[len(vwaoVals)-1]

		// Strong trend detection
		if strong, err := suite.vwao.IsStrongTrend(); err == nil && strong {
			if lastVWAO > 60 {
				bull += 0.7 // Strong uptrend with volume
			} else if lastVWAO < -60 {
				bear += 0.7 // Strong downtrend with volume
			}
		}
		// VWAO direction bias
		if lastVWAO > 30 {
			bull += 0.3 // Moderate bullish bias
		} else if lastVWAO < -30 {
			bear += 0.3 // Moderate bearish bias
		}
	}

	/* ---- MACD (histogram cross) ---- */
	histVals := suite.macd.GetHistogramValues()
	if len(histVals) >= 2 {
		histLen := len(histVals)
		curHist := histVals[histLen-1]
		prevHist := histVals[histLen-2]

		// Histogram zero-line crossover (strong signal)
		if prevHist < 0 && curHist > 0 {
			bull += 1.1 * trendScale
		} else if prevHist > 0 && curHist < 0 {
			bear += 1.1 * trendScale
		}

		// Histogram direction (momentum)
		if curHist > 0 {
			bull += 0.25 * trendScale
		} else if curHist < 0 {
			bear += 0.25 * trendScale
		}

		// Histogram momentum acceleration (scalping edge)
		if histLen >= 3 {
			prev2Hist := histVals[histLen-3]
			// Accelerating bullish: histogram increasing
			if curHist > prevHist && prevHist > prev2Hist && curHist > 0 {
				bull += 0.2
			}
			// Accelerating bearish: histogram decreasing
			if curHist < prevHist && prevHist < prev2Hist && curHist < 0 {
				bear += 0.2
			}
		}
	}

	/* ---- HMA (low-lag trend) ---- */
	// HMA crossovers are excellent for scalping due to minimal lag
	if bullish, err := suite.hma.IsBullishCrossover(); err == nil && bullish {
		bull += 1.1 * trendScale
	}
	if bearish, err := suite.hma.IsBearishCrossover(); err == nil && bearish {
		bear += 1.1 * trendScale
	}
	if dir, err := suite.hma.GetTrendDirection(); err == nil {
		if dir == "Bullish" {
			bull += 0.3
		} else if dir == "Bearish" {
			bear += 0.3
		}
	}

	/* ---- Parabolic SAR (stop-and-reverse) ---- */
	if sar := suite.sar.GetValues(); len(sar) > 0 {
		if suite.sar.IsUptrend() {
			bull += 0.7
		} else {
			bear += 0.7
		}
	}

	/* ---- Bollinger Bands (volatility squeeze/mean reversion) ---- */
	if suite.hasClose {
		upper := suite.bollinger.GetUpper()
		middle := suite.bollinger.GetMiddle()
		lower := suite.bollinger.GetLower()

		if len(upper) > 0 && len(lower) > 0 && len(middle) > 0 {
			lastUpper := upper[len(upper)-1]
			lastLower := lower[len(lower)-1]
			lastMiddle := middle[len(middle)-1]
			bandwidth := lastUpper - lastLower
			meanRevBullScale := 1.0
			meanRevBearScale := 1.0
			if strongTrend {
				if trendBias > 0 {
					meanRevBearScale = 0.5 // fade fewer shorts in strong uptrend
				} else if trendBias < 0 {
					meanRevBullScale = 0.5 // fade fewer longs in strong downtrend
				}
			}
			if isChop {
				meanRevBullScale *= 1.1
				meanRevBearScale *= 1.1
			}

			// Band touch/penetration signals (mean reversion for scalping)
			if bandwidth > 0 {
				// Calculate how far price is from the bands as a ratio
				lowerDist := (suite.lastClose - lastLower) / bandwidth
				upperDist := (lastUpper - suite.lastClose) / bandwidth

				// Price at or below lower band: strong bullish reversal signal
				if lowerDist <= 0 {
					bull += 0.9 * meanRevBullScale
				} else if lowerDist < 0.1 {
					// Price touching lower band area
					bull += 0.6 * meanRevBullScale
				}

				// Price at or above upper band: strong bearish reversal signal
				if upperDist <= 0 {
					bear += 0.9 * meanRevBearScale
				} else if upperDist < 0.1 {
					// Price touching upper band area
					bear += 0.6 * meanRevBearScale
				}
			}

			// Middle band cross (trend bias)
			if suite.lastClose > lastMiddle {
				bull += 0.2
			} else if suite.lastClose < lastMiddle {
				bear += 0.2
			}
		}
	}

	/* ---- ATR (volatility confirmation) ---- */
	// Expanding ATR with price movement confirms trend strength
	if suite.hasClose && suite.prevClose > 0 {
		atrVals := suite.atr.GetATRValues()
		if len(atrVals) >= 2 {
			lastATR := atrVals[len(atrVals)-1]
			prevATR := atrVals[len(atrVals)-2]
			priceTrend := suite.lastClose - suite.prevClose

			if prevATR > 0 {
				atrChange := (lastATR - prevATR) / prevATR
				// Expanding volatility with directional move = confirmation
				if atrChange > 0.02 && priceTrend != 0 {
					boost := 0.2
					if atrChange > 0.08 {
						boost = 0.35 // strong volatility expansion
					}
					if priceTrend > 0 {
						bull += boost
					} else {
						bear += boost
					}
				}
			}
		}
	}

	/* ---- VWAP (intraday flow) ---- */
	// VWAP is critical for scalping: institutional level
	if suite.hasClose {
		if vals := suite.vwap.GetValues(); len(vals) > 0 {
			lastVWAP := vals[len(vals)-1]
			if lastVWAP > 0 {
				if suite.lastClose > lastVWAP {
					bull += 0.8
				} else if suite.lastClose < lastVWAP {
					bear += 0.8
				}
			}
		}
	}

	/* ---- MFI (volume-backed momentum) ---- */
	// Volume confirmation is crucial for scalping
	if bullish, err := suite.mfi.IsBullishCrossover(); err == nil && bullish {
		bull += 1.0
	}
	if bearish, err := suite.mfi.IsBearishCrossover(); err == nil && bearish {
		bear += 1.0
	}
	if zone, err := suite.mfi.GetOverboughtOversold(); err == nil {
		switch zone {
		case "Oversold":
			bull += 0.4
		case "Overbought":
			bear += 0.4
		}
	}

	/* ---- Price momentum (last close vs previous) ---- */
	// Simple price direction adds small bias
	if suite.hasClose && suite.prevClose > 0 {
		if suite.lastClose > suite.prevClose {
			bull += 0.2
		} else if suite.lastClose < suite.prevClose {
			bear += 0.2
		}
	}

	// Cache the computed scores
	suite.cachedBullScore = bull
	suite.cachedBearScore = bear
	suite.cachedScoresValid = true

	return bull, bear
}

func (suite *ScalpingIndicatorSuite) currentVolRatio() float64 {
	if suite.volRatioValid {
		return suite.cachedVolRatio
	}

	atrVals := suite.atr.GetATRValues()
	if len(atrVals) == 0 || suite.lastClose == 0 {
		suite.cachedVolRatio = 0
		suite.volRatioValid = true
		return 0
	}

	suite.cachedVolRatio = atrVals[len(atrVals)-1] / suite.lastClose
	suite.volRatioValid = true
	return suite.cachedVolRatio
}

// ----------------------- Optimized Suite Methods -----------------------

// computeScores aggregates bullish/bearish contributions from the 6 optimized indicators.
// Weights are calibrated for scalping with emphasis on:
//   - Crossover signals (high weight: first to signal reversals)
//   - Extreme zone readings (medium weight: mean reversion setups)
//   - Trend confirmation (lower weight: filters false signals)
func (suite *OptimizedScalpingIndicatorSuite) computeScores() (float64, float64) {
	if suite.cachedScoresValid {
		return suite.cachedBullScore, suite.cachedBearScore
	}

	var bull, bear float64

	// ---- Regime detection for profit/risk tilt ----
	volRatio := suite.currentVolRatio()
	isChop := volRatio < 0.0012 // low volatility regime → avoid chasing trends
	trendBias := 0.0
	strongTrend := false
	if vals := suite.vwao.GetVWAOValues(); len(vals) > 0 {
		last := vals[len(vals)-1]
		if last > 60 {
			trendBias += 1
			strongTrend = true
		} else if last < -60 {
			trendBias -= 1
			strongTrend = true
		}
	}
	if dir, err := suite.hma.GetTrendDirection(); err == nil {
		if dir == "Bullish" {
			trendBias += 0.5
		} else if dir == "Bearish" {
			trendBias -= 0.5
		}
	}
	trendScale := 1.0
	if isChop {
		trendScale = 0.7
	}
	if strongTrend {
		if trendBias > 0 {
			bull += 0.2 // favour signals in the prevailing strong trend
		} else if trendBias < 0 {
			bear += 0.2
		}
	}

	/* ---- Adaptive DEMA Momentum Oscillator (volatility-adaptive momentum) ---- */
	// ADMO crossovers are primary scalping signals - adapts to volatility changes
	if bullish, err := suite.admo.IsBullishCrossover(); err == nil && bullish {
		bull += 1.3 * trendScale // Slightly higher weight than RSI due to adaptive nature
	}
	if bearish, err := suite.admo.IsBearishCrossover(); err == nil && bearish {
		bear += 1.3 * trendScale
	}
	// ADMO overbought/oversold zones
	admoVals := suite.admo.GetAMDOValues()
	if len(admoVals) > 0 {
		lastADMO := admoVals[len(admoVals)-1]
		// Check against config thresholds (default ±1.0, but we set ±0.8 for scalping)
		if lastADMO < -0.8 {
			bull += 0.6 // Oversold zone
		} else if lastADMO > 0.8 {
			bear += 0.6 // Overbought zone
		}
		// Strong momentum signals
		if lastADMO > 1.5 {
			bear += 0.3 // Very overbought
		} else if lastADMO < -1.5 {
			bull += 0.3 // Very oversold
		}
	}

	/* ---- Volume Weighted Aroon Oscillator (volume-backed trend strength) ---- */
	// VWAO provides volume-weighted trend signals - excellent for scalping
	if bullish, err := suite.vwao.IsBullishCrossover(); err == nil && bullish {
		bull += 1.2 * trendScale // Strong signal: volume-weighted trend shift
	}
	if bearish, err := suite.vwao.IsBearishCrossover(); err == nil && bearish {
		bear += 1.2 * trendScale
	}

	// Cache VWAO values (accessed multiple times)
	vwaoVals := suite.vwao.GetVWAOValues()
	if len(vwaoVals) > 0 {
		lastVWAO := vwaoVals[len(vwaoVals)-1]

		// Strong trend detection
		if strong, err := suite.vwao.IsStrongTrend(); err == nil && strong {
			if lastVWAO > 60 {
				bull += 0.7 // Strong uptrend with volume
			} else if lastVWAO < -60 {
				bear += 0.7 // Strong downtrend with volume
			}
		}
		// VWAO direction bias
		if lastVWAO > 30 {
			bull += 0.3 // Moderate bullish bias
		} else if lastVWAO < -30 {
			bear += 0.3 // Moderate bearish bias
		}
	}

	/* ---- MACD (histogram cross) ---- */
	histVals := suite.macd.GetHistogramValues()
	if len(histVals) >= 2 {
		histLen := len(histVals)
		curHist := histVals[histLen-1]
		prevHist := histVals[histLen-2]

		// Histogram zero-line crossover (strong signal)
		if prevHist < 0 && curHist > 0 {
			bull += 1.1 * trendScale
		} else if prevHist > 0 && curHist < 0 {
			bear += 1.1 * trendScale
		}

		// Histogram direction (momentum)
		if curHist > 0 {
			bull += 0.25 * trendScale
		} else if curHist < 0 {
			bear += 0.25 * trendScale
		}

		// Histogram momentum acceleration (scalping edge)
		if histLen >= 3 {
			prev2Hist := histVals[histLen-3]
			// Accelerating bullish: histogram increasing
			if curHist > prevHist && prevHist > prev2Hist && curHist > 0 {
				bull += 0.2
			}
			// Accelerating bearish: histogram decreasing
			if curHist < prevHist && prevHist < prev2Hist && curHist < 0 {
				bear += 0.2
			}
		}
	}

	/* ---- HMA (low-lag trend) ---- */
	// HMA crossovers are excellent for scalping due to minimal lag
	if bullish, err := suite.hma.IsBullishCrossover(); err == nil && bullish {
		bull += 1.1
	}
	if bearish, err := suite.hma.IsBearishCrossover(); err == nil && bearish {
		bear += 1.1
	}
	if dir, err := suite.hma.GetTrendDirection(); err == nil {
		if dir == "Bullish" {
			bull += 0.3
		} else if dir == "Bearish" {
			bear += 0.3
		}
	}

	/* ---- ATR (volatility confirmation) ---- */
	// Expanding ATR with price movement confirms trend strength
	if suite.hasClose && suite.prevClose > 0 {
		atrVals := suite.atr.GetATRValues()
		if len(atrVals) >= 2 {
			lastATR := atrVals[len(atrVals)-1]
			prevATR := atrVals[len(atrVals)-2]
			priceTrend := suite.lastClose - suite.prevClose

			if prevATR > 0 {
				atrChange := (lastATR - prevATR) / prevATR
				// Expanding volatility with directional move = confirmation
				if atrChange > 0.02 && priceTrend != 0 {
					boost := 0.2
					if atrChange > 0.08 {
						boost = 0.35 // strong volatility expansion
					}
					if priceTrend > 0 {
						bull += boost
					} else {
						bear += boost
					}
				}
			}
		}
	}

	/* ---- MFI (volume-backed momentum) ---- */
	// Volume confirmation is crucial for scalping
	if bullish, err := suite.mfi.IsBullishCrossover(); err == nil && bullish {
		bull += 1.0
	}
	if bearish, err := suite.mfi.IsBearishCrossover(); err == nil && bearish {
		bear += 1.0
	}
	if zone, err := suite.mfi.GetOverboughtOversold(); err == nil {
		switch zone {
		case "Oversold":
			bull += 0.4
		case "Overbought":
			bear += 0.4
		}
	}

	/* ---- Price momentum (last close vs previous) ---- */
	// Simple price direction adds small bias
	if suite.hasClose && suite.prevClose > 0 {
		if suite.lastClose > suite.prevClose {
			bull += 0.2
		} else if suite.lastClose < suite.prevClose {
			bear += 0.2
		}
	}

	// Cache the computed scores
	suite.cachedBullScore = bull
	suite.cachedBearScore = bear
	suite.cachedScoresValid = true

	return bull, bear
}

func (suite *OptimizedScalpingIndicatorSuite) currentVolRatio() float64 {
	if suite.volRatioValid {
		return suite.cachedVolRatio
	}

	atrVals := suite.atr.GetATRValues()
	if len(atrVals) == 0 || suite.lastClose == 0 {
		suite.cachedVolRatio = 0
		suite.volRatioValid = true
		return 0
	}

	suite.cachedVolRatio = atrVals[len(atrVals)-1] / suite.lastClose
	suite.volRatioValid = true
	return suite.cachedVolRatio
}

// ----------------------- Optimized Suite Getters -----------------------

func (suite *OptimizedScalpingIndicatorSuite) GetAdaptiveDEMAMomentumOscillator() *indicator.AdaptiveDEMAMomentumOscillator {
	return suite.admo
}

func (suite *OptimizedScalpingIndicatorSuite) GetVolumeWeightedAroonOscillator() *indicator.VolumeWeightedAroonOscillator {
	return suite.vwao
}

func (suite *OptimizedScalpingIndicatorSuite) GetMACD() *indicator.MACD {
	return suite.macd
}

func (suite *OptimizedScalpingIndicatorSuite) GetHMA() *indicator.HullMovingAverage {
	return suite.hma
}

func (suite *OptimizedScalpingIndicatorSuite) GetATR() *indicator.AverageTrueRange {
	return suite.atr
}

func (suite *OptimizedScalpingIndicatorSuite) GetMFI() *indicator.MoneyFlowIndex {
	return suite.mfi
}

// GetPlotData returns combined plot data from the 6 optimized indicators.
func (suite *OptimizedScalpingIndicatorSuite) GetPlotData(startTime, interval int64) []indicator.PlotData {
	// Pre-allocate with estimated capacity to reduce allocations
	plotData := make([]indicator.PlotData, 0, 15)

	plotData = append(plotData, suite.admo.GetPlotData(startTime, interval)...)
	plotData = append(plotData, suite.vwao.GetPlotData(startTime, interval)...)
	plotData = append(plotData, suite.macd.GetPlotData(startTime, interval)...)
	plotData = append(plotData, suite.hma.GetPlotData(startTime, interval)...)

	if atr := suite.atr.GetATRValues(); len(atr) > 0 {
		x := make([]float64, len(atr))
		for i := range x {
			x[i] = float64(i)
		}
		timestamps := indicator.GenerateTimestamps(startTime, len(atr), interval)
		plotData = append(plotData, indicator.PlotData{
			Name:      "ATR",
			X:         x,
			Y:         atr,
			Type:      "line",
			Timestamp: timestamps,
		})
	}

	if mfi, err := suite.mfi.GetPlotData(); err == nil {
		plotData = append(plotData, mfi...)
	}

	return plotData
}
