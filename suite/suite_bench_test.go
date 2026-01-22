package suite

import (
	"testing"

	"github.com/evdnx/goti/config"
)

// BenchmarkScalpingIndicatorSuite_Add tests the performance of adding data to the suite
func BenchmarkScalpingIndicatorSuite_Add(b *testing.B) {
	suite, err := NewScalpingIndicatorSuiteWithConfig(config.DefaultConfig())
	if err != nil {
		b.Fatalf("Failed to create suite: %v", err)
	}

	// Use deterministic data to avoid allocations in the benchmark loop
	high, low, close, volume := 100.0, 95.0, 98.0, 1000.0

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Vary the input slightly to prevent compiler optimizations
		h := high + float64(i%10)*0.1
		l := low + float64(i%10)*0.1
		c := close + float64(i%10)*0.1
		v := volume + float64(i%10)*10

		if err := suite.Add(h, l, c, v); err != nil {
			b.Fatalf("Add failed: %v", err)
		}
	}
}

// BenchmarkScalpingIndicatorSuite_GetCombinedSignal tests the performance of signal calculation
func BenchmarkScalpingIndicatorSuite_GetCombinedSignal(b *testing.B) {
	suite, err := NewScalpingIndicatorSuiteWithConfig(config.DefaultConfig())
	if err != nil {
		b.Fatalf("Failed to create suite: %v", err)
	}

	// Pre-fill with enough data for meaningful calculations
	high, low, close, volume := 100.0, 95.0, 98.0, 1000.0
	for i := 0; i < 100; i++ {
		h := high + float64(i%20)*0.5
		l := low + float64(i%20)*0.5
		c := close + float64(i%20)*0.5
		v := volume + float64(i%20)*50

		if err := suite.Add(h, l, c, v); err != nil {
			b.Fatalf("Pre-fill Add failed: %v", err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := suite.GetCombinedSignal()
		if err != nil {
			b.Fatalf("GetCombinedSignal failed: %v", err)
		}
	}
}

// BenchmarkScalpingIndicatorSuite_FullCycle tests the full Add + GetCombinedSignal cycle
func BenchmarkScalpingIndicatorSuite_FullCycle(b *testing.B) {
	suite, err := NewScalpingIndicatorSuiteWithConfig(config.DefaultConfig())
	if err != nil {
		b.Fatalf("Failed to create suite: %v", err)
	}

	// Pre-fill with some initial data
	high, low, close, volume := 100.0, 95.0, 98.0, 1000.0
	for i := 0; i < 50; i++ {
		h := high + float64(i%10)*0.2
		l := low + float64(i%10)*0.2
		c := close + float64(i%10)*0.2
		v := volume + float64(i%10)*20

		if err := suite.Add(h, l, c, v); err != nil {
			b.Fatalf("Pre-fill Add failed: %v", err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Add new data
		h := high + float64(i%10)*0.1
		l := low + float64(i%10)*0.1
		c := close + float64(i%10)*0.1
		v := volume + float64(i%10)*10

		if err := suite.Add(h, l, c, v); err != nil {
			b.Fatalf("Add failed: %v", err)
		}

		// Get signal
		_, err := suite.GetCombinedSignal()
		if err != nil {
			b.Fatalf("GetCombinedSignal failed: %v", err)
		}
	}
}

// BenchmarkScalpingIndicatorSuite_GetPlotData tests plot data generation performance
func BenchmarkScalpingIndicatorSuite_GetPlotData(b *testing.B) {
	suite, err := NewScalpingIndicatorSuiteWithConfig(config.DefaultConfig())
	if err != nil {
		b.Fatalf("Failed to create suite: %v", err)
	}

	// Pre-fill with enough data for plot generation
	high, low, close, volume := 100.0, 95.0, 98.0, 1000.0
	for i := 0; i < 200; i++ {
		h := high + float64(i%30)*0.3
		l := low + float64(i%30)*0.3
		c := close + float64(i%30)*0.3
		v := volume + float64(i%30)*30

		if err := suite.Add(h, l, c, v); err != nil {
			b.Fatalf("Pre-fill Add failed: %v", err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = suite.GetPlotData(1625097600000, 60_000)
	}
}

// BenchmarkOptimizedScalpingIndicatorSuite_Add tests the performance of adding data to the optimized suite
func BenchmarkOptimizedScalpingIndicatorSuite_Add(b *testing.B) {
	suite, err := NewOptimizedScalpingIndicatorSuiteWithConfig(config.DefaultConfig())
	if err != nil {
		b.Fatalf("Failed to create optimized suite: %v", err)
	}

	// Use deterministic data to avoid allocations in the benchmark loop
	high, low, close, volume := 100.0, 95.0, 98.0, 1000.0

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Vary the input slightly to prevent compiler optimizations
		h := high + float64(i%10)*0.1
		l := low + float64(i%10)*0.1
		c := close + float64(i%10)*0.1
		v := volume + float64(i%10)*10

		if err := suite.Add(h, l, c, v); err != nil {
			b.Fatalf("Add failed: %v", err)
		}
	}
}

// BenchmarkOptimizedScalpingIndicatorSuite_GetCombinedSignal tests the performance of signal calculation for optimized suite
func BenchmarkOptimizedScalpingIndicatorSuite_GetCombinedSignal(b *testing.B) {
	suite, err := NewOptimizedScalpingIndicatorSuiteWithConfig(config.DefaultConfig())
	if err != nil {
		b.Fatalf("Failed to create optimized suite: %v", err)
	}

	// Pre-fill with enough data for meaningful calculations
	high, low, close, volume := 100.0, 95.0, 98.0, 1000.0
	for i := 0; i < 100; i++ {
		h := high + float64(i%20)*0.5
		l := low + float64(i%20)*0.5
		c := close + float64(i%20)*0.5
		v := volume + float64(i%20)*50

		if err := suite.Add(h, l, c, v); err != nil {
			b.Fatalf("Pre-fill Add failed: %v", err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := suite.GetCombinedSignal()
		if err != nil {
			b.Fatalf("GetCombinedSignal failed: %v", err)
		}
	}
}

// BenchmarkOptimizedScalpingIndicatorSuite_FullCycle tests the full Add + GetCombinedSignal cycle for optimized suite
func BenchmarkOptimizedScalpingIndicatorSuite_FullCycle(b *testing.B) {
	suite, err := NewOptimizedScalpingIndicatorSuiteWithConfig(config.DefaultConfig())
	if err != nil {
		b.Fatalf("Failed to create optimized suite: %v", err)
	}

	// Pre-fill with some initial data
	high, low, close, volume := 100.0, 95.0, 98.0, 1000.0
	for i := 0; i < 50; i++ {
		h := high + float64(i%10)*0.2
		l := low + float64(i%10)*0.2
		c := close + float64(i%10)*0.2
		v := volume + float64(i%10)*20

		if err := suite.Add(h, l, c, v); err != nil {
			b.Fatalf("Pre-fill Add failed: %v", err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Add new data
		h := high + float64(i%10)*0.1
		l := low + float64(i%10)*0.1
		c := close + float64(i%10)*0.1
		v := volume + float64(i%10)*10

		if err := suite.Add(h, l, c, v); err != nil {
			b.Fatalf("Add failed: %v", err)
		}

		// Get signal
		_, err := suite.GetCombinedSignal()
		if err != nil {
			b.Fatalf("GetCombinedSignal failed: %v", err)
		}
	}
}

// BenchmarkOptimizedScalpingIndicatorSuite_GetPlotData tests plot data generation performance for optimized suite
func BenchmarkOptimizedScalpingIndicatorSuite_GetPlotData(b *testing.B) {
	suite, err := NewOptimizedScalpingIndicatorSuiteWithConfig(config.DefaultConfig())
	if err != nil {
		b.Fatalf("Failed to create optimized suite: %v", err)
	}

	// Pre-fill with enough data for plot generation
	high, low, close, volume := 100.0, 95.0, 98.0, 1000.0
	for i := 0; i < 200; i++ {
		h := high + float64(i%30)*0.3
		l := low + float64(i%30)*0.3
		c := close + float64(i%30)*0.3
		v := volume + float64(i%30)*30

		if err := suite.Add(h, l, c, v); err != nil {
			b.Fatalf("Pre-fill Add failed: %v", err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = suite.GetPlotData(1625097600000, 60_000)
	}
}