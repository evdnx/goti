package volatility

import "testing"

func TestBollingerBands_Calculation(t *testing.T) {
	bb, err := NewBollingerBandsWithParams(3, 2)
	if err != nil {
		t.Fatalf("constructor error: %v", err)
	}

	closes := []float64{10, 12, 14}
	for i, c := range closes {
		if err := bb.Add(c); err != nil {
			t.Fatalf("Add failed at idx %d: %v", i, err)
		}
	}

	upper, mid, lower, err := bb.Calculate()
	if err != nil {
		t.Fatalf("Calculate returned error: %v", err)
	}

	if upper != 16 || mid != 12 || lower != 8 {
		t.Fatalf("unexpected bands: upper %.2f, mid %.2f, lower %.2f (want 16,12,8)", upper, mid, lower)
	}
}

func TestBollingerBands_InvalidPrice(t *testing.T) {
	bb, _ := NewBollingerBandsWithParams(3, 2)
	if err := bb.Add(-1); err == nil {
		t.Fatal("expected error for negative price")
	}
}
