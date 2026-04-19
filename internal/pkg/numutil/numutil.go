package numutil

import "math"

func AbsInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func AbsFloat(v float64) float64 {
	return math.Abs(v)
}
