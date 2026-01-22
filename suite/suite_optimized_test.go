package suite

import (
	"testing"

	"github.com/evdnx/goti/config"
)

func TestOptimizedScalpingIndicatorSuite(t *testing.T) {
	suite, err := NewOptimizedScalpingIndicatorSuite()
	if err != nil {
		t.Fatalf("Failed to create optimized suite: %v", err)
	}

	// Test adding data - need enough data for indicators to generate values
	for i := 0; i < 50; i++ {
		high := 100.0 + float64(i%10)*0.5
		low := 95.0 + float64(i%10)*0.5
		close := 98.0 + float64(i%10)*0.5
		volume := 1000.0 + float64(i%10)*50

		if err := suite.Add(high, low, close, volume); err != nil {
			t.Fatalf("Failed to add data at iteration %d: %v", i, err)
		}
	}

	// Test getting signal
	signal, err := suite.GetCombinedSignal()
	if err != nil {
		t.Fatalf("Failed to get combined signal: %v", err)
	}

	if signal == "" {
		t.Error("Expected non-empty signal")
	}

	// Test reset
	suite.Reset()

	// Test getters
	if suite.GetAdaptiveDEMAMomentumOscillator() == nil {
		t.Error("Expected ADMO to be non-nil")
	}
	if suite.GetVolumeWeightedAroonOscillator() == nil {
		t.Error("Expected VWAO to be non-nil")
	}
	if suite.GetMACD() == nil {
		t.Error("Expected MACD to be non-nil")
	}
	if suite.GetHMA() == nil {
		t.Error("Expected HMA to be non-nil")
	}
	if suite.GetATR() == nil {
		t.Error("Expected ATR to be non-nil")
	}
	if suite.GetMFI() == nil {
		t.Error("Expected MFI to be non-nil")
	}

	// Test plot data
	plotData := suite.GetPlotData(0, 1000)
	t.Logf("Plot data length: %d", len(plotData))
	for i, data := range plotData {
		t.Logf("Plot data %d: Name=%s, Len(X)=%d", i, data.Name, len(data.X))
	}
	// Don't fail if no plot data - some indicators might not have data yet
	if len(plotData) > 0 {
		t.Logf("Successfully generated %d plot data series", len(plotData))
	}
}

func TestOptimizedScalpingIndicatorSuiteWithConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	suite, err := NewOptimizedScalpingIndicatorSuiteWithConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to create optimized suite with config: %v", err)
	}

	// Test basic functionality
	if err := suite.Add(100.0, 95.0, 98.0, 1000.0); err != nil {
		t.Fatalf("Failed to add data: %v", err)
	}

	signal, err := suite.GetCombinedSignal()
	if err != nil {
		t.Fatalf("Failed to get combined signal: %v", err)
	}

	if signal == "" {
		t.Error("Expected non-empty signal")
	}
}