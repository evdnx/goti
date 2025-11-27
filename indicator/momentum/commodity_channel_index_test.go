package momentum

import (
	"math"
	"testing"
)

func TestCommodityChannelIndex_Calculation(t *testing.T) {
	cci, err := NewCommodityChannelIndexWithParams(3)
	if err != nil {
		t.Fatalf("constructor error: %v", err)
	}

	bars := []struct {
		h, l, c float64
	}{
		{10, 8, 9},
		{11, 9, 10},
		{12, 10, 11}, // first CCI
	}

	for i, b := range bars {
		if err := cci.Add(b.h, b.l, b.c); err != nil {
			t.Fatalf("Add failed at idx %d: %v", i, err)
		}
	}

	val, err := cci.Calculate()
	if err != nil {
		t.Fatalf("Calculate returned error: %v", err)
	}

	// With the three bars above, TP values are [9,10,11]:
	//   SMA = 10, mean deviation = 2/3, CCI = (11-10)/(0.015*(2/3)) = 100.
	if math.Abs(val-100) > 1e-6 {
		t.Fatalf("unexpected CCI: got %.6f, want 100", val)
	}
	ob, err := cci.IsOverbought()
	if err != nil {
		t.Fatalf("IsOverbought error: %v", err)
	}
	if !ob {
		t.Fatal("expected overbought at +100")
	}

	// Add one more bar to swing the CCI negative.
	if err := cci.Add(10, 8, 9); err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	val, err = cci.Calculate()
	if err != nil {
		t.Fatalf("Calculate returned error: %v", err)
	}
	if math.Abs(val+100) > 1e-6 {
		t.Fatalf("unexpected CCI after drop: got %.6f, want -100", val)
	}
	os, err := cci.IsOversold()
	if err != nil {
		t.Fatalf("IsOversold error: %v", err)
	}
	if !os {
		t.Fatal("expected oversold at -100")
	}
}

func TestCommodityChannelIndex_InvalidPeriod(t *testing.T) {
	if _, err := NewCommodityChannelIndexWithParams(0); err == nil {
		t.Fatal("expected error for period < 1")
	}
}
