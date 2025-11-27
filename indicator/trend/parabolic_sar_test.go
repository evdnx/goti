package trend

import "testing"

func TestParabolicSAR_UptrendCalculation(t *testing.T) {
	sar, err := NewParabolicSAR()
	if err != nil {
		t.Fatalf("constructor error: %v", err)
	}

	data := []struct {
		h, l float64
	}{
		{10, 9},
		{11, 10},
		{12, 11},
		{13, 12},
	}

	for i, d := range data {
		if err := sar.Add(d.h, d.l); err != nil {
			t.Fatalf("Add failed at idx %d: %v", i, err)
		}
	}

	val, err := sar.Calculate()
	if err != nil {
		t.Fatalf("Calculate returned error: %v", err)
	}
	if !approxEqual(val, 9.12) {
		t.Fatalf("unexpected SAR: got %.4f, want ~9.12", val)
	}
	if !sar.IsUptrend() {
		t.Fatal("expected ongoing uptrend")
	}
}

func TestParabolicSAR_ReversalToDowntrend(t *testing.T) {
	sar, _ := NewParabolicSAR()

	data := []struct {
		h, l float64
	}{
		{10, 9},
		{11, 10},
		{12, 11},
		{13, 12},
		{12, 8}, // drop -> reversal
	}

	for _, d := range data {
		if err := sar.Add(d.h, d.l); err != nil {
			t.Fatalf("Add failed: %v", err)
		}
	}

	val, err := sar.Calculate()
	if err != nil {
		t.Fatalf("Calculate returned error: %v", err)
	}
	if !approxEqual(val, 13) {
		t.Fatalf("unexpected SAR after reversal: got %.4f, want 13", val)
	}
	if sar.IsUptrend() {
		t.Fatal("expected downtrend after reversal")
	}
}
