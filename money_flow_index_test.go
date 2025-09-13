package goti

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helper – creates a MFI instance with a deterministic config
// ---------------------------------------------------------------------------
func newTestMFI(t *testing.T) *MoneyFlowIndex {
	cfg := DefaultConfig()
	// Use a small, easy‑to‑verify scale factor
	cfg.MFIVolumeScale = 1.0
	mfi, err := NewMoneyFlowIndexWithParams(3, cfg)
	require.NoError(t, err)
	return mfi
}

// ---------------------------------------------------------------------------
// Constructor tests
// ---------------------------------------------------------------------------
func TestNewMoneyFlowIndex_Validation(t *testing.T) {
	// Invalid period
	_, err := NewMoneyFlowIndexWithParams(0, DefaultConfig())
	assert.Error(t, err)

	// Overbought <= oversold
	badCfg := DefaultConfig()
	badCfg.MFIOverbought = 20
	badCfg.MFIOversold = 30
	_, err = NewMoneyFlowIndexWithParams(5, badCfg)
	assert.Error(t, err)

	// Invalid config (ATSEMAperiod <= 0)
	badCfg = DefaultConfig()
	badCfg.ATSEMAperiod = 0
	_, err = NewMoneyFlowIndexWithParams(5, badCfg)
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// Add validation tests
// ---------------------------------------------------------------------------
func TestMoneyFlowIndex_Add_Validation(t *testing.T) {
	mfi := newTestMFI(t)

	// High < low
	err := mfi.Add(10, 12, 11, 1000)
	assert.Error(t, err)

	// Negative close
	err = mfi.Add(12, 10, -5, 1000)
	assert.Error(t, err)

	// Negative volume
	err = mfi.Add(12, 10, 11, -100)
	assert.Error(t, err)

	// Valid first entry – should not produce an MFI yet
	err = mfi.Add(12, 10, 11, 1000)
	assert.NoError(t, err)

	// Still no MFI because we need period+1 = 4 samples
	_, err = mfi.Calculate()
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// MFI calculation – basic known sequence
// ---------------------------------------------------------------------------
func TestMoneyFlowIndex_Calculation_Basic(t *testing.T) {
	mfi := newTestMFI(t)

	// Use a deterministic dataset where the expected MFI can be hand‑computed.
	// Period = 3, so we need 4 samples.
	data := []struct {
		high, low, close, vol float64
	}{
		{10, 8, 9, 1000},
		{11, 9, 10, 1200},
		{12, 10, 11, 1500},
		{13, 11, 12, 1800},
	}
	for _, d := range data {
		require.NoError(t, mfi.Add(d.high, d.low, d.close, d.vol))
	}

	// After the fourth sample we should have exactly one MFI value.
	val, err := mfi.Calculate()
	require.NoError(t, err)

	// Manually compute expected MFI:
	//
	// Typical price (TP) = (H+L+C)/3
	// Money flow = TP * volume (scale = 1)
	//
	// Sample 1 TP = (10+8+9)/3 = 9
	// Sample 2 TP = (11+9+10)/3 = 10
	// Sample 3 TP = (12+10+11)/3 = 11
	// Sample 4 TP = (13+11+12)/3 = 12
	//
	// Positive MF = sum(TP_i * vol_i) where close_i > close_{i-1}
	//   2→3 : close rises (11 > 10) → TP3*1500 = 11*1500 = 16500
	//   3→4 : close rises (12 > 11) → TP4*1800 = 12*1800 = 21600
	// Positive MF = 38100
	//
	// Negative MF = sum where close falls – none in this upward run → 0
	//
	// Money Ratio = ∞ → MFI = 100 (our implementation returns 100 when negativeMF==0)
	expected := 100.0
	assert.InDelta(t, expected, val, 1e-9)
}

// ---------------------------------------------------------------------------
// Edge‑case handling – zero positive or negative money flow
// ---------------------------------------------------------------------------
func TestMoneyFlowIndex_EdgeCases_ZeroFlows(t *testing.T) {
	mfi := newTestMFI(t)

	// Downward price movement → only negative money flow
	down := []struct{ h, l, c, v float64 }{
		{10, 8, 9, 1000},
		{9, 7, 8, 1100},
		{8, 6, 7, 1200},
		{7, 5, 6, 1300},
	}
	for _, d := range down {
		require.NoError(t, mfi.Add(d.h, d.l, d.c, d.v))
	}
	val, err := mfi.Calculate()
	require.NoError(t, err)
	assert.Equal(t, 0.0, val) // pure negative flow → 0

	// Reset and test the neutral case (price unchanged → both flows zero)
	mfi.Reset()
	flat := []struct{ h, l, c, v float64 }{
		{10, 8, 9, 1000},
		{10, 8, 9, 1000},
		{10, 8, 9, 1000},
		{10, 8, 9, 1000},
	}
	for _, f := range flat {
		require.NoError(t, mfi.Add(f.h, f.l, f.c, f.v))
	}
	val, err = mfi.Calculate()
	require.NoError(t, err)
	assert.Equal(t, 50.0, val) // neutral case → 50
}

// ---------------------------------------------------------------------------
// Crossover detection
// ---------------------------------------------------------------------------
func TestMoneyFlowIndex_Crossovers(t *testing.T) {
	mfi := newTestMFI(t)

	// Build a sequence that goes from oversold (<20) to above oversold (cross up)
	seq := []struct{ h, l, c, v float64 }{
		{10, 8, 9, 1000},
		{11, 9, 10, 1100},
		{12, 10, 11, 1200},
		{13, 11, 12, 1300}, // after this MFI = 100 (above oversold)
	}
	for _, d := range seq {
		require.NoError(t, mfi.Add(d.h, d.l, d.c, d.v))
	}
	bull, err := mfi.IsBullishCrossover()
	require.NoError(t, err)
	assert.True(t, bull)

	// Now add a point that pushes MFI above overbought (80) then drops below it
	// to trigger a bearish crossover.
	add := []struct{ h, l, c, v float64 }{
		{14, 12, 13, 1400}, // still upward → MFI stays 100
		{13, 11, 12, 1500}, // price falls → negative flow appears, MFI will drop
	}
	for _, d := range add {
		require.NoError(t, mfi.Add(d.h, d.l, d.c, d.v))
	}
	bear, err := mfi.IsBearishCrossover()
	require.NoError(t, err)
	assert.True(t, bear)
}

// ---------------------------------------------------------------------------
// Overbought / Oversold zone reporting
// ---------------------------------------------------------------------------
func TestMoneyFlowIndex_ZoneReporting(t *testing.T) {
	mfi := newTestMFI(t)

	// Force an overbought value (>80)
	over := []struct{ h, l, c, v float64 }{
		{10, 8, 9, 1000},
		{11, 9, 10, 1100},
		{12, 10, 11, 1200},
		{13, 11, 12, 1300},
	}
	for _, d := range over {
		require.NoError(t, mfi.Add(d.h, d.l, d.c, d.v))
	}
	zone, err := mfi.GetOverboughtOversold()
	require.NoError(t, err)
	assert.Equal(t, "Overbought", zone)

	// Reset and force an oversold value (<20)
	mfi.Reset()
	under := []struct{ h, l, c, v float64 }{
		{13, 11, 12, 1300},
		{12, 10, 11, 1200},
		{11, 9, 10, 1100},
		{10, 8, 9, 1000},
	}
	for _, d := range under {
		require.NoError(t, mfi.Add(d.h, d.l, d.c, d.v))
	}
	zone, err = mfi.GetOverboughtOversold()
	require.NoError(t, err)
	assert.Equal(t, "Oversold", zone)
}

// ---------------------------------------------------------------------------
// Reset functionality
// ---------------------------------------------------------------------------
func TestMoneyFlowIndex_Reset(t *testing.T) {
	mfi := newTestMFI(t)

	seq := []struct{ h, l, c, v float64 }{
		{10, 8, 9, 1000},
		{11, 9, 10, 1100},
		{12, 10, 11, 1200},
		{13, 11, 12, 1300},
	}
	for _, d := range seq {
		require.NoError(t, mfi.Add(d.h, d.l, d.c, d.v))
	}
	// Ensure we have data before resetting
	_, err := mfi.Calculate()
	require.NoError(t, err)

	mfi.Reset()

	// All slices should be empty and lastValue zero.
	assert.Empty(t, mfi.GetValues())
	assert.Equal(t, 0.0, mfi.GetLastValue())

	_, err = mfi.Calculate()
	assert.Error(t, err) // no data after reset
}

// ---------------------------------------------------------------------------
// Divergence detection
// ---------------------------------------------------------------------------

func TestMoneyFlowIndex_Divergence(t *testing.T) {
	// ---------------------------------------------------------------------------
	// Helper to create a MoneyFlowIndex with a custom period.
	// ---------------------------------------------------------------------------
	newMFI := func(period int) *MoneyFlowIndex {
		mfi, err := NewMoneyFlowIndexWithParams(period, DefaultConfig())
		if err != nil {
			t.Fatalf("failed to create MoneyFlowIndex: %v", err)
		}
		return mfi
	}

	// ---------------------------------------------------------------------------
	// Helper to feed OHLCV samples into the indicator.
	// ---------------------------------------------------------------------------
	addSamples := func(mfi *MoneyFlowIndex, samples [][4]float64) {
		for _, s := range samples {
			if err := mfi.Add(s[0], s[1], s[2], s[3]); err != nil {
				t.Fatalf("Add failed: %v", err)
			}
		}
	}

	// ---------------------------------------------------------------
	// 1️⃣ Bullish classic divergence
	//    – price makes a lower low, MFI makes a higher low.
	// ---------------------------------------------------------------
	t.Run("BullishClassic", func(t *testing.T) {
		mfi := newMFI(2) // short look‑back so MFI appears early
		//
		//  price  vol
		//  10.0   1000   (no MFI yet)
		//   9.0   1000   (down day → negative MF)
		//   9.5   3000   (up day   → big positive MF)
		//   8.5   1000   (down day → negative MF)
		//
		// After the 3rd sample the first MFI is computed.
		// After the 4th sample we have a second MFI that is higher,
		// while the price has made a lower low → bullish classic divergence.
		samples := [][4]float64{
			{10, 9, 9.5, 1000},
			{9, 8, 8.5, 1000},
			{9.5, 8.5, 9.0, 3000},
			{8.5, 7.5, 8.0, 1000},
		}
		addSamples(mfi, samples)

		div, err := mfi.IsDivergence()
		if err != nil {
			t.Fatalf("DetectClassicDivergence returned error: %v", err)
		}
		if div != "bullish" {
			t.Fatalf("expected classic bullish, got %s", div)
		}
	})

	// ---------------------------------------------------------------
	// 2️⃣ Bearish classic divergence
	//    – price makes a higher high, MFI makes a lower high.
	// ---------------------------------------------------------------
	t.Run("BearishClassic", func(t *testing.T) {
		mfi := newMFI(2)
		//
		//  price  vol
		//   8.0   1000   (no MFI yet)
		//   9.0   1000   (up day → positive MF)
		//   8.5   5000   (down day → large negative MF)
		//   9.5   100    (up day → tiny positive MF)
		//
		// First MFI after the 3rd sample is relatively high
		// (because the negative MF dominates). After the 4th sample
		// the second MFI drops, while price has made a higher high →
		// bearish classic divergence.
		samples := [][4]float64{
			{8, 7, 7.5, 1000},
			{9, 8, 8.5, 1000},
			{8.5, 8, 8.0, 5000},
			{9.5, 9, 9.0, 100},
		}
		addSamples(mfi, samples)

		div, err := mfi.IsDivergence()
		if err != nil {
			t.Fatalf("DetectClassicDivergence returned error: %v", err)
		}
		if div != "bearish" {
			t.Fatalf("expected classic bearish, got %s", div)
		}
	})
}

// ---------------------------------------------------------------------------
// Plot data generation – sanity checks
// ---------------------------------------------------------------------------
func TestMoneyFlowIndex_GetPlotData(t *testing.T) {
	mfi := newTestMFI(t)

	seq := []struct{ h, l, c, v float64 }{
		{10, 8, 9, 1000},
		{11, 9, 10, 1100},
		{12, 10, 11, 1200},
		{13, 11, 12, 1300},
	}
	for _, d := range seq {
		require.NoError(t, mfi.Add(d.h, d.l, d.c, d.v))
	}
	plots, err := mfi.GetPlotData()
	require.NoError(t, err)
	require.Len(t, plots, 2)

	// First series should be the line with the same number of points as MFI values.
	line := plots[0]
	assert.Equal(t, "MFI", line.Name)
	assert.Equal(t, "line", line.Type)
	assert.Len(t, line.Y, len(mfi.GetValues()))
	assert.Len(t, line.X, len(mfi.GetValues()))

	// Second series should be the scatter with signal markers.
	sig := plots[1]
	assert.Equal(t, "Signals", sig.Name)
	assert.Equal(t, "scatter", sig.Type)
	assert.Len(t, sig.Y, len(mfi.GetValues()))
}

// ---------------------------------------------------------------------------
// JSON marshalling sanity – ensures PlotData structs are serialisable
// ---------------------------------------------------------------------------
func TestPlotData_JSONMarshalling(t *testing.T) {
	p := PlotData{
		Name: "test",
		X:    []float64{0, 1, 2},
		Y:    []float64{10, 20, 30},
		Type: "line",
	}
	b, err := json.Marshal(p)
	require.NoError(t, err)
	var decoded PlotData
	require.NoError(t, json.Unmarshal(b, &decoded))
	assert.Equal(t, p, decoded)
}

// ---------------------------------------------------------------------------
// Ensure that calling Calculate without any data returns a clear error.
// ---------------------------------------------------------------------------
func TestMoneyFlowIndex_Calculate_NoData(t *testing.T) {
	mfi := newTestMFI(t)
	_, err := mfi.Calculate()
	assert.True(t, errors.Is(err, errors.New("no MFI data")))
}
