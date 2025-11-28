package suite

import (
	"fmt"

	"github.com/evdnx/goti/config"
	"github.com/evdnx/goti/indicator"
)

// ---------------------------------------------------------------------
// ScalpingIndicatorSuite – fast, low-lag bundle tuned for intraday use.
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

	lastClose float64
	prevClose float64
	lastHigh  float64
	lastLow   float64
	hasClose  bool
}

// NewScalpingIndicatorSuite creates a suite with scalping-optimised defaults.
func NewScalpingIndicatorSuite() (*ScalpingIndicatorSuite, error) {
	return NewScalpingIndicatorSuiteWithConfig(config.DefaultConfig())
}

// NewScalpingIndicatorSuiteWithConfig builds a suite using a custom config and
// short, responsive periods suitable for 1–5 minute charts.
func NewScalpingIndicatorSuiteWithConfig(cfg config.IndicatorConfig) (*ScalpingIndicatorSuite, error) {
	// Tighten thresholds for faster reversals.
	cfg.RSIOverbought = 68
	cfg.RSIOversold = 32
	cfg.MFIOverbought = 75
	cfg.MFIOversold = 25

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	rsi, err := indicator.NewRelativeStrengthIndexWithParams(7, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create RSI: %w", err)
	}

	stochastic, err := indicator.NewStochasticOscillatorWithParams(9, 3)
	if err != nil {
		return nil, fmt.Errorf("failed to create stochastic oscillator: %w", err)
	}

	macd, err := indicator.NewMACDWithParams(5, 13, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to create MACD: %w", err)
	}

	cci, err := indicator.NewCommodityChannelIndexWithParams(10)
	if err != nil {
		return nil, fmt.Errorf("failed to create CCI: %w", err)
	}

	hma, err := indicator.NewHullMovingAverageWithParams(8)
	if err != nil {
		return nil, fmt.Errorf("failed to create HMA: %w", err)
	}

	sar, err := indicator.NewParabolicSARWithParams(0.018, 0.18)
	if err != nil {
		return nil, fmt.Errorf("failed to create Parabolic SAR: %w", err)
	}

	bollinger, err := indicator.NewBollingerBandsWithParams(14, 2.0)
	if err != nil {
		return nil, fmt.Errorf("failed to create Bollinger Bands: %w", err)
	}

	atr, err := indicator.NewAverageTrueRangeWithParams(7)
	if err != nil {
		return nil, fmt.Errorf("failed to create ATR: %w", err)
	}

	vwap := indicator.NewVWAP()

	mfi, err := indicator.NewMoneyFlowIndexWithParams(7, cfg)
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
	if high < low ||
		!indicator.IsValidPrice(high) ||
		!indicator.IsValidPrice(low) ||
		!indicator.IsNonNegativePrice(close) ||
		!indicator.IsValidVolume(volume) {
		return fmt.Errorf("invalid price or volume")
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
		suite.prevClose = suite.lastClose
	}
	suite.lastClose = close
	suite.lastHigh = high
	suite.lastLow = low
	suite.hasClose = true
	return nil
}

