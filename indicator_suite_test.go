package goti

import (
	"strings"
	"testing"
)

func TestIndicatorSuite(t *testing.T) {
	// Test initialization
	config := DefaultConfig()
	suite, err := NewIndicatorSuiteWithConfig(config)
	if err != nil {
		t.Fatalf("NewIndicatorSuiteWithConfig failed: %v", err)
	}

	// Test adding invalid data
	err = suite.Add(100, 101, 100, 1000)
	if err == nil || !strings.Contains(err.Error(), "invalid price") {
		t.Errorf("Expected error containing 'invalid price', got: %v", err)
	}

	// Test combined signal (bullish)
	for i := 0; i < 45; i++ {
		high := float64(10000 - i*100 + 1)
		low := float64(10000 - i*100 - 1)
		close := float64(10000 - i*100)
		volume := 1000000.0
		err := suite.Add(high, low, close, volume)
		if err != nil {
			t.Errorf("Add failed at index %d: %v", i, err)
		}
	}
	suite.Add(101, 99, 100, 1000000)
	suite.Add(301, 299, 300, 1000000)
	suite.Add(601, 599, 600, 1000000)
	suite.Add(1001, 999, 1000, 1000000)
	suite.Add(1501, 1499, 1500, 1000000)
	rsiBullish, _ := suite.rsi.IsBullishCrossover()
	mfiBullish, _ := suite.mfi.IsBullishCrossover()
	vwaoBullish, _ := suite.vwao.IsBullishCrossover()
	hmaBullish, _ := suite.hma.IsBullishCrossover()
	amdoBullish, _ := suite.amdo.IsBullishCrossover()
	atsoBullish := suite.atso.IsBullishCrossover()
	t.Logf("RSI: %v, MFI: %v, VWAO: %v, HMA: %v, AMDO: %v, ATSO: %v", rsiBullish, mfiBullish, vwaoBullish, hmaBullish, amdoBullish, atsoBullish)
	signal, err := suite.GetCombinedSignal()
	if err != nil {
		t.Errorf("GetCombinedSignal failed: %v", err)
	}
	if signal != "Strong Bullish" {
		t.Errorf("Expected Strong Bullish signal, got: %s", signal)
	}

	// Test combined signal (neutral)
	suite.Reset()
	for i := 0; i < 50; i++ {
		high := float64(100 + i*20 + 1)
		low := float64(100 + i*20 - 1)
		close := float64(100 + i*20)
		volume := 1000000.0
		err := suite.Add(high, low, close, volume)
		if err != nil {
			t.Errorf("Add failed at index %d: %v", i, err)
		}
	}
	signal, err = suite.GetCombinedSignal()
	if err != nil {
		t.Errorf("GetCombinedSignal failed: %v", err)
	}
	if signal != "Neutral" {
		t.Errorf("Expected Neutral signal, got: %s", signal)
	}

	// Test plot data
	plotData := suite.GetPlotData(1625097600000, 86400000)
	if len(plotData) == 0 {
		t.Error("Expected non-empty plot data")
	}
}
