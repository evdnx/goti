package momentum

import "testing"

func TestStochasticOscillator_Calculation(t *testing.T) {
	stoch, err := NewStochasticOscillatorWithParams(3, 2)
	if err != nil {
		t.Fatalf("constructor error: %v", err)
	}

	data := []struct {
		h, l, c float64
	}{
		{10, 5, 7},
		{12, 6, 11},
		{14, 5, 13}, // first %K
		{15, 9, 10}, // second %K and first %D
	}

	for i, d := range data {
		if err := stoch.Add(d.h, d.l, d.c); err != nil {
			t.Fatalf("Add failed at idx %d: %v", i, err)
		}
	}

	k, d, err := stoch.Calculate()
	if err != nil {
		t.Fatalf("Calculate returned error: %v", err)
	}

	// Expected values based on the manual calculation in the comment above.
	// After the third bar: %K ≈ 88.8889
	// After the fourth bar: %K ≈ 50, %D = average(88.8889, 50) ≈ 69.4444
	if !approxEqual(k, 50) {
		t.Fatalf("unexpected %%K: got %.6f, want 50", k)
	}
	if !approxEqual(d, 69.444444) {
		t.Fatalf("unexpected %%D: got %.6f, want ~69.4444", d)
	}
}

func TestStochasticOscillator_OverboughtOversold(t *testing.T) {
	stoch, err := NewStochasticOscillatorWithParams(2, 2)
	if err != nil {
		t.Fatalf("constructor error: %v", err)
	}

	// First two bars produce a %K of 100 (overbought).
	if err := stoch.Add(10, 9, 10); err != nil {
		t.Fatalf("Add 1 failed: %v", err)
	}
	if err := stoch.Add(12, 9, 12); err != nil {
		t.Fatalf("Add 2 failed: %v", err)
	}
	ob, err := stoch.IsOverbought()
	if err != nil {
		t.Fatalf("IsOverbought error: %v", err)
	}
	if !ob {
		t.Fatal("expected overbought after strong up move")
	}

	// A sharp drop should push %K to 0 (oversold).
	if err := stoch.Add(12, 8, 8); err != nil {
		t.Fatalf("Add 3 failed: %v", err)
	}
	os, err := stoch.IsOversold()
	if err != nil {
		t.Fatalf("IsOversold error: %v", err)
	}
	if !os {
		t.Fatal("expected oversold after drop")
	}
}
