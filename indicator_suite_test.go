package goti

import (
	"strings"
	"testing"
)

func TestScalpingIndicatorSuite(t *testing.T) {
	cfg := DefaultConfig()
	suite, err := NewScalpingIndicatorSuiteWithConfig(cfg)
	if err != nil {
		t.Fatalf("NewScalpingIndicatorSuiteWithConfig failed: %v", err)
	}

	t.Run("invalid add is rejected", func(t *testing.T) {
		err := suite.Add(100, 101, 100, 1000) // high < low
		if err == nil || !strings.Contains(err.Error(), "invalid price") {
			t.Fatalf("expected validation error, got %v", err)
		}
	})

	t.Run("bullish bias after recovery", func(t *testing.T) {
		suite.Reset()
		price := 200.0
		// Dip first to prime oversold conditions.
		for i := 0; i < 15; i++ {
			price -= 1.5
			if err := suite.Add(price+1, price-1, price, 5000+float64(i)*50); err != nil {
				t.Fatalf("add during dip failed at %d: %v", i, err)
			}
		}
		// Then drive a fast rebound to trigger low-lag crossovers.
		for i := 0; i < 40; i++ {
			price += 2.5
			if err := suite.Add(price+1.2, price-1.2, price, 9000+float64(i)*80); err != nil {
				t.Fatalf("add during rebound failed at %d: %v", i, err)
			}
		}

		signal, err := suite.GetCombinedSignal()
		if err != nil {
			t.Fatalf("GetCombinedSignal failed: %v", err)
		}
		if !strings.Contains(signal, "Bullish") {
			t.Fatalf("expected bullish signal, got %s", signal)
		}

		plotData := suite.GetPlotData(1625097600000, 60_000)
		if len(plotData) == 0 {
			t.Fatal("expected non-empty plot data")
		}
	})

	t.Run("bearish bias on sustained drop", func(t *testing.T) {
		suite.Reset()
		price := 420.0
		for i := 0; i < 60; i++ {
			price -= 2.0
			if err := suite.Add(price+1.5, price-1.5, price, 8000+float64(i)*30); err != nil {
				t.Fatalf("add during selloff failed at %d: %v", i, err)
			}
		}

		signal, err := suite.GetCombinedSignal()
		if err != nil {
			t.Fatalf("GetCombinedSignal failed: %v", err)
		}
		if !strings.Contains(signal, "Bearish") {
			t.Fatalf("expected bearish signal, got %s", signal)
		}
	})
}
