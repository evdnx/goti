package momentum

import "testing"

func TestNewMACD_InvalidPeriods(t *testing.T) {
	if _, err := NewMACDWithParams(0, 10, 3); err == nil {
		t.Fatal("expected error for fast period < 1")
	}
	if _, err := NewMACDWithParams(10, 5, 3); err == nil {
		t.Fatal("expected error when fast >= slow")
	}
}

func TestMACD_NotReady(t *testing.T) {
	macd, err := NewMACD()
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	// Fewer than the slow period – no MACD values yet.
	for i := 1; i <= 5; i++ {
		if err := macd.Add(float64(i)); err != nil {
			t.Fatalf("Add failed: %v", err)
		}
	}
	if _, _, _, err := macd.Calculate(); err == nil {
		t.Fatal("expected error when MACD not ready")
	}
}

func TestMACD_AddAndCalculate(t *testing.T) {
	// Use small periods so values appear quickly.
	macd, err := NewMACDWithParams(3, 6, 3)
	if err != nil {
		t.Fatalf("constructor error: %v", err)
	}

	// Eight steadily increasing closes yield a stable MACD of 1.5 when using
	// 3/6 EMAs. The first signal value (period 3) therefore equals 1.5 and the
	// histogram is zero.
	closes := []float64{1, 2, 3, 4, 5, 6, 7, 8}
	for _, c := range closes {
		if err := macd.Add(c); err != nil {
			t.Fatalf("Add(%v) failed: %v", c, err)
		}
	}

	macdVal, sigVal, histVal, err := macd.Calculate()
	if err != nil {
		t.Fatalf("Calculate returned error: %v", err)
	}

	if !approxEqual(macdVal, 1.5) {
		t.Fatalf("MACD mismatch: got %.6f, want 1.5", macdVal)
	}
	if !approxEqual(sigVal, 1.5) {
		t.Fatalf("Signal mismatch: got %.6f, want 1.5", sigVal)
	}
	if !approxEqual(histVal, 0) {
		t.Fatalf("Histogram mismatch: got %.6f, want 0", histVal)
	}
}
