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
	rsi        *indicator.RelativeStrengthIndex
	stochastic *indicator.StochasticOscillator
	macd       *indicator.MACD
	cci        *indicator.CommodityChannelIndex
	hma        *indicator.HullMovingAverage
	sar        *indicator.ParabolicSAR
	bollinger  *indicator.BollingerBands
	atr        *indicator.AverageTrueRange
	vwap       *indicator.VWAP
	mfi        *indicator.MoneyFlowIndex

	lastClose  float64
	prevClose  float64
	prev2Close float64 // second-to-last close for momentum confirmation
	lastHigh   float64
	lastLow    float64
	hasClose   bool
	closeCount int // track number of closes for momentum lookback
}

// NewScalpingIndicatorSuite creates a suite with scalping-optimised defaults.
func NewScalpingIndicatorSuite() (*ScalpingIndicatorSuite, error) {
	return NewScalpingIndicatorSuiteWithConfig(config.DefaultConfig())
}

// NewScalpingIndicatorSuiteWithConfig builds a suite using a custom config and
// short, responsive periods suitable for 1–5 minute charts.
//
// Period rationale for scalping:
//   - RSI(5): Faster than default 14, captures micro-reversals
//   - Stochastic(7,3): Quick K with smooth D for crossover signals
//   - MACD(5,12,4): Tight fast/slow spread with responsive signal line
//   - CCI(8): Short cycle detection for overbought/oversold extremes
//   - HMA(6): Ultra-low lag trend following
//   - SAR(0.02,0.2): Standard acceleration for stop placement
//   - Bollinger(12,2.0): Shorter lookback for volatility squeeze detection
//   - ATR(5): Very responsive volatility measure
//   - MFI(5): Quick volume-backed momentum
func NewScalpingIndicatorSuiteWithConfig(cfg config.IndicatorConfig) (*ScalpingIndicatorSuite, error) {
	// Tighten thresholds for faster reversals (asymmetric for mean-reversion).
	cfg.RSIOverbought = 65
	cfg.RSIOversold = 35
	cfg.MFIOverbought = 72
	cfg.MFIOversold = 28

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// RSI: 5-period for rapid momentum shifts
	rsi, err := indicator.NewRelativeStrengthIndexWithParams(5, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create RSI: %w", err)
	}

	// Stochastic: 7/3 for fast K with smoothed D
	stochastic, err := indicator.NewStochasticOscillatorWithParams(7, 3)
	if err != nil {
		return nil, fmt.Errorf("failed to create stochastic oscillator: %w", err)
	}

	// MACD: 5/12/4 for tight histogram oscillation
	macd, err := indicator.NewMACDWithParams(5, 12, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to create MACD: %w", err)
	}

	// CCI: 8-period for quick cycle extremes
	cci, err := indicator.NewCommodityChannelIndexWithParams(8)
	if err != nil {
		return nil, fmt.Errorf("failed to create CCI: %w", err)
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
		rsi:        rsi,
		stochastic: stochastic,
		macd:       macd,
		cci:        cci,
		hma:        hma,
		sar:        sar,
		bollinger:  bollinger,
		atr:        atr,
		vwap:       vwap,
		mfi:        mfi,
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

	if err := suite.rsi.Add(close); err != nil {
		return fmt.Errorf("RSI add failed: %w", err)
	}
	if err := suite.stochastic.Add(high, low, close); err != nil {
		return fmt.Errorf("stochastic add failed: %w", err)
	}
	if err := suite.macd.Add(close); err != nil {
		return fmt.Errorf("MACD add failed: %w", err)
	}
	if err := suite.cci.Add(high, low, close); err != nil {
		return fmt.Errorf("CCI add failed: %w", err)
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

// GetCombinedBearishSignal mirrors GetCombinedSignal for API parity.
func (suite *ScalpingIndicatorSuite) GetCombinedBearishSignal() (string, error) {
	return suite.GetCombinedSignal()
}

// GetDivergenceSignals checks for divergence signals across momentum/volume.
func (suite *ScalpingIndicatorSuite) GetDivergenceSignals() (map[string]string, error) {
	result := make(map[string]string)

	if rsiDiv, rsiSignal, err := suite.rsi.IsDivergence(); err == nil && rsiDiv {
		result["RSI"] = rsiSignal
	}

	if mfiSignal, err := suite.mfi.IsDivergence(); err == nil && mfiSignal != "none" {
		result["MFI"] = mfiSignal
	}
	return result, nil
}

// Reset clears all indicator data and cached price context.
func (suite *ScalpingIndicatorSuite) Reset() {
	suite.rsi.Reset()
	suite.stochastic.Reset()
	suite.macd.Reset()
	suite.cci.Reset()
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
}

// ----------------------- Indicator getters -----------------------

func (suite *ScalpingIndicatorSuite) GetRSI() *indicator.RelativeStrengthIndex {
	return suite.rsi
}

func (suite *ScalpingIndicatorSuite) GetStochastic() *indicator.StochasticOscillator {
	return suite.stochastic
}

func (suite *ScalpingIndicatorSuite) GetMACD() *indicator.MACD {
	return suite.macd
}

func (suite *ScalpingIndicatorSuite) GetCCI() *indicator.CommodityChannelIndex {
	return suite.cci
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
	var plotData []indicator.PlotData

	plotData = append(plotData, suite.rsi.GetPlotData(startTime, interval)...)
	plotData = append(plotData, suite.stochastic.GetPlotData(startTime, interval)...)
	plotData = append(plotData, suite.macd.GetPlotData(startTime, interval)...)
	plotData = append(plotData, suite.cci.GetPlotData(startTime, interval)...)
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
	var bull, bear float64

	/* ---- RSI (fast reversal) ---- */
	// RSI crossovers are primary scalping signals
	if bullish, err := suite.rsi.IsBullishCrossover(); err == nil && bullish {
		bull += 1.2
	}
	if bearish, err := suite.rsi.IsBearishCrossover(); err == nil && bearish {
		bear += 1.2
	}
	if zone, err := suite.rsi.GetOverboughtOversold(); err == nil {
		switch zone {
		case "Oversold":
			bull += 0.5 // reduced: zone alone is weaker than crossover
		case "Overbought":
			bear += 0.5
		}
	}

	/* ---- Stochastic Oscillator (fast cross) ---- */
	// Fixed: proper handling of K/D array alignment
	kVals := suite.stochastic.GetKValues()
	dVals := suite.stochastic.GetDValues()

	// Zone signals from %K
	if len(kVals) > 0 {
		k := kVals[len(kVals)-1]
		// Tighter zones for scalping: 22/78 instead of 25/75
		if k < 22 {
			bull += 0.6
		} else if k > 78 {
			bear += 0.6
		}
	}

	// K/D crossover detection with proper array bounds checking
	if len(kVals) >= 2 && len(dVals) >= 2 {
		// Get the aligned values: D values start later than K values
		kLen, dLen := len(kVals), len(dVals)
		// Both arrays have at least 2 elements, get the last two from each
		curK := kVals[kLen-1]
		prevK := kVals[kLen-2]
		curD := dVals[dLen-1]
		prevD := dVals[dLen-2]

		// Bullish crossover: K crosses above D (allow equal on previous)
		if prevK <= prevD && curK > curD {
			bull += 1.0
		}
		// Bearish crossover: K crosses below D (allow equal on previous)
		if prevK >= prevD && curK < curD {
			bear += 1.0
		}
	}

	/* ---- MACD (histogram cross) ---- */
	histVals := suite.macd.GetHistogramValues()
	if len(histVals) >= 2 {
		curHist := histVals[len(histVals)-1]
		prevHist := histVals[len(histVals)-2]

		// Histogram zero-line crossover (strong signal)
		if prevHist < 0 && curHist > 0 {
			bull += 1.1
		} else if prevHist > 0 && curHist < 0 {
			bear += 1.1
		}

		// Histogram direction (momentum)
		if curHist > 0 {
			bull += 0.25
		} else if curHist < 0 {
			bear += 0.25
		}

		// Histogram momentum acceleration (scalping edge)
		if len(histVals) >= 3 {
			prev2Hist := histVals[len(histVals)-3]
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

	/* ---- CCI (cycle extremes) ---- */
	cciVals := suite.cci.GetValues()
	if len(cciVals) > 0 {
		lastCCI := cciVals[len(cciVals)-1]
		// Tighter CCI extremes for scalping: ±80 instead of ±90
		switch {
		case lastCCI <= -80:
			bull += 0.6
		case lastCCI >= 80:
			bear += 0.6
		}
		// Zero-line cross (trend shift)
		if len(cciVals) >= 2 {
			prevCCI := cciVals[len(cciVals)-2]
			if prevCCI < 0 && lastCCI > 0 {
				bull += 0.35
			} else if prevCCI > 0 && lastCCI < 0 {
				bear += 0.35
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

	/* ---- Parabolic SAR (stop-and-reverse) ---- */
	if sar := suite.sar.GetValues(); len(sar) > 0 {
		if suite.sar.IsUptrend() {
			bull += 0.7
		} else {
			bear += 0.7
		}
	}

	/* ---- Bollinger Bands (volatility squeeze/mean reversion) ---- */
	upper := suite.bollinger.GetUpper()
	middle := suite.bollinger.GetMiddle()
	lower := suite.bollinger.GetLower()
	if suite.hasClose && len(upper) > 0 && len(lower) > 0 && len(middle) > 0 {
		lastUpper := upper[len(upper)-1]
		lastLower := lower[len(lower)-1]
		lastMiddle := middle[len(middle)-1]
		bandwidth := lastUpper - lastLower

		// Band touch/penetration signals (mean reversion for scalping)
		if bandwidth > 0 {
			// Calculate how far price is from the bands as a ratio
			lowerDist := (suite.lastClose - lastLower) / bandwidth
			upperDist := (lastUpper - suite.lastClose) / bandwidth

			// Price at or below lower band: strong bullish reversal signal
			if lowerDist <= 0 {
				bull += 0.9
			} else if lowerDist < 0.1 {
				// Price touching lower band area
				bull += 0.6
			}

			// Price at or above upper band: strong bearish reversal signal
			if upperDist <= 0 {
				bear += 0.9
			} else if upperDist < 0.1 {
				// Price touching upper band area
				bear += 0.6
			}
		}

		// Middle band cross (trend bias)
		if suite.lastClose > lastMiddle {
			bull += 0.2
		} else if suite.lastClose < lastMiddle {
			bear += 0.2
		}
	}

	/* ---- ATR (volatility confirmation) ---- */
	// Expanding ATR with price movement confirms trend strength
	atrVals := suite.atr.GetATRValues()
	if len(atrVals) >= 2 && suite.hasClose && suite.prevClose > 0 {
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

	/* ---- VWAP (intraday flow) ---- */
	// VWAP is critical for scalping: institutional level
	if vals := suite.vwap.GetValues(); len(vals) > 0 && suite.hasClose {
		lastVWAP := vals[len(vals)-1]
		if lastVWAP > 0 {
			if suite.lastClose > lastVWAP {
				bull += 0.8
			} else if suite.lastClose < lastVWAP {
				bear += 0.8
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

	return bull, bear
}

func (suite *ScalpingIndicatorSuite) currentVolRatio() float64 {
	atrVals := suite.atr.GetATRValues()
	if len(atrVals) == 0 || suite.lastClose == 0 {
		return 0
	}
	return atrVals[len(atrVals)-1] / suite.lastClose
}
