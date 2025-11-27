package goti

import "github.com/evdnx/goti/indicator"

// Unexported helpers bridged to the indicator package so existing tests keep working.
func clamp(value, min, max float64) float64 {
	return indicator.Clamp(value, min, max)
}

func calculateSlope(y2, y1 float64) float64 {
	return indicator.CalculateSlope(y2, y1)
}

func calculateStandardDeviation(data []float64, mean float64) float64 {
	return indicator.CalculateStandardDeviation(data, mean)
}

func calculateEMA(data []float64, period int, prevEMA float64) (float64, error) {
	return indicator.CalculateEMA(data, period, prevEMA)
}

func calculateWMA(data []float64, period int) (float64, error) {
	return indicator.CalculateWMA(data, period)
}

func keepLast[T any](s []T, n int) []T {
	return indicator.KeepLast(s, n)
}

func isValidPrice(price float64) bool { return indicator.IsValidPrice(price) }

func isNonNegativePrice(price float64) bool { return indicator.IsNonNegativePrice(price) }

func isValidVolume(volume float64) bool { return indicator.IsValidVolume(volume) }
