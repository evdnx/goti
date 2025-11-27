package indicator

import (
	"math"
	"testing"
)

/*
-------------------------------------------------------------

	Helper – generate a deterministic OHLC series
	-------------------------------------------------------------
*/
func generateOHLC(start, step float64, n int) (highs, lows, closes []float64) {
	highs = make([]float64, n)
	lows = make([]float64, n)
	closes = make([]float64, n)

	for i := 0; i < n; i++ {
		base := start + float64(i)*step
		highs[i] = base + 1.0 // high = base + 1
		lows[i] = base - 1.0  // low  = base - 1
		closes[i] = base      // close = base
	}
	return
}

/*
-------------------------------------------------------------

	Constructor tests
	-------------------------------------------------------------
*/
func TestNewAverageTrueRange_Default(t *testing.T) {
	atr, err := NewAverageTrueRange()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if atr.period != 14 {
		t.Fatalf("expected default period 14, got %d", atr.period)
	}
}

func TestNewAverageTrueRange_WithCustomPeriod(t *testing.T) {
	atr, err := NewAverageTrueRangeWithParams(7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if atr.period != 7 {
		t.Fatalf("expected period 7, got %d", atr.period)
	}
}

func TestNewAverageTrueRange_InvalidPeriod(t *testing.T) {
	if _, err := NewAverageTrueRangeWithParams(0); err == nil {
		t.Fatalf("expected error for period < 1")
	}
}

/*
-------------------------------------------------------------

	Option handling – close validation
	-------------------------------------------------------------
*/
func TestWithCloseValidation_Disabled(t *testing.T) {
	atr, err := NewAverageTrueRangeWithParams(5, WithCloseValidation(false))
	if err != nil {
		t.Fatalf("constructor error: %v", err)
	}
	// Close far outside high/low – should be accepted because validation is off.
	if err := atr.AddCandle(10, 9, 20); err != nil {
		t.Fatalf("expected AddCandle to succeed when validation disabled, got %v", err)
	}
}

/*
-------------------------------------------------------------

	Input validation
	-------------------------------------------------------------
*/
func TestAddCandle_InvalidHighLow(t *testing.T) {
	atr, _ := NewAverageTrueRange()
	if err := atr.AddCandle(5, 6, 5.5); err == nil {
		t.Fatalf("expected error when high < low")
	}
}

func TestAddCandle_InvalidPrices(t *testing.T) {
	atr, _ := NewAverageTrueRange()
	invalid := []float64{math.NaN(), math.Inf(1), -1.0}
	for _, v := range invalid {
		if err := atr.AddCandle(v, 1, 1); err == nil {
			t.Fatalf("expected error for invalid high %v", v)
		}
		if err := atr.AddCandle(1, v, 1); err == nil {
			t.Fatalf("expected error for invalid low %v", v)
		}
		if err := atr.AddCandle(1, 0, v); err == nil {
			t.Fatalf("expected error for invalid close %v", v)
		}
	}
}

func TestAddCandle_CloseOutOfBounds(t *testing.T) {
	atr, _ := NewAverageTrueRange()
	// close lower than low
	if err := atr.AddCandle(10, 9, 8); err == nil {
		t.Fatalf("expected error when close < low")
	}
	// close higher than high
	if err := atr.AddCandle(10, 9, 11); err == nil {
		t.Fatalf("expected error when close > high")
	}
}

/*
-------------------------------------------------------------

	Core ATR calculation – compare against hand‑computed values
	-------------------------------------------------------------
*/
func TestATR_Calculation_SimpleSeries(t *testing.T) {
	// Use a tiny period so we can verify easily.
	period := 3
	atr, err := NewAverageTrueRangeWithParams(period)
	if err != nil {
		t.Fatalf("constructor error: %v", err)
	}

	// Deterministic series: high = base+1, low = base-1, close = base
	highs, lows, closes := generateOHLC(10, 1, 6) // 6 candles → enough for 2 ATR outputs

	// Feed data
	for i := 0; i < len(highs); i++ {
		if err := atr.AddCandle(highs[i], lows[i], closes[i]); err != nil {
			t.Fatalf("AddCandle failed at i=%d: %v", i, err)
		}
	}

	// After feeding 6 candles with period=3 we should have (6‑(3+1)+1)=3 ATR values.
	if len(atr.GetATRValues()) != 3 {
		t.Fatalf("expected 3 ATR values, got %d", len(atr.GetATRValues()))
	}

	/*
	   Manual calculation (period = 3):
	   TR_i = max( high_i‑low_i,
	                |high_i‑close_{i‑1}|,
	                |low_i‑close_{i‑1}| )
	   For our synthetic series:
	     high_i‑low_i = 2
	     |high_i‑close_{i‑1}| = |(base+1)‑(base‑1)| = 2
	     |low_i‑close_{i‑1}| = |(base‑1)‑(base‑1)| = 0
	   => TR_i = 2 for every i ≥ 1
	   ATR = average(TR over last 3 periods) = 2
	*/

	expectedATR := 2.0
	val, err := atr.Calculate()
	if err != nil {
		t.Fatalf("Calculate error: %v", err)
	}
	if math.Abs(val-expectedATR) > 1e-9 {
		t.Fatalf("ATR mismatch: expected %.2f, got %.6f", expectedATR, val)
	}
}

/*
-------------------------------------------------------------

	Period change – should reset internal state
	-------------------------------------------------------------
*/
func TestATR_SetPeriod_ResetsState(t *testing.T) {
	atr, _ := NewAverageTrueRangeWithParams(5)
	highs, lows, closes := generateOHLC(1, 1, 7)
	for i := 0; i < len(highs); i++ {
		_ = atr.AddCandle(highs[i], lows[i], closes[i])
	}
	if len(atr.GetATRValues()) == 0 {
		t.Fatalf("expected some ATR values before period change")
	}
	// Change period – internal slices must be cleared.
	if err := atr.SetPeriod(3); err != nil {
		t.Fatalf("SetPeriod error: %v", err)
	}
	if atr.period != 3 {
		t.Fatalf("period not updated")
	}
	if len(atr.GetATRValues()) != 0 || len(atr.GetHighs()) != 0 {
		t.Fatalf("state not cleared after SetPeriod")
	}
}

/*
-------------------------------------------------------------

	Reset – clears everything but retains period
	-------------------------------------------------------------
*/
func TestATR_Reset(t *testing.T) {
	atr, _ := NewAverageTrueRangeWithParams(4)
	highs, lows, closes := generateOHLC(5, 0.5, 6)
	for i := 0; i < len(highs); i++ {
		_ = atr.AddCandle(highs[i], lows[i], closes[i])
	}
	if len(atr.GetATRValues()) == 0 {
		t.Fatalf("expected ATR values before Reset")
	}
	atr.Reset()
	if atr.lastValue != 0 {
		t.Fatalf("lastValue not cleared")
	}
	if len(atr.GetATRValues()) != 0 || len(atr.GetHighs()) != 0 {
		t.Fatalf("internal slices not cleared after Reset")
	}
	if atr.period != 4 {
		t.Fatalf("period should stay unchanged after Reset")
	}
}

/*
-------------------------------------------------------------

	Defensive copy getters – modifying the returned slice must not
	affect the internal state.
	-------------------------------------------------------------
*/
func TestATR_Getters_DefensiveCopy(t *testing.T) {
	atr, _ := NewAverageTrueRangeWithParams(2)
	highs, lows, closes := generateOHLC(10, 1, 4)
	for i := 0; i < len(highs); i++ {
		_ = atr.AddCandle(highs[i], lows[i], closes[i])
	}
	origHighs := atr.GetHighs()
	origLows := atr.GetLows()
	origCloses := atr.GetCloses()
	origATR := atr.GetATRValues()

	// Mutate the returned slices
	origHighs[0] = -999
	origLows[0] = -999
	origCloses[0] = -999
	origATR[0] = -999

	// Ensure internal slices stayed intact
	if atr.highs[0] == -999 || atr.lows[0] == -999 || atr.closes[0] == -999 {
		t.Fatalf("internal slice modified through getter")
	}
	if len(atr.atrValues) > 0 && atr.atrValues[0] == -999 {
		t.Fatalf("ATR slice modified through getter")
	}
}

/*
-------------------------------------------------------------

	Error paths – Calculate before any ATR value is produced
	-------------------------------------------------------------
*/
func TestATR_Calculate_NoData(t *testing.T) {
	atr, _ := NewAverageTrueRange()
	if _, err := atr.Calculate(); err == nil {
		t.Fatalf("expected error when Calculate called before any ATR data")
	}
}

/*
-------------------------------------------------------------

	Edge case – period = 1 (ATR reduces to true range of latest bar)
	-------------------------------------------------------------
*/
func TestATR_PeriodOne(t *testing.T) {
	atr, _ := NewAverageTrueRangeWithParams(1)
	// First candle – not enough data yet (need period+1 = 2 closes)
	if err := atr.AddCandle(10, 9, 9.5); err != nil {
		t.Fatalf("first AddCandle failed: %v", err)
	}
	if _, err := atr.Calculate(); err == nil {
		t.Fatalf("expected error after only one candle with period=1")
	}
	// Second candle – now we have two closes, ATR = true range of second bar
	if err := atr.AddCandle(11, 9, 10); err != nil {
		t.Fatalf("second AddCandle failed: %v", err)
	}
	val, err := atr.Calculate()
	if err != nil {
		t.Fatalf("Calculate error: %v", err)
	}
	/*
	   true range for second bar:
	     high‑low = 2
	     |high‑prevClose| = |11‑9.5| = 1.5
	     |low‑prevClose|  = |9‑9.5| = 0.5
	     TR = max(2,1.5,0.5) = 2
	*/
	if math.Abs(val-2.0) > 1e-9 {
		t.Fatalf("period=1 ATR mismatch: expected 2.0, got %.6f", val)
	}
}

/*
-------------------------------------------------------------

	Stress test – feed many candles and ensure slices never grow
	beyond their configured capacity.
	-------------------------------------------------------------
*/
func TestATR_SliceLimits(t *testing.T) {
	const period = 10
	atr, _ := NewAverageTrueRangeWithParams(period)

	// Feed 1 000 random‑looking candles
	for i := 0; i < 1000; i++ {
		h := float64(100 + i%5) // high
		l := h - 2.0            // low
		c := l + 1.0            // close in the middle
		if err := atr.AddCandle(h, l, c); err != nil {
			t.Fatalf("AddCandle failed at i=%d: %v", i, err)
		}
		// Verify slice caps
		if len(atr.highs) > period+1 || len(atr.lows) > period+1 || len(atr.closes) > period+1 {
			t.Fatalf("OHLC slices exceeded cap after i=%d", i)
		}
		if len(atr.atrValues) > period {
			t.Fatalf("ATR values slice exceeded cap after i=%d", i)
		}
	}
}
