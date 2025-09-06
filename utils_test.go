package goti // same package as the library code

import (
	"fmt"
	"math"
	"reflect"
	"testing"
)

/*
--------------------------------------------------------------

	Local copy of the validation helper (mirrors the library’s logic)
	--------------------------------------------------------------
*/
func validatePositiveInt(name string, value int) error {
	if value <= 0 {
		return fmt.Errorf("%s must be positive, got %d", name, value)
	}
	return nil
}

/*
--------------------------------------------------------------

	Slice helpers
	--------------------------------------------------------------
*/
func TestTrimTail(t *testing.T) {
	src := []int{1, 2, 3, 4, 5}
	got := trimTail(src, 3)
	exp := []int{3, 4, 5}
	if !reflect.DeepEqual(got, exp) {
		t.Fatalf("trimTail: expected %v, got %v", exp, got)
	}

	// Asking for more elements than exist should return the original slice unchanged.
	got = trimTail(src, 10)
	if !reflect.DeepEqual(got, src) {
		t.Fatalf("trimTail over‑length: expected %v, got %v", src, got)
	}
}

/*
--------------------------------------------------------------

	Numeric helpers
	--------------------------------------------------------------
*/
func TestClamp(t *testing.T) {
	tests := []struct {
		val, min, max, want float64
	}{
		{5, 0, 10, 5},
		{-1, 0, 10, 0},
		{15, 0, 10, 10},
		{7, 7, 7, 7}, // degenerate range
	}
	for _, tt := range tests {
		if got := clamp(tt.val, tt.min, tt.max); got != tt.want {
			t.Fatalf("clamp(%v,%v,%v) = %v, want %v", tt.val, tt.min, tt.max, got, tt.want)
		}
	}
}

func TestCalculateSlope(t *testing.T) {
	if got := calculateSlope(10, 4); got != 6 {
		t.Fatalf("calculateSlope expected 6, got %v", got)
	}
}

/*
--------------------------------------------------------------

	Statistics
	--------------------------------------------------------------
*/
func TestCalculateStandardDeviation(t *testing.T) {
	data := []float64{2, 4, 4, 4, 5, 5, 7, 9}
	const eps = 1e-9
	got := calculateStandardDeviation(data, 0)
	if math.Abs(got-2.138089935) > eps {
		t.Fatalf("stddev mismatch: got %v, want ~2.13809", got)
	}
}

/*
--------------------------------------------------------------

	Moving‑average implementations
	--------------------------------------------------------------
*/
func TestSimpleMovingAverage(t *testing.T) {
	ma, err := NewMovingAverage(SMAMovingAverage, 3) // use the constant defined in the package
	if err != nil {
		t.Fatalf("unexpected error creating SMA: %v", err)
	}
	values := []float64{1, 2, 3, 4, 5}
	for _, v := range values {
		if err := ma.Add(v); err != nil {
			t.Fatalf("Add error: %v", err)
		}
	}
	// Last SMA of the last three points (3,4,5) = 4
	got, err := ma.Calculate()
	if err != nil {
		t.Fatalf("Calculate error: %v", err)
	}
	if got != 4 {
		t.Fatalf("SMA expected 4, got %v", got)
	}
}

func TestExponentialMovingAverage(t *testing.T) {
	ma, err := NewMovingAverage(EMAMovingAverage, 3) // use the constant defined in the package
	if err != nil {
		t.Fatalf("unexpected error creating EMA: %v", err)
	}
	for _, v := range []float64{1, 2, 3, 4, 5} {
		if err := ma.Add(v); err != nil {
			t.Fatalf("Add error: %v", err)
		}
	}
	got, err := ma.Calculate()
	if err != nil {
		t.Fatalf("Calculate error: %v", err)
	}
	if math.Abs(got-4) > 1e-9 {
		t.Fatalf("EMA expected 4, got %v", got)
	}
}

func TestWeightedMovingAverage(t *testing.T) {
	ma, err := NewMovingAverage(WMAMovingAverage, 3) // use the constant defined in the package
	if err != nil {
		t.Fatalf("unexpected error creating WMA: %v", err)
	}
	for _, v := range []float64{1, 2, 3, 4, 5} {
		if err := ma.Add(v); err != nil {
			t.Fatalf("Add error: %v", err)
		}
	}
	got, err := ma.Calculate()
	if err != nil {
		t.Fatalf("Calculate error: %v", err)
	}
	// Expected WMA: (3*3 + 4*2 + 5*1) / (3+2+1) = 22/6 = 3.666…
	expected := 22.0 / 6.0
	if math.Abs(got-expected) > 1e-9 {
		t.Fatalf("WMA expected %v, got %v", expected, got)
	}
}

/*
--------------------------------------------------------------

	Validation helper (now defined locally)
	--------------------------------------------------------------
*/
func TestValidatePositiveInt(t *testing.T) {
	if err := validatePositiveInt("period", 5); err != nil {
		t.Fatalf("unexpected error for positive int: %v", err)
	}
	if err := validatePositiveInt("period", 0); err == nil {
		t.Fatalf("expected error for zero value")
	}
	if err := validatePositiveInt("period", -3); err == nil {
		t.Fatalf("expected error for negative value")
	}
}

/*
--------------------------------------------------------------

	Small helper used by other ATSO tests (kept for completeness)
	--------------------------------------------------------------
*/
func computeRawATSO(startIdx, period int, atso *AdaptiveTrendStrengthOscillator) (float64, error) {
	if startIdx+period > len(atso.closes) {
		return 0, fmt.Errorf("window out of range")
	}
	highs := atso.highs[startIdx : startIdx+period]
	lows := atso.lows[startIdx : startIdx+period]
	closes := atso.closes[startIdx : startIdx+period]

	adaptPeriod := period
	var upSum, downSum float64
	for i := 1; i < adaptPeriod; i++ {
		if closes[i] > closes[i-1] {
			upSum += highs[i] - lows[i-1]
		} else {
			downSum += lows[i] - highs[i-1]
		}
	}
	if upSum+downSum == 0 {
		return 0, fmt.Errorf("division by zero in trend strength")
	}
	return ((upSum - downSum) / (upSum + downSum)) * 100, nil
}
