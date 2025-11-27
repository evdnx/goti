package volume

import (
	"math"
	"testing"
)

func TestVWAP_Calculation(t *testing.T) {
	vwap := NewVWAP()

	candles := []struct {
		h, l, c, v float64
	}{
		{10, 8, 9, 2},
		{11, 9, 10, 1},
	}

	for i, c := range candles {
		if err := vwap.Add(c.h, c.l, c.c, c.v); err != nil {
			t.Fatalf("Add failed at idx %d: %v", i, err)
		}
	}

	val, err := vwap.Calculate()
	if err != nil {
		t.Fatalf("Calculate returned error: %v", err)
	}

	// Expected VWAP: ((9*2) + (10*1)) / (2+1) = 28/3 ≈ 9.3333
	if math.Abs(val-9.333333) > 1e-6 {
		t.Fatalf("unexpected VWAP: got %.6f, want ~9.333333", val)
	}
}

func TestVWAP_InvalidInput(t *testing.T) {
	vwap := NewVWAP()
	if err := vwap.Add(10, 9, 9.5, -1); err == nil {
		t.Fatal("expected error for negative volume")
	}
}
