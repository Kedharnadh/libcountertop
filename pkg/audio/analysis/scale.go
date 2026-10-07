package analysis

import (
	"math"
)

// Scale multiplies samples in place, clamping rather than wrapping.
func Scale(samples []int16, by float32) {
	if by == 1 {
		return
	}

	for i, v := range samples {
		s := float32(v) * by
		switch {
		case s > math.MaxInt16:
			samples[i] = math.MaxInt16
		case s < math.MinInt16:
			samples[i] = math.MinInt16
		default:
			samples[i] = int16(s)
		}
	}
}
