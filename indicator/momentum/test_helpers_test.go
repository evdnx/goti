package momentum

import "math"

func approxEqual(a, b float64) bool {
	const eps = 1e-6
	return math.Abs(a-b) <= eps
}