// GetCombinedSignal returns the aggregated scalping bias.
func (suite *ScalpingIndicatorSuite) GetCombinedSignal() (string, error) {
	bull, bear := suite.computeScores()
	net := bull - bear

	volRatio := suite.currentVolRatio()
	strong := 2.4
	normal := 1.4
	weak := 0.6

	// Looser thresholds when volatility is expanding, tighter when it is contracting.
	switch {
	case volRatio > 0.004:
		strong -= 0.2
		normal -= 0.15
		weak -= 0.1
	case volRatio < 0.0015:
		strong += 0.2
		normal += 0.15
		weak += 0.1
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
	suite.lastHigh = 0
	suite.lastLow = 0
	suite.hasClose = false
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
func (suite *ScalpingIndicatorSuite) computeScores() (float64, float64) {
	var bull, bear float64

	/* ---- RSI (fast reversal) ---- */
	if bullish, err := suite.rsi.IsBullishCrossover(); err == nil && bullish {
		bull += 1.1
	}
	if bearish, err := suite.rsi.IsBearishCrossover(); err == nil && bearish {
		bear += 1.1
	}
	if zone, err := suite.rsi.GetOverboughtOversold(); err == nil {
		switch zone {
		case "Oversold":
			bull += 0.6
		case "Overbought":
			bear += 0.6
		}
	}

	/* ---- Stochastic Oscillator (fast cross) ---- */
	kVals := suite.stochastic.GetKValues()
	dVals := suite.stochastic.GetDValues()
	if len(kVals) > 0 {
		k := kVals[len(kVals)-1]
		if k < 25 {
			bull += 0.7
		} else if k > 75 {
			bear += 0.7
		}
	}
	if len(dVals) >= 2 && len(kVals) >= len(dVals) {
		offset := len(kVals) - len(dVals)
		curK := kVals[offset+len(dVals)-1]
		prevK := kVals[offset+len(dVals)-2]
		curD := dVals[len(dVals)-1]
		prevD := dVals[len(dVals)-2]
		if prevK <= prevD && curK > curD {
			bull += 1.0
		}
		if prevK >= prevD && curK < curD {
			bear += 1.0
		}
	}

	/* ---- MACD (histogram cross) ---- */
	macdVals := suite.macd.GetMACDValues()
	signalVals := suite.macd.GetSignalValues()
	histVals := suite.macd.GetHistogramValues()
	if len(histVals) >= 2 {
		curHist := histVals[len(histVals)-1]
		prevHist := histVals[len(histVals)-2]
		if prevHist <= 0 && curHist > 0 {
			bull += 1.1
		}
		if prevHist >= 0 && curHist < 0 {
			bear += 1.1
		}
		if curHist > 0 {
			bull += 0.3
		} else {
			bear += 0.3
		}
	} else if len(macdVals) >= 2 && len(signalVals) >= 2 && len(macdVals) >= len(signalVals) {
		offset := len(macdVals) - len(signalVals)
		curMACD := macdVals[offset+len(signalVals)-1]
		prevMACD := macdVals[offset+len(signalVals)-2]
		curSignal := signalVals[len(signalVals)-1]
		prevSignal := signalVals[len(signalVals)-2]
		if prevMACD <= prevSignal && curMACD > curSignal {
			bull += 1.0
		}
		if prevMACD >= prevSignal && curMACD < curSignal {
			bear += 1.0
		}
	}

	/* ---- CCI (cycle extremes) ---- */
	cciVals := suite.cci.GetValues()
	if len(cciVals) > 0 {
		lastCCI := cciVals[len(cciVals)-1]
		switch {
		case lastCCI <= -90:
			bull += 0.7
		case lastCCI >= 90:
			bear += 0.7
		}
		if len(cciVals) >= 2 {
			prevCCI := cciVals[len(cciVals)-2]
			if prevCCI < 0 && lastCCI > 0 {
				bull += 0.3
			} else if prevCCI > 0 && lastCCI < 0 {
				bear += 0.3
			}
		}
	}

	/* ---- HMA (low-lag trend) ---- */
	if bullish, err := suite.hma.IsBullishCrossover(); err == nil && bullish {
		bull += 1.0
	}
	if bearish, err := suite.hma.IsBearishCrossover(); err == nil && bearish {
		bear += 1.0
	}
	if dir, err := suite.hma.GetTrendDirection(); err == nil {
		if dir == "Bullish" {
			bull += 0.35
		} else if dir == "Bearish" {
			bear += 0.35
		}
	}

	/* ---- Parabolic SAR (stop-and-reverse) ---- */
	if sar := suite.sar.GetValues(); len(sar) > 0 {
		if suite.sar.IsUptrend() {
			bull += 0.8
		} else {
			bear += 0.8
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
		switch {
		case suite.lastClose <= lastLower:
			bull += 0.9
		case suite.lastClose >= lastUpper:
			bear += 0.9
		}
		if suite.lastClose > lastMiddle {
			bull += 0.25
		} else if suite.lastClose < lastMiddle {
			bear += 0.25
		}
	}

	/* ---- ATR (volatility confirmation) ---- */
	atrVals := suite.atr.GetATRValues()
	if len(atrVals) >= 2 && suite.hasClose && suite.prevClose > 0 {
		lastATR := atrVals[len(atrVals)-1]
		prevATR := atrVals[len(atrVals)-2]
		atrTrend := lastATR - prevATR
		priceTrend := suite.lastClose - suite.prevClose
		if atrTrend > 0 && priceTrend != 0 && prevATR > 0 {
			boost := 0.2
			if atrTrend/prevATR > 0.05 {
				boost = 0.3
			}
			if priceTrend > 0 {
				bull += boost
			} else {
				bear += boost
			}
		}
	}

	/* ---- VWAP (intraday flow) ---- */
	if vals := suite.vwap.GetValues(); len(vals) > 0 && suite.hasClose {
		lastVWAP := vals[len(vals)-1]
		if suite.lastClose > lastVWAP {
			bull += 0.9
		} else if suite.lastClose < lastVWAP {
			bear += 0.9
		}
	}

	/* ---- MFI (volume-backed momentum) ---- */
	if bullish, err := suite.mfi.IsBullishCrossover(); err == nil && bullish {
		bull += 0.9
	}
	if bearish, err := suite.mfi.IsBearishCrossover(); err == nil && bearish {
		bear += 0.9
	}
	if zone, err := suite.mfi.GetOverboughtOversold(); err == nil {
		switch zone {
		case "Oversold":
			bull += 0.35
		case "Overbought":
			bear += 0.35
		}
	}

	/* ---- Price momentum (last close vs previous) ---- */
	if suite.hasClose && suite.prevClose > 0 {
		switch {
		case suite.lastClose > suite.prevClose:
			bull += 0.25
		case suite.lastClose < suite.prevClose:
			bear += 0.25
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
